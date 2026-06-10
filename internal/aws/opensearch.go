package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	ssmservice "github.com/aws/aws-sdk-go-v2/service/ssm"
	awscconfig "github.com/blontic/awsc/internal/config"
	"github.com/blontic/awsc/internal/debug"
	"github.com/blontic/awsc/internal/ui"
)

// OpenSearchClient interface for mocking
type OpenSearchClient interface {
	ListDomainNames(ctx context.Context, params *opensearch.ListDomainNamesInput, optFns ...func(*opensearch.Options)) (*opensearch.ListDomainNamesOutput, error)
	DescribeDomain(ctx context.Context, params *opensearch.DescribeDomainInput, optFns ...func(*opensearch.Options)) (*opensearch.DescribeDomainOutput, error)
}

type OpenSearchManager struct {
	opensearchClient OpenSearchClient
	ec2Client        EC2Client
	ssmClient        *ssmservice.Client
	region           string
}

type OpenSearchDomain struct {
	Name     string
	Endpoint string
	Port     int32
	Version  string
}

type OpenSearchManagerOptions struct {
	OpenSearchClient OpenSearchClient
	EC2Client        EC2Client
	SSMClient        *ssmservice.Client
	Region           string
}

func NewOpenSearchManager(ctx context.Context, opts ...OpenSearchManagerOptions) (*OpenSearchManager, error) {
	if len(opts) > 0 && opts[0].OpenSearchClient != nil {
		// Use provided clients (for testing)
		return &OpenSearchManager{
			opensearchClient: opts[0].OpenSearchClient,
			ec2Client:        opts[0].EC2Client,
			ssmClient:        opts[0].SSMClient,
			region:           opts[0].Region,
		}, nil
	}

	// Production path
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return nil, err
	}

	return &OpenSearchManager{
		opensearchClient: opensearch.NewFromConfig(cfg),
		ec2Client:        ec2.NewFromConfig(cfg),
		ssmClient:        ssmservice.NewFromConfig(cfg),
		region:           cfg.Region,
	}, nil
}

