# cloud-playground

A comprehensive local development platform orchestrating **Kind** (Kubernetes), **LocalStack** (AWS emulation), **OpenTofu** (IaC), and **cpctl** CLI—giving you a production-like environment on your laptop **for free**.

## ⚡ 5-Minute Start
```bash
# Clone + Build
git clone https://github.com/afeldman/cloud-playground.git && cd cloud-playground && cd cli/cpctl && go build .

# Start everything
task up

# Check status
task status

# Connect to PostgreSQL
task tunnel:pg:start

# Tear down
task down
```

## 📦 What's Included

| Component | Purpose |
|-----------|---------|
| **Kind** | Kubernetes cluster (1 control + 2 workers) |
| **LocalStack** | AWS services emulation (S3, Lambda, Batch, DynamoDB, etc.) |
| **OpenTofu** | Declarative infrastructure-as-code |
| **PostgreSQL** | Database in K8s |
| **cpctl** | Master CLI for orchestration |
| **Prometheus + Grafana** | (Phase 10) Observability stack |
| **Ollama / LM Studio** | (Phase 6) Local LLM for debugging |

## 📖 Documentation

**Start here**: [**docs/README.md**](./docs/README.md) — complete index with all guides

Quick links:
- 🚀 [QUICKSTART](./docs/QUICKSTART.md) — 5-minute setup
- 🏗️ [ARCHITECTURE](./docs/ARCHITECTURE.md) — design + philosophy
- 🔌 [TUNNEL_GUIDE](./docs/TUNNEL_GUIDE.md) — database access
- ☁️ [MIRROR_CLOUD_GUIDE](./docs/MIRROR_CLOUD_GUIDE.md) — ephemeral AWS
- 🆘 [TROUBLESHOOTING](./docs/TROUBLESHOOTING.md) — 20+ solutions
- 📋 [CLI_REFERENCE](./docs/CLI_REFERENCE.md) — all commands
- 📊 [OBSERVABILITY](./docs/OBSERVABILITY_GUIDE.md) — monitoring setup

## 🎯 Use Cases

### Local Development
```bash
task up                    # Start K8s + LocalStack + Postgres
task tunnel:pg:start       # Connect to DB
go run ./my-app/main.go    # Run your code locally
task down                  # Cleanup
```

### Testing AWS-Dependent Code
```bash
# Deploy Lambda function locally
cpctl lambda deploy my-func.zip --name test --runtime python3.11 --handler index.handler

# Invoke it
cpctl lambda invoke test --payload '{"key":"value"}'

# Monitor Batch job
cpctl batch watch --job-id abc123
```

### GitHub Actions CI/CD
```bash
# Test workflows locally before pushing
task act:ci
task act:release
```

### Multi-Project Development
```bash
# Sequential project development (Phase 9)
cpctl project deploy airbyte      # Deploy project 1
# ... develop on it ...
cpctl project teardown airbyte    # Clean up
cpctl project deploy datalynq     # Deploy project 2
```

## 🔄 Development Stages

| Stage | Environment | Suitable For |
|-------|-------------|--------------|
| **0** | Local IDE | Unit tests, code |
| **1** | Kind + LocalStack (this repo) | Integration tests |
| **2** | Ephemeral AWS (Mirror-Cloud) | E2E tests, production validation |
| **3** | Dev/Staging/Prod | Live traffic |

## 📊 Architecture Overview

```
┌─────────────────────────────────────────┐
│ Your Machine (Free, Local)              │
├─────────────────────────────────────────┤
│ Docker                                  │
│ ├─ Kind Cluster (Kubernetes)            │
│ │  ├─ PostgreSQL (StatefulSet)          │
│ │  ├─ App Pods                          │
│ │  └─ Service Mesh (optional)           │
│ └─ LocalStack (AWS emulation)           │
│    ├─ S3, SQS, DynamoDB                 │
│    ├─ Lambda, Batch                     │
│    └─ SSM, IAM, Secrets                 │
├─────────────────────────────────────────┤
│ cpctl CLI                               │
│ ├─ Kubernetes operations                │
│ ├─ Tunnel management (port-forward)     │
│ ├─ Environment provisioning             │
│ ├─ Lambda/Batch deployment              │
│ └─ AI debugging (LLM integration)       │
└─────────────────────────────────────────┘
```

## ✨ Key Features

✅ **Zero AWS Cost** — Everything runs locally  
✅ **Production-Like** — Real K8s + real AWS services (emulated)  
✅ **Fast Iteration** — No cloud deploy delays  
✅ **Reproducible** — Same setup on every machine  
✅ **Observable** — Prometheus + Grafana + Logs  
✅ **AI-Assisted** — Chat with local LLM for debugging  
✅ **DevOps Ready** — Test CI/CD workflows locally with `act`  
✅ **Multi-Project** — Sequential development of 13+ projects (DHW2)

## 🤖 LLM Integration (Phase 6)

```bash
# Ask AI assistant about your setup
cpctl ai chat
# → "What's running in the cluster?"
# → "Debug Lambda deployment failures"
# → "Show me recent errors"

# Or get issue-specific suggestions
cpctl ai suggest --issue "high-memory"
```

## 📊 Observability (Phase 10)

```bash
# Deploy Prometheus + Grafana
task obs:up

# Open dashboards
task obs:dashboard

# Watch alerts in real-time
cpctl alerts watch
```

## 🚀 Release & Distribution

**Multi-platform binaries**:
```bash
# Automatic via GoReleaser (CI/CD)
# Generates: cpctl_v1.0.0_linux_amd64, cpctl_v1.0.0_darwin_arm64, etc.
# Docker images: ghcr.io/afeldman/cpctl:v1.0.0 (amd64, arm64)
```

## 💬 Support

- **Docs**: [docs/](./docs/)
- **Issues**: [GitHub Issues](https://github.com/afeldman/cloud-playground/issues)
- **Discussions**: [GitHub Discussions](https://github.com/afeldman/cloud-playground/discussions)

## 📜 License

Apache License 2.0 — see [LICENSE](./LICENSE)

---

**v1.0.0-beta.9** | [CHANGELOG](./CHANGELOG.md) | [Release Notes](https://github.com/afeldman/cloud-playground/releases)

