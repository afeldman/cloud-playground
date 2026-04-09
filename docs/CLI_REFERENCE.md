# CLI_REFERENCE — Complete cpctl Command Reference

## Command Structure

```
cpctl [global-flags] <command> [subcommand] [flags] [args]
```

### Global Flags
```
  -v, --verbose         Enable verbose logging (debug level)
  -q, --quiet           Suppress non-essential output
      --config string   Path to .cpctl.yaml (default: ./.cpctl.yaml)
      --json           Output as JSON (for scripting)
      --no-color       Disable colored output
  -h, --help            Show help for command
```

---

## Command Overview

| Command | Purpose | Main Subcommands |
|---------|---------|-----------------|
| `env` | Manage environments (Kind + LocalStack) | up, down, status, logs |
| `tunnel` | Port-forward services | start, stop, ls, status, logs |
| `lambda` | Deploy & invoke Lambda functions | deploy, invoke, logs, ls |
| `batch` | Submit & monitor Batch jobs | submit, status, logs, wait |
| `obs` | Observability tools | metrics, logs, traces |
| `ai` | LLM-powered debugging | ask, explain, suggest |
| `status` | Overall system status | (shows TUI dashboard) |
| `config` | Manage configuration | show, validate, init |
| `version` | Show version | - |

---

## env — Environment Lifecycle

### env up — Start Environment

**Start local Kind cluster + LocalStack**:
```bash
cpctl env up

# With custom cluster name (default: cpctl-local)
cpctl env up --name my-cluster

# With verbose logging
cpctl env up -v

# Skip infrastructure checks (unsafe)
cpctl env up --skip-checks
```

**Start ephemeral AWS environment (Mirror-Cloud)**:
```bash
cpctl env up mirror

# With custom TTL (auto-cleanup)
cpctl env up mirror --ttl 4h

# Specific AWS profile + region
cpctl env up mirror --profile staging --region us-west-2

# Custom resource sizing
cpctl env up mirror --instance-type t3.large --rds-size db.t3.medium
```

**Examples**:
```bash
# One-liner: local + LocalStack
cpctl env up

# Development (real AWS for 8 hours, then auto-cleanup)
cpctl env up mirror --ttl 8h --profile dev

# CI/CD (ephemeral, minimal resources)
cpctl env up --name ci-test --skip-checks
```

### env down — Destroy Environment

**Destroy local environment**:
```bash
cpctl env down

# Force destroy (ignore errors)
cpctl env down --force

# Destroy only infrastructure (keep data)
cpctl env down --keep-data

# Dry-run (show what would be deleted)
cpctl env down --dry-run
```

**Destroy Mirror-Cloud environment**:
```bash
cpctl env down mirror

# Specific env by ID
cpctl env down mirror --env-id mirror-xyz123
```

### env status — Check Status

```bash
cpctl env status

# Output:
# ╔───────────────────────────────────────╗
# ║ Environment Status                    ║
# ╠───────────────────────────────────────╣
# ║ Local (Kind)                    ✓     ║
# ║   Cluster: cpctl-local                ║
# ║   Nodes: 3 ready                      ║
# ║   Resources: 4 CPU, 8 GB RAM          ║
# ║                                       ║
# ║ LocalStack                      ✓     ║
# ║   Endpoint: 127.0.0.1:4566            ║
# ║   Services: S3, SQS, DynamoDB, etc    ║
# ║                                       ║
# ║ Observability                   ✓     ║
# ║   Prometheus: localhost:9090          ║
# ║   Grafana: localhost:3000             ║
# ║   Loki: localhost:3100                ║
# ║                                       ║
# ║ Next: cpctl status (full dashboard)   ║
# ╚───────────────────────────────────────╝
```

### env logs — View Environment Logs

```bash
# Last 100 lines
cpctl env logs

# Follow logs
cpctl env logs -f

# Specific component
cpctl env logs --component localstack
cpctl env logs --component kind

# Since timestamp
cpctl env logs --since "2024-01-15 10:30:00"

# Save to file
cpctl env logs > env.log
```

