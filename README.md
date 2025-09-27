# SWA (AWS backwards)

[![CI](https://github.com/blontic/swa/actions/workflows/ci.yml/badge.svg)](https://github.com/blontic/swa/actions/workflows/ci.yml)

A Go-based CLI tool for AWS operations including SSO authentication, account switching, and RDS port forwarding.

## Features

- **SSO Authentication**: Interactive account and role selection with force re-auth option
- **RDS Port Forwarding**: Connect to RDS instances through bastion hosts with automatic discovery
- **EC2 Remote Sessions**: Connect to EC2 instances via SSM sessions with agent detection
- **Secrets Manager**: List and view AWS Secrets Manager secrets with JSON formatting
- **AWS Context Display**: Shows current account, role, and region at start of each command
- **Configuration Management**: Simple setup with `swa config init`
- **Credential Management**: Automatic credential setup in dedicated swa profile (never overwrites default)
- **Interactive UI**: Arrow key navigation with graceful fallbacks

## Prerequisites

- **AWS Session Manager Plugin** must be installed for RDS port forwarding:
  - macOS: `brew install --cask session-manager-plugin`
  - Linux: Download from AWS and install .deb package
  - Windows: Download installer from AWS

## Usage

### First Time Setup

1. Configure SSO:

```bash
./swa config init
```

This will prompt you for:

- SSO Start URL (e.g., https://your-org.awsapps.com/start)
- SSO Region (e.g., us-east-1)
- Default AWS Region (e.g., us-east-1)

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

## Global Options

All commands support these global flags:

```bash
# Override AWS region for any command
./swa --region us-west-2 secrets list
./swa --region eu-west-1 rds connect
./swa --region ap-southeast-1 ec2 connect

# Use alternate config file
./swa --config ~/.swa-dev/config.yaml login

# Enable verbose debugging output
./swa --verbose rds connect
./swa -v ec2 connect

# Force re-authentication
./swa login --force

# Combine flags
./swa --verbose --region us-west-2 rds connect
```

## Configuration

The configuration file is stored at `~/.swa/config.yaml`:

```yaml
default_region: us-east-1
sso:
  region: us-east-1
  start_url: https://your-org.awsapps.com/start
```

## Local Development

1. Install dependencies and build locally:

```bash
make dev
```
