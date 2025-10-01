# SWA Architecture Rules

## Self-Contained Binary Design
- Everything built into the binary using Go AWS SDK v2
- **External dependency**: session-manager-plugin required for SSM operations
- Must work cross-platform (Windows, macOS, Linux)
- Pure Go implementation for all AWS operations except SSM protocol handling

## Configuration Management
- Use `~/.swa/config.yaml` for configuration storage
- Access config via `github.com/spf13/viper`
- Required fields: `sso.start_url`, `sso.region`, `default_region`
- Load AWS config with region from viper configuration
- Support global `--region` flag to override config region
- Support global `--config` flag to specify alternate config file

## Authentication & Credentials
- **CredentialsManager**: Handles SSO authentication, token caching, credential setup
- **SSOManager**: Pure account/role listing operations (requires access token)
- SDK-based SSO authentication using device authorization flow
- Cache tokens in `~/.aws/sso/cache/` with secure permissions (0600)
- Write credentials to `~/.aws/config` (swa profile)
- All AWS service managers use LoadSWAConfigWithProfile() to load swa profile
- Detect authentication errors with IsAuthError() function
- Clean separation: authentication vs listing vs credential management
- **MANDATORY**: All AWS operations must handle auth errors with automatic re-authentication prompt
- **MANDATORY**: Long-running operations (polling, streaming) must check auth errors on each iteration
- **MANDATORY**: All managers must reload AWS clients after successful re-authentication
- **Auto-reauth flow**: "Credentials expired. Re-authenticate? (y/n)" → Run login automatically → Reload clients → Retry operation

## Output Design
- **stderr**: Interactive messages, status updates, success notifications
- **stdout**: Only export commands for clean `eval $(swa command)` usage
- **AWS Context Display**: Show AccountID, Role, Region at start of each command
- **Verbose Mode**: Global `--verbose` flag for detailed debugging output
- No credential leakage in logs or output
- Separate user feedback from shell eval output

## Interactive UI
- Primary: Bubble Tea for arrow key navigation
- Fallback: Simple numbered selection for non-interactive environments
- Graceful degradation when TTY not available
- Handle Ctrl+C gracefully

## Command Design Pattern
- **MANDATORY**: All commands must support direct parameter access with fallback to interactive selection
- **Pattern**: `./swa [service] [action]` (interactive) and `./swa [service] [action] --name [resource]` (direct)
- **Error Handling**: When direct access fails, show error and fall back to interactive list
- **Consistency**: Use "Selected: [resource]" format after all selections
- See `command-patterns.md` for detailed implementation requirements

## Manager Pattern
- Create manager structs for AWS services (SSO, RDS, Secrets, etc.) in `internal/aws/`
- Initialize with context and AWS config using `LoadSWAConfigWithProfile()`
- Store clients and region in manager struct
- Provide high-level methods for operations
- Include all required service clients in manager (e.g., RDSManager has rdsClient, ec2Client, ssmClient)
- Use manager methods for complex workflows (e.g., FindBastionHosts, StartPortForwarding)

## Constructor Pattern
- **ALWAYS use optional parameter pattern** for manager constructors
- **NEVER create separate test-only constructors** (e.g., NewManagerWithClients)
- Create `ManagerOptions` struct with all injectable dependencies
- Use variadic parameters: `func NewManager(ctx context.Context, opts ...ManagerOptions)`
- Production usage: `NewManager(ctx)` - loads real AWS clients
- Test usage: `NewManager(ctx, ManagerOptions{Client: mockClient, Region: "us-east-1"})`
- Keep test-only code out of production files

## Configuration Loading
- See `implementation-preferences.md` for detailed config loading patterns

## Global Flags
- **`--region`**: Override AWS region for any command
- **`--config`**: Specify alternate SWA config file
- **`--verbose`**: Enable detailed debug output via debug package
- **`--force`**: Force re-authentication (login command)

## Debug System
- **Debug Package**: `internal/debug` for controlled verbose output
- **Verbose Flag**: Global `--verbose` or `-v` flag
- **Clean Output**: Debug info only shown when requested
- **Stderr Output**: Debug messages go to stderr, not stdout

## AWS Services Integration
- **SSO**: Account and role listing operations only (no authentication)
- **Credentials**: SSO authentication, token management, account/role selection workflow
- **RDS**: Database instance listing, bastion host discovery, port forwarding
- **Secrets Manager**: Secret listing, retrieval, and display with JSON formatting
- **EC2**: Instance discovery for bastion hosts, security group analysis
- **SSM**: Session creation for port forwarding via external plugin
- **CloudWatch Logs**: Log group listing, log tailing with follow mode

## Security Requirements
- See `coding-standards.md` for detailed security standards

## External Plugin SSM Implementation
- Use AWS SDK SSM StartSession for session creation
- Use external session-manager-plugin for protocol handling
- Same requirement and compatibility as AWS CLI
- Provide clear installation instructions when plugin missing