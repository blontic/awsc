# SWA Architecture Rules

## Self-Contained Binary Design
- Everything built into the binary using Go AWS SDK v2
- **NO external dependencies** - no AWS CLI, no session manager plugin
- Zero dependencies beyond the binary itself
- Must work cross-platform (Windows, macOS, Linux)
- Pure Go implementation for all AWS operations

## Configuration Management
- Use `~/.swa/config.yaml` for configuration storage
- Access config via `github.com/spf13/viper`
- Required fields: `sso.start_url`, `sso.region`, `default_region`
- Load AWS config with region from viper configuration
- Support global `--region` flag to override config region
- Support global `--config` flag to specify alternate config file

## Authentication & Credentials
- SDK-based SSO authentication using device authorization flow
- Cache tokens in `~/.aws/sso/cache/` with SHA1 filename
- Write credentials to `~/.aws/config` (swa profile)
- All AWS managers use LoadSWAConfig() to load swa profile
- Detect authentication errors with IsAuthError() function
- Auto-retry authentication when credentials fail

## Output Design
- **stderr**: Interactive messages, status updates, success notifications
- **stdout**: Only export commands for clean `eval $(swa command)` usage
- No credential leakage in logs or output
- Separate user feedback from shell eval output

## Interactive UI
- Primary: Bubble Tea for arrow key navigation
- Fallback: Simple numbered selection for non-interactive environments
- Graceful degradation when TTY not available
- Handle Ctrl+C gracefully

## Manager Pattern
- Create manager structs for AWS services (SSO, RDS, Secrets, etc.) in `internal/aws/`
- Initialize with context and AWS config
- Store clients and region in manager struct
- Provide high-level methods for operations
- Include all required service clients in manager (e.g., RDSManager has rdsClient, ec2Client, ssmClient)
- Use manager methods for complex workflows (e.g., FindBastionHosts, StartPortForwarding)

## AWS Services Integration
- **SSO**: Authentication, account/role selection, credential management
- **RDS**: Database instance listing, bastion host discovery, port forwarding
- **Secrets Manager**: Secret listing, retrieval, and display with JSON formatting
- **EC2**: Instance discovery for bastion hosts, security group analysis
- **SSM**: Session creation for port forwarding via external plugin

## Native SSM Implementation
- Use AWS SDK SSM StartSession for port forwarding
- Implement WebSocket connection handling in pure Go
- Native TCP proxy between local port and SSM WebSocket
- No external session manager plugin dependency