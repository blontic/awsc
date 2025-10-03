# AWSC Project Maintenance

## Rule Management
- **CRITICAL**: Update project rules whenever implementing new features or patterns
- Rules must reflect current project state and decisions
- Keep rules minimal and actionable
- Remove obsolete rules when patterns change

## Documentation Requirements
- **Update README.md when adding new features** - Always document new commands and functionality
- **Dual usage patterns** - MUST document both interactive and direct parameter usage for every command
- **Parameter examples** - Show `--name` and other parameter usage with real examples
- **Global flags** - Show how global options work with specific commands
- **Prerequisites** - List all external dependencies (like session-manager-plugin)

## Documentation Standards
- Use consistent command examples with `./awsc` prefix
- **MANDATORY**: Show both interactive and direct parameter usage for every command
- Follow pattern: `./awsc [service] [action]` then `./awsc [service] [action] --name [resource]`
- Include global flag examples for major commands
- Keep language clear and concise
- Use code blocks for all command examples

## Implementation Process
1. Analyze requirement against existing architecture
2. Update relevant rule files before coding (if needed)
3. Implement using established patterns
4. Verify implementation follows all rules
5. Update documentation (README) with new functionality

## When to Update Rules
- Adding new AWS services or operations
- Introducing new dependencies
- Changing output patterns or user interaction
- Modifying authentication or configuration flows
- Adding new command structures or CLI patterns
- Creating commands that don't follow established patterns

## When to Update Documentation
- Adding new CLI commands or subcommands
- Adding new global flags (`--region`, `--config`, etc.)
- Changing command syntax or behavior
- Adding new external dependencies
- Modifying configuration file structure
- Adding new AWS service integrations

## Continuous Improvement
- Rules should evolve with the codebase
- Consolidate similar rules to avoid duplication
- Keep rules focused on current active development
- Remove duplicate or conflicting guidance