# SWA (AWS backwards)

[![CI](https://github.com/blontic/swa/actions/workflows/ci.yml/badge.svg)](https://github.com/blontic/swa/actions/workflows/ci.yml)

A CLI tool for AWS SSO authentication, RDS port forwarding, EC2 sessions, and secrets management.

## Prerequisites

- **AWS Session Manager Plugin** for RDS/EC2 connections:
  - macOS: `brew install --cask session-manager-plugin`
  - Linux: Download from AWS and install .deb package
  - Windows: Download installer from AWS

## Setup

```bash
# Initial configuration
./swa config init

# Login to AWS SSO
./swa login
```

## Commands

All commands support both interactive selection and direct parameter access:

```bash
# SSO Authentication
./swa login                    # Select account and role interactively
./swa login --force           # Force browser re-authentication
./swa login --account my-account --role my-role  # Login to specific account and role directly

# RDS Port Forwarding
./swa rds connect             # List and select RDS instances interactively
./swa rds connect --name my-db-instance  # Connect to specific RDS instance directly

# EC2 Sessions
./swa ec2 connect             # List and select EC2 instances for SSM session
./swa ec2 connect --instance-id i-1234567890abcdef0  # Connect to specific instance directly
./swa ec2 rdp                 # List and select Windows instances for RDP port forwarding
./swa ec2 rdp --instance-id i-1234567890abcdef0     # RDP to specific Windows instance directly

# Secrets Manager
./swa secrets show            # List and select secrets interactively
./swa secrets show --name my-secret  # Show specific secret directly

# CloudWatch Logs
./swa logs tail               # List and select log group to tail interactively
./swa logs tail --group /aws/lambda/my-function  # Tail specific log group directly
./swa logs tail --group /aws/lambda/my-function --since 5m   # Show logs from last 5 minutes
./swa logs tail --group /aws/lambda/my-function --follow     # Follow log output continuously
./swa logs tail --group /aws/lambda/my-function --since 1h --follow  # Show last hour and follow

# Configuration
./swa config init             # Initial setup
./swa config show             # Show current configuration
```

### Command Pattern

All resource commands follow a consistent pattern:
- **Interactive mode**: Run without parameters to see a list and select interactively
- **Direct mode**: Use `--name` or `--instance-id` to access resources directly
- **Fallback behavior**: If a specified resource isn't found, shows error and falls back to interactive list

## Global Options

```bash
# Override AWS region for any command
./swa --region us-west-2 secrets show --name my-secret
./swa --region eu-west-1 rds connect --name my-db
./swa --region ap-southeast-1 ec2 connect --instance-id i-1234567890abcdef0
./swa --region us-west-2 logs tail --group /aws/lambda/my-function

# Use alternate config file
./swa --config ~/.swa-dev/config.yaml login

# Enable verbose debugging output
./swa --verbose rds connect --name my-db
./swa -v ec2 connect

# Force re-authentication
./swa login --force

# Direct login to specific account and role
./swa login --account production-account --role admin-role
./swa login --force --account dev-account --role developer-role

# Combine flags
./swa --verbose --region us-west-2 rds connect --name production-db
./swa --region ap-southeast-2 secrets show --name /prod/api-keys
./swa --verbose --region us-east-1 logs tail --group /aws/ecs/my-service --since 30m
```

## Configuration

Config stored at `~/.swa/config.yaml`:

```yaml
default_region: us-east-1
sso:
  region: us-east-1
  start_url: https://your-org.awsapps.com/start
```

## Development

```bash
make dev
```
