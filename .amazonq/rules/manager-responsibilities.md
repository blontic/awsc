# SWA Manager Responsibilities

## CredentialsManager (`internal/aws/credentials.go`)
- **Authentication**: SSO device flow, browser opening, token polling
- **Token Management**: Cache SSO tokens in `~/.aws/sso/cache/` with secure permissions
- **Credential Setup**: Write AWS credentials to `~/.aws/config` (swa profile)
- **User Workflow**: Account/role selection, interactive UI for login process
- **Error Detection**: `IsAuthError()` for credential expiration detection
- **Config Loading**: Uses `config.LoadAWSConfig()` for SSO client initialization

## SSOManager (`internal/aws/sso.go`)
- **Pure Listing**: Account listing, role listing, credential retrieval
- **No Authentication**: Requires access token to be provided
- **Stateless Operations**: No token caching or user interaction
- **Clean Interface**: All methods take access token as parameter
- **Config Loading**: Uses `config.LoadAWSConfig()` for region configuration

## Service Managers (RDS, EC2, Secrets, Logs)
- **AWS Operations**: Service-specific operations using AWS SDK
- **Credential Loading**: Use `LoadSWAConfigWithProfile()` to load swa profile
- **Error Handling**: Detect auth errors and guide user to run `swa login`
- **Client Reload**: MANDATORY reloadClient() method to refresh AWS clients after re-authentication
- **Business Logic**: Complex workflows like bastion discovery, port forwarding, log tailing
- **Config Loading**: Uses `LoadSWAConfigWithProfile()` for authenticated operations

## Config Package (`internal/config/setup.go`)
- **Shared Utilities**: `LoadAWSConfig()` for region override logic
- **Configuration Management**: Config file creation and validation
- **Region Priority**: `--region` flag > `default_region` > `sso.region`
- **No AWS Operations**: Pure configuration and utility functions

## Separation Principles
- **Authentication vs Listing**: CredentialsManager handles auth, SSOManager handles listing
- **Config vs Operations**: Config package for utilities, AWS package for operations
- **Stateful vs Stateless**: CredentialsManager manages state, SSOManager is stateless
- **User Interaction vs API Calls**: CredentialsManager handles UI, SSOManager handles API