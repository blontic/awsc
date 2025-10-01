# SWA Command Patterns

## Mandatory Command Design Pattern
**ALL commands MUST support direct parameter access with fallback to interactive selection**

## Standard Command Pattern
Every command that operates on AWS resources must follow this pattern:

```bash
# Interactive selection (no parameters)
./swa [service] [action]

# Direct access (with parameters)
./swa [service] [action] --name [resource-name]
```

## Implementation Requirements

### Command Structure
- **Interactive Mode**: When no specific parameters provided, show list for user selection
- **Direct Mode**: When `--name` or equivalent parameter provided, attempt direct access
- **Fallback Behavior**: When direct access fails (resource not found), show error and fall back to interactive list
- **Consistent Messaging**: Use "Selected: [resource]" format after selection

### Error Handling Pattern
```go
// If specific resource name provided, try direct access
if resourceName != "" {
    // Attempt direct access
    if resource found and accessible {
        // Process directly and return
        return processResource(resource)
    } else {
        // Show error and fall through to interactive list
        fmt.Printf("Resource '%s' not found. Available resources:\n\n", resourceName)
    }
}

// Interactive selection (either no name provided or resource not found)
// Show list and let user select
```

## Current Command Compliance

### ✅ Compliant Commands
- `logs tail` - Supports `--group` flag with fallback
- `secrets show` - Supports `--name` flag with fallback
- `rds connect` - Supports `--name` flag with fallback  
- `ec2 connect` - Supports `--instance-id` parameter with fallback
- `ec2 rdp` - Supports `--instance-id` parameter with fallback
- `login` - Supports `--account` and `--role` flags with fallback

### 🔄 Commands to Update
- Any new commands must follow this pattern
- Existing commands should be updated to this pattern when modified

## Flag Naming Standards
- **Primary resource identifier**: `--name` (preferred)
- **Alternative identifiers**: `--id`, `--identifier` (when name is not appropriate)
- **Boolean modifiers**: `--force`, `--verbose` (for behavior modification)

## User Experience Requirements
- **No parameter**: Show interactive list immediately
- **Valid parameter**: Process directly, show "Selected: [resource]" 
- **Invalid parameter**: Show error message, then interactive list
- **Empty list**: Show helpful message about no resources found
- **Consistent selection format**: All interactive selections use same UI pattern

## Documentation Requirements
- **README examples**: Show both interactive and direct usage for each command
- **Help text**: Command descriptions should mention both modes
- **Global flags**: All commands support `--region`, `--config`, `--verbose`

## Benefits of This Pattern
- **Scriptable**: Commands can be used in automation with direct parameters
- **Interactive**: User-friendly for exploration and discovery
- **Consistent**: Same pattern across all commands reduces learning curve
- **Robust**: Graceful fallback when direct access fails
- **Discoverable**: Users can explore available resources when direct access fails
