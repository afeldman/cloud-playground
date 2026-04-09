# Database Tunneling Guide

Learn how to securely access cloud databases from your local development environment using **cpctl tunnel**.

## Overview

Database tunneling creates secure port-forward connections to cloud databases without exposing them publicly.

**Two methods supported**:
- **Kubernetes Port-Forward**: For in-cluster PostgreSQL (fast, local only)
- **AWS SSM Session Manager**: For RDS, Aurora, managed databases (secure, encrypted)

## Quick Start

### Connect to Local Kubernetes PostgreSQL

```bash
# Start tunnel
task tunnel:pg:start
# → Binds localhost:5432 → K8s postgres service

# Connect with psql
psql -h localhost -p 5432 -U postgres

# Stop tunnel
task tunnel:pg:stop
```

### Connect to AWS RDS (Mirror-Cloud)

```bash
# Start Mirror-Cloud environment first
task env:mirror:up --ttl 4h

# Create RDS tunnel
task tunnel:rds:start
# → Binds localhost:5433 → AWS RDS endpoint

# Connect with psql
psql -h localhost -p 5433 -U postgres

# Clean up
task tunnel:rds:stop
task env:mirror:down
```

## Tunnel Configuration

Tunnels are defined in `.cpctl.yaml`:

```yaml
tunnels:
  pg:
    type: kubernetes          # K8s port-forward
    method: port-forward
    namespace: default
    service: postgres
    local_port: 5432
    remote_port: 5432
    auto_start: false
    
  rds:
    type: aws-ssm             # RDS via SSM
    method: ssm
    local_port: 5433
    remote_port: 5432
    remote_host: rds.mirror-cloud.aws.internal
    ssh_user: ssm-user
    auto_start: false
```

## How It Works

### Kubernetes Port-Forward

1. **cpctl** connects to Kind cluster (kubeconfig ~/.kube/config)
2. Spawns `kubectl port-forward -n <ns> svc/<service> <local>:<remote>`
3. Process PID stored in `data/tunnels/postgres.pid`
4. Keeps running in background
5. `stop` kills process gracefully

**Advantages**:
- Fast (no encryption overhead for local dev)
- Direct access to in-cluster services
- No AWS credentials needed

**Limitations**:
- K8s cluster must be running (Kind)
- Only works for services in same cluster
- Network limited to localhost

### AWS SSM Session Manager

1. **cpctl** authenticates to AWS (IAM role via `~/.aws/credentials`)
2. Creates SSM Session to EC2/RDS endpoint
3. Opens encrypted tunnel through Systems Manager
4. Binds localhost port
5. Keeps running until stopped

**Advantages**:
- Secure (encrypted in-transit)
- Works from anywhere (VPN not needed)
- EC2 instance doesn't need public IP
- Audit trail in CloudTrail

**Limitations**:
- Requires AWS credentials configured
- Requires IAM permissions (AmazonSSMManagedInstanceCore)
- EC2 instance must have SSM agent running

## Monitoring Tunnels

### List Active Tunnels

```bash
task tunnel:list
```

Output:
```
🔗 Active Tunnels:

PID    Name   Service      Local    Remote         Status
2847   pg     postgres     :5432   K8s postgres   ✅ Healthy
4521   rds    RDS          :5433   AWS RDS        ✅ Healthy
```

### Check Tunnel Health

```bash
task tunnel:status
```

Tests connectivity with TCP dial + latency check:
```
pg:   ✅ Connected (4ms)
rds:  ✅ Connected (87ms)
```

Red flags:
- ❌ Connection refused → Service not running
- ⏱️ Timeout → Network unreachable
- 🔓 Port already in use → Kill process manually

## Troubleshooting

### "Port already in use"

```bash
# Kill process using the port
lsof -i :5432
kill -9 <PID>

# Or start on alternate port
task tunnel:pg:start -- --local-port 5435
```

### "Connection refused"

```bash
# Check K8s service exists
kubectl get svc postgres

# Check RDS endpoint reachable
aws rds describe-db-instances --query 'DBInstances[0].Endpoint'
```

### "Tunnel died unexpectedly"

```bash
# Check logs
tail -f ~/.cpctl/logs/tunnels.log

# Restart
task tunnel:pg:stop
task tunnel:pg:start
```

## Advanced Usage

### SSH Tunnel (Alternative)

For manual SSH port-forward without cpctl:

```bash
ssh -i ~/.ssh/aws-key.pem -N -L 5433:rds-endpoint.aws.amazon.com:5432 ubuntu@bastion-host
```

Then connect locally:
```bash
psql -h localhost -p 5433 -U postgres
```

### Multiple Tunnels

cpctl supports opening multiple tunnels simultaneously:

```bash
task tunnel:pg:start
task tunnel:rds:start
task tunnel:list
```

Each runs in its own process, managed independently.

### Persistent Tunnels

Auto-start on `task up`:

```yaml
tunnels:
  pg:
    auto_start: true  # Start automatically with 'task up'
```

Then:
```bash
task up  # Starts Kind + LocalStack + tunnels
```

## Security Best Practices

✅ **DO**:
- Use SSM for production database access
- Rotate SSH keys regularly
- Limit tunnel lifetime with `--ttl`
- Monitor tunnel activity in CloudTrail
- Use IAM roles (not static keys)

❌ **DON'T**:
- Leave tunnels running 24/7 (close after use)
- Share RDS credentials via tunnels
- Use port-forward over untrusted networks
- Store credentials in `.cpctl.yaml` (use IAM roles)

## See Also

- [Cloud Playground Architecture](./ARCHITECTURE.md)
- [Mirror-Cloud Guide](./MIRROR_CLOUD_GUIDE.md)
- [Troubleshooting](./TROUBLESHOOTING.md)
