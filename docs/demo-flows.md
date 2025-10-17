# AWSC Demo Flows

## SSO Login Flow

```bash
$ awsc login
Starting SSO authentication...
Opening browser for SSO authentication...

Select AWS Account:
▶ production-account (123456789012)
  development-account (987654321098)
  staging-account (555666777888)

Selected: production-account

Select role for production-account:
▶ AdminRole
  ReadOnlyRole
  DeveloperRole

Selected: AdminRole
Assumed role AdminRole in account production-account
Credentials written to AWS config profile 'awsc'
```

## RDS Connect with Switch Account

```bash
$ awsc rds connect -s
Select AWS Account:
▶ production-account (123456789012)
  development-account (987654321098)

Selected: production-account

Select role for production-account:
▶ AdminRole
  ReadOnlyRole

Selected: AdminRole

Select RDS Instance:
▶ prod-mysql-db (mysql:3306)
  analytics-cluster (writer) (aurora-mysql:3306) [Writer]
  analytics-cluster (reader) (aurora-mysql:3306) [Reader]

Selected: prod-mysql-db
Using bastion: web-server-1
Starting port forwarding via i-1234567890abcdef0...

Local port forwarding: localhost:3306 -> prod-mysql-db.cluster-xyz.us-east-1.rds.amazonaws.com:3306
Press Ctrl+C to stop...
```

## EC2 Connect Flow

```bash
$ awsc ec2 connect
Select EC2 Instance:
▶ web-server-1 (i-1234567890abcdef0) - Linux - running
  api-server-2 (i-0987654321fedcba0) - Linux - running
  worker-node-3 (i-abcdef1234567890) - Linux - stopped

Selected: web-server-1

Starting session with i-1234567890abcdef0...

Starting session with SessionId: user-0123456789abcdef0
sh-4.2$ 
```

## OpenSearch Domain Connection

```bash
$ awsc opensearch connect
Select OpenSearch Domain:
▶ search-logs-prod (OpenSearch_2.3)
  analytics-dev (OpenSearch_1.3)
  metrics-staging (OpenSearch_2.5)

Selected: search-logs-prod
Using bastion: web-server-prod
Starting port forwarding via i-0a1b2c3d4e5f67890...

Local port forwarding: localhost:443 -> vpc-search-logs-prod-xyz.us-east-1.es.amazonaws.com:443
Press Ctrl+C to stop...
```


## Direct Parameter Usage

```bash
$ awsc rds connect --name "analytics-cluster (reader)" --local-port 5432
Selected: analytics-cluster (reader)
Using bastion: web-server-1
Starting port forwarding via i-1234567890abcdef0...

Local port forwarding: localhost:5432 -> analytics-cluster.cluster-ro-xyz.us-east-1.rds.amazonaws.com:3306
Press Ctrl+C to stop...
```

```bash
$ awsc ec2 connect --instance-id i-1234567890abcdef0
Selected: web-server-1

Starting session with i-1234567890abcdef0...
Starting session with SessionId: user-0123456789abcdef0
sh-4.2$ 
```

```bash
$ awsc opensearch connect --name search-logs-prod --local-port 9200
Selected: search-logs-prod
Using bastion: web-server-prod
Starting port forwarding via i-0a1b2c3d4e5f67890...

Local port forwarding: localhost:9200 -> vpc-search-logs-prod-xyz.us-east-1.es.amazonaws.com:443
Press Ctrl+C to stop...
```