# SWA Development Workflow Rules

## Rule Management
- **CRITICAL**: Update project rules whenever implementing new features or patterns
- Rules must reflect current project state and decisions
- Add new rule files for new domains (e.g., database operations, networking)
- Keep rules minimal and actionable

## Implementation Process
1. Analyze requirement against existing architecture
2. Update relevant rule files before coding
3. Implement using established patterns
4. Verify implementation follows all rules
5. Update rules if new patterns emerge

## Rule Categories
- `architecture.md` - Core design principles and patterns
- `coding-standards.md` - Go coding conventions and practices  
- `implementation-preferences.md` - AWS SDK usage and dependencies
- `constructor-patterns.md` - Mandatory constructor patterns for managers
- `pagination-patterns.md` - Mandatory pagination patterns for AWS APIs
- `development-workflow.md` - This file - process and workflow rules

## Rule Updates Required When:
- Adding new AWS services or operations
- Introducing new dependencies
- Changing output patterns or user interaction
- Modifying authentication or configuration flows
- Adding new command structures or CLI patterns
- Adding new global flags or configuration options

## Documentation Updates Required When:
- Adding new CLI commands or subcommands
- Adding new global flags (`--region`, `--config`, etc.)
- Changing command syntax or behavior
- Adding new external dependencies
- Modifying configuration file structure
- Adding new AWS service integrations

## Continuous Improvement
- Rules should evolve with the codebase
- Remove obsolete rules when patterns change
- Consolidate similar rules to avoid duplication
- Keep rules focused on current active development