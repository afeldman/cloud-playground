# Local Development

Single-tool workflow powered by `cpctl`.

## 1. Build cpctl (once)

```bash
cd cli/cpctl && go build -o ../../cpctl . && cd ../..
# or just use: go run ./cli/cpctl <command>
```

## 2. Start everything

```bash
cpctl up
```

Starts Kind cluster (`birdy-playground`) + LocalStack, waits for readiness, runs Terraform, then shows a step-by-step progress view:

```
Starting playground

  ✓  Removing previous cluster
  ✓  Creating Kind cluster
  ✓  Starting LocalStack
  ✓  Waiting for LocalStack
  ✓  Terraform init
  ✓  Terraform apply

Playground is up:
  Kind cluster  : birdy-playground
  LocalStack    : http://localhost:4566
  LocalStack UI : http://localhost:8080
```

## 3. Push config & secrets

```bash
cpctl sync
```

Reads config from `data/params/global/` and secrets from `data/secrets/local/`, creates ConfigMap + Secret in the `services` namespace.

Options:
```bash
cpctl sync --dry-run              # print manifests, don't apply
cpctl sync --source aws-ssm --ssm-path /cp/services
cpctl sync --diff                 # exit 2 if drift detected (CI)
```

## 4. Apply manifests

```bash
cpctl apply
```

Applies `manifests/bootstrap/` then `manifests/sanitized/services/`.

## 5. Check status

```bash
cpctl status
```

Opens an interactive TUI dashboard showing Kind cluster health, LocalStack service states, and Terraform resource count. Refreshes every 10 seconds.

Keys: `r` refresh, `t` terraform plan, `q` quit.

## 6. Tear down

```bash
cpctl down
```

## Terraform-managed AWS resources

LocalStack resources are provisioned declaratively via Terraform (`terraform/localstack/`). `cpctl up` and `cpctl localstack up` run `terraform init` + `terraform apply` automatically after LocalStack is ready.

**Resources created:**

| Service | Resources |
|---|---|
| S3 | `development-bucket`, `test-bucket`, `artifacts-bucket` |
| SQS | `development-queue`, `test-queue`, `dead-letter-queue` |
| DynamoDB | `Users` (hash: `id`), `Orders` (hash: `orderId`) |
| IAM | `LambdaExecutionRole` + AWSLambdaBasicExecutionRole policy |
| SSM | `/development/database/host`, `/development/database/port`, `/development/api/key` (SecureString) |
| SNS | `development-topic`, `notifications-topic` |
| CloudWatch Logs | `/aws/lambda/development`, `/aws/lambda/test` |

**Add your own resources:** create a new `.tf` file in `terraform/localstack/` and run `cpctl localstack up` (or `task terraform:plan` to preview first).

**State location:** `terraform/localstack/terraform.tfstate` (gitignored, local only)

```bash
# Preview changes without applying
task terraform:plan

# Verify idempotency
terraform -chdir=terraform/localstack plan
# → "No changes."

# Check a specific resource
aws --endpoint-url http://localhost:4566 ssm get-parameter \
  --name "/development/api/key" --with-decryption
```

## AWS-only mode (no cluster)

```bash
cpctl localstack up    # start LocalStack + terraform apply
cpctl localstack down  # stop it
```

Useful for local AWS SDK development without a Kubernetes cluster.

## Mirror config from AWS

```bash
cpctl mirror aws <profile>
```

Mirrors AWS SSM parameters to `data/params/` for local use.

## MCP server (Claude Desktop integration)

```bash
cpctl mcp serve
```

Starts an MCP server over stdio. Add to `~/.config/claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "cloud-playground": {
      "command": "cpctl",
      "args": ["mcp", "serve"]
    }
  }
}
```

## Environment variables

Put overrides in a `.env` file at the repo root — cpctl loads it automatically:

```bash
# .env
AWS_ENDPOINT_URL=http://localhost:4566
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
AWS_DEFAULT_REGION=us-east-1
```

Shell environment always takes precedence over `.env`.

## AWS CLI against LocalStack

```bash
aws --endpoint-url http://localhost:4566 s3 ls
# or with profile:
AWS_PROFILE=localstack aws --endpoint-url http://localhost:4566 s3 ls
```

## Troubleshooting

```bash
# LocalStack health
curl http://localhost:4566/_localstack/health | jq .

# Logs
docker compose -f localstack/docker-compose.yml logs --tail=50

# Re-create cluster
cpctl down && cpctl up
```
