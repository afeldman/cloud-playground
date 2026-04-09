# Mirror-Cloud Guide: Ephemeral AWS for Testing

Learn how to provision temporary AWS infrastructure for testing features that LocalStack doesn't support.

## What is Mirror-Cloud?

**Mirror-Cloud** = Copy of your infrastructure on real AWS, auto-destroys after TTL.

Use cases:
- Test RDS features (AWS Batch, Aurora, EC2)
- Test real VPC networking
- Performance testing before production
- End-to-end testing with real AWS services

**Cost**: $0.50–$5 per test (scales with compute + storage)
**TTL**: Auto-destroys after 1–24h (prevents runaway costs)

## Quick Start

### 1. Configure AWS Mirror Account

Edit `.cpctl.yaml`:

```yaml
mirror:
  aws_profile: mirror-account  # AWS profile with mirror account creds
  aws_region: eu-central-1
  default_ttl: 4h              # Auto-cleanup after 4 hours
  resources:
    lambda: true
    batch: true
    rds: false               # Optional: disable RDS to save costs
    vpc: true
```

Verify AWS credentials:

```bash
aws sts get-caller-identity --profile mirror-account
# Output: Account: 123456789012, User: arn:aws:iam::123456789012:user/mirror-user
```

### 2. Bring Up Mirror-Cloud

```bash
task env:mirror:up --ttl 4h
# → Creates VPC, Batch compute env, Lambda functions
# → Auto-cleanup in 4 hours

task env:status
# Shows endpoints, resources, TTL countdown
```

Output:
```
🌐 Mirror-Cloud Environments

mirror-prod-abc123      Up (0m30s / 4h TTL)
├── VPC: vpc-a1b2c3d4
├── Subnets: 3 (AZ-redundant)
├── Batch Compute Env: batch-mirror-123
│   └── Instances: 2 (t3.medium)
├── Lambda Functions: 3 deployed
├── RDS: Not enabled (will save ~$2/h)
└── [Auto-teardown in 3h29m]
```

### 3. Deploy Your Application

Deploy the same Lambda/Batch code to Mirror-Cloud:

```bash
task lambda:deploy --stage mirror
# → Same command, but deploys to real AWS

task batch:submit --stage mirror --job process-data
# → Runs on real AWS Batch (not local K8s)

task tunnel:rds:start
# If RDS enabled, tunnel to database
```

### 4. Test

```bash
# Run end-to-end tests
task test:e2e STAGE=mirror

# Monitor AWS CloudWatch
aws logs tail /aws/batch/job --follow --profile mirror-account
```

### 5. Cleanup (Automatic)

After 4h: **cpctl automatically destroys all resources**

Or manually:

```bash
task env:mirror:down
# → Immediate cleanup (no waiting)
```

## Mirror-Cloud Architecture

### Infrastructure as Code (Terraform)

Two module flavors:

**LocalStack** (`tofu/localstack/`):
```hcl
provider "localstack" {
  endpoint_url = "http://localhost:4566"
}
resource "aws_lambda_function" "example" { ... }
```

**Mirror-Cloud** (`tofu/mirror/`):
```hcl
provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile  # mirror-account
}
resource "aws_lambda_function" "example" { ... }
# Same code, different provider!
```

Both use **identical resource definitions** (DRY principle).

### Resource Limits

To prevent runaway costs, Mirror-Cloud enforces limits:

```yaml
mirror:
  max_ttl: 24h                    # Never more than 24 hours
  max_instances: 20               # Max EC2 instances
  cost_limit: 100                 # $100 daily hard limit
```

Request override (requires approval):

```bash
task env:mirror:up --ttl 48h --approve-costs
# Needs human review before exceeding limits
```

## Scaling Mirror-Cloud

### Increase Compute Resources

```bash
task env:scale mirror --workers 5
# → Updates Batch compute env to 5 instances

task env:status mirror
# Shows new endpoint URLs
```

### Adjust Storage

Modify `.cpctl.yaml`:

```yaml
mirror:
  resources:
    lambda:
      memory: 512    # Default 128 MB
      timeout: 60    # Default 30s
```

