# AWSC Demo Flows

These examples show representative terminal interactions. Account names, IDs,
endpoints, and instance IDs are illustrative. Status and selection messages are
printed to stderr; only `secrets show` writes the secret value to stdout.

Once port forwarding starts, the AWS `session-manager-plugin` takes over the
terminal and prints its own connection details; press `Ctrl+C` to stop it.

## SSO Login Flow

```bash
$ awsc login
Starting SSO authentication...
Opening browser to: https://my-org.awsapps.com/start/...

Select AWS Account:
▶ production-account (123456789012)
  development-account (987654321098)
  staging-account (555666777888)

✓ Selected: production-account

Select role for production-account:
▶ AdminRole
  ReadOnlyRole
  DeveloperRole

✓ Selected: AdminRole

Successfully authenticated to production-account (123456789012) as AdminRole
This terminal is now using profile awsc-production-account automatically.

To pin this profile explicitly (e.g. in another terminal), export:
  export AWSC_PROFILE=awsc-production-account
To use it with the AWS CLI:
  aws --profile awsc-production-account <command>
```

## RDS Connect with Switch Account

```bash
$ awsc rds connect -s
Select AWS Account:
▶ production-account (123456789012)
  development-account (987654321098)

✓ Selected: production-account

Select role for production-account:
▶ AdminRole
  ReadOnlyRole

✓ Selected: AdminRole

Select RDS Instance:
▶ prod-mysql-db (mysql:3306)
  analytics-cluster (writer) (aurora-mysql:3306) [Writer]
  analytics-cluster (reader) (aurora-mysql:3306) [Reader]

✓ Selected: prod-mysql-db
Using bastion: web-server-1 (i-1234567890abcdef0)
Starting port forwarding...
```

## EC2 Connect Flow

```bash
$ awsc ec2 connect
Select EC2 Instance:
▶ web-server-1 (i-1234567890abcdef0) - Linux - running
  api-server-2 (i-0987654321fedcba0) - Linux - running
  worker-node-3 (i-abcdef1234567890) - Linux - stopped

✓ Selected: web-server-1
```

## OpenSearch Domain Connection

```bash
$ awsc opensearch connect
Select OpenSearch Domain:
▶ search-logs-prod (OpenSearch_2.3)
  analytics-dev (OpenSearch_1.3)
  metrics-staging (OpenSearch_2.5)

✓ Selected: search-logs-prod
Using bastion: web-server-prod (i-0a1b2c3d4e5f67890)
Starting port forwarding...
```

## Direct Parameter Usage

```bash
$ awsc rds connect --name "analytics-cluster (reader)" --local-port 5432
Connecting to RDS instance: analytics-cluster (reader)
✓ Selected: analytics-cluster (reader)
Using bastion: web-server-1 (i-1234567890abcdef0)
Starting port forwarding...
```

```bash
$ awsc ec2 connect --instance-id i-1234567890abcdef0
Connecting to instance: web-server-1 (i-1234567890abcdef0)
```

```bash
$ awsc opensearch connect --name search-logs-prod --local-port 9200
Connecting to OpenSearch domain: search-logs-prod
✓ Selected: search-logs-prod
Using bastion: web-server-prod (i-0a1b2c3d4e5f67890)
Starting port forwarding...
```

## Secrets Manager

`secrets show` prints status to stderr and the secret value to stdout, so it can
be redirected cleanly:

```bash
$ awsc secrets show --name /prod/api-key > key.txt
Showing secret: /prod/api-key
# key.txt now contains only the secret value
```
