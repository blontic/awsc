package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	ssmservice "github.com/aws/aws-sdk-go-v2/service/ssm"
	awscconfig "github.com/blontic/awsc/internal/config"
	"github.com/blontic/awsc/internal/debug"
	"github.com/blontic/awsc/internal/ui"
)

// RDSClient interface for mocking
type RDSClient interface {
	DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
	DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)
}

// EC2Client interface for mocking
type EC2Client interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
}

type RDSManager struct {
	rdsClient RDSClient
	ec2Client EC2Client
	ssmClient *ssmservice.Client
	region    string
}

type RDSInstance struct {
	Identifier   string
	Endpoint     string
	Port         int32
	Engine       string
	EndpointType string // "instance", "cluster-writer", "cluster-reader"
	ClusterName  string // For cluster endpoints
}

type BastionHost struct {
	InstanceId       string
	Name             string
	SecurityGroupIds []string
}

type RDSManagerOptions struct {
	RDSClient RDSClient
	EC2Client EC2Client
	SSMClient *ssmservice.Client
	Region    string
}

func NewRDSManager(ctx context.Context, opts ...RDSManagerOptions) (*RDSManager, error) {
	if len(opts) > 0 && opts[0].RDSClient != nil {
		// Use provided clients (for testing)
		return &RDSManager{
			rdsClient: opts[0].RDSClient,
			ec2Client: opts[0].EC2Client,
			ssmClient: opts[0].SSMClient,
			region:    opts[0].Region,
		}, nil
	}

	// Production path
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return nil, err
	}

	return &RDSManager{
		rdsClient: rds.NewFromConfig(cfg),
		ec2Client: ec2.NewFromConfig(cfg),
		ssmClient: ssmservice.NewFromConfig(cfg),
		region:    cfg.Region,
	}, nil
}

func (r *RDSManager) RunConnect(ctx context.Context, instanceName string, localPort int32, listBastions bool) error {
	// List RDS instances
	instances, err := r.ListRDSInstances(ctx)
	if err != nil {
		return fmt.Errorf("error listing RDS instances: %v", err)
	}

	if len(instances) == 0 {
		return fmt.Errorf("no RDS instances found")
	}

	var selectedInstance RDSInstance

	// If instance name provided, try to connect directly
	if instanceName != "" {
		var targetInstance *RDSInstance
		for _, instance := range instances {
			if instance.Identifier == instanceName {
				targetInstance = &instance
				break
			}
		}

		if targetInstance != nil {
			fmt.Printf("Connecting to RDS instance: %s\n", targetInstance.Identifier)
			selectedInstance = *targetInstance
		} else {
			return fmt.Errorf("RDS instance '%s' not found", instanceName)
		}
	}

	// If no instance name provided, show interactive selection
	if instanceName == "" {
		// Create instance options for selection
		instanceOptions := make([]string, len(instances))
		for i, instance := range instances {
			switch instance.EndpointType {
			case "cluster-writer":
				instanceOptions[i] = fmt.Sprintf("%s (%s:%d) [Writer]", instance.Identifier, instance.Engine, instance.Port)
			case "cluster-reader":
				instanceOptions[i] = fmt.Sprintf("%s (%s:%d) [Reader]", instance.Identifier, instance.Engine, instance.Port)
			default:
				instanceOptions[i] = fmt.Sprintf("%s (%s:%d)", instance.Identifier, instance.Engine, instance.Port)
			}
		}

		// Interactive instance selection
		selectedIndex, err := ui.RunSelector("Select RDS Instance:", instanceOptions)
		if err != nil {
			return fmt.Errorf("error selecting instance: %v", err)
		}
		if selectedIndex == -1 {
			return fmt.Errorf("no instance selected")
		}

		selectedInstance = instances[selectedIndex]
		fmt.Printf("✓ Selected: %s\n", selectedInstance.Identifier)
	} else {
		fmt.Printf("✓ Selected: %s\n", selectedInstance.Identifier)
	}

	// Find bastion hosts
	bastions, err := r.FindBastionHosts(ctx, selectedInstance, listBastions)
	if err != nil {
		return err
	}

	if len(bastions) == 0 {
		return fmt.Errorf("no bastion hosts available for %s", selectedInstance.Identifier)
	}

	var bastion BastionHost
	if listBastions && len(bastions) > 1 {
		bastionOptions := make([]string, len(bastions))
		for i, b := range bastions {
			bastionOptions[i] = fmt.Sprintf("%s (%s)", b.Name, b.InstanceId)
		}
		selectedIndex, err := ui.RunSelector("Select Bastion Host:", bastionOptions)
		if err != nil {
			return fmt.Errorf("error selecting bastion: %v", err)
		}
		if selectedIndex == -1 {
			return fmt.Errorf("no bastion selected")
		}
		bastion = bastions[selectedIndex]
		fmt.Printf("✓ Selected bastion: %s (%s)\n", bastion.Name, bastion.InstanceId)
	} else {
		bastion = bastions[0]
		fmt.Printf("Using bastion: %s (%s)\n", bastion.Name, bastion.InstanceId)
	}

	// Use default local port if not specified
	if localPort == 0 {
		localPort = selectedInstance.Port
	}

	// Start port forwarding
	return r.StartPortForwarding(ctx, bastion.InstanceId, selectedInstance.Endpoint, selectedInstance.Port, localPort)
}

