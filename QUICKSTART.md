# QUICKSTART.md — Get cloud-playground running in 5 minutes

## 🎯 Goal
Get from zero to a working local development environment in ≤5 minutes.

## ✅ Prerequisites Checklist
- **Docker** installed + running
- **Kind**: `brew install kind` (macOS) or see [Kind docs](https://kind.sigs.k8s.io/)
- **Go 1.23+** (for building `cpctl` locally)
- **OpenTofu**: `brew install opentofu` (macOS) or `apt install tofu` (Linux)
- **LocalStack CLI**: `brew install localstack` (macOS) or `pip install localstack` (Linux)
- **(Optional)** Ollama or LM Studio for AI assistant features

## 🚀 Step 1: Clone & Build (1 min)
```bash
# Clone the repository
git clone https://github.com/your-org/cloud-playground.git
cd cloud-playground

# Build the cpctl CLI
cd cli/cpctl
go build -o cpctl .

# Verify it works
./cpctl --help
```

## 🏗️ Step 2: Start Local Stack (2 min)
```bash
# Start everything with one command
task env:up

# Or manually:
# go run ./cli/cpctl env up

# Check status
task env:status
# or:
# go run ./cli/cpctl env status
```

## 🔌 Step 3: Connect to PostgreSQL (1 min)
```bash
# Start port-forward for PostgreSQL
task tunnel:pg:start

# Now available at localhost:5432
psql -h localhost -p 5432 -U postgres

# List active tunnels
task tunnel:list
```

## ⚡ Step 4: Deploy a Lambda Function (Optional, 1 min)
```bash
# Create a simple Python Lambda function
mkdir -p /tmp/lambda && cd /tmp/lambda
cat > index.py << 'EOF'
def handler(event, context):
    return {"statusCode": 200, "body": "Hello from Lambda!"}
EOF
zip function.zip index.py

# Deploy to local stack
go run ./cli/cpctl lambda deploy function.zip \
  --name hello-world \
  --runtime python3.11 \
  --handler index.handler \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --stage localstack
```

## 🤖 Step 5: Chat with AI Assistant (Optional)
```bash
# Check AI backend connectivity
task ai:doctor

# Start interactive chat with local LLM
task ai:chat

# In the chat, ask questions like:
# > what's my local setup status?
# > show me Kubernetes pod status
# > debug my Lambda deployment
```

## 🛠️ Troubleshooting

### ❌ Docker not running?
```bash
# macOS
open -a Docker
# or via brew services
brew services start docker

# Linux (systemd)
sudo systemctl start docker
```

### ❌ Kind cluster fails to start?
```bash
# Clean up and retry
task clean
task env:up

# Check Docker resources
docker ps -a
docker logs kind-control-plane
```

### ❌ Database tunnel won't connect?
```bash
# Check tunnel status
task tunnel:status

# Restart tunnel
task tunnel:stop pg
task tunnel:pg:start

# Verify PostgreSQL is running in cluster
kubectl get pods -n playground | grep postgres
```

### ❌ LocalStack not responding?
```bash
# Check LocalStack container
docker logs birdy-localstack

# Restart LocalStack
docker restart birdy-localstack

# Verify AWS services
aws --endpoint-url=http://localhost:4566 s3 ls
```

### ❌ Go build fails?
```bash
# Ensure Go 1.23+ is installed
go version

# Clean module cache
go clean -modcache

# Download dependencies
go mod download
```

## 📚 Next Steps

### Advanced Guides
- **Database Access**: Read [LOCAL_DEVELOPMENT.md](./docs/LOCAL_DEVELOPMENT.md) for advanced database tunneling
- **Cloud Testing**: Check [test_pyramid.md](./docs/test_pyramid.md) for testing strategy across stages
- **Full Reference**: Read [README.md](./README.md) for architecture and complete feature list

### What to Try Next
1. **Deploy Kubernetes manifests**: `task apply`
2. **Sync config & secrets**: `task sync`
3. **Test AWS Batch jobs**: `task batch:submit --help`
4. **Mirror real AWS resources**: `task env:up:mirror`
5. **Run CI checks locally**: `task ci:check`

### Development Workflow
```bash
# Watch for changes and rebuild
task dev:watch

# Run tests on file changes
task dev:test:watch

# Check code quality
task ci:lint
```

## 🎥 Video Demo
*Coming soon: 10-minute walkthrough video*

---

**⏱️ Time to Working Setup**: ~5 minutes  
**🎯 Next Milestone**: Deploy + test complete application in 10 minutes  
**🔄 Maintenance**: Run `task update` to keep components current

## 📞 Need Help?
- Check the [troubleshooting section](#-troubleshooting) above
- Review [LOCAL_DEVELOPMENT.md](./docs/LOCAL_DEVELOPMENT.md) for detailed workflows
- Open an issue on GitHub for bugs or feature requests

---

*Last updated: April 2026*  
*Tested with: Docker 27+, Kind 0.25+, Go 1.23+, OpenTofu 1.8+*