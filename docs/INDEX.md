# Documentation Index — cloud-playground

Complete guide to the **cloud-playground** local + cloud development platform.

---

## 🚀 Getting Started (5 minutes)

1. **[QUICKSTART.md](./QUICKSTART.md)** — Get cloud-playground running locally
   - Prerequisites (Docker, Kubernetes, Go)
   - `cpctl env up` → One command startup
   - First Lambda + Batch job examples

2. **[README.md](./README.md)** — Overview & project status
   - What is cloud-playground?
   - Why use it?
   - Roadmap & known limitations

---

## 🏗️ Understanding the System

3. **[ARCHITECTURE.md](./ARCHITECTURE.md)** — Design & internals
   - Three-stage pipeline (Local → Mirror-Cloud → Production)
   - Component interactions
   - Container orchestration (Kind + LocalStack)
   - Infrastructure-as-Code approach
   - Design decisions & philosophy

4. **[LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md)** — Day-to-day workflows
   - Development loop: Code → Deploy → Test
   - Debugging techniques
   - Best practices
   - IDE integrations

---

## 🔧 Reference Documentation

5. **[CLI_REFERENCE.md](./CLI_REFERENCE.md)** — Complete command reference
   - All `cpctl` commands with examples
   - Global flags
   - Environment management
   - Function, job, and tunnel operations
   - Common workflows

6. **[TUNNEL_GUIDE.md](./TUNNEL_GUIDE.md)** — Port-forwarding & remote access
   - Kubernetes tunnels (K8s services)
   - AWS SSM tunnels (EC2 instances)
   - Configuration
   - Security best practices
   - IDE integration

7. **[MIRROR_CLOUD_GUIDE.md](./MIRROR_CLOUD_GUIDE.md)** — Real AWS testing
   - When to use Mirror-Cloud
   - Setup & configuration
   - Cost management (TTL-based cleanup)
   - Multi-stage testing (Local → Mirror → Production)
   - Troubleshooting Mirror-specific issues

---

## 🐛 Troubleshooting & Diagnostics

8. **[TROUBLESHOOTING.md](./TROUBLESHOOTING.md)** — Common issues & solutions
   - Quick diagnostics (gather logs)
   - Environment setup failures
   - Tunnel connection issues
   - Lambda deployment errors
   - Batch job failures
   - Performance optimization

9. **[ROLLING_LOGGER.md](./ROLLING_LOGGER.md)** — Observability & logging
   - Rolling log system
   - Metrics collection
   - Log aggregation
   - Performance monitoring

10. **[test_pyramid.md](./test_pyramid.md)** — Testing strategy
    - Unit → Integration → E2E test layers
    - When to use each layer
    - LocalStack testing best practices

---

## 📋 Quick Reference Tables

### Command Categories

| Category | Purpose | Main Commands |
|----------|---------|---|
| **Environment** | Manage local Kind cluster + LocalStack | `env up`, `env down`, `env status` |
| **Tunnels** | Port-forward K8s services & EC2 | `tunnel start`, `tunnel stop`, `tunnel ls` |
| **Serverless** | Deploy & invoke Lambda functions | `lambda deploy`, `lambda invoke`, `lambda logs` |
| **Batch** | Submit & monitor batch jobs | `batch submit`, `batch status`, `batch logs` |
| **Observability** | View metrics, logs, traces | `obs metrics`, `obs logs`, `obs traces` |
| **AI** | LLM-powered debugging | `ai ask`, `ai explain`, `ai suggest` |
| **System** | Configuration & status | `status`, `config`, `version` |

### Services Available Locally

| Service | Purpose | Endpoint | Port |
|---------|---------|----------|------|
| **Kind** | Kubernetes cluster | - | - |
| **LocalStack** | AWS emulation | `http://localstack:4566` | 4566 |
| **PostgreSQL** | Main database | `localhost:5432` | 5432 |
| **Redis** | Cache/session store | `localhost:6379` | 6379 |
| **Prometheus** | Metrics database | `http://localhost:9090` | 9090 |
| **Grafana** | Dashboards | `http://localhost:3000` | 3000 |
| **Loki** | Log aggregation | `localhost:3100` | 3100 |

### LocalStack AWS Services

```
✓ S3               Object storage
✓ SQS              Message queues
✓ DynamoDB         NoSQL database
✓ SSM              Parameter store, secrets
✓ IAM              Identity & access management
✓ Lambda           Serverless functions
✓ Batch            Job processing
✓ SNS              Pub/sub messaging
✓ CloudWatch       Logs & metrics
✓ ECR              Container registry (basic)
```

