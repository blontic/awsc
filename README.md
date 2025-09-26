# SWA (AWS backwards)

[![CI](https://github.com/blontic/swa/actions/workflows/ci.yml/badge.svg)](https://github.com/blontic/swa/actions/workflows/ci.yml)

A Go-based CLI tool for AWS operations including SSO authentication, account switching, and RDS port forwarding.

## Project Structure

```
.
├── main.go              # Entry point
├── cmd/                 # Cobra commands
│   ├── root.go         # Root command and configuration
│   ├── sso.go          # SSO-related commands
│   └── rds.go          # RDS-related commands
├── internal/            # Internal packages
│   ├── aws/            # AWS-specific functionality
│   │   ├── sso.go      # SSO manager
│   │   ├── credentials.go # Credential management
│   │   ├── rds.go      # RDS manager
│   │   └── externalplugin.go # Session manager plugin wrapper
│   ├── config/         # Configuration management
│   └── ui/             # Terminal UI components
```

## Setup

1. Install dependencies:
```bash
make deps
```

2. Build the tool:
```bash
make build
```

3. Configure SSO settings:
```bash
./swa config init
```

This will prompt you for:
- SSO Start URL (e.g., https://your-org.awsapps.com/start)
- SSO Region (e.g., us-east-1)
- Default AWS Region (e.g., us-east-1)

## Usage

### First Time Setup
1. Configure SSO:
```bash
./swa config init
```

2. Authenticate with SSO:
```bash
./swa login
```

### Daily Usage

**SSO Login:**
```bash
./swa login
```

This will:
1. List all available AWS accounts from your SSO
2. Let you select an account
3. List available roles for that account
4. Let you select a role
5. Set up credentials in ~/.aws/config (swa profile)

**RDS Port Forwarding:**
```bash
./swa rds connect
```

This will:
1. List available RDS instances
2. Find compatible bastion hosts
3. Set up port forwarding tunnel
4. Allow direct connection to RDS via localhost

**Secrets Manager:**
```bash
./swa secrets list
```

This will:
1. List all secrets in AWS Secrets Manager
2. Let you select a secret to view
3. Display the secret value with JSON formatting

**EC2 Remote Sessions:**
```bash
./swa ec2 connect
```

This will:
1. List available EC2 instances with SSM agent
2. Let you select an instance
3. Start an interactive SSM session for remote shell access

## Prerequisites

- **AWS Session Manager Plugin** must be installed for RDS port forwarding:
  - macOS: `brew install --cask session-manager-plugin`
  - Linux: Download from AWS and install .deb package
  - Windows: Download installer from AWS
- The tool stores configuration in `~/.swa/config.yaml`

## Features

- **SSO Authentication**: Interactive account and role selection
- **RDS Port Forwarding**: Connect to RDS instances through bastion hosts
- **EC2 Remote Sessions**: Connect to EC2 instances via SSM sessions
- **Secrets Manager**: List and view AWS Secrets Manager secrets
- **Configuration Management**: Simple setup with `swa config init`
- **Credential Management**: Automatic credential setup in dedicated swa profile (never overwrites default)
- **Interactive UI**: Arrow key navigation with graceful fallbacks
- **Global Options**: Region override and config file selection

## Global Options

All commands support these global flags:

```bash
# Override AWS region for any command
./swa --region us-west-2 secrets list
./swa --region eu-west-1 rds connect
./swa --region ap-southeast-1 ec2 connect

# Use alternate config file
./swa --config ~/.swa-dev/config.yaml login

# Force re-authentication
./swa login --force-auth
```

## Configuration

The configuration file is stored at `~/.swa/config.yaml`:

```yaml
default_region: us-east-1
sso:
  region: us-east-1
  start_url: https://your-org.awsapps.com/start
```

## Available Commands

- `swa login` - Authenticate with AWS SSO and select account/role
- `swa rds connect` - Connect to RDS instances via bastion hosts
- `swa ec2 connect` - Connect to EC2 instances via SSM sessions
- `swa secrets list` - List and view AWS Secrets Manager secrets
- `swa config init` - Initialize configuration file

## Development

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Clean build artifacts
make clean

# Development workflow (deps + test + build)
make dev
```