---

## tunnel — Port-Forward & Remote Access

### tunnel start — Create Tunnel

**Start single tunnel**:
```bash
cpctl tunnel start postgres

# Custom local port
cpctl tunnel start postgres --local-port 15432

# Custom remote port
cpctl tunnel start postgres --remote-port 3306

# With TTL (auto-close)
cpctl tunnel start postgres --ttl 1h

# SSM tunnel to AWS
cpctl tunnel start bastion --type ssm
```

**Start multiple tunnels**:
```bash
cpctl tunnel start postgres redis app-debug

# Start all tunnels configured in .cpctl.yaml
cpctl tunnel start --all
```

### tunnel stop — Destroy Tunnel

```bash
cpctl tunnel stop postgres

# Stop all tunnels
cpctl tunnel stop --all

# Force stop (don't wait for graceful shutdown)
cpctl tunnel stop postgres --force
```

### tunnel ls — List Tunnels

```bash
cpctl tunnel ls

# Output:
# ┌──────────────┬─────────┬──────────┬──────────────────────┐
# │ Name         │ Type    │ Status   │ Local Address        │
# ├──────────────┼─────────┼──────────┼──────────────────────┤
# │ postgres     │ k8s     │ active   │ 127.0.0.1:5432       │
# │ redis        │ k8s     │ active   │ 127.0.0.1:6379       │
# │ app-debug    │ k8s     │ active   │ 127.0.0.1:3000       │
# │ bastion      │ ssm     │ inactive │ -                    │
# └──────────────┴─────────┴──────────┴──────────────────────┘
```

### tunnel status — Detailed Tunnel Info

```bash
cpctl tunnel status postgres

# Output:
# Name: postgres
# Type: Kubernetes port-forward
# Status: active (running for 23m)
# Local: 127.0.0.1:5432
# Remote: default/postgres:5432
# PID: 12456
# Last error: -
```

### tunnel logs — View Tunnel Logs

```bash
cpctl tunnel logs postgres

# Follow logs
cpctl tunnel logs postgres -f

# Last N lines
cpctl tunnel logs postgres --tail 50

# Since timestamp
cpctl tunnel logs postgres --since 10m
```

---

## lambda — Serverless Functions

### lambda deploy — Deploy Function

**Deploy function**:
```bash
cpctl lambda deploy function.zip --name my-func

# To LocalStack
cpctl lambda deploy function.zip --name my-func --stage localstack

# To Mirror-Cloud
cpctl lambda deploy function.zip --name my-func --stage mirror

# With custom handler
cpctl lambda deploy function.zip --name my-func --handler index.handler

# With environment variables
cpctl lambda deploy function.zip --name my-func --env FOO=bar,BAZ=qux

# With IAM role
cpctl lambda deploy function.zip --name my-func --role arn:aws:iam::xxx:role/my-role

# With timeout
cpctl lambda deploy function.zip --name my-func --timeout 30

# With memory
cpctl lambda deploy function.zip --name my-func --memory 512

# Verbose output
cpctl lambda deploy function.zip --name my-func -v
```

### lambda invoke — Call Function

```bash
# Simple invocation
cpctl lambda invoke my-func

# With payload
cpctl lambda invoke my-func --payload '{"key": "value"}'

# From file
cpctl lambda invoke my-func --payload-file payload.json

# Async invocation
cpctl lambda invoke my-func --async

# With specific stage
cpctl lambda invoke my-func --stage mirror

# Get full response (including logs)
cpctl lambda invoke my-func --full
```

### lambda logs — View Logs

```bash
cpctl lambda logs my-func

# Follow logs
cpctl lambda logs my-func -f

# Last N lines
cpctl lambda logs my-func --tail 50

# Since time
cpctl lambda logs my-func --since 5m

# Filter by request ID
cpctl lambda logs my-func --request-id xyz123
```

### lambda ls — List Functions

