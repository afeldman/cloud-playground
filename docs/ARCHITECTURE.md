# ARCHITECTURE.md — Design Decisions & Deep Dive

## Table of Contents
1. Philosophy
2. Component Architecture
3. Stage Progression (Local → Mirror → Production)
4. Configuration Management
5. State & Persistence
6. Security Model
7. Extensibility

## Part 1: Philosophy

### Principle 1: Zero-Cost Local Development
- All local services run on developer's machine (Docker + Kind)
- No AWS account required to start development
- Rapid feedback loop (<1 sec deploy cycles)

### Principle 2: Ephemeral Staging (Mirror-Cloud)
- Real AWS account, temporary resources
- TTL-based auto-cleanup prevents cost overruns
- Feature validation against production-like infrastructure

### Principle 3: Production Out of Scope
- cpctl never touches prod/staging accounts
- cpctl only supports read-only diagnostics on prod
- Manual approval process for real infrastructure

## Part 2: Component Architecture

### CLI Interface Layer (cpctl)
- **Framework**: Cobra (modular command groups)
- **Commands**: env, tunnel, lambda, batch, ai, status
- **Design**: Each command group is minimal; business logic in internal/ packages

### Stage Abstraction Layer
- **Pattern**: Stage enum (local, mirror, production)
- **Routing**: Commands detect stage from --stage flag or .cpctl.yaml
- **Benefit**: User doesn't care about LocalStack vs AWS; CLI hides complexity

### Infrastructure Provisioning (OpenTofu)
- **Local** (`tofu/localstack/`): LocalStack provider + dummy resources
- **Mirror** (`tofu/mirror/`): Real AWS provider + ephemeral resources
- **Pattern**: Shared variable names + different backends

### LLM Integration (MCP)
- **Primary Path**: cpctl MCP (stdio server)
- **Fallback Path**: Docker MCP (container diagnostics)
- **Design**: LLM talks to cpctl first; escalates to Docker for deep debugging

## Part 3: Stage Progression

### Local Development
```
Developer Machine
├── Kind Cluster (Kubernetes)
│   ├── PostgreSQL (StatefulSet)
│   └── App Deployments
├── LocalStack (Docker Container)
│   ├── S3, SQS, DynamoDB
│   ├── Lambda (execution in Docker)
│   ├── Batch (queue only; execution via Kind)
│   └── CloudWatch
└── cpctl (CLI Interface)
```

### Mirror-Cloud (AWS)
```
AWS Account (mirror-*-*)
├── VPC (temp)
│   ├── Public Subnets
│   ├── Batch Compute Env (EC2)
│   └── RDS (PostgreSQL)
├── Lambda (real AWS Lambda)
├── Batch Workers (real EC2)
└── TTL Controller (cleanup on expiry)
```

### Production (Managed Separately)
- cpctl does NOT provision/manage
- cpctl CAN inspect via SSM tunnels
- cpctl CAN stream logs
- cpctl CANNOT mutate

## Part 4: Configuration Management

### Single Source of Truth: .cpctl.yaml
```yaml
playground:
  name: birdy-cloud-playground
  data_dir: ./data

localstack:
  enabled: true
  endpoint: http://localhost:4566

kind:
  enabled: true
  cluster_name: birdy-local

mirror:
  enabled: true
  aws_profile: mirror-account
  default_ttl: 4h

ai:
  enabled: true
  endpoint: http://localhost:1234/v1
  model: mistralai/ministral-3-3b
```

### Validation
- Schema validation on init (config/validation.go)
- Required fields: playground.name, playground.data_dir, kind.cluster_name
- Optional fields: auto-detect or default
- Migration logic: future schema versions supported

## Part 5: State & Persistence

### data/ Directory Structure
```
data/
├── tunnels/
│   ├── pg.pid        (PID of port-forward process)
│   └── rds.pid
├── envs/
│   └── mirror/
│       ├── .tofu-state   (OpenTofu state)
│       ├── .ttl-expiry   (timestamp)
│       └── endpoints.json
└── secrets/
    ├── localstack-credentials   (local cache)
    └── rds-password
```

