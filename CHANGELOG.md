# CHANGELOG

All notable changes to cloud-playground are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Phase 8**: Complete documentation suite (QUICKSTART, guides, troubleshooting)
- **Rolling Logger**: Production-grade logging with zap + lumberjack (file rotation, compression)
- **GoReleaser**: Multi-platform Docker image builds (amd64, arm64, linux, darwin, windows)
- **Docker Compose**: Orchestrated services (LocalStack, PostgreSQL, Prometheus, Grafana, AlertManager)
- **Docker MCP**: LLM-driven container orchestration via Model Context Protocol
- **Phase 9 (Planned)**: Multi-project orchestration for sequential DHW2 development
- **Phase 10 (Planned)**: Telemetry + observability (Prometheus, Grafana, AlertManager)

### Changed
- Migrated logging from slog to zap + lumberjack for production-grade file rotation
- Updated `.cpctl.yaml` with observability and logging configuration sections
- Enhanced AGENT.md with phase status tracking and roadmap

### Fixed
- Backward compatibility maintained for all existing slog API calls
- Lambda commands now properly handle timeout/memory flag types (int vs string)

---

## [1.0.0-beta.9] — 2026-04-09

### Added
- **Phase 7 Complete**: Config validation + migration guardrails
  - Config schema validation on startup
  - Config migration logic for forward compatibility
  - JSON schema export for tooling integration
- **Lambda + Batch Commands (Phase 5)**:
  - `cpctl lambda deploy` — build and deploy Lambda functions
  - `cpctl lambda invoke` — invoke with JSON payloads
  - `cpctl lambda logs` — stream CloudWatch logs (tail mode)
  - `cpctl batch submit` — submit batch jobs
  - `cpctl batch watch` — monitor job progress
  - `cpctl batch logs` — stream job logs
  - `cpctl batch cancel` — cancel jobs
- **LLM Integration (Phase 6)**:
  - `cpctl ai chat` — interactive REPL with local LLM
  - `cpctl ai debug` — error-focused analysis
  - `cpctl ai suggest` — issue-specific suggestions
  - `cpctl ai doctor` — diagnostics and backend detection
- **MCP Integration (Phase 6.5)**:
  - Dual MCP servers: cpctl (orchestration) + Docker (diagnostics)
  - 20+ MCP tools for environment, tunnel, Lambda, Batch management
  - Confirmation semantics for destructive operations
  - Safety tiers: read-only vs mutating tools

### Changed
- Refined `cpctl env` for ephemeral Mirror-Cloud environments
- Reorganized Taskfile.yml into 10+ categories (environment, tunnels, build, CI/CD, etc.)
- act.sh wrapper for GitHub Actions local testing

### Verified
- All 4 logger tests passing (TestZapLoggerWithRolling, TestLoggerFormats, TestFallbackLogger, TestLogLevels)
- Build successful: `go build ./...`
- Lambda + Batch command structure validated

---

## [1.0.0-beta.8] — 2026-04-08

### Added
- **Phase 4 Complete**: Task Automation & Local CI/CD
  - `task act:ci` — test CI workflows locally
  - `task act:release` — test release workflows locally
  - `act.sh` wrapper script with command discovery
  - Secrets management (GITHUB_TOKEN, AWS_*, DOCKER_*)

### Changed
- Taskfile.yml: 50+ tasks in 10 organized categories
- `.actrc` configuration for Docker-based act execution

---

## [1.0.0-beta.7] — 2026-04-07

### Added
- **Phase 3 Complete**: env command (Mirror-Cloud)
  - `cpctl env up mirror --ttl 4h` — provision ephemeral AWS
  - `cpctl env down mirror` — teardown
  - `cpctl env status` — show endpoints
  - `cpctl env scale` — adjust compute
  - TTL-based auto-cleanup

### Changed
- OpenTofu modules for both LocalStack and AWS

---

## [1.0.0-beta.6] — 2026-04-06

### Added
- **Phase 2 Complete**: tunnel command (Kubernetes + SSM)
  - `cpctl tunnel start pg` — port-forward PostgreSQL
  - `cpctl tunnel list` — show active tunnels
  - `cpctl tunnel status` — health check
  - `cpctl tunnel stop pg` — stop tunnel
  - PID tracking and process management

---

## [1.0.0-beta.5] — 2026-04-05

### Added
- **Phase 1 Complete**: Terraform → OpenTofu migration + Architecture
  - `tofu/` directory structure (localstack/, mirror/)
  - S3, SQS, DynamoDB, Lambda, Batch modules
  - `.cpctl.yaml` configuration schema
  - Initial tunnels/manager.go for PID tracking

---

## [1.0.0-beta.1] — 2026-04-01

### Added
- Initial cloud-playground architecture design
- Kind cluster setup (1 control, 2 workers, 1 batch-worker)
- LocalStack integration (AWS emulation)
- cpctl skeleton (main commands)
- Taskfile.yml stubs
- Documentation scaffolding

---

## Notes

### Versioning
- **Major** — Breaking changes to CLI or architecture
- **Minor** — New features (backward compatible)
- **Patch** — Bug fixes and documentation

### Release Checklist
- [ ] All tests passing
- [ ] Documentation updated
- [ ] CHANGELOG.md updated with all changes
- [ ] Version bumped (via git tag)
- [ ] Docker images built and pushed
- [ ] Release notes generated (GoReleaser)
- [ ] Announce on channels

### Future Roadmap
- **Phase 9**: Multi-project orchestration (DHW2 ecosystem)
- **Phase 10**: Telemetry + observability (Prometheus + Grafana)
- **Phase 11**: Performance profiling + optimization
- **Phase 12**: Advanced networking (service mesh exploration)
