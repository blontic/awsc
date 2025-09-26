# SWA Implementation Preferences

## AWS Operations
- **ALWAYS use AWS SDK Go v2** for AWS API calls
- Use `github.com/aws/aws-sdk-go-v2/service/*` packages
- Load config with swa profile: `LoadSWAConfig(ctx)` - includes region override and swa profile
- Create service clients from config: `service.NewFromConfig(cfg)`
- Pass context to all AWS API calls
- For SSM operations: Use StartSession to create sessions, external plugin for protocol
- For RDS operations: Use DescribeDBInstances for listing instances
- For EC2 operations: Use DescribeInstances, DescribeSecurityGroups for bastion discovery

## SSM Port Forwarding
- **External plugin approach**: Use official session-manager-plugin binary
- **SDK for session creation**: Use AWS SDK to call StartSession
- **Plugin for protocol**: Let official plugin handle WebSocket/TCP complexity
- **Same requirement as AWS CLI**: Users need session-manager-plugin installed
- **100% compatibility**: Uses exact same code path as AWS CLI

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
- External binary dependency:
  - `session-manager-plugin` - Official AWS plugin for SSM protocol

## Security
- Never log or print credentials
- Use secure token caching
- Handle sensitive data carefully
- No hardcoded values or credentials

## Credential Handling
- **Use dedicated swa profile**: Never overwrite default profile in ~/.aws/credentials
- **LoadSWAConfig()**: All managers use this to load swa profile with region override
- **IsAuthError()**: Detect authentication/credential errors in AWS responses
- **Auto-retry pattern**: Try operation → Detect auth error → Run handleAuthenticationError() → Recreate manager → Retry operation
- **Consistent pattern**: Create manager → Try AWS operation → Handle auth errors → Recreate manager → Retry
- **Profile isolation**: Keep swa credentials separate from user's existing AWS setup

## Global Options
- **Region override**: `--region` flag available on all commands to override config region
- **Priority order**: Command line `--region` > config `default_region` > config `sso.region` > AWS default
- **Consistent behavior**: All AWS service managers respect region override

## User Experience
- Provide clear, actionable error messages
- **Minimal verbose output**: Remove unnecessary progress messages like "Fetching...", "Finding..."
- **Selection confirmation**: Show "Selected: [item]" after user selections (consistent across all commands)
- **Graceful quit**: Handle 'q' key without showing exit status or error messages
- **Credential guidance**: When credentials missing/expired, show "Please run: swa login"
- **Clean output**: No emojis in output headers, keep formatting minimal and professional
- Use interactive selection when possible
- Graceful fallback for non-interactive environments
- Clean, minimal output