# AWSC (AWS Connect)

[![CI](https://github.com/blontic/awsc/actions/workflows/ci.yml/badge.svg)](https://github.com/blontic/awsc/actions/workflows/ci.yml)

A CLI tool for AWS SSO authentication, RDS port forwarding, EC2 sessions, and Secrets Manager operations.

> 📺 **[View Demo Flows](docs/demo-flows.md)** - See terminal interactions

## Features

- **SSO Authentication** - Seamless AWS SSO login with account/role selection and credential caching
- **RDS Port Forwarding** - Connect to private RDS instances and Aurora clusters with automatic bastion host discovery and security group analysis
- **EC2 Sessions** - Interactive SSH sessions via AWS Systems Manager with automatic SSM agent detection
- **Windows RDP** - Port forwarding for Windows instances with RDP protocol support
- **OpenSearch Connections** - Connect to private OpenSearch domains via bastion hosts with automatic endpoint discovery
- **Secrets Manager** - View and manage AWS Secrets Manager secrets
- **Multi-Profile Support** - Work with multiple AWS accounts simultaneously in different terminal windows

## Prerequisites

- **Platform**: macOS or Linux (Windows users should use WSL)
- **AWS Session Manager Plugin** for RDS/EC2 connections:
  - macOS: `brew install --cask session-manager-plugin`
  - Linux: Download from AWS and install .deb package

## Installation

### Install script (macOS & Linux)

Downloads the correct binary for your OS/architecture, verifies its checksum, and installs it to a directory on your `PATH`:

```bash
curl -fsSL https://raw.githubusercontent.com/blontic/awsc/main/install.sh | sh
```

Options:

- `AWSC_VERSION=v0.4.1` — install a specific release instead of the latest.
- `AWSC_INSTALL_DIR=~/.local/bin` — install somewhere you own (no `sudo` needed).

**Updating:** re-run the same command — it installs the latest release over the existing one.

### Manual download

Grab the archive for your platform from the [latest release](https://github.com/blontic/awsc/releases/latest), extract it, and move the `awsc` binary onto your `PATH` (the binary inside the archive is already named `awsc`):

```bash
tar -xzf awsc_<version>_<os>_<arch>.tar.gz
sudo mv awsc /usr/local/bin/
```

### Homebrew

Available from the [`blontic/homebrew-tap`](https://github.com/blontic/homebrew-tap) tap (published on each release):

```bash
brew install blontic/tap/awsc
brew upgrade --cask awsc
```

> Distributed as a Homebrew Cask. The install strips the macOS quarantine flag so the unsigned binary runs without a Gatekeeper prompt.

### Build from source

Requires Go (see `go.mod` for the version). Produces `./awsc`:

```bash
make build
```

## Setup

```bash
# Initial configuration
./awsc config init

# Login to AWS SSO
./awsc login
```

## Multi-Profile Support

AWSC supports multiple AWS accounts simultaneously using a hybrid approach:

### Automatic Per-Terminal Sessions (PPID Tracking)

Each terminal window automatically tracks its own AWS session:

```bash
# Terminal 1
$ ./awsc login
# Select prod-account → Creates awsc-prod-account profile
$ ./awsc ec2 connect
# Automatically uses awsc-prod-account

# Terminal 2 (different terminal window)
$ ./awsc login
# Select dev-account → Creates awsc-dev-account profile
$ ./awsc rds connect
# Automatically uses awsc-dev-account

# Both terminals work independently!
```

### Explicit Profile Override

Use the `AWSC_PROFILE` environment variable to explicitly set which profile to use:

```bash
# Override for specific terminal session
$ export AWSC_PROFILE=awsc-prod-account
$ ./awsc ec2 connect  # Uses awsc-prod-account
$ ./awsc rds connect  # Still uses awsc-prod-account

# Switch to different profile
$ export AWSC_PROFILE=awsc-dev-account
$ ./awsc secrets show  # Now uses awsc-dev-account
```

### Automatic Error Recovery

AWSC automatically handles authentication errors:

- **Missing profile**: If the profile is deleted from `~/.aws/config`, awsc will detect it and prompt you to login again
- **Expired credentials**: When credentials expire, awsc prompts for re-authentication
- **No active session**: First-time users are automatically guided through login

All commands automatically recover from authentication errors without manual intervention.

### AWS CLI Integration

All profiles created by AWSC can be used with the AWS CLI:

```bash
$ aws s3 ls --profile awsc-prod-account
$ aws ec2 describe-instances --profile awsc-dev-account
$ aws rds describe-db-instances --profile awsc-staging-account
```

### Profile Naming

Profiles are automatically named `awsc-{accountName}` where `{accountName}` is your AWS account name. Credentials are stored in `~/.aws/config` and work until they expire.

### Platform Support

Multi-profile support works on **macOS and Linux only**. Windows users should use WSL (Windows Subsystem for Linux) for the best experience.

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
./awsc rds connect -s --name my-db  # Switch AWS account first, then connect
./awsc rds connect -l --name my-db  # List and select bastion host manually

# EC2 Sessions
./awsc ec2 connect             # List and select EC2 instances for SSM session
./awsc ec2 connect --instance-id i-1234567890abcdef0  # Connect to specific instance directly
./awsc ec2 connect -s --instance-id i-123  # Switch AWS account first, then connect
./awsc ec2 rdp                 # List and select Windows instances for RDP port forwarding
./awsc ec2 rdp --instance-id i-1234567890abcdef0     # RDP to specific Windows instance directly
./awsc ec2 rdp --instance-id i-1234567890abcdef0 --local-port 13389  # RDP with custom local port
./awsc ec2 rdp -s --instance-id i-123 --local-port 13389  # Switch account first, then RDP

# OpenSearch Connections
./awsc opensearch connect      # List and select OpenSearch domains interactively
./awsc opensearch connect --name my-domain  # Connect to specific OpenSearch domain directly
./awsc opensearch connect --name my-domain --local-port 9200  # Connect with custom local port
./awsc opensearch connect -s --name prod-domain  # Switch AWS account first, then connect
./awsc opensearch connect -l --name my-domain  # List and select bastion host manually

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
./awsc --region us-west-2 opensearch connect --name my-domain
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

# Switch AWS account before connecting (uses existing SSO session)
./awsc rds connect --switch-account --name prod-db
./awsc ec2 connect -s --instance-id i-123
./awsc ec2 rdp -s --instance-id i-456
./awsc opensearch connect -s --name prod-opensearch

# Combine flags
./awsc --verbose --region us-west-2 rds connect --name production-db
./awsc --region ap-southeast-2 secrets show --name /prod/api-keys
./awsc --verbose --region us-east-1 secrets show --name /prod/database-password
./awsc -s --region us-west-2 rds connect --name prod-db  # Switch account + region
./awsc --verbose --region eu-west-1 opensearch connect --name prod-search
```

## Configuration

Configuration is stored at `~/.awsc/config.yaml` and is created interactively by `awsc config init`. It contains three required values:

| Key | Description | Example |
| --- | --- | --- |
| `sso.start_url` | Your AWS SSO start URL (`https://<org>.awsapps.com/start`) | `https://my-org.awsapps.com/start` |
| `sso.region` | Region where AWS SSO / IAM Identity Center is configured | `us-east-1` |
| `default_region` | Default region for RDS/EC2/OpenSearch/Secrets operations (overridable with `--region`) | `us-east-1` |

Example `~/.awsc/config.yaml`:

```yaml
sso:
  start_url: https://my-org.awsapps.com/start
  region: us-east-1
default_region: us-east-1
```

Use a different config file for a command with the global `--config` flag, and view the active configuration with `awsc config show`.

## Development

```bash
make dev          # mocks + deps + test + build
make build        # build ./awsc (version injected via ldflags)
make test         # go test ./...
make vuln         # govulncheck vulnerability scan
make mocks        # regenerate internal/aws/mocks (after changing a *Client interface)
```

The only external runtime dependency is `session-manager-plugin` (used for SSM sessions and port forwarding); there is no dependency on the AWS CLI.