func (r *RDSManager) ListRDSInstances(ctx context.Context) ([]RDSInstance, error) {
	var instances []RDSInstance

	// Get standalone DB instances
	dbInstances, err := r.getDBInstances(ctx)
	if err != nil {
		return nil, err
	}
	instances = append(instances, dbInstances...)

	// Get Aurora cluster endpoints
	clusterEndpoints, err := r.getClusterEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	instances = append(instances, clusterEndpoints...)

	return instances, nil
}

func (r *RDSManager) getDBInstances(ctx context.Context) ([]RDSInstance, error) {
	var allDBInstances []rdstypes.DBInstance
	var marker *string

	for {
		result, err := r.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			Marker: marker,
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					if reloadErr := r.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					result, err = r.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
						Marker: marker,
					})
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		allDBInstances = append(allDBInstances, result.DBInstances...)

		if result.Marker == nil {
			break
		}
		marker = result.Marker
	}

	var instances []RDSInstance
	for _, db := range allDBInstances {
		if db.DBInstanceStatus != nil && *db.DBInstanceStatus == "available" && db.DBClusterIdentifier == nil {
			// Only include standalone instances (not part of a cluster).
			// Endpoint can be nil for instances that report "available" but
			// have no resolvable endpoint yet; skip those rather than panic.
			if db.Endpoint == nil || db.Endpoint.Address == nil || db.Endpoint.Port == nil {
				continue
			}
			instances = append(instances, RDSInstance{
				Identifier:   aws.ToString(db.DBInstanceIdentifier),
				Endpoint:     aws.ToString(db.Endpoint.Address),
				Port:         aws.ToInt32(db.Endpoint.Port),
				Engine:       aws.ToString(db.Engine),
				EndpointType: "instance",
			})
		}
	}

	return instances, nil
}

func (r *RDSManager) getClusterEndpoints(ctx context.Context) ([]RDSInstance, error) {
	var allClusters []rdstypes.DBCluster
	var marker *string

	for {
		result, err := r.rdsClient.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
			Marker: marker,
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					if reloadErr := r.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					result, err = r.rdsClient.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
						Marker: marker,
					})
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		allClusters = append(allClusters, result.DBClusters...)

		if result.Marker == nil {
			break
		}
		marker = result.Marker
	}

	var instances []RDSInstance
	for _, cluster := range allClusters {
		if cluster.Status != nil && *cluster.Status == "available" {
			clusterID := aws.ToString(cluster.DBClusterIdentifier)
			// Add cluster writer endpoint
			if cluster.Endpoint != nil {
				instances = append(instances, RDSInstance{
					Identifier:   clusterID + " (writer)",
					Endpoint:     aws.ToString(cluster.Endpoint),
					Port:         aws.ToInt32(cluster.Port),
					Engine:       aws.ToString(cluster.Engine),
					EndpointType: "cluster-writer",
					ClusterName:  clusterID,
				})
			}

			// Add cluster reader endpoint
			if cluster.ReaderEndpoint != nil {
				instances = append(instances, RDSInstance{
					Identifier:   clusterID + " (reader)",
					Endpoint:     aws.ToString(cluster.ReaderEndpoint),
					Port:         aws.ToInt32(cluster.Port),
					Engine:       aws.ToString(cluster.Engine),
					EndpointType: "cluster-reader",
					ClusterName:  clusterID,
				})
			}
		}
	}

	return instances, nil
}