### Why Separate State?
- PID tracking: know which process to kill
- TTL tracking: enforce cleanup
- Credentials: ephemeral; regenerate on env up/down
- Endpoints: cache to avoid repeated AWS queries

## Part 6: Security Model

### Local (Implicit Trust)
- No authentication on localhost K8s/LocalStack
- PID-based access control (tunnel processes)
- Credentials stored in `/data` (local .gitignore)

### Mirror-Cloud (Explicit Scoping)
- AWS credentials from ~/.aws/ (via AWS_PROFILE)
- IAM policy: restrict mirror account (separate from prod)
- Temporary resources auto-cleanup (TTL)

### Production (Read-Only + SSM)
- SSM Session Manager for tunnel access
- No cpctl mutations allowed
- All actions logged to CloudTrail

## Part 7: Extensibility

### Adding a New Command Group
```go
// 1. Create cmd/myfeature.go
var myfeatureCmd = &cobra.Command{...}

// 2. Register subcommands
myfeatureCmd.AddCommand(myfeatureSubCmd1)
myfeatureCmd.AddCommand(myfeatureSubCmd2)

// 3. Register with root in init()
rootCmd.AddCommand(myfeatureCmd)
```

### Adding a New Stage
```go
// 1. Update env.go
const (
    StageLocalStack = "localstack"
    StageMirror = "mirror"
    StageFoo = "foo"  // NEW
)

// 2. Update stage detection logic
func (m *EnvironmentManager) GetStage() Stage {...}

// 3. Add tofu/foo/ module for IaC
```

### Adding a New MCP Tool
```go
// 1. Register in mcpserver/server.go
registerMyNewTools()

// 2. Define tool handler
func myToolHandler(req *mcp.CallToolRequest) *mcp.CallToolResult {...}

// 3. Add safety tier gating if mutating
if profileAllowsMutating() {
    registerMyNewMutatingTools()
}
```

## Design Rationale Document

### Why OpenTofu (not Terraform)?
- Terraform licensing concerns → OpenTofu addresses with MPL 2.0
- Feature parity with Terraform 1.6+
- Smaller footprint for local CI/CD

### Why LocalStack (not moto)?
- LocalStack: AWS API parity (S3, Lambda, Batch, etc.)
- moto: Python-only, lower API completeness
- LocalStack Docker image: drop-in single container

### Why Kind (not Minikube/Docker Desktop)?
- Kind: lightweight, multiple nodes, no VM overhead
- LocalStack integration: native docker-compose support
- Batch simulation: K8s job → simulates AWS Batch

### Why MCP over raw stdio commands?
- MCP standardizes tool format for LLMs
- Enables multi-LLM support (Claude, OpenAI, Ollama)
- Tool descriptions + safety profiles built-in

## Performance Characteristics

| Operation | Local | Mirror |
|-----------|-------|---------|
| `cpctl up` | 2-3 min | 5-10 min (IaC) |
| Lambda deploy | 5 sec | 15-30 sec |
| Batch job submit | 2 sec | 5-10 sec |
| Tunnel start | 1 sec | 3-5 sec (SSM) |

## Future Roadmap

### Phase 9: Multi-Region Mirror-Cloud
- Deploy to multiple AWS regions simultaneously
- Load-test across regions

### Phase 10: Cost Optimization
- Cost-aware scheduling (run jobs during off-peak)
- Resource pooling (shared RDS across environments)

### Phase 11: GitOps Integration
- ArgoCD sync for declarative env management
- Auto-revert on manifest drift

## Next Steps
- [TUNNEL_GUIDE](./TUNNEL_GUIDE.md) — Database tunneling implementation
- [MIRROR_CLOUD_GUIDE](./MIRROR_CLOUD_GUIDE.md) — Ephemeral AWS environments
- [TROUBLESHOOTING](./TROUBLESHOOTING.md) — Common issues and solutions
- [QUICKSTART](./QUICKSTART.md) — Foundational setup (if available)
```