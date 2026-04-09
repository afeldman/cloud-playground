# QUICKSTART — Get cloud-playground running in 5 minutes

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
git clone https://github.com/afeldman/cloud-playground.git
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
task up

# Check status
task status
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
cpctl lambda deploy function.zip \
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

# Linux (systemd)
sudo systemctl start docker
```

### ❌ Kind cluster fails to start?
```bash
# Clean up and retry
task clean
task up
```

### ❌ Database tunnel won't connect?
```bash
# Check tunnel status
task tunnel:status

# Verify PostgreSQL is running
kubectl get pods -n default | grep postgres
```

### ❌ Go build fails?
```bash
# Ensure Go 1.23+ is installed
go version

# Download dependencies
go mod download
```

## 📚 Next Steps

- **Advanced Guides**: See [docs/README.md](./README.md)  
- **Troubleshooting**: [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)  
- **Full Architecture**: [ARCHITECTURE.md](./ARCHITECTURE.md)

---

**⏱️ Time to Working Setup**: ~5 minutes  
*Last updated: April 2026*