func (o *OpenSearchManager) RunConnect(ctx context.Context, domainName string, localPort int32, listBastions bool) error {
	// List OpenSearch domains
	domains, err := o.ListOpenSearchDomains(ctx)
	if err != nil {
		return fmt.Errorf("error listing OpenSearch domains: %v", err)
	}

	if len(domains) == 0 {
		return fmt.Errorf("no OpenSearch domains found")
	}

	var selectedDomain OpenSearchDomain

	// If domain name provided, try to connect directly
	if domainName != "" {
		var targetDomain *OpenSearchDomain
		for _, domain := range domains {
			if domain.Name == domainName {
				targetDomain = &domain
				break
			}
		}

		if targetDomain != nil {
			fmt.Printf("Connecting to OpenSearch domain: %s\n", targetDomain.Name)
			selectedDomain = *targetDomain
		} else {
			return fmt.Errorf("OpenSearch domain '%s' not found", domainName)
		}
	}

	// If no domain name provided, show interactive selection
	if domainName == "" {
		// Create domain options for selection
		domainOptions := make([]string, len(domains))
		for i, domain := range domains {
			domainOptions[i] = fmt.Sprintf("%s (%s)", domain.Name, domain.Version)
		}

		// Interactive domain selection
		selectedIndex, err := ui.RunSelector("Select OpenSearch Domain:", domainOptions)
		if err != nil {
			return fmt.Errorf("error selecting domain: %v", err)
		}
		if selectedIndex == -1 {
			return fmt.Errorf("no domain selected")
		}

		selectedDomain = domains[selectedIndex]
		fmt.Printf("✓ Selected: %s\n", selectedDomain.Name)
	} else {
		fmt.Printf("✓ Selected: %s\n", selectedDomain.Name)
	}

	// Find bastion hosts
	bastions, err := o.FindBastionHosts(ctx, selectedDomain, listBastions)
	if err != nil {
		return err
	}

	if len(bastions) == 0 {
		return fmt.Errorf("no bastion hosts available for %s", selectedDomain.Name)
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

	// Start port forwarding
	return o.StartPortForwarding(ctx, bastion.InstanceId, selectedDomain.Endpoint, selectedDomain.Port, localPort)
}

func (o *OpenSearchManager) ListOpenSearchDomains(ctx context.Context) ([]OpenSearchDomain, error) {
	// List domain names
	result, err := o.opensearchClient.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
	if err != nil {
		if IsAuthError(err) {
			if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
				if reloadErr := o.reloadClients(ctx); reloadErr != nil {
					return nil, reloadErr
				}
				result, err = o.opensearchClient.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
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

	var domains []OpenSearchDomain
	for _, domainInfo := range result.DomainNames {
		if domainInfo.DomainName == nil {
			continue
		}

		// Get domain details
		domainDetail, err := o.opensearchClient.DescribeDomain(ctx, &opensearch.DescribeDomainInput{
			DomainName: domainInfo.DomainName,
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					if reloadErr := o.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					domainDetail, err = o.opensearchClient.DescribeDomain(ctx, &opensearch.DescribeDomainInput{
						DomainName: domainInfo.DomainName,
					})
					if err != nil {
						debug.Printf("Error describing domain %s: %v\n", *domainInfo.DomainName, err)
						continue
					}
				} else {
					return nil, err
				}
			} else {
				debug.Printf("Error describing domain %s: %v\n", *domainInfo.DomainName, err)
				continue
			}
		}

		domain := domainDetail.DomainStatus
		if domain == nil || domain.DomainEndpointOptions == nil || domain.DomainEndpointOptions.EnforceHTTPS == nil || !*domain.DomainEndpointOptions.EnforceHTTPS {
			continue // Skip domains without HTTPS enforcement
		}

		// Only include domains that are active and have VPC endpoints
		if domain.Processing != nil && *domain.Processing {
			continue // Skip domains that are being processed
		}

		if domain.VPCOptions == nil || len(domain.VPCOptions.SecurityGroupIds) == 0 {
			continue // Skip domains without VPC configuration
		}

		var endpoint string
		var port int32 = 443 // Default HTTPS port

		if domain.Endpoints != nil {
			if vpcEndpoint, exists := domain.Endpoints["vpc"]; exists {
				endpoint = vpcEndpoint
			}
		}

		if endpoint == "" && domain.DomainEndpointOptions != nil && domain.DomainEndpointOptions.CustomEndpoint != nil {
			endpoint = *domain.DomainEndpointOptions.CustomEndpoint
		}

		if endpoint == "" {
			continue // Skip domains without accessible endpoints
		}

		// Remove https:// prefix if present
		endpoint = strings.TrimPrefix(endpoint, "https://")

		var version string
		if domain.EngineVersion != nil {
			version = *domain.EngineVersion
		}

		domains = append(domains, OpenSearchDomain{
			Name:     *domainInfo.DomainName,
			Endpoint: endpoint,
			Port:     port,
			Version:  version,
		})
	}

	return domains, nil
}

func (o *OpenSearchManager) FindBastionHosts(ctx context.Context, domain OpenSearchDomain, findAll bool) ([]BastionHost, error) {
	// Get OpenSearch security groups
	opensearchSecurityGroups, err := o.getOpenSearchSecurityGroups(ctx, domain)
	if err != nil {
		return nil, err
	}

	debug.Printf("OpenSearch %s security groups: %v\n", domain.Name, opensearchSecurityGroups)

	// Pre-fetch all SG inbound rules once (avoids repeated API calls per instance)
	sgRulesCache, err := o.fetchSecurityGroupRules(ctx, opensearchSecurityGroups)
	if err != nil {
		return nil, err
	}

	// Find EC2 instances that can connect to OpenSearch
	var allReservations []types.Reservation
	var nextToken *string

	for {
		result, err := o.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			NextToken: nextToken,
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					if reloadErr := o.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					result, err = o.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
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
					stoppedInstanceNames = append(stoppedInstanceNames, o.getInstanceName(instance.Tags))
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

			name := o.getInstanceName(instance.Tags)
			ec2SgIds := o.getSecurityGroupIds(instance.SecurityGroups)
			debug.Printf("Checking if EC2 instance %s (%s) can reach OpenSearch — EC2 security groups: %v\n", name, *instance.InstanceId, ec2SgIds)

			if o.canConnectWithCachedRules(instance.SecurityGroups, sgRulesCache, domain.Port) {
				debug.Printf("✓ EC2 instance %s can connect to OpenSearch %s\n", name, domain.Name)
				bastion := BastionHost{
					InstanceId:       *instance.InstanceId,
					Name:             name,
					SecurityGroupIds: ec2SgIds,
				}
				if !findAll {
					return []BastionHost{bastion}, nil
				}
				bastions = append(bastions, bastion)
			} else {
				debug.Printf("✗ EC2 instance %s cannot connect to OpenSearch %s\n", name, domain.Name)
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
		fmt.Printf("No running EC2 instances found in region %s.\n", o.region)
		fmt.Printf("To use OpenSearch port forwarding, you need a running EC2 instance with:\n")
		fmt.Printf("- SSM agent installed and configured\n")
		fmt.Printf("- Network access to the OpenSearch domain\n")
		if stoppedInstances > 0 {
			fmt.Printf("\nYou can start one of the stopped instances above and try again.\n")
			return nil, fmt.Errorf("no running bastion hosts found - %d stopped instances available", stoppedInstances)
		}
		fmt.Printf("\nAlternatively, you can connect directly if your OpenSearch domain is publicly accessible.\n")
		return nil, fmt.Errorf("no running EC2 instances found in region %s", o.region)
	}

	fmt.Printf("Found %d running EC2 instances but none can connect to OpenSearch %s.\n", runningInstances, domain.Name)
	fmt.Printf("This usually means the security groups don't allow the connection.\n")
	return nil, fmt.Errorf("no suitable bastion hosts found - security groups may not allow connection")
}

func (o *OpenSearchManager) StartPortForwarding(ctx context.Context, bastionId, opensearchEndpoint string, opensearchPort, localPort int32) error {
	// Create port forwarder
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	pf := NewExternalPluginForwarder(cfg)

	fmt.Printf("Starting port forwarding...\n")

	// Start port forwarding to remote host through bastion
	return pf.StartPortForwardingToRemoteHost(ctx, bastionId, opensearchEndpoint, int(opensearchPort), int(localPort))
}

func (o *OpenSearchManager) getOpenSearchSecurityGroups(ctx context.Context, domain OpenSearchDomain) ([]string, error) {
	result, err := o.opensearchClient.DescribeDomain(ctx, &opensearch.DescribeDomainInput{
		DomainName: aws.String(domain.Name),
	})
	if err != nil {
		if IsAuthError(err) {
			if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
				if reloadErr := o.reloadClients(ctx); reloadErr != nil {
					return nil, reloadErr
				}
				result, err = o.opensearchClient.DescribeDomain(ctx, &opensearch.DescribeDomainInput{
					DomainName: aws.String(domain.Name),
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

	if result.DomainStatus == nil || result.DomainStatus.VPCOptions == nil {
		return nil, fmt.Errorf("OpenSearch domain not found or not in VPC")
	}

	return result.DomainStatus.VPCOptions.SecurityGroupIds, nil
}

func (o *OpenSearchManager) fetchSecurityGroupRules(ctx context.Context, sgIds []string) (map[string][]types.IpPermission, error) {
	cache := make(map[string][]types.IpPermission, len(sgIds))
	for _, sgId := range sgIds {
		result, err := o.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{sgId},
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					if reloadErr := o.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					result, err = o.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
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

func (o *OpenSearchManager) canConnectWithCachedRules(ec2SecurityGroups []types.GroupIdentifier, sgRulesCache map[string][]types.IpPermission, port int32) bool {
	ec2SgIds := make(map[string]bool)
	for _, sg := range ec2SecurityGroups {
		ec2SgIds[*sg.GroupId] = true
	}

	for sgId, rules := range sgRulesCache {
		debug.Printf("  Checking OpenSearch security group %s inbound rules for port %d\n", sgId, port)
		for _, rule := range rules {
			if o.ruleMatchesPort(rule, port) {
				debug.Printf("    Rule allows port %d\n", port)
				for _, pair := range rule.UserIdGroupPairs {
					if pair.GroupId != nil {
						var ec2SgList []string
						for id := range ec2SgIds {
							ec2SgList = append(ec2SgList, id)
						}
						if ec2SgIds[*pair.GroupId] {
							debug.Printf("      OpenSearch SG allows %s → EC2 has %s ✓ match!\n", *pair.GroupId, strings.Join(ec2SgList, ", "))
							return true
						}
						debug.Printf("      OpenSearch SG allows %s → EC2 has %s — no match\n", *pair.GroupId, strings.Join(ec2SgList, ", "))
					}
				}
				for _, ipRange := range rule.IpRanges {
					if ipRange.CidrIp != nil {
						if *ipRange.CidrIp == "0.0.0.0/0" {
							debug.Printf("      OpenSearch SG allows 0.0.0.0/0 (open access) ✓\n")
							return true
						}
						debug.Printf("      OpenSearch SG allows CIDR %s — not matched (only SG-to-SG supported)\n", *ipRange.CidrIp)
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

func (o *OpenSearchManager) ruleMatchesPort(rule types.IpPermission, port int32) bool {
	// All-traffic rules (protocol -1) have nil ports and match everything
	if rule.FromPort == nil || rule.ToPort == nil {
		return rule.IpProtocol != nil && *rule.IpProtocol == "-1"
	}
	return *rule.FromPort <= port && port <= *rule.ToPort
}

func (o *OpenSearchManager) getInstanceName(tags []types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	return "Unnamed"
}

func (o *OpenSearchManager) getSecurityGroupIds(sgs []types.GroupIdentifier) []string {
	var ids []string
	for _, sg := range sgs {
		ids = append(ids, *sg.GroupId)
	}
	return ids
}

func (o *OpenSearchManager) reloadClients(ctx context.Context) error {
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return err
	}

	o.opensearchClient = opensearch.NewFromConfig(cfg)
	o.ec2Client = ec2.NewFromConfig(cfg)
	o.ssmClient = ssmservice.NewFromConfig(cfg)
	o.region = cfg.Region

	return nil
}
