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

```bash
# SSO Authentication
./swa login                    # Select account and role
./swa login --force           # Force browser re-authentication

# RDS Port Forwarding
./swa rds connect             # Connect to RDS via bastion host

# EC2 Sessions
./swa ec2 connect             # Start SSM session to EC2 instance

# Secrets Manager
./swa secrets list            # View AWS secrets

# Configuration
./swa config init             # Initial setup
```

## Global Options

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
