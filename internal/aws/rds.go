package aws

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	ssmservice "github.com/aws/aws-sdk-go-v2/service/ssm"
)

type RDSManager struct {
	rdsClient *rds.Client
	ec2Client *ec2.Client
	ssmClient *ssmservice.Client
	region    string
}

type RDSInstance struct {
	Identifier string
	Endpoint   string
	Port       int32
	Engine     string
}

type BastionHost struct {
	InstanceId       string
	Name             string
	SecurityGroupIds []string
}

func NewRDSManager(ctx context.Context) (*RDSManager, error) {
	cfg, err := LoadSWAConfig(ctx)
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

func (r *RDSManager) ListRDSInstances(ctx context.Context) ([]RDSInstance, error) {
	result, err := r.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, err
	}

	var instances []RDSInstance
	for _, db := range result.DBInstances {
		if db.DBInstanceStatus != nil && *db.DBInstanceStatus == "available" {
			instances = append(instances, RDSInstance{
				Identifier: *db.DBInstanceIdentifier,
				Endpoint:   *db.Endpoint.Address,
				Port:       *db.Endpoint.Port,
				Engine:     *db.Engine,
			})
		}
	}

	return instances, nil
}

func (r *RDSManager) FindBastionHosts(ctx context.Context, rdsInstance RDSInstance) ([]BastionHost, error) {
	// Get RDS security groups
	rdsSecurityGroups, err := r.getRDSSecurityGroups(ctx, rdsInstance.Identifier)
	if err != nil {
		return nil, err
	}

	// Find EC2 instances that can connect to RDS
	result, err := r.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var bastions []BastionHost
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if r.canConnectToRDS(ctx, instance.SecurityGroups, rdsSecurityGroups, rdsInstance.Port) {
				name := r.getInstanceName(instance.Tags)
				bastions = append(bastions, BastionHost{
					InstanceId:       *instance.InstanceId,
					Name:             name,
					SecurityGroupIds: r.getSecurityGroupIds(instance.SecurityGroups),
				})
			}
		}
	}

	return bastions, nil
}

func (r *RDSManager) StartPortForwarding(ctx context.Context, bastionId, rdsEndpoint string, rdsPort, localPort int32) error {
	// Create port forwarder
	cfg, err := LoadSWAConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	pf := NewExternalPluginForwarder(cfg)
	
	fmt.Printf("Starting port forwarding via %s...\n", bastionId)
	
	// Start port forwarding to remote host through bastion
	err = pf.StartPortForwardingToRemoteHost(ctx, bastionId, rdsEndpoint, int(rdsPort), int(localPort))
	if err != nil {
		fmt.Printf("Port forwarding failed: %v\n", err)
		return r.fallbackToCommand(bastionId, rdsEndpoint, rdsPort, localPort)
	}
	return nil
}

func (r *RDSManager) fallbackToCommand(bastionId, rdsEndpoint string, rdsPort, localPort int32) error {
	fmt.Printf("\nWould you like to run the command manually? (y/N): ")
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" {
		parameters := fmt.Sprintf(`{"host":["%s"],"portNumber":["%d"],"localPortNumber":["%d"]}`, 
			rdsEndpoint, rdsPort, localPort)
		
		fmt.Printf("\nRun this command:\n\n")
		fmt.Printf("aws ssm start-session --target %s --document-name AWS-StartPortForwardingSessionToRemoteHost --parameters '%s' --region %s\n\n", 
			bastionId, parameters, r.region)
	}
	
	return nil
}

func (r *RDSManager) getRDSSecurityGroups(ctx context.Context, dbIdentifier string) ([]string, error) {
	result, err := r.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbIdentifier),
	})
	if err != nil {
		return nil, err
	}

	if len(result.DBInstances) == 0 {
		return nil, fmt.Errorf("RDS instance not found")
	}

	var sgIds []string
	for _, sg := range result.DBInstances[0].VpcSecurityGroups {
		sgIds = append(sgIds, *sg.VpcSecurityGroupId)
	}

	return sgIds, nil
}

func (r *RDSManager) canConnectToRDS(ctx context.Context, ec2SecurityGroups []types.GroupIdentifier, rdsSecurityGroups []string, rdsPort int32) bool {
	ec2SgIds := make(map[string]bool)
	for _, sg := range ec2SecurityGroups {
		ec2SgIds[*sg.GroupId] = true
	}

	// Check if any RDS security group allows inbound from EC2 security groups
	for _, rdsSgId := range rdsSecurityGroups {
		if r.checkSecurityGroupRules(ctx, rdsSgId, ec2SgIds, rdsPort) {
			return true
		}
	}

	return false
}

func (r *RDSManager) checkSecurityGroupRules(ctx context.Context, rdsSgId string, ec2SgIds map[string]bool, port int32) bool {
	result, err := r.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{rdsSgId},
	})
	if err != nil {
		return false
	}

	if len(result.SecurityGroups) == 0 {
		return false
	}

	for _, rule := range result.SecurityGroups[0].IpPermissions {
		if r.ruleMatchesPort(rule, port) {
			// Check if rule allows access from EC2 security groups
			for _, userIdGroupPair := range rule.UserIdGroupPairs {
				if userIdGroupPair.GroupId != nil && ec2SgIds[*userIdGroupPair.GroupId] {
					return true
				}
			}
			// Check for open access (0.0.0.0/0)
			for _, ipRange := range rule.IpRanges {
				if ipRange.CidrIp != nil && *ipRange.CidrIp == "0.0.0.0/0" {
					return true
				}
			}
		}
	}

	return false
}

func (r *RDSManager) ruleMatchesPort(rule types.IpPermission, port int32) bool {
	if rule.FromPort == nil || rule.ToPort == nil {
		return false
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
		ids = append(ids, *sg.GroupId)
	}
	return ids
}