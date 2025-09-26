# SWA Documentation Rules

## README Maintenance
- **Update README.md when adding new features** - Always document new commands and functionality
- **Keep examples current** - Update usage examples to reflect actual command syntax
- **Document global options** - Show how `--region` and `--config` flags work with all commands
- **Maintain feature list** - Add new capabilities to the Features section
- **Update prerequisites** - Document any new external dependencies

## Documentation Requirements
- **Command descriptions** - Each new command needs usage examples in README
- **Configuration changes** - Document new config fields and their purpose
- **Global flags** - Show how global options work with specific commands
- **Prerequisites** - List all external dependencies (like session-manager-plugin)
- **Installation steps** - Keep setup instructions current and accurate

## Documentation Structure
- **Overview** - Brief description of what SWA does
- **Setup** - Installation and initial configuration steps
- **Usage** - Daily usage patterns with examples
- **Features** - List of all capabilities
- **Global Options** - How to use `--region` and `--config` flags
- **Available Commands** - Complete list of all commands
- **Prerequisites** - External dependencies and installation
- **Configuration** - Config file format and required fields

## When to Update Documentation
- Adding new CLI commands or subcommands
- Adding new global flags or options
- Changing command syntax or behavior
- Adding new external dependencies
- Modifying configuration file structure
- Adding new AWS service integrations

## Documentation Standards
- Use consistent command examples with `./swa` prefix
- Show both the command and what it does (numbered steps)
- Include global flag examples for major commands
- Keep language clear and concise
- Use code blocks for all command examples
- Document both success and error scenarios where relevant