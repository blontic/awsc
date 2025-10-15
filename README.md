# AWSC (AWS Connect)

[![CI](https://github.com/blontic/awsc/actions/workflows/ci.yml/badge.svg)](https://github.com/blontic/awsc/actions/workflows/ci.yml)

A CLI tool for AWS SSO authentication, RDS port forwarding, EC2 sessions, and Secrets Manager operations.

## Features

- **SSO Authentication** - Seamless AWS SSO login with account/role selection and credential caching
- **RDS Port Forwarding** - Connect to private RDS instances and Aurora clusters with automatic bastion host discovery and security group analysis
- **EC2 Sessions** - Interactive SSH sessions via AWS Systems Manager with automatic SSM agent detection
- **Windows RDP** - Port forwarding for Windows instances with RDP protocol support
- **Secrets Manager** - View and manage AWS Secrets Manager secrets
- **Cross-Platform** - Works on macOS, Linux, and Windows without AWS CLI dependency

## Prerequisites

- **AWS Session Manager Plugin** for RDS/EC2 connections:
  - macOS: `brew install --cask session-manager-plugin`
  - Linux: Download from AWS and install .deb package
  - Windows: Download installer from AWS

## Setup

```bash
# Initial configuration
./awsc config init

# Login to AWS SSO
./awsc login
```

## Commands

All commands support both interactive selection and direct parameter access:

```bash
# SSO Authentication
./awsc login                    # Select account and role interactively
./awsc login --force           # Force browser re-authentication
./awsc login --account my-account --role my-role  # Login to specific account and role directly

# RDS Port Forwarding
./awsc rds connect             # List and select RDS instances and Aurora clusters interactively
./awsc rds connect --name my-db-instance  # Connect to specific RDS instance directly
./awsc rds connect --name "my-cluster (reader)"  # Connect to Aurora cluster reader endpoint
./awsc rds connect --name my-db-instance --local-port 5432  # Connect with custom local port

# EC2 Sessions
./awsc ec2 connect             # List and select EC2 instances for SSM session
./awsc ec2 connect --instance-id i-1234567890abcdef0  # Connect to specific instance directly
./awsc ec2 rdp                 # List and select Windows instances for RDP port forwarding
./awsc ec2 rdp --instance-id i-1234567890abcdef0     # RDP to specific Windows instance directly
./awsc ec2 rdp --instance-id i-1234567890abcdef0 --local-port 13389  # RDP with custom local port

# Secrets Manager
./awsc secrets show            # List and select secrets interactively
./awsc secrets show --name my-secret  # Show specific secret directly

# Configuration
./awsc config init             # Initial setup
./awsc config show             # Show current configuration
```

### Command Pattern

All resource commands follow a consistent pattern:

- **Interactive mode**: Run without parameters to see a list and select interactively
- **Direct mode**: Use `--name` or `--instance-id` to access resources directly
- **Fallback behavior**: If a specified resource isn't found, shows error and falls back to interactive list

## Global Options

```bash
# Override AWS region for any command
./awsc --region us-west-2 secrets show --name my-secret
./awsc --region eu-west-1 rds connect --name my-db
./awsc --region ap-southeast-1 ec2 connect --instance-id i-1234567890abcdef0
# Use alternate config file
./awsc --config ~/.awsc-dev/config.yaml login

# Enable verbose debugging output
./awsc --verbose rds connect --name my-db
./awsc -v ec2 connect

# Force re-authentication
./awsc login --force

# Direct login to specific account and role
./awsc login --account production-account --role admin-role
./awsc login --force --account dev-account --role developer-role

# Combine flags
./awsc --verbose --region us-west-2 rds connect --name production-db
./awsc --region ap-southeast-2 secrets show --name /prod/api-keys
./awsc --verbose --region us-east-1 secrets show --name /prod/database-password
```

## Configuration

Config stored at `~/.awsc/config.yaml`:

## Development

```bash
make dev
```