```bash
cpctl lambda ls

# Specific stage
cpctl lambda ls --stage mirror

# JSON output (for scripting)
cpctl lambda ls --json

# Filter by pattern
cpctl lambda ls --filter "my-*"
```

---

## batch — Batch Job Processing

### batch submit — Submit Job

```bash
# Simple job (default job definition)
cpctl batch submit --container-image myimage:latest

# Named job
cpctl batch submit --container-image myimage:latest --job-name data-proc-1

# With command
cpctl batch submit --container-image myimage:latest --command "python process.py"

# With arguments
cpctl batch submit --container-image myimage:latest --args "arg1,arg2,arg3"

# With environment variables
cpctl batch submit --container-image myimage:latest --env FOO=bar,BAZ=qux

# With resource requests
cpctl batch submit --container-image myimage:latest --vcpus 4 --memory 8192

# To specific queue
cpctl batch submit --container-image myimage:latest --job-queue high-priority

# With timeout
cpctl batch submit --container-image myimage:latest --timeout 1h

# Array job (run 100 tasks)
cpctl batch submit --container-image myimage:latest --array-size 100
```

### batch status — Check Job Status

```bash
cpctl batch status job-12345

# If no ID shown, list all jobs
cpctl batch status

# Specific state filter
cpctl batch status --state RUNNING
cpctl batch status --state SUCCEEDED
cpctl batch status --state FAILED

# Watch updates
cpctl batch status job-12345 --follow
```

### batch logs — View Job Logs

```bash
cpctl batch logs job-12345

# Follow logs
cpctl batch logs job-12345 -f

# Specific array task
cpctl batch logs job-12345 --task 42

# Last N lines
cpctl batch logs job-12345 --tail 100

# All logs to file
cpctl batch logs job-12345 > job.log
```

### batch wait — Wait for Job Completion

```bash
# Wait until job completes (any state)
cpctl batch wait job-12345

# Wait with timeout
cpctl batch wait job-12345 --timeout 1h

# Show progress
cpctl batch wait job-12345 --progress

# Exit code: 0 if succeeded, non-zero if failed
cpctl batch wait job-12345
echo $?  # Check exit code
```

---

## obs — Observability

### obs metrics — Query Metrics

```bash
cpctl obs metrics

# Shows interactive dashboard with:
# - Pod CPU/memory usage
# - HTTP request latency
# - Error rates
# - Custom application metrics

# Query specific metric
cpctl obs metrics --query 'rate(http_requests_total[5m])'

# Export metrics to Prometheus format
cpctl obs metrics --export
```

### obs logs — View Application Logs

```bash
cpctl obs logs

# Specific pod
cpctl obs logs -n default -p app-pod-1

# All pods in namespace
cpctl obs logs -n default

# Follow logs
cpctl obs logs -n default -f

# Specific time range
cpctl obs logs --since 1h --until 30m
```

### obs traces — View Distributed Traces

```bash
cpctl obs traces

# Specific service
cpctl obs traces --service app

# Specific operation
cpctl obs traces --service app --operation /api/users

# Error traces only
cpctl obs traces --service app --errors-only

# Export as JSON
cpctl obs traces --export traces.json
```

---

## ai — LLM-Powered Debugging

### ai ask — Ask Question About System

```bash
cpctl ai ask "Why is the Lambda function failing?"

# With context
cpctl ai ask "Debug this error" --context logs

# Specific resource
cpctl ai ask "What's wrong with this pod?" --pod app-pod-1

# Get explanation + fix suggestion
cpctl ai ask "Why is my app hanging?" --explain --suggest-fix
```

### ai explain — Explain Error

```bash
cpctl ai explain "error message here"

# From logs
cpctl ai explain --from logs

# From specific pod
cpctl ai explain --from pod app-pod-1

# Show fix suggestions
cpctl ai explain --suggest-fix
```

### ai suggest — Get Suggestions