func (r *RDSManager) FindBastionHosts(ctx context.Context, rdsInstance RDSInstance, findAll bool) ([]BastionHost, error) {
	// Get RDS security groups
	rdsSecurityGroups, err := r.getRDSSecurityGroups(ctx, rdsInstance)
	if err != nil {
		return nil, err
	}

	debug.Printf("RDS %s security groups: %v\n", rdsInstance.Identifier, rdsSecurityGroups)

	// Pre-fetch all SG inbound rules once (avoids repeated API calls per instance)
	sgRulesCache, err := r.fetchSecurityGroupRules(ctx, rdsSecurityGroups)
	if err != nil {
		return nil, err
	}

	// Find EC2 instances that can connect to RDS
	var allReservations []types.Reservation
	var nextToken *string

	for {
		result, err := r.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			NextToken: nextToken,
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					if reloadErr := r.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					result, err = r.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
						NextToken: nextToken,
					})
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		allReservations = append(allReservations, result.Reservations...)

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Count and categorize instances
	runningInstances := 0
	stoppedInstances := 0
	var stoppedInstanceNames []string

	for _, reservation := range allReservations {
		for _, instance := range reservation.Instances {
			if instance.State != nil {
				if instance.State.Name == "running" {
					runningInstances++
				} else if instance.State.Name == "stopped" {
					stoppedInstances++
					stoppedInstanceNames = append(stoppedInstanceNames, r.getInstanceName(instance.Tags))
				}
			}
		}
	}
	debug.Printf("Found %d running and %d stopped EC2 instances\n", runningInstances, stoppedInstances)

	// Check running instances
	var bastions []BastionHost
	for _, reservation := range allReservations {
		for _, instance := range reservation.Instances {
			if instance.State == nil || instance.State.Name != "running" {
				continue
			}

			name := r.getInstanceName(instance.Tags)
			ec2SgIds := r.getSecurityGroupIds(instance.SecurityGroups)
			debug.Printf("Checking if EC2 instance %s (%s) can reach RDS — EC2 security groups: %v\n", name, aws.ToString(instance.InstanceId), ec2SgIds)

			if r.canConnectWithCachedRules(instance.SecurityGroups, sgRulesCache, rdsInstance.Port) {
				debug.Printf("✓ EC2 instance %s can connect to RDS %s\n", name, rdsInstance.Identifier)
				bastion := BastionHost{
					InstanceId:       aws.ToString(instance.InstanceId),
					Name:             name,
					SecurityGroupIds: ec2SgIds,
				}
				if !findAll {
					return []BastionHost{bastion}, nil
				}
				bastions = append(bastions, bastion)
			} else {
				debug.Printf("✗ EC2 instance %s cannot connect to RDS %s\n", name, rdsInstance.Identifier)
			}
		}
	}

	if len(bastions) > 0 {
		return bastions, nil
	}

	// No bastion found - show helpful error
	if stoppedInstances > 0 {
		fmt.Printf("\nFound %d stopped EC2 instance(s):\n", stoppedInstances)
		for _, name := range stoppedInstanceNames {
			fmt.Printf("- %s (stopped)\n", name)
		}
		fmt.Printf("\n")
	}

	if runningInstances == 0 {
		fmt.Printf("No running EC2 instances found in region %s.\n", r.region)
		fmt.Printf("To use RDS port forwarding, you need a running EC2 instance with:\n")
		fmt.Printf("- SSM agent installed and configured\n")
		fmt.Printf("- Network access to the RDS instance\n")
		if stoppedInstances > 0 {
			fmt.Printf("\nYou can start one of the stopped instances above and try again.\n")
			return nil, fmt.Errorf("no running bastion hosts found - %d stopped instances available", stoppedInstances)
		}
		fmt.Printf("\nAlternatively, you can connect directly if your RDS is publicly accessible.\n")
		return nil, fmt.Errorf("no running EC2 instances found in region %s", r.region)
	}

	fmt.Printf("Found %d running EC2 instances but none can connect to RDS %s.\n", runningInstances, rdsInstance.Identifier)
	fmt.Printf("This usually means the security groups don't allow the connection.\n")
	return nil, fmt.Errorf("no suitable bastion hosts found - security groups may not allow connection")
}

func (r *RDSManager) StartPortForwarding(ctx context.Context, bastionId, rdsEndpoint string, rdsPort, localPort int32) error {
	// Create port forwarder
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	pf := NewExternalPluginForwarder(cfg)

	fmt.Printf("Starting port forwarding...\n")

	// Start port forwarding to remote host through bastion
	return pf.StartPortForwardingToRemoteHost(ctx, bastionId, rdsEndpoint, int(rdsPort), int(localPort))
}

