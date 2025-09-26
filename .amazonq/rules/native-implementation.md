# SWA SSM Implementation Rules

## External Plugin Approach
- **Use external session-manager-plugin**: Call the official AWS plugin binary
- **SDK for session creation**: Use AWS SDK v2 to create SSM sessions
- **Plugin for protocol handling**: Let the official plugin handle WebSocket/TCP
- **Same requirement as AWS CLI**: Users need session-manager-plugin installed
- **100% compatibility**: Uses exact same code path as `aws ssm start-session`

## Implementation Pattern
- Use `ssm.StartSession()` to create session and get credentials
- Pass session data to external `session-manager-plugin` binary
- Use same arguments as AWS CLI for perfect compatibility
- Provide clear installation instructions when plugin is missing

## Error Handling
- Detect missing session-manager-plugin and provide installation instructions
- Return descriptive errors for session creation failures
- Provide fallback manual commands as last resort
- Handle plugin execution failures gracefully

## Current Implementation
- `internal/aws/externalplugin.go` - External plugin wrapper
- Simple, reliable, and fully compatible with AWS CLI

## Plugin Installation
- **macOS**: `brew install --cask session-manager-plugin`
- **Linux**: Download and install .deb package
- **Windows**: Download and run installer
- Same requirement as AWS CLI