---

## 🎯 Common Workflows

### Workflow 1: Local Development (30 min)

```
1. cpctl env up                           # Start local environment
2. cpctl tunnel start postgres            # Forward database
3. Code/test locally with IDE
4. cpctl lambda deploy function.zip       # Deploy Lambda
5. cpctl lambda invoke my-func            # Test function
6. View logs: cpctl lambda logs my-func -f
7. Iterate on code
8. cpctl env down                         # Cleanup
```

**Docs**: [QUICKSTART.md](./QUICKSTART.md), [LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md)

### Workflow 2: Mirror-Cloud Testing (2 hours)

```
1. cpctl env up mirror --ttl 8h           # Real AWS (ephemeral)
2. cpctl lambda deploy function.zip       # Deploy to real AWS
3. Integration tests (against real S3, RDS, etc.)
4. cpctl batch submit --container-image=... --args=large-dataset
5. Monitor in Grafana: localhost:3000
6. cpctl batch logs job-12345 -f
7. Validate cost: $0.50 estimated
8. cpctl env down mirror                  # Auto-cleanup at TTL
```

**Docs**: [MIRROR_CLOUD_GUIDE.md](./MIRROR_CLOUD_GUIDE.md)

### Workflow 3: Debug Failing Job

```
1. cpctl batch status                     # List recent jobs
2. cpctl batch logs job-12345             # View error logs
3. cpctl ai explain "error message"       # LLM helps diagnose
4. cpctl tunnel start prometheus grafana  # Performance debugging
5. Check metrics: localhost:9090
6. Fix code
7. cpctl batch submit ...                 # Retry
```

**Docs**: [TROUBLESHOOTING.md](./TROUBLESHOOTING.md), [CLI_REFERENCE.md](./CLI_REFERENCE.md)

### Workflow 4: IDE Debugging

```
1. cpctl tunnel start postgres redis      # Forward services
2. DataGrip: Connect to localhost:5432
3. IDE debugger: Connect to localhost:3000
4. Set breakpoints, step through code
5. Query DB in DataGrip while debugging
6. cpctl obs logs -f                      # Watch pod logs
```

**Docs**: [LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md), [TUNNEL_GUIDE.md](./TUNNEL_GUIDE.md)

---

## 📚 Document Roadmap

### Level 1: Beginner (Start here!)
1. README.md
2. QUICKSTART.md
3. LOCAL_DEVELOPMENT.md (first 50%)

### Level 2: Intermediate (Want more control?)
1. ARCHITECTURE.md (overview)
2. CLI_REFERENCE.md (browse commands)
3. TUNNEL_GUIDE.md
4. LOCAL_DEVELOPMENT.md (rest)

### Level 3: Advanced (Power user?)
1. ARCHITECTURE.md (full deep-dive)
2. MIRROR_CLOUD_GUIDE.md
3. test_pyramid.md
4. ROLLING_LOGGER.md

### Level 4: Expert (Operating at scale?)
1. All docs
2. Source code exploration
3. Customizing OpenTofu modules
4. Contributing to cloud-playground

---

## 🔍 Finding What You Need

### By Question

**Q: "How do I start?"**
→ [QUICKSTART.md](./QUICKSTART.md)

**Q: "How do I forward a database?"**
→ [TUNNEL_GUIDE.md](./TUNNEL_GUIDE.md) → Workflow 1

**Q: "What commands are available?"**
→ [CLI_REFERENCE.md](./CLI_REFERENCE.md) → Command Overview

**Q: "Why is my Lambda failing?"**
→ [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) → Lambda Issues

**Q: "How do I test on real AWS?"**
→ [MIRROR_CLOUD_GUIDE.md](./MIRROR_CLOUD_GUIDE.md)

**Q: "How is this thing designed?"**
→ [ARCHITECTURE.md](./ARCHITECTURE.md)

**Q: "How do I debug performance?"**
→ [LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md) → Debugging Techniques

**Q: "What's my best testing strategy?"**
→ [test_pyramid.md](./test_pyramid.md)

### By Tool

**Using `cpctl command`?**
→ [CLI_REFERENCE.md](./CLI_REFERENCE.md) → search command name

**Using tunnels?**
→ [TUNNEL_GUIDE.md](./TUNNEL_GUIDE.md)

**Using LocalStack?**
→ [ARCHITECTURE.md](./ARCHITECTURE.md) → AWS Emulation

**Using Kind?**
→ [ARCHITECTURE.md](./ARCHITECTURE.md) → Container Orchestration

