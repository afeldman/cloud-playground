# MIRROR_CLOUD_GUIDE.md — Testing on Real AWS with Auto-Cleanup

## Overview
Mirror-Cloud = ephemeral AWS environment for features LocalStack can't emulate:
- RDS (real PostgreSQL on AWS, not LocalStack)
- EC2 + Auto Scaling Groups
- CloudFormation + advanced IAM policies
- Fargate (for serverless Batch jobs)
- Cost Control: TTL-based auto-teardown (default 4h)

## Part 1: Provision Mirror-Cloud

### Quick Start
```bash
# Provision 4-hour environment
cpctl env up mirror --ttl 4h

# Wait for Terraform apply (~5-10 min)
# Endpoints will be printed to stdout
```

### View Environment Status
```bash
cpctl env status
# Output:
# ✅ Environment Status
#
# 📊 Details:
#    Stage:        mirror
#    Status:       ready
#    Resources:    8
#    Tunnels:      0
#    Last Updated: 2026-04-09 12:00:00
#
# 💡 Next steps:
#    cpctl tunnel list          # View available tunnels
#    cpctl tunnel start pg      # Connect to database
#    cpctl status               # Full system status
```

## Part 2: Deploy + Test on Mirror-Cloud

### Deploy Lambda to Mirror
```bash
cpctl lambda deploy function.zip \\
  --name my-function \\
  --stage mirror \\
  --runtime python3.11 \\
  --handler index.handler \\
  --role arn:aws:iam::999999999999:role/birdy-mirror-lambda
```

### Invoke Lambda on Mirror
```bash
cpctl lambda invoke my-function \\
  --stage mirror \\
  --payload '{"test": true}'
```

### Submit Batch Job on Mirror
```bash
cpctl batch submit my-job \\
  --stage mirror \\
  --image my-repo/my-job:latest \\
  --vcpu 2 \\
  --memory 4096
```

### Connect to RDS on Mirror
```bash
# Start tunnel
cpctl tunnel start rds

# Connect
psql -h localhost -p 5433 -U admin -d postgres

# Query
SELECT version();
```

## Part 3: Scaling & Resource Management

### Scale Batch Compute
```bash
# Increase workers (Planned feature - Phase 4)
cpctl env scale mirror --workers 5 --vcpu-per-worker 4

# Check status
cpctl env status
```

### Extend TTL (Prevent Auto-Cleanup)
```bash
# Extend by 2 more hours (Planned feature - Phase 4)
cpctl env scale mirror --extend-ttl 2h

# New expiry: original-expiry + 2h
```

## Part 4: Cost Management

### Estimate Costs
```bash
# Show cost breakdown for current environment (Planned feature - Phase 4)
cpctl env cost-estimate mirror
# Output:
# ✅ Cost Estimate (4h TTL):
#   NAT Gateway (1): $0.77
#   RDS (db.t4g.micro): $3.20
#   EC2 Batch Workers (1): $0.50
#   Data Transfer: $0.12
#   ────────────────────
#   TOTAL: ~$4.59 / 4-hour window
```

### Set Budget Alert
```bash
cpctl env up mirror --ttl 4h --max-cost 10.00
# If costs exceed $10, deployment will fail (Planned feature - Phase 4)
# Prevents runaway spend
```

## Part 5: Cleanup

### Manual Teardown
```bash
cpctl env down mirror

# Confirm? (y/n): y
# Destroying all resources...
# ✅ Mirror-Cloud destroyed
```

### Automatic Cleanup (TTL Expiry)
- Timer runs in background (goroutine)
- When TTL expires → automatic `env down`
- Logs cleanup to `data/envs/mirror/.cleanup-log`

## Troubleshooting

### "AWS credentials not found"
```bash
# Set AWS_PROFILE for mirror account
export AWS_PROFILE=mirror-account
cpctl env up mirror --ttl 4h

# Or configure .cpctl.yaml
cat .cpctl.yaml
# shows: mirror.aws_profile: mirror-account
```

### "RDS creation failed"
```bash
# Check Terraform logs
cat data/envs/mirror/.tofu-apply.log

# Common: VPC quota, security group rules, IAM permissions
# Solution: increase AWS quotas or simplify environment
cpctl env down mirror
# Fix issue, then retry
```

### "Batch job stuck in SUBMITTED"
```bash
cpctl batch watch --stage mirror --job-id abc123
# Search for compute env issues
# Common: IAM role permissions, task definition can't pull image
```

## Configuration Reference

### Mirror Configuration in .cpctl.yaml
```yaml
mirror:
  enabled: true
  aws_profile: mirror-account
  default_ttl: 4h
  resources:
    rds: true
    batch: true
    lambda: true
    ec2: false  # Disable EC2 to reduce costs
```

### Environment-Specific Variables
```yaml
development:
  stage: mirror  # Default stage for commands
  auto_start_tunnels: true  # Auto-start RDS tunnel on env up
```

## Next Steps
- [TUNNEL_GUIDE](./TUNNEL_GUIDE.md) — RDS tunnel setup
- [TROUBLESHOOTING](./TROUBLESHOOTING.md) — Advanced debugging
- [ARCHITECTURE](./ARCHITECTURE.md) — Design decisions
- [QUICKSTART](./QUICKSTART.md) — Foundational setup (if available)
```