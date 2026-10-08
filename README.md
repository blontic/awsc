# AWSC (AWS Connect)

[![CI](https://github.com/blontic/awsc/actions/workflows/ci.yml/badge.svg)](https://github.com/blontic/awsc/actions/workflows/ci.yml)

A CLI for AWS SSO login, RDS/OpenSearch port forwarding, EC2 sessions and RDP, and Secrets Manager.

> 📺 **[View Demo Flows](docs/demo-flows.md)** - See terminal interactions

## Features

- **SSO login** - Pick an account and role; works with multiple orgs (IAM Identity Center start URLs)
- **RDS** - Port forward to private RDS instances and Aurora clusters through an automatically chosen bastion
- **EC2** - Shell sessions via SSM, and RDP port forwarding to Windows instances
- **OpenSearch** - Port forward to private OpenSearch domains through a bastion
- **Secrets Manager** - Find and show secrets
- **Per-terminal accounts** - Different terminals can use different accounts and orgs at the same time

No access keys are stored: awsc writes standard AWS SSO profiles that also work with the AWS CLI.

## Prerequisites

- macOS or Linux (on Windows, use WSL)
- [AWS Session Manager Plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) for RDS, EC2 and OpenSearch connections (macOS: `brew install --cask session-manager-plugin`)

## Installation

### Homebrew (macOS & Linux)

```bash
brew install blontic/tap/awsc
```

Update with `brew upgrade --cask awsc`. The cask removes the macOS quarantine flag so the unsigned binary runs without a Gatekeeper prompt.

### Install script (macOS & Linux)

Downloads the right binary for your OS and architecture, verifies its checksum, and installs it on your `PATH`. Re-run to update.

```bash
curl -fsSL https://raw.githubusercontent.com/blontic/awsc/main/install.sh | sh
```

Set `AWSC_VERSION=v0.4.1` to install a specific release, or `AWSC_INSTALL_DIR=~/.local/bin` to install without `sudo`.

### Build from source

Requires Go (version in `go.mod`):

```bash
make build   # produces ./awsc
```

Or download an archive from the [latest release](https://github.com/blontic/awsc/releases/latest).

## Setup

```bash
awsc config add   # SSO start URL and regions (also prompted on first use)
awsc login        # pick an account and role
```

## Shell Completions

Homebrew installs completions automatically. Otherwise:

```bash
source <(awsc completion zsh)                                          # zsh, current shell
awsc completion zsh > "$(brew --prefix)/share/zsh/site-functions/_awsc" # zsh, persisted
awsc completion bash | sudo tee /etc/bash_completion.d/awsc > /dev/null # bash, persisted
```

## Commands

Every command can be run interactively (pick from a list) or directly with flags, which makes it scriptable. A name that doesn't exist is an error that says which account and region were searched.

```bash
# Login
awsc login                                  # pick account and role
awsc login --account my-account --role Admin
awsc login --force                          # new browser login
awsc logout                                 # end the org's SSO session and clear this terminal
awsc logout --all                           # every org and every terminal

# RDS
awsc rds connect                            # pick an instance or Aurora endpoint
awsc rds connect --name my-db --local-port 5432
awsc rds connect --name "my-cluster (reader)"
awsc rds connect -l --name my-db            # pick the bastion yourself

# EC2
awsc ec2 connect                            # SSM shell session
awsc ec2 connect --instance-id i-1234567890abcdef0
awsc ec2 rdp --instance-id i-1234567890abcdef0 --local-port 13389

# OpenSearch
awsc opensearch connect --name my-domain --local-port 9200

# Secrets Manager (the value is the only output on stdout)
awsc secrets show
awsc secrets show --name my-secret > secret.txt

# Configuration
awsc config list                            # * = default
awsc config add [org]
awsc config use <org>                       # set default and switch this terminal
awsc config show [org]
awsc config remove <org>
```

### Options

| Flag | Applies to | Description |
| --- | --- | --- |
| `-s`, `--switch-account` | `rds`, `ec2`, `opensearch`, `secrets` | Pick another account/role first |
| `--region <region>` | all | Override the region for this command |
| `--org <name>` | all | Use another org (logging in switches this terminal to it) |
| `-v`, `--verbose` | all | Debug output |

| Environment variable | Description |
| --- | --- |
| `AWSC_ORG` | Org to use |
| `AWSC_PROFILE` | Profile to use instead of the terminal's session |

## Accounts, terminals and profiles

Each terminal remembers its own org, account and role, so different windows can work in different accounts at once. A terminal can switch with `awsc login` or `-s` at any time.

Logging in writes an SSO profile to `~/.aws/config` named `awsc-<account>` (or `awsc-<org>-<account>` if another org has an account with the same name), so you can also use it directly:

```bash
aws s3 ls --profile awsc-my-account
```

awsc keeps its `awsc-*` sections in `~/.aws/config` in sync with its orgs and never touches anything else in that file.

## Configuration

`~/.awsc/config.yaml` holds one or more orgs:

```yaml
default_org: my-org
orgs:
  my-org:
    sso:
      start_url: https://my-org.awsapps.com/start
      region: us-east-1        # IAM Identity Center region
    default_region: us-east-1  # region for RDS/EC2/OpenSearch/Secrets (override with --region)
  partner:
    sso:
      start_url: https://partner.awsapps.com/start
      region: eu-west-1
    default_region: eu-west-1
```

The org is chosen by: `--org`, then `AWSC_ORG`, then the org this terminal last used, then `default_org`.

**Upgrading from awsc 0.5 or earlier:** the old config is converted automatically (backup at `config.yaml.bak`) and you log in once. The `--config` flag has been replaced by orgs (`awsc config add`, `--org`).

## Development

```bash
make dev     # mocks + deps + test + build
make build   # build ./awsc (version injected via ldflags)
make test    # go test ./...
make vuln    # govulncheck
make check-mod  # go.mod/go.sum are tidy and verified (CI and releases fail otherwise)
make mocks   # regenerate internal/aws/mocks after changing a *Client interface
```

The only runtime dependency is `session-manager-plugin`; awsc does not need the AWS CLI.