```bash
cpctl ai suggest "optimize this Lambda"

# Specific resource
cpctl ai suggest --pod app-pod-1

# Performance optimization
cpctl ai suggest --pod app-pod-1 --type performance

# Cost optimization
cpctl ai suggest --type cost
```

---

## status — System Dashboard

```bash
cpctl status

# Opens interactive TUI showing:
# ┌─────────────────────────────────────────┐
# │ 📊 CLOUD-PLAYGROUND STATUS              │
# ├─────────────────────────────────────────┤
# │ Environment: ✓ HEALTHY                  │
# │ │ Kind Cluster: 3/3 nodes ready         │
# │ │ Containers: 12 running, 0 failed      │
# │ │ LocalStack: ✓ All services ready      │
# │                                         │
# │ Tunnels: 2 active                       │
# │ │ postgres → 127.0.0.1:5432             │
# │ │ redis → 127.0.0.1:6379                │
# │                                         │
# │ Applications: 5 ready                   │
# │ │ app (3 replicas, all healthy)         │
# │ │ batch-worker (1 replica, healthy)     │
# │                                         │
# │ Observability: ✓ All systems up         │
# │ │ Prometheus: 1234 time-series          │
# │ │ Grafana: 8 dashboards                 │
# │                                         │
# │ Recent Errors: None                     │
# └─────────────────────────────────────────┘

# Refresh dashboard (default 2s)
cpctl status --refresh 5s

# Non-interactive JSON output
cpctl status --json
```

---

## config — Configuration Management

### config show — Display Configuration

```bash
cpctl config show

# Specific section
cpctl config show --section tunnels
cpctl config show --section environments

# As JSON
cpctl config show --json

# Validate current config
cpctl config validate
```

### config init — Initialize Configuration

```bash
cpctl config init

# Interactive setup wizard:
# What's your project name? → cloud-playground
# Docker daemon socket path? → /var/run/docker.sock
# Kubernetes context? → kind-cpctl-local
# ... (more questions)

# Creates/updates .cpctl.yaml
```

---

## version — Show Version

```bash
cpctl version

# Output:
# cpctl version 1.0.0
# Go version: go1.21.0
# Built: 2024-01-15T10:30:45Z
# Commit: abc123def456

# Check for updates
cpctl version --check-updates
```

---

## Examples

### Example 1: Quick Start

```bash
# Start everything
cpctl env up

# Forward PostgreSQL
cpctl tunnel start postgres

# Connect with psql
psql -h localhost -p 5432 -U postgres

# Stop when done
cpctl env down
```

### Example 2: Lambda Development

```bash
# Start env
cpctl env up

# Deploy function
cpctl lambda deploy function.zip --name my-func --stage localstack

# Invoke
cpctl lambda invoke my-func --payload '{"x": 5}'

# View logs
cpctl lambda logs my-func -f

# Redeploy after code change
cpctl lambda deploy function.zip --name my-func --stage localstack
```

### Example 3: Batch Processing

```bash
# Start env
cpctl env up

# Forward tunnels for monitoring
cpctl tunnel start prometheus grafana

# Submit batch job
cpctl batch submit --container-image myimage:latest --job-name data-proc-1

# Monitor job
cpctl batch status --watch

# View logs when done
cpctl batch logs job-12345

# Open Grafana to see performance
# → http://localhost:3000/grafana
```

### Example 4: Multi-Stage Testing

```bash
# Local testing
cpctl env up --name local
cpctl lambda deploy function.zip --name my-func --stage localstack
cpctl lambda invoke my-func --stage localstack
cpctl env down

# Staging testing (real AWS)
cpctl env up mirror --ttl 8h --profile staging
cpctl lambda deploy function.zip --name my-func --stage mirror
cpctl lambda invoke my-func --stage mirror
cpctl env down mirror  # Auto-cleanup or manual

# Production (separate workflow)
# → Use terraform/CDK directly
```

---

See also:
- [LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md) — Workflows
- [TUNNEL_GUIDE.md](./TUNNEL_GUIDE.md) — Tunnel details
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) — Common issues