Then redeploy:

```bash
task env:mirror:down
task env:mirror:up
```

## Comparing Local vs Mirror

| Feature | Local (Kind + LocalStack) | Mirror (Real AWS) |
|---------|---------------------------|------------------|
| **Cost** | $0 (uses Docker) | $1–5 per test |
| **Setup** | 2 min (docker-compose) | 5 min (ToFu apply) |
| **Performance** | Fast (localhost) | Realistic (cloud) |
| **Networking** | Single host | Multi-AZ VPC |
| **RDS** | EmulatedDynamoDB only | Real PostgreSQL RDS |
| **Batch** | K8s Jobs (not real Batch API) | Real AWS Batch |
| **CI/CD** | Not supported locally | Full GitHub Actions test |

**Rule of thumb**:
- **Local**: Rapid iteration (loop 1–10 min)
- **Mirror**: Pre-production validation (loop 30 min–2 h)

## Troubleshooting

### "Insufficient IAM permissions"

```bash
# Check mirror account permissions
aws iam get-user --profile mirror-account

# Required policy:
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["ec2:*", "batch:*", "lambda:*", "rds:*", "logs:*", "ssm:*"],
      "Resource": "*"
    }
  ]
}
```

### "Terraform apply failed"

```bash
# Check Terraform state
cd tofu/mirror
tofu plan -profile mirror-account

# Manually destroy + retry
tofu destroy --auto-approve
cd - && task env:mirror:up --ttl 4h
```

### "Network unreachable"

```bash
# Verify VPC + security groups
aws ec2 describe-security-groups --filters Name=vpc-id,Values=vpc-xyz

# Tunnel to Bastion host if needed
task tunnel:rds:start
```

### "Auto-cleanup didn't work"

Check PID file:

```bash
ps aux | grep cpctl
# Look for auto-cleanup goroutine

# Manual cleanup
task env:mirror:down
```

## Cost Optimization Tips

✅ **DO**:
- Use default 4h TTL (saves 93% vs 24h)
- Disable unused resources (RDS, extra instances)
- Run tests in batch (don't create 10 envs)
- Monitor CloudWatch costs weekly

❌ **DON'T**:
- Leave environments running overnight
- Create t3.xlarge instances for testing (use t3.small)
- Deploy 10 Lambda copies when 1 will do
- Use RDS multi-AZ (single AZ is cheaper)

**Estimated costs**:
- 1-hour test: $0.50–$1
- 4-hour test: $2–$5
- 24-hour test: $20–$100

## Advanced: Custom Mirror Configs

Create a custom environment profile:

```bash
cp tofu/mirror/variables.tf tofu/mirror/custom.tfvars
```

Edit `custom.tfvars`:

```hcl
aws_region = "us-west-2"
instance_count = 10
rds_enabled = true
rds_instance_class = "db.r5.xlarge"
```

Deploy:

```bash
task env:mirror:up --config custom.tfvars
```

## Monitoring

### CloudWatch Dashboard

cpctl auto-generates CloudWatch dashboard:

```bash
task dashboard:open mirror
# Opens in browser (Lambda invocations, Batch jobs, RDS connections)
```

### Cost Alerts

Setup SNS alerts in AWS:

```bash
aws sns subscribe --topic-arn arn:aws:sns:region:account:cpctl-costs \
  --protocol email --notification-endpoint your@email.com
```

Then cpctl will email you cost warnings.

## Integration with CI/CD

Automatically test against Mirror-Cloud in GitHub Actions:

```yaml
# .github/workflows/e2e-test.yml
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to Mirror-Cloud
        run: task env:mirror:up --ttl 2h --profile ci-mirror
      
      - name: Run tests
        run: task test:e2e STAGE=mirror
      
      - name: Cleanup
        if: always()
        run: task env:mirror:down
        env:
          AWS_PROFILE: ci-mirror
```

## See Also

- [Database Tunneling Guide](./TUNNEL_GUIDE.md)
- [Cloud Playground Architecture](./ARCHITECTURE.md)
- [QUICKSTART.md](./QUICKSTART.md)
- [OpenTofu Documentation](https://opentofu.org/)
