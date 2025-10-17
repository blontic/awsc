# AWSC Core Architecture Rules

## Self-Contained Binary Design

- **Pure Go implementation** using AWS SDK v2 only
- **NO AWS CLI dependencies** - tool must work independently without AWS CLI installed
- **NO fallback commands** - when operations fail, return clear errors without suggesting manual AWS CLI commands
- **External dependency**: session-manager-plugin required for SSM operations only
- Must work cross-platform (macOS and Linux only - Windows users should use WSL)

## Multi-Profile Support

- **Profile Naming**: `awsc-{accountName}` format stored in `~/.aws/config`
- **Session Tracking**: PPID-based sessions in `~/.awsc/sessions/session-{ppid}.json`
- **Hybrid Selection Priority**:
  1. `AWSC_PROFILE` environment variable (explicit override)
  2. PPID session file (automatic per-terminal)
  3. Return "no active session" error
- **Auto-Login**: All commands detect "no active session" and auto-trigger login
- **Platform**: macOS and Linux only (PPID tracking requires Unix-like systems)
- **Profile Manager**: Located in `internal/config/profile.go` (local state management)

## Configuration Management

- Use `~/.awsc/config.yaml` for configuration storage
- Access config via `github.com/spf13/viper`
- Required fields: `sso.start_url`, `sso.region`, `default_region`
- Support global `--region` and `--config` flags
- Auto-setup: Check for config on first run, guide user through setup if missing

## Authentication & Credentials

- **CredentialsManager**: Handles SSO authentication, token caching, credential setup
- **SSOManager**: Pure account/role listing operations (requires access token)
- SDK-based SSO authentication using device authorization flow
- Cache tokens in `~/.aws/sso/cache/` with secure permissions (0600)
- Write credentials to `~/.aws/config` (awsc profile)
- All AWS service managers use `LoadAWSConfigWithProfile()` to load awsc profile
- **MANDATORY**: All AWS operations must handle auth errors with automatic re-authentication prompt
- **MANDATORY**: Long-running operations must check auth errors on each iteration
- **MANDATORY**: All managers must reload AWS clients after successful re-authentication
- **Auto-reauth flow**: "Credentials expired. Re-authenticate? (y/n)" → Run login automatically → Reload clients → Retry operation

## Manager Pattern & Constructor Requirements

- Create manager structs for AWS services in `internal/aws/`
- **MANDATORY Constructor Pattern**: `func NewManager(ctx context.Context, opts ...ManagerOptions)`
- **NEVER create separate test constructors** (e.g., NewManagerWithClients)
- Use `ManagerOptions` struct with all injectable dependencies
- Production usage: `NewManager(ctx)` - loads real AWS clients
- Test usage: `NewManager(ctx, ManagerOptions{Client: mockClient, Region: "region"})`
- Initialize with context and AWS config using `LoadAWSConfigWithProfile()`
- Include all required service clients in manager (e.g., RDSManager has rdsClient, ec2Client, ssmClient)

### Auth Error Handling at Manager Creation

Commands must handle "no active session" errors when creating managers:

```go
manager, err := aws.NewManager(ctx)
if err != nil {
    if aws.IsAuthError(err) {
        shouldReauth, reAuthErr := aws.PromptForReauth(ctx)
        if reAuthErr != nil {
            fmt.Printf("Error during re-authentication: %v\n", reAuthErr)
            os.Exit(1)
        }
        if !shouldReauth {
            fmt.Printf("Authentication cancelled\n")
            os.Exit(1)
        }
        // Retry creating manager after successful login
        manager, err = aws.NewManager(ctx)
        if err != nil {
            fmt.Printf("Error creating manager after re-authentication: %v\n", err)
            os.Exit(1)
        }
    } else {
        fmt.Printf("Error creating manager: %v\n", err)
        os.Exit(1)
    }
}
```

**Auth errors detected:**
- "no active session" - No PPID session or AWSC_PROFILE set
- "failed to get shared config profile" - Profile missing from ~/.aws/config
- "ExpiredToken", "InvalidToken" - Credentials expired
- "failed to refresh cached credentials" - Token refresh failed
- Other AWS credential errors

**Auto-recovery:** All auth errors trigger automatic re-authentication flow

### Auth Error Handling During Operations

Managers must handle auth errors during AWS operations:

```go
result, err := m.client.Operation(ctx, input)
if err != nil {
    if IsAuthError(err) {
        if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
            // MANDATORY: Reload clients with fresh credentials
            if reloadErr := m.reloadClients(ctx); reloadErr != nil {
                return reloadErr
            }
            // Retry the operation
            result, err = m.client.Operation(ctx, input)
            if err != nil {
                return err
            }
        } else {
            return err
        }
    } else {
        return err
    }
}
```

## AWS Operations & Pagination

- **ALWAYS use AWS SDK Go v2** for AWS API calls
- **MANDATORY**: All AWS list/describe operations must handle pagination (NextToken/Marker)
- **Pattern**: Use for loop with NextToken/Marker until no more pages
- **Never assume single page**: AWS APIs are paginated by default
- Config loading patterns:
  - `config.LoadAWSConfig(ctx)`: For SSO operations (region override only)
  - `config.LoadAWSConfigWithProfile(ctx)`: For service operations (awsc profile + region override)

## Command Design Pattern

- **MANDATORY**: All commands must support direct parameter access with fallback to interactive selection
- **Pattern**: `./awsc [service] [action]` (interactive) and `./awsc [service] [action] --name [resource]` (direct)
- **Error Handling**: When direct access fails, show error and fall back to interactive list
- **Consistency**: Use "Selected: [resource]" format after all selections
- **Empty Resource Handling**: Show "No [resources] found" when lists are empty

## Output Design

- **stderr**: Interactive messages, status updates, success notifications
- **stdout**: Only export commands for clean `eval $(awsc command)` usage
- **AWS Context Display**: Show AccountID, Role, Region at start of each command
- **Verbose Mode**: Global `--verbose` flag for detailed debugging output
- No credential leakage in logs or output

## Manager Responsibilities

- **CredentialsManager**: Authentication, token management, credential setup, user workflow
- **SSOManager**: Pure listing operations (accounts, roles, credentials) - stateless
- **Service Managers**: AWS operations using `LoadAWSConfigWithProfile()`, auth error handling, client reload
- **Config Package**: Shared utilities, configuration management, region priority logic

## SSM Implementation

- Use AWS SDK SSM StartSession for session creation
- Use external session-manager-plugin for protocol handling
- Same requirement and compatibility as AWS CLI
- Provide clear installation instructions when plugin missing

## Global Flags

- **`--region`**: Override AWS region for any command
- **`--config`**: Specify alternate AWSC config file
- **`--verbose`**: Enable detailed debug output via debug package
- **`--force`**: Force re-authentication (login command)
