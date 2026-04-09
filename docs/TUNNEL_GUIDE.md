# TUNNEL_GUIDE.md — Database Access Without Public IPs

## Overview
Two tunnel types for database access:
- **Kubernetes Port-Forward** (local K8s PostgreSQL)
- **AWS SSM Session Manager** (remote RDS, requires AWS credentials)

## Part 1: Kubernetes Port-Forward Tunnel (LocalStack)

### Start PostgreSQL Tunnel
```bash
# Start tunnel
cpctl tunnel start pg

# Status
cpctl tunnel status
# Output:
# ✅ pg: localhost:5432 → postgres:5432 (Kind) [PID: 12345]
```

### Connect with psql
```bash
# Default credentials from Kind manifest
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d cloud_playground

# Or via .pgpass file
echo "localhost:5432:cloud_playground:postgres:postgres" > ~/.pgpass
chmod 600 ~/.pgpass
psql -h localhost -p 5432 -U postgres -d cloud_playground
```

### Use in Application Code
```go
// Go example
connStr := "postgres://postgres:postgres@localhost:5432/cloud_playground"
db, err := sql.Open("postgres", connStr)
```

### Stop Tunnel
```bash
cpctl tunnel stop pg
```

## Part 2: AWS SSM Session Manager Tunnel (RDS on Mirror-Cloud)

### Prerequisites
- Mirror-Cloud environment: `cpctl env up mirror --ttl 4h`
- AWS credentials configured: `aws configure`
- Target RDS security group allows SSM access

### Start RDS Tunnel
```bash
# Start tunnel
cpctl tunnel start rds

# Status
cpctl tunnel status
# Output:
# ✅ rds: localhost:5433 → rds-instance.xxx.amazonaws.com:5432 (SSM) [PID: 12346]
```

### Connect to Mirror-Cloud RDS
```bash
# Get RDS password from AWS Secrets Manager
aws secretsmanager get-secret-value --secret-id birdy-mirror-rds-password \\
  --query 'SecretString' --output text

# Connect via tunnel
psql -h localhost -p 5433 -U admin -d postgres
```

## Part 3: Multiple Tunnels & Health Checks

### List Active Tunnels
```bash
cpctl tunnel list
# Output:
# Active Tunnels:
# ┌──────────────┬──────┬────────────┬──────────────────┬─────────────┐
# │ Name         │ PID  │ Local Port │ Remote Host      │ Remote Port │
# ├──────────────┼──────┼────────────┼──────────────────┼─────────────┤
# │ pg           │ 12345│ 5432       │ postgres         │ 5432        │
# │ rds          │ 12346│ 5433       │ rds.amazonaws.com│ 5432        │
# └──────────────┴──────┴────────────┴──────────────────┴─────────────┘
```

### Health Check All Tunnels
```bash
cpctl tunnel status
# Shows connection status + latest error for each tunnel
```

### Troubleshooting Tunnel Failures

#### "Connection refused"
```bash
# 1. Check if tunnel process is running
ps aux | grep cpctl

# 2. Check Kind cluster health
kubectl get nodes -A

# 3. Check pod status
kubectl get pods -n default | grep postgres

# 4. Restart tunnel
cpctl tunnel stop pg
cpctl tunnel start pg
```

#### "Access denied" (RDS)
```bash
# 1. Verify SSM permissions
aws sts get-caller-identity

# 2. Check RDS security group allows SSM + EC2
aws ec2 describe-security-groups --group-ids sg-xxxxx

# 3. Test SSM Session directly
aws ssm start-session --target i-xxxxx
```

## Part 4: Advanced: Custom Tunnels

### Define Custom Tunnel in .cpctl.yaml
```yaml
tunnels:
  custom-db:
    type: ssm
    method: ssm-session-manager
    namespace: default
    remote_host: my-rds.xxxxx.amazonaws.com
    remote_port: 5432
    local_port: 5434
    auto_start: false
```

### Start Custom Tunnel
```bash
cpctl tunnel start custom-db
psql -h localhost -p 5434 -U admin
```

## Configuration Reference

### Kubernetes Tunnel
```yaml
tunnels:
  pg:
    type: kubernetes
    method: port-forward
    namespace: default
    service: postgres
    remote_port: 5432
    local_port: 5432
    auto_start: true
```

### SSM Tunnel
```yaml
tunnels:
  rds:
    type: ssm
    method: ec2-instance
    remote_host: bastion.internal
    remote_port: 3306
    local_port: 3307
    ssh_user: ec2-user
    auto_start: false
```

## Next Steps
- [MIRROR_CLOUD_GUIDE](./MIRROR_CLOUD_GUIDE.md) — Provision environments for testing
- [TROUBLESHOOTING](./TROUBLESHOOTING.md) — Common connection issues
- [QUICKSTART](./QUICKSTART.md) — Foundational setup (if available)
- [ARCHITECTURE](./ARCHITECTURE.md) — Design decisions behind tunnel implementation
```