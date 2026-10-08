# AWSC Demo Flows

Representative terminal sessions; names and IDs are illustrative. Pickers show the
current context on their first line and disappear once you choose. Status messages go
to stderr; only `secrets show` writes to stdout (the secret value).

Once port forwarding starts, `session-manager-plugin` takes over the terminal; press
`Ctrl+C` to stop it.

## Login

```text
$ awsc login
Logging in to org "my-org". Confirm this code in the browser: LFRW-MMDP
Press Enter to open https://my-org.awsapps.com/start/#/device?user_code=LFRW-MMDP (Ctrl+C to cancel)
Waiting for approval in the browser (expires in 10 minutes)...
....
Authentication successful!

Org: my-org | Region: ap-southeast-2

Select AWS Account:

▶ production-account (123456789012)
  development-account (987654321098)

↑/↓ navigate · Enter select · type to filter · Esc clear filter / quit
```

```text
✓ Selected: production-account
✓ Selected: AdminRole

This terminal is now using production-account (123456789012) as AdminRole

To use it with the AWS CLI in this terminal:
  export AWS_PROFILE=awsc-production-account
  export AWS_REGION=ap-southeast-2
```

The browser step only appears when there is no valid SSO login for the org.

## RDS, switching account first

```text
$ awsc rds connect -s
✓ Selected: development-account
✓ Selected: DeveloperRole

This terminal is now using development-account (987654321098) as DeveloperRole

Org: my-org | Account: development-account | Role: DeveloperRole | Region: ap-southeast-2

Select RDS Instance:

▶ dev-postgres (postgres:5432)
  analytics-cluster (writer) (aurora-postgresql:5432) [Writer]
  analytics-cluster (reader) (aurora-postgresql:5432) [Reader]
```

```text
✓ Selected: dev-postgres
Using bastion: bastion-1 (i-1234567890abcdef0)
Starting port forwarding...
```

## Direct (no prompts)

```text
$ awsc rds connect --name "analytics-cluster (reader)" --local-port 5432
Connecting to RDS instance: analytics-cluster (reader)
Using bastion: bastion-1 (i-1234567890abcdef0)
Starting port forwarding...

$ awsc ec2 connect --instance-id i-1234567890abcdef0
Connecting to instance: web-server-1 (i-1234567890abcdef0)

$ awsc opensearch connect --name search-logs --local-port 9200
Connecting to OpenSearch domain: search-logs
Using bastion: bastion-1 (i-1234567890abcdef0)
Starting port forwarding...
```

## Secrets

```text
$ awsc secrets show --name /prod/api-key > key.txt
Showing secret: /prod/api-key
```

`key.txt` contains only the secret value.

## Nothing found

```text
$ awsc opensearch connect

✗ Error: no OpenSearch domains found in development-account (ap-southeast-2) as DeveloperRole; use -s to switch account or --region to change region
```