func (r *RDSManager) getRDSSecurityGroups(ctx context.Context, rdsInstance RDSInstance) ([]string, error) {
	if rdsInstance.EndpointType == "cluster-writer" || rdsInstance.EndpointType == "cluster-reader" {
		// Get security groups from cluster
		result, err := r.rdsClient.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(rdsInstance.ClusterName),
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					if reloadErr := r.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					result, err = r.rdsClient.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
						DBClusterIdentifier: aws.String(rdsInstance.ClusterName),
					})
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		if len(result.DBClusters) == 0 {
			return nil, fmt.Errorf("RDS cluster not found")
		}

		var sgIds []string
		for _, sg := range result.DBClusters[0].VpcSecurityGroups {
			if sg.VpcSecurityGroupId != nil {
				sgIds = append(sgIds, *sg.VpcSecurityGroupId)
			}
		}
		return sgIds, nil
	} else {
		// Get security groups from instance
		result, err := r.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(rdsInstance.Identifier),
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					if reloadErr := r.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					result, err = r.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
						DBInstanceIdentifier: aws.String(rdsInstance.Identifier),
					})
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		if len(result.DBInstances) == 0 {
			return nil, fmt.Errorf("RDS instance not found")
		}

		var sgIds []string
		for _, sg := range result.DBInstances[0].VpcSecurityGroups {
			if sg.VpcSecurityGroupId != nil {
				sgIds = append(sgIds, *sg.VpcSecurityGroupId)
			}
		}
		return sgIds, nil
	}
}

func (r *RDSManager) fetchSecurityGroupRules(ctx context.Context, sgIds []string) (map[string][]types.IpPermission, error) {
	cache := make(map[string][]types.IpPermission, len(sgIds))
	for _, sgId := range sgIds {
		result, err := r.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{sgId},
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					if reloadErr := r.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					result, err = r.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
						GroupIds: []string{sgId},
					})
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
		if len(result.SecurityGroups) > 0 {
			cache[sgId] = result.SecurityGroups[0].IpPermissions
		}
	}
	return cache, nil
}

func (r *RDSManager) canConnectWithCachedRules(ec2SecurityGroups []types.GroupIdentifier, sgRulesCache map[string][]types.IpPermission, port int32) bool {
	ec2SgIds := make(map[string]bool)
	for _, sg := range ec2SecurityGroups {
		if sg.GroupId != nil {
			ec2SgIds[*sg.GroupId] = true
		}
	}

	for sgId, rules := range sgRulesCache {
		debug.Printf("  Checking RDS security group %s inbound rules for port %d\n", sgId, port)
		for _, rule := range rules {
			if r.ruleMatchesPort(rule, port) {
				debug.Printf("    Rule allows port %d\n", port)
				for _, pair := range rule.UserIdGroupPairs {
					if pair.GroupId != nil {
						var ec2SgList []string
						for id := range ec2SgIds {
							ec2SgList = append(ec2SgList, id)
						}
						if ec2SgIds[*pair.GroupId] {
							debug.Printf("      RDS SG allows %s → EC2 has %s ✓ match!\n", *pair.GroupId, strings.Join(ec2SgList, ", "))
							return true
						}
						debug.Printf("      RDS SG allows %s → EC2 has %s — no match\n", *pair.GroupId, strings.Join(ec2SgList, ", "))
					}
				}
				for _, ipRange := range rule.IpRanges {
					if ipRange.CidrIp != nil {
						if *ipRange.CidrIp == "0.0.0.0/0" {
							debug.Printf("      RDS SG allows 0.0.0.0/0 (open access) ✓\n")
							return true
						}
						debug.Printf("      RDS SG allows CIDR %s — not matched (only SG-to-SG supported)\n", *ipRange.CidrIp)
					}
				}
			} else {
				if rule.FromPort != nil && rule.ToPort != nil {
					debug.Printf("    Rule for port range %d-%d does not cover port %d\n", *rule.FromPort, *rule.ToPort, port)
				} else {
					debug.Printf("    Rule does not cover port %d\n", port)
				}
			}
		}
	}
	return false
}

func (r *RDSManager) ruleMatchesPort(rule types.IpPermission, port int32) bool {
	// All-traffic rules (protocol -1) have nil ports and match everything
	if rule.FromPort == nil || rule.ToPort == nil {
		return rule.IpProtocol != nil && *rule.IpProtocol == "-1"
	}
	return *rule.FromPort <= port && port <= *rule.ToPort
}

func (r *RDSManager) getInstanceName(tags []types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	return "Unnamed"
}

func (r *RDSManager) getSecurityGroupIds(sgs []types.GroupIdentifier) []string {
	var ids []string
	for _, sg := range sgs {
		if sg.GroupId != nil {
			ids = append(ids, *sg.GroupId)
		}
	}
	return ids
}

func (r *RDSManager) reloadClients(ctx context.Context) error {
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return err
	}

	r.rdsClient = rds.NewFromConfig(cfg)
	r.ec2Client = ec2.NewFromConfig(cfg)
	r.ssmClient = ssmservice.NewFromConfig(cfg)
	r.region = cfg.Region

	return nil
}
