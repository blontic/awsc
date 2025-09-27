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
- All AWS service managers use LoadSWAConfig() to load swa profile
- Detect authentication errors with IsAuthError() function
- Clean separation: authentication vs listing vs credential management

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

## Manager Pattern
- Create manager structs for AWS services (SSO, RDS, Secrets, etc.) in `internal/aws/`
- Initialize with context and AWS config using `LoadAWSConfig()` or `LoadSWAConfig()`
- Store clients and region in manager struct
- Provide high-level methods for operations
- Include all required service clients in manager (e.g., RDSManager has rdsClient, ec2Client, ssmClient)
- Use manager methods for complex workflows (e.g., FindBastionHosts, StartPortForwarding)

## Configuration Loading
- **`config.LoadSWAConfig()`**: AWS config with region override for SSO operations (no profile)
- **`config.LoadSWAConfigWithProfile()`**: AWS config with swa profile for service operations
- Region priority: `--region` flag > `default_region` config > `sso.region` config
- All managers use appropriate config loader for their purpose

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

## Security Requirements
- Use secure file permissions: 0700 for directories, 0600 for files
- Replace SHA-1 with SHA-256 for hashing operations
- Add nil pointer checks before dereferencing AWS SDK responses
- Validate file paths to prevent path traversal attacks
- Use strings.Contains() instead of custom contains() functions

## External Plugin SSM Implementation
- Use AWS SDK SSM StartSession for session creation
- Use external session-manager-plugin for protocol handling
- Same requirement and compatibility as AWS CLI
- Provide clear installation instructions when plugin missing