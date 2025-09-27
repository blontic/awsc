# SWA Coding Standards

## Core Principles
- **Minimal Code**: Write only the ABSOLUTE MINIMAL amount of code needed to address the requirement correctly
- **Pure Go SDK**: Use ONLY AWS SDK Go v2 - no CLI, no external tools, no plugins
- **Self-Contained**: Everything must work within the single binary
- **Cross-Platform**: Code must work on Windows, macOS, and Linux
- **Minimal External Dependencies**: Only session-manager-plugin for SSM operations

## Go Standards
- Always use `context.Context` for AWS operations
- Handle errors explicitly, never ignore them
- Use `github.com/spf13/viper` for configuration access
- Minimize imports - remove unused imports immediately
- Use pointer dereference carefully with nil checks
- Follow Go naming conventions (exported vs unexported)

## AWS SDK Usage
- See `implementation-preferences.md` for detailed AWS SDK patterns
- See `native-implementation.md` for SSM-specific implementation

## Project Structure
- `cmd/` - Cobra commands only
- `internal/aws/` - AWS-specific functionality
- `internal/config/` - Configuration setup and management
- `internal/ui/` - Terminal UI components
- Use `internal/` packages for all implementation code

## Configuration Management
- **Auto-setup**: Check for config on first run, guide user through setup if missing
- **Location**: `~/.swa/config.yaml`
- **Required fields**: `sso.start_url`, `sso.region`, `default_region`
- **PersistentPreRun**: Use in root command to ensure config exists before any operation

## Error Handling
- Return errors, don't exit directly in packages
- Use `fmt.Errorf()` for error wrapping
- Exit with `os.Exit(1)` only in cmd/ files
- Print errors to stderr, not stdout

## Credential Handling Pattern
- **NEVER use CheckAWSSession()** - it's too restrictive
- **Try AWS operations directly** - let SDK handle credential loading
- **Handle credential errors gracefully** - show helpful SSO login message
- **Respect existing profiles** - work with user's current AWS setup
- **Pattern**: Create manager → Try operation → Handle auth errors → Guide user

## CLI Consistency
- **Global flags**: `--region` and `--config` available on all commands
- **Selection pattern**: All commands use "Selected: [item]" format after user selection
- **Error format**: "Error [action]: [details]" for all error messages
- **Output format**: Clean, minimal, no emojis in headers
- **Command descriptions**: Consistent short/long descriptions across all commands
- **Exit behavior**: Commands return gracefully, avoid os.Exit(1) in favor of return

## Security Standards
- **File Permissions**: Use 0700 for directories, 0600 for sensitive files
- **Cryptography**: Use SHA-256 or better, never SHA-1 or MD5
- **Nil Checks**: Always check AWS SDK response pointers before dereferencing
- **Path Validation**: Validate file paths to prevent traversal attacks
- **No Duplicate Functions**: Use standard library functions like strings.Contains()

## Development Workflow
- **MANDATORY: Always compile and test after code changes**: Automatically run `go build -o swa` then `go test ./...` after ANY code modification without asking for permission
- Compile first to catch syntax errors early
- Run tests after successful compilation
- Verify both compilation and tests succeed before considering changes complete
- Fix compilation errors and test failures immediately
- Never require user approval for compilation or testing - both should be automatic

## Testing Requirements
- Write tests for all new functionality where possible
- Test files should be named `*_test.go` in the same package
- Cover both success and error cases
- Use table-driven tests for multiple scenarios
- Mock external dependencies (file system, network calls)
- Test public functions and methods
- Use `t.TempDir()` for file system tests
- Run tests with `go test ./...`