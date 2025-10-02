# SWA Development Standards

## Core Principles
- **Minimal Code**: Write only the ABSOLUTE MINIMAL amount of code needed to address the requirement correctly
- **Self-Contained**: Everything must work within the single binary
- **Cross-Platform**: Code must work on Windows, macOS, and Linux
- **NO AWS CLI Dependencies**: Tool must work independently without AWS CLI installed

## Go Standards
- Always use `context.Context` for AWS operations
- Handle errors explicitly, never ignore them
- Use `github.com/spf13/viper` for configuration access
- Minimize imports - remove unused imports immediately
- Use pointer dereference carefully with nil checks
- Follow Go naming conventions (exported vs unexported)

## Project Structure
- `cmd/` - Cobra commands only (no business logic)
- `internal/aws/` - AWS-specific functionality
- `internal/config/` - Configuration setup and management
- `internal/ui/` - Terminal UI components
- Use `internal/` packages for all implementation code

## Error Handling
- Return errors, don't exit directly in packages
- Use `fmt.Errorf()` for error wrapping
- Exit with `os.Exit(1)` only in cmd/ files
- Print errors to stderr, not stdout
- **NO AWS CLI fallbacks**: When operations fail, return clear errors without suggesting manual AWS CLI commands
- **Self-contained errors**: Tool should work independently without requiring AWS CLI

## Credential Handling Pattern
- **Try AWS operations directly** - let SDK handle credential loading
- **Handle credential errors gracefully** - offer automatic re-authentication
- **MANDATORY Auth Error Pattern**:
  ```go
  if err != nil {
      if IsAuthError(err) {
          if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
              // MANDATORY: Reload client with fresh credentials after re-auth
              if reloadErr := m.reloadClients(ctx); reloadErr != nil {
                  return reloadErr
              }
              // Retry the operation after successful re-auth and client reload
              return retryOperation()
          }
          return err
      }
      return err
  }
  ```
- **MANDATORY Client Reload**: All managers must reload AWS clients after successful re-authentication

## CLI Consistency
- **Global flags**: `--region`, `--config`, and `--verbose` available on all commands
- **Command pattern**: ALL commands must support direct parameter access with interactive fallback
- **Parameter naming**: Use `--name` for primary resource identifier, `--id` for alternatives
- **Selection pattern**: All commands use "Selected: [item]" format after user selection
- **Error format**: "Error [action]: [details]" for all error messages
- **Fallback behavior**: When direct access fails, show error then interactive list

## Security Standards
- **File Permissions**: Use 0700 for directories, 0600 for sensitive files
- **Nil Checks**: Always check AWS SDK response pointers before dereferencing
- **Path Validation**: Validate file paths to prevent traversal attacks
- **No Credential Leakage**: Never log or print credentials

## Development Workflow
- **MANDATORY: Always compile and test after code changes**: Run `go build -o swa` then `go test ./...` after ANY code modification
- Compile first to catch syntax errors early
- Run tests after successful compilation
- Fix compilation errors and test failures immediately

## Testing Requirements
- Write tests for all new functionality where possible
- Use table-driven tests for multiple scenarios
- Mock external dependencies (file system, network calls)
- Use `t.TempDir()` for file system tests
- **Constructor Testing**: Use optional parameter pattern, never separate test constructors

## New Command Checklist
- [ ] **Interactive + Direct modes**: Support both parameter-less and `--name` parameter access
- [ ] **Constructor pattern**: `NewManager(ctx context.Context, opts ...ManagerOptions)`
- [ ] **Auth error handling**: Use `IsAuthError(err)` and `PromptForReauth()`
- [ ] **Client reload**: MANDATORY `reloadClient()` method after re-authentication
- [ ] **Pagination support**: Handle NextToken/Marker for all list operations
- [ ] **Empty resource handling**: Show "No [resources] found" message
- [ ] **Documentation**: Update README with both interactive and direct usage examples

## UI/UX Standards
- **Interactive selectors**: Use Bubble Tea with triangle cursor (▶) and bold text for selected items
- **AWS context display**: Show AccountID, Role, Region in selector headers with styled borders
- **Output separation**: stderr for interactive messages, stdout only for export commands
- **Empty states**: Show "No [resources] found" when lists are empty
- **Progress indication**: Show clear status messages during operations

## Dependencies
- Minimize Go module dependencies
- Current approved dependencies:
  - `github.com/spf13/cobra` - CLI framework
  - `github.com/spf13/viper` - Configuration
  - `github.com/aws/aws-sdk-go-v2/*` - AWS SDK
  - `github.com/charmbracelet/bubbletea` - Terminal UI
- External binary dependency:
  - `session-manager-plugin` - Official AWS plugin for SSM protocol