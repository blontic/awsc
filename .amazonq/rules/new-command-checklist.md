# New Command Implementation Checklist

## MANDATORY Requirements for All New Commands

### 1. Command Structure
- [ ] **Interactive mode**: Works without parameters, shows resource list
- [ ] **Direct mode**: Supports `--name` or equivalent parameter for direct access
- [ ] **Fallback behavior**: When direct access fails, shows error then interactive list
- [ ] **Selection confirmation**: Shows "Selected: [resource]" after user selection

### 2. Manager Implementation
- [ ] **Constructor pattern**: `NewManager(ctx context.Context, opts ...ManagerOptions)`
- [ ] **Config loading**: Uses `config.LoadSWAConfigWithProfile(ctx)` for production
- [ ] **Test support**: Optional parameters for mock clients in tests
- [ ] **Region support**: Respects `--region` flag override

### 3. Authentication Handling
- [ ] **Try-first approach**: Attempt AWS operations without pre-validation
- [ ] **Auth error detection**: Use `IsAuthError(err)` for credential issues
- [ ] **Auto-reauth prompt**: Ask "Credentials expired. Re-authenticate? (y/n)" and run login automatically
- [ ] **Client reload**: MANDATORY reloadClient() method to refresh AWS clients after re-authentication
- [ ] **NEVER skip reload**: ALWAYS call reloadClient() after successful PromptForReauth before retrying operations
- [ ] **Operation retry**: Retry original operation after successful re-authentication and client reload
- [ ] **Long-running operations**: Check auth errors in loops/polling/streaming and reload clients after re-auth
- [ ] **Multiple clients**: If manager has multiple AWS clients, reload ALL of them in reloadClient() method

### 4. Pagination Support
- [ ] **All list operations**: Must handle NextToken/Marker pagination
- [ ] **Complete results**: Collect all pages before processing
- [ ] **Proper loops**: Continue until NextToken/Marker is nil

### 5. Empty Resource Handling
- [ ] **Empty lists**: Show "No [resources] found" message
- [ ] **Helpful feedback**: Don't show blank output when no resources exist
- [ ] **Both modes**: Handle empty results in interactive and direct modes

### 6. Error Handling
- [ ] **Descriptive errors**: Clear messages for common failure scenarios
- [ ] **Graceful fallback**: When direct access fails, show available options
- [ ] **Consistent format**: "Error [action]: [details]" pattern

### 7. Testing Requirements
- [ ] **Constructor tests**: Verify optional parameter pattern works
- [ ] **Command structure tests**: Verify flags and subcommands exist
- [ ] **Flag validation**: Test all command-line flags are properly defined

### 8. Documentation Updates
- [ ] **README examples**: Show both interactive and direct usage patterns
- [ ] **Global flags**: Document how `--region` and `--config` work with command
- [ ] **Command descriptions**: Consistent short/long descriptions

### 9. Code Quality
- [ ] **Minimal implementation**: Only code needed for the requirement
- [ ] **Consistent patterns**: Follow established SWA architectural patterns
- [ ] **No duplicate code**: Reuse existing functions where possible
- [ ] **Proper imports**: Only import what's needed, remove unused imports

### 10. Integration
- [ ] **Root command**: Register new command in `cmd/root.go` init()
- [ ] **Dependencies**: Add required AWS SDK services to go.mod
- [ ] **Build verification**: `go build -o swa` succeeds
- [ ] **Test verification**: `go test ./...` passes

## Command Examples to Follow
- **logs tail**: Interactive selection, direct access, follow mode, auth handling
- **secrets show**: Direct parameter with fallback, empty resource handling
- **rds connect**: Complex workflows, bastion discovery, port forwarding
- **ec2 connect/rdp**: Instance filtering, parameter validation

## Anti-Patterns to Avoid
- ❌ Separate test constructors (use optional parameters)
- ❌ Pre-validating credentials (try operations first)
- ❌ Ignoring pagination (always handle NextToken/Marker)
- ❌ Silent failures on empty resources (show helpful messages)
- ❌ Missing auth error handling in long-running operations
- ❌ Skipping client reload after re-authentication (causes continued auth failures)
- ❌ Partial client reload (must reload ALL clients in multi-client managers)