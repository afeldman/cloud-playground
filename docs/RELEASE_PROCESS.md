# Release Process with GoReleaser

This document describes the automated release process for cpctl using GoReleaser and GitHub Actions.

## Overview

The release process is fully automated using:
- **GoReleaser**: Builds binaries, creates archives, builds Docker images, and generates changelogs
- **GitHub Actions**: Triggers releases on tag pushes and handles the entire workflow
- **GitHub Container Registry**: Stores Docker images
- **GitHub Releases**: Hosts binary releases

## Workflows

### 1. Snapshot Releases (`snapshot.yml`)
**Trigger**: Push to `main`/`master` branch
**Purpose**: Build and test snapshot releases without creating actual releases
**Output**: 
- Artifacts uploaded as workflow artifacts (7-day retention)
- No GitHub Release created
- Docker images tagged as `SNAPSHOT-<commit>`

### 2. Full Releases (`release.yml`)
**Trigger**: Push of a tag starting with `v` (e.g., `v1.2.3`)
**Purpose**: Create a full release with all artifacts
**Output**:
- GitHub Release with binaries for all platforms
- Docker images tagged with version and `latest`
- Checksums and signatures (if configured)
- Homebrew tap update (if configured)

### 3. Manual Releases
**Trigger**: Manual workflow dispatch from GitHub UI
**Purpose**: Create a release without pushing a tag first
**Process**: 
1. Go to "Actions" → "Release with GoReleaser" → "Run workflow"
2. Enter version (e.g., `v1.2.3`)
3. Workflow creates tag and release

## Release Artifacts

Each release includes:

### Binaries
- **Linux**: amd64, arm64 (tar.gz)
- **macOS**: amd64, arm64 (tar.gz)  
- **Windows**: amd64 (zip)
- **Checksums**: SHA256 checksums file
- **Signatures**: GPG signatures (optional)

### Docker Images
- Multi-architecture images for `linux/amd64` and `linux/arm64`
- Tags: `{version}` and `latest` (for non-snapshot releases)
- Available at: `ghcr.io/afeldman/cloud-playground`

### Documentation
- README.md
- LOCAL_DEVELOPMENT.md
- AGENT.md
- QUICKSTART.md

## Configuration Files

### `.goreleaser.yml`
Main GoReleaser configuration:
- Build settings (Go version, platforms, ldflags)
- Archive formats and included files
- Docker image configuration
- Changelog generation
- Homebrew tap configuration

### `.github/workflows/release.yml`
Release workflow:
- Triggers on tags and manual dispatch
- Sets up Go, Docker Buildx
- Runs GoReleaser with proper permissions
- Handles GitHub Container Registry login

### `.github/workflows/snapshot.yml`
Snapshot workflow:
- Runs on pushes to main branch
- Builds snapshot artifacts for testing
- Validates build process in PRs

### `Dockerfile.goreleaser`
Multi-stage Dockerfile:
- Uses distroless base image
- Platform-aware artifact copying
- Non-root user for security

## How to Create a Release

### Automatic Release (Recommended)
1. Update version in code if needed
2. Commit changes to `main` branch
3. Create and push a tag:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```
4. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Create Docker images
   - Generate changelog from git history
   - Create GitHub Release
   - Push to GitHub Container Registry

### Manual Release
1. Go to GitHub repository → Actions
2. Select "Release with GoReleaser"
3. Click "Run workflow"
4. Enter version (e.g., `v1.2.3`)
5. Click "Run workflow"

### Snapshot Release
1. Push to `main` branch
2. Workflow automatically builds snapshot
3. Check artifacts in workflow run

## Versioning

- Follows [Semantic Versioning](https://semver.org/)
- Tags must start with `v` (e.g., `v1.2.3`)
- Pre-release versions: `v1.2.3-alpha.1`, `v1.2.3-beta.1`
- Snapshot versions: `{version}-SNAPSHOT-{commit}`

## Changelog Generation

GoReleaser automatically generates changelogs from git commits using conventional commit messages:

```
feat: Add new feature
fix: Fix bug
perf: Improve performance
docs: Update documentation
chore: Maintenance tasks
ci: CI/CD changes
build: Build system changes
```

## Optional Configuration

### GPG Signing
To enable GPG signing of releases:

1. Generate GPG key if not already done
2. Add public key to GitHub account
3. Add secret to repository:
   - Name: `GPG_FINGERPRINT`
   - Value: Fingerprint of your GPG key

### Homebrew Tap
To publish to Homebrew:

1. Create a tap repository: `homebrew-tap`
2. Add secret to repository:
   - Name: `HOMEBREW_TAP_GITHUB_TOKEN`
   - Value: GitHub token with repo permissions

## Troubleshooting

### Common Issues

1. **Release fails with permission errors**
   - Ensure workflow has `contents: write` and `packages: write` permissions
   - Check that GitHub token has sufficient scope

2. **Docker build fails**
   - Verify Dockerfile exists and is valid
   - Check that base image is accessible
   - Ensure Buildx is properly set up

3. **Changelog empty**
   - Check commit messages follow conventional format
   - Verify git history is fetched with `fetch-depth: 0`

4. **Binaries not uploaded**
   - Check artifact size limits (2GB per file on GitHub)
   - Verify network connectivity during upload

### Debugging
- Check workflow run logs in GitHub Actions
- Run GoReleaser locally: `goreleaser release --snapshot --clean`
- Test Docker build locally: `docker build -f Dockerfile.goreleaser .`

## Local Testing

### Test GoReleaser Configuration
```bash
# Install GoReleaser
brew install goreleaser/tap/goreleaser

# Dry run
goreleaser check

# Build snapshot locally
goreleaser release --snapshot --clean
```

### Test with act (GitHub Actions locally)
```bash
# Install act
brew install act

# List workflows
act --list

# Run snapshot workflow
act push -W .github/workflows/snapshot.yml
```

## Security Considerations

- Docker images use non-root user
- Binaries are statically linked with CGO disabled
- Checksums provided for verification
- Optional GPG signing for authenticity
- Secrets stored in GitHub Secrets, not in code

## Related Documentation

- [GoReleaser Documentation](https://goreleaser.com/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Conventional Commits](https://www.conventionalcommits.org/)