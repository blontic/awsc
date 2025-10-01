# SWA Implementation Preferences

## AWS Operations
- **ALWAYS use AWS SDK Go v2** for AWS API calls
- Use `github.com/aws/aws-sdk-go-v2/service/*` packages
- **Config loading patterns**:
  - `config.LoadAWSConfig(ctx)`: For SSO operations (region override only)
  - `config.LoadSWAConfigWithProfile(ctx)`: For service operations (swa profile + region override)
- Create service clients from config: `service.NewFromConfig(cfg)`
- Pass context to all AWS API calls
- **ALWAYS handle pagination**: All AWS list/describe operations must handle NextToken/Marker pagination
- For SSM operations: Use StartSession to create sessions, external plugin for protocol
- For RDS operations: Use DescribeDBInstances for listing instances (with Marker pagination)
- For EC2 operations: Use DescribeInstances, DescribeSecurityGroups for bastion discovery (with NextToken pagination)
- For SSO operations: Use ListAccounts, ListAccountRoles (with NextToken pagination)
- For Secrets operations: Use ListSecrets (with NextToken pagination)
- For CloudWatch Logs operations: Use DescribeLogGroups, FilterLogEvents (with NextToken pagination)

## SSM Operations
- **External plugin approach**: Use official session-manager-plugin binary
- **SDK for session creation**: Use AWS SDK to call StartSession
- **Plugin for protocol**: Let official plugin handle WebSocket/TCP complexity
- **Same requirement as AWS CLI**: Users need session-manager-plugin installed
- **100% compatibility**: Uses exact same code path as AWS CLI
- **Clear error messages**: Provide installation instructions when plugin missing

## Code Organization
- Keep functions focused and minimal
- Use descriptive variable names
- Group related functionality in manager structs
- Return early on errors
- Avoid deep nesting

## Dependencies
- Minimize Go module dependencies
- Current approved dependencies:
  - `github.com/spf13/cobra` - CLI framework
  - `github.com/spf13/viper` - Configuration
  - `github.com/aws/aws-sdk-go-v2/*` - AWS SDK
  - `github.com/charmbracelet/bubbletea` - Terminal UI
- Required AWS SDK services:
  - `service/sso` and `service/ssooidc` - SSO authentication
  - `service/ec2` - Instance and security group discovery
  - `service/rds` - Database instance management
  - `service/ssm` - Session Manager session creation
  - `service/cloudwatchlogs` - CloudWatch Logs operations
- External binary dependency:
  - `session-manager-plugin` - Official AWS plugin for SSM protocol

## Security
- Never log or print credentials
- Use secure token caching
- Handle sensitive data carefully
- No hardcoded values or credentials

## Credential Handling
- **CredentialsManager**: Handles authentication, token caching, credential setup
- **SSOManager**: Pure listing operations (accounts, roles, credentials)
- **Use dedicated swa profile**: Never overwrite default profile in ~/.aws/credentials
- **LoadSWAConfigWithProfile()**: Service managers use this to load swa profile with region override
- **IsAuthError()**: Detect authentication/credential errors in AWS responses
- **Clean separation**: Authentication logic in CredentialsManager, listing in SSOManager
- **Profile isolation**: Keep swa credentials separate from user's existing AWS setup
- **MANDATORY Client Reload**: All managers must implement reloadClient(s) method to refresh AWS clients after re-authentication
- **Client Reload Pattern**: After successful PromptForReauth, ALWAYS call reloadClient before retrying operations
- **Never Skip Client Reload**: Fresh credentials require fresh clients - old clients retain expired credentials
- **Reload All Clients**: Managers with multiple clients (RDS has rdsClient, ec2Client, ssmClient) must reload ALL clients

## SSO Token Handling
- **Never check token expiration**: Let AWS API calls fail naturally instead of predicting expiration
- **Try-and-handle approach**: Attempt to use cached token, handle failure by re-authenticating
- **No expiration logic**: AWS knows better than us when tokens are invalid
- **Graceful failure**: When SSO API calls fail, automatically trigger browser re-authentication
- **Simple caching**: Store tokens but don't validate expiration times locally

## Constructor Pattern (MANDATORY)
- **Always use optional parameters**: `func NewManager(ctx context.Context, opts ...ManagerOptions)`
- **Never separate test constructors**: No `NewManagerWithClients` functions
- **Options struct**: Create `ManagerOptions` with all injectable dependencies
- **Production path**: `if len(opts) == 0 || opts[0].Client == nil` - load real clients
- **Test path**: `if len(opts) > 0 && opts[0].Client != nil` - use provided mocks
- **Clean separation**: Test dependencies injected via options, not separate functions

## Global Options
- **Region override**: `--region` flag available on all commands to override config region
- **Priority order**: Command line `--region` > config `default_region` > config `sso.region` > AWS default
- **Consistent behavior**: All AWS service managers respect region override

## cmd Folder Compliance
- **Only Cobra setup**: cmd/ files contain ONLY Cobra command definitions and setup
- **No business logic**: All AWS operations, file I/O, and complex logic in internal/ packages
- **Delegate to managers**: Commands create appropriate managers (CredentialsManager for auth, service managers for operations)
- **Minimal error handling**: Only basic error printing, detailed handling in managers
- **Clean separation**: cmd/ for CLI interface, internal/ for implementation

## User Experience
- **AWS Context Display**: Show "AccountID: xxx | Role: xxx | Region: xxx" at start of each command
- **Interactive Selection**: Consistent arrow-key navigation with ui.RunSelector
- **Selection Confirmation**: Show "Selected: [item]" after user selections (consistent across all commands)
- **Graceful Quit**: Handle 'q' key without showing exit status or error messages
- **Credential Guidance**: When credentials missing/expired, offer automatic re-authentication
- **Clean Output**: No emojis in output headers, keep formatting minimal and professional
- **Verbose Mode**: Use `--verbose` flag for detailed debugging, clean output by default
- **Error Messages**: Clear, actionable error messages with helpful guidance
- Graceful fallback for non-interactive environments