**Using Grafana/Prometheus?**
→ [ROLLING_LOGGER.md](./ROLLING_LOGGER.md)

### By Error

**"Connection refused"**
→ [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) → Common Error Messages

**"Port already in use"**
→ [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) → Tunnel Issues

**"Lambda deployment fails"**
→ [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) → Lambda Issues

**"Batch job stuck"**
→ [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) → Batch Issues

---

## 🎓 Learning Paths

### Path A: I just want to deploy a Lambda ⚡
1. [QUICKSTART.md](./QUICKSTART.md) (5 min)
2. [CLI_REFERENCE.md](./CLI_REFERENCE.md) → `lambda` section (10 min)
3. Deploy! 🚀

**Time**: 15 minutes

### Path B: I want full-featured local development 💻
1. [QUICKSTART.md](./QUICKSTART.md) (5 min)
2. [ARCHITECTURE.md](./ARCHITECTURE.md) → Overview (15 min)
3. [LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md) (30 min)
4. [TUNNEL_GUIDE.md](./TUNNEL_GUIDE.md) (20 min)
5. Practice with examples (30 min)

**Time**: ~2 hours

### Path C: I want to debug issues & optimize 🔍
1. Path B (first) (2 hours)
2. [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) (30 min)
3. [ROLLING_LOGGER.md](./ROLLING_LOGGER.md) (20 min)
4. [test_pyramid.md](./test_pyramid.md) (20 min)

**Time**: ~3.5 hours

### Path D: I want to understand everything 🎓
1-10. Read all documents in order
11. Explore source code: `cmd/`, `core/`, `tofu/`
12. Join discussions & contribute

**Time**: Several days (ongoing learning)

---

## 🔗 Cross-References

Each document links to related docs:
- Quick navigation footer in every guide
- Cross-linked examples
- Command references with doc links

### Most-Linked Docs
1. [CLI_REFERENCE.md](./CLI_REFERENCE.md) — Used by nearly every other doc
2. [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) — Referenced from QUICKSTART, LOCAL_DEVELOPMENT, TUNNEL_GUIDE
3. [ARCHITECTURE.md](./ARCHITECTURE.md) — Provides foundational understanding
4. [LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md) — Daily reference

---

## 📞 Getting Help

### Stuck?
1. Check [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) for your error
2. Search this index for your question
3. Run diagnostic: `cpctl status --json`
4. Check [CLI_REFERENCE.md](./CLI_REFERENCE.md) for command usage
5. Read [ARCHITECTURE.md](./ARCHITECTURE.md) for design understanding

### Found a bug?
1. Reproduce with `cpctl -v`
2. Collect logs: `cpctl env logs > env.log`
3. Check existing issues
4. Report with full error + command used

### Want to contribute?
1. Read [ARCHITECTURE.md](./ARCHITECTURE.md) (design)
2. Review source code: `cmd/`, `core/`
3. Check issues marked "good-first-issue"
4. Submit PR with tests

---

## 📋 Document Statistics

| Document | Purpose | Sections | Audience |
|----------|---------|----------|----------|
| README | Overview | 5 | Everyone |
| QUICKSTART | Getting started | 8 | Beginners |
| ARCHITECTURE | Deep design | 12 | Intermediate+ |
| LOCAL_DEVELOPMENT | Day-to-day workflows | 10 | Intermediate |
| TUNNEL_GUIDE | Port-forwarding | 8 | Intermediate |
| CLI_REFERENCE | Command reference | 15 | Everyone |
| MIRROR_CLOUD_GUIDE | Real AWS testing | 9 | Advanced |
| TROUBLESHOOTING | Debugging | 12 | Everyone |
| ROLLING_LOGGER | Observability | 6 | Intermediate+ |
| test_pyramid | Testing strategy | 5 | Advanced |

**Total**: 90+ sections across 10 docs
**Depth**: From 5-minute quickstart to multi-hour deep-dives
**Coverage**: Setup → Development → Testing → Production

---

## 🚀 Next Steps

### First Time?
→ Start with [QUICKSTART.md](./QUICKSTART.md)

### Development?
→ Read [LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md)

### Stuck?
→ Check [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)

### Need Command Help?
→ See [CLI_REFERENCE.md](./CLI_REFERENCE.md)

### Want to Learn Everything?
→ Follow [Learning Paths](#-learning-paths) above

---

**Last Updated**: 2024-01-15
**Version**: 1.0.0
**Status**: Complete & production-ready ✅

See [README.md](./README.md) for project status and roadmap.
