# TROUBLESHOOTING — Common Issues & Solutions

## Quick Diagnostics

Before investigating deep, run:

```bash
cpctl status              # Overall health check
cpctl env logs -f         # Follow environment logs
docker ps                 # Check container status
kubectl get all -n default  # Check all K8s resources
```

---

## Environment Issues

### ✗ `cpctl env up` Fails

**Symptom**: Command exits with error, environment not created

**Causes & Solutions**:

#### 1. Docker daemon not running
```bash
# Check if Docker is running
docker ps

# If not, start it:
open /Applications/Docker.app  # macOS
systemctl start docker         # Linux
```

#### 2. Insufficient disk space
```bash
# Check available disk space (need >10GB)
df -h

# Free up space:
docker system prune               # Remove unused containers/images
docker system prune --volumes    # Also remove unused volumes
```

#### 3. Insufficient memory
```bash
# Check available RAM
free -h              # Linux
vm_stat              # macOS

# Docker needs ≥8GB. Increase Docker memory:
# Docker Desktop → Settings → Resources → Memory: 12GB
```

#### 4. Port conflict (default: 4566, 5432, 6379, etc.)
```bash
# Find what's using port
lsof -i :4566 | head -20

# Stop the process or use different port:
cpctl env up --localstack-port 5566

# List all configured ports:
cpctl config show | grep -i port
```

#### 5. Kubernetes context issues
```bash
# Verify Kind context exists
kubectl config get-contexts | grep kind

# If missing, recreate:
cpctl env down --force
cpctl env up
```

**Debug Output**:
```bash
# Run with verbose logging
cpctl env up -v

# Save logs to file
cpctl env up -v > env-setup.log 2>&1

# Check Docker daemon logs
docker logs <container-id>

# Check Kind cluster status
kind get clusters
kubectl get nodes
```

---

### ✗ LocalStack Services Not Responding

**Symptom**: `Connection refused` or `timeout` when accessing LocalStack

**Diagnosis**:
```bash
# Check LocalStack container
docker ps | grep localstack

# Check if service is responding
curl http://localhost:4566/_localstack/health

# Check logs
docker logs $(docker ps -qf "name=localstack")
```

**Solutions**:

#### 1. LocalStack container crashed
```bash
# Restart LocalStack
docker restart $(docker ps -qf "name=localstack")

# Or recreate entirely
cpctl env down --force
cpctl env up
```

#### 2. Port not forwarded correctly
```bash
# Verify port forward exists
docker port $(docker ps -qf "name=localstack") | grep 4566

# If missing, restart
docker restart $(docker ps -qf "name=localstack")
```

#### 3. Wrong endpoint URL in code
```bash
# Correct endpoint in local code:
endpoint = "http://localhost:4566"          # From laptop
endpoint = "http://localstack:4566"         # From within K8s pod

# Check what your code uses:
grep -r "endpoint" src/ | grep -i localstack
```

#### 4. Invalid AWS credentials (LocalStack doesn't validate)
```bash
# LocalStack accepts ANY credentials. If still failing:
export AWS_ACCESS_KEY_ID=testing
export AWS_SECRET_ACCESS_KEY=testing
export AWS_SECURITY_TOKEN=testing
export AWS_SESSION_TOKEN=testing

aws --endpoint-url http://localhost:4566 s3 ls
```

---

## Tunnel Issues

### ✗ `cpctl tunnel start` Fails

**Symptom**: `Error: tunnel failed to start`, port already in use

**Solutions**:

#### 1. Port already in use
```bash
# Find what's using the port
lsof -i :5432

# Kill the process
kill -9 $(lsof -t -i :5432)

# Or use different port
cpctl tunnel start postgres --local-port 15432

# Then connect with different port
psql -h localhost -p 15432
```

#### 2. Service/pod doesn't exist
```bash
# Check service exists
kubectl get svc postgres

# If missing, create it:
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
EOF
```

#### 3. Kubernetes context not set
```bash
# Check current context
kubectl config current-context

# If wrong, switch to Kind context
kubectl config use-context kind-cpctl-local

# Try tunnel again
cpctl tunnel start postgres
```

### ✗ Tunnel Works But Can't Connect

**Symptom**: `cpctl tunnel start` succeeds, but `psql localhost:5432` times out

**Diagnosis**:
```bash
# 1. Verify tunnel is running
cpctl tunnel ls

# 2. Test connection with nc
nc -zv localhost 5432

# 3. Check if service is actually listening
kubectl exec -it svc/postgres -- netstat -tuln | grep 5432

# 4. Try from inside the pod
kubectl exec -it svc/postgres -- psql -c "SELECT 1"
```

**Solutions**:

#### 1. Service not actually listening
```bash
# Pod may not be ready
kubectl get pods -l app=postgres

# Check pod status
kubectl describe pod <pod-name>

# Check pod logs
kubectl logs <pod-name>

# Wait for pod to start
kubectl wait --for=condition=ready pod -l app=postgres --timeout=60s
```

#### 2. Firewall blocking connection
```bash
# On macOS, Docker uses a VM
# Ensure port is actually forwarded from Docker Desktop

# Try connecting through Docker
docker exec -it $(docker ps -qf "name=postgres") \
  psql -h localhost -p 5432

# If that works but local doesn't, Docker daemon issue
```

#### 3. Wrong tunnel configuration
```bash
# Check tunnel config
cpctl config show --section tunnels

# Verify remote port matches service port
kubectl get svc postgres -o yaml | grep port

# Recreate tunnel with correct ports
cpctl tunnel stop postgres
cpctl tunnel start postgres --remote-port 3306
```

### ✗ SSM Tunnel Not Working

**Symptom**: `cpctl tunnel start bastion --type ssm` fails

**Prerequisites**:
```bash
# 1. AWS credentials configured
aws sts get-caller-identity

# 2. EC2 instance has SSM role
aws ec2 describe-instances --instance-ids i-xxx \
  --query 'Reservations[0].Instances[0].IamInstanceProfile'

# 3. SSM agent running on instance
aws ssm describe-instance-information --filters \
  "Key=InstanceIds,Values=i-xxx"
```

**Solutions**:

#### 1. Instance not reachable by SSM
```bash
# Verify instance has SSM role
aws ec2 describe-instances --instance-ids i-xxx \
  --query 'Reservations[0].Instances[0].IamInstanceProfile'

# If missing, add it
aws ec2 associate-iam-instance-profile \
  --instance-id i-xxx \
  --iam-instance-profile Name=EC2-SSM-Role

# Wait for agent to register (1-2 minutes)
sleep 120
aws ssm describe-instance-information | grep i-xxx
```

#### 2. No SSH key configured
```bash
# For SSM tunnel to enable SSH, you need SSH key on instance
# Copy your public key to instance:
aws ssm start-session --target i-xxx

# Then add public key to ~/.ssh/authorized_keys
echo "ssh-rsa AAAA... user@host" >> ~/.ssh/authorized_keys
exit
```

#### 3. Tunnel configuration incomplete
```bash
# Check SSM tunnel config
cpctl config show --section tunnels | grep -A5 bastion

# Required fields:
# - type: ssm
# - instanceId: i-xxx (or instanceName: xxx)
# - localPort: XXXX
# - remotePort: 22 (for SSH)

# Update config if needed
cpctl config init
```

---

## Lambda Issues

### ✗ `cpctl lambda deploy` Fails

**Symptom**: Error uploading or deploying Lambda function

**Solutions**:

#### 1. Function.zip doesn't exist
```bash
# Check file
ls -lh function.zip

# Create if missing
zip -r function.zip index.js node_modules/
```

#### 2. ZIP file too large (>52MB)
```bash
# Check size
du -h function.zip

# For larger functions, upload to S3 first
aws s3 cp function.zip s3://my-bucket/my-func.zip

cpctl lambda deploy s3://my-bucket/my-func.zip \
  --name my-func --from-s3
```

#### 3. LocalStack Lambda service not ready
```bash
# Check LocalStack health
curl http://localhost:4566/_localstack/health | jq '.services.lambda'

# Should show "running". If not, restart LocalStack
cpctl env down --force
cpctl env up
```

#### 4. Invalid handler or runtime
```bash
# Verify handler exists
unzip -l function.zip | grep -i index

# Verify runtime is supported
# Node: node14.x, node16.x, node18.x, etc.
# Python: python3.8, python3.9, python3.10, etc.
# Go: provided, provided.al2

cpctl lambda deploy function.zip --name my-func \
  --handler index.handler --runtime nodejs18.x
```

### ✗ Lambda Invocation Fails

**Symptom**: `cpctl lambda invoke` returns error

**Diagnosis**:
```bash
# Check function exists
cpctl lambda ls

# Check function details
aws --endpoint-url http://localhost:4566 \
  lambda get-function --function-name my-func

# Check logs
cpctl lambda logs my-func
```

**Solutions**:

#### 1. Function doesn't exist
```bash
# Deploy it first
cpctl lambda deploy function.zip --name my-func
```

#### 2. Payload format error
```bash
# Valid JSON only
cpctl lambda invoke my-func --payload '{"key": "value"}'

# Not valid:
cpctl lambda invoke my-func --payload "{key: value}"

# Use file for complex payloads
echo '{"key": "value"}' > payload.json
cpctl lambda invoke my-func --payload-file payload.json
```

#### 3. Function hits timeout
```bash
# Increase timeout
cpctl lambda deploy function.zip --name my-func --timeout 60

# Then invoke
cpctl lambda invoke my-func --payload '...'

# Check logs to see where it's slow
cpctl lambda logs my-func -f
```

#### 4. Missing environment or IAM role
```bash
# Deploy with required env vars
cpctl lambda deploy function.zip --name my-func \
  --env DB_HOST=postgres,DB_PORT=5432

# Deploy with role
cpctl lambda deploy function.zip --name my-func \
  --role arn:aws:iam::000000000000:role/lambda-role
```

---

## Batch Issues

### ✗ Batch Job Fails Immediately

**Symptom**: Job status shows `FAILED` within seconds

**Diagnosis**:
```bash
# Check job status in detail
cpctl batch status job-12345

# Get raw AWS response
aws --endpoint-url http://localhost:4566 \
  batch describe-jobs --jobs job-12345

# Check job logs
cpctl batch logs job-12345
```

**Solutions**:

#### 1. Container image not found
```bash
# Verify image exists locally
docker images | grep myimage

# Or pull from registry
docker pull myimage:latest

# When submitting, use full image reference:
cpctl batch submit --container-image myimage:latest
```

#### 2. Batch queue doesn't exist
```bash
# List queues
aws --endpoint-url http://localhost:4566 \
  batch describe-job-queues

# If missing, create it
cpctl env up  # Should create default queues

# Or specify existing queue
cpctl batch submit --container-image myimage:latest \
  --job-queue default
```

#### 3. Insufficient compute resources
```bash
# Check compute environment
aws --endpoint-url http://localhost:4566 \
  batch describe-compute-environments

# Check available instances
docker ps  # See Docker resource usage

# Increase Docker memory or CPU in Docker Desktop settings
```

#### 4. Container command fails
```bash
# Test command locally first
docker run myimage:latest python process.py arg1 arg2

# If it works locally, check environment in Batch:
cpctl batch logs job-12345 -f

# Check stdout/stderr for errors
cpctl batch logs job-12345 | tail -50
```

### ✗ Batch Job Hangs

**Symptom**: Job status stuck in `RUNNABLE` or `RUNNING` for hours

**Diagnosis**:
```bash
# Check if container is actually running
docker ps | grep my-job

# Check job details
aws --endpoint-url http://localhost:4566 \
  batch describe-jobs --jobs job-12345

# Check LocalStack Batch logs
docker logs $(docker ps -qf "name=localstack") | grep -i batch
```

**Solutions**:

#### 1. Container waiting for input (blocking I/O)
```bash
# Verify container doesn't expect stdin
docker run -it myimage:latest python process.py

# If it's waiting for input, modify Dockerfile:
# FROM myimage:latest
# ENTRYPOINT ["python", "process.py", "--no-stdin"]

# Or pass /dev/null as stdin in Batch
```

#### 2. LocalStack Batch service stuck
```bash
# Kill hanging job
aws --endpoint-url http://localhost:4566 \
  batch terminate-job --job-id job-12345

# Restart LocalStack
docker restart $(docker ps -qf "name=localstack")

# Try again
cpctl batch submit ...
```

---

## Observability Issues

### ✗ Prometheus/Grafana Not Accessible

**Symptom**: Can't reach `http://localhost:9090` or `http://localhost:3000`

**Diagnosis**:
```bash
# Check tunnels
cpctl tunnel ls

# Check if port-forward working
nc -zv localhost 9090
nc -zv localhost 3000

# Check if pods are running
kubectl get pods | grep prometheus
kubectl get pods | grep grafana
```

**Solutions**:

#### 1. Tunnels not started
```bash
cpctl tunnel start prometheus grafana
```

#### 2. Pods not running
```bash
# Check pod status
kubectl describe pod prometheus-0

# Check logs
kubectl logs prometheus-0

# Restart
kubectl rollout restart statefulset prometheus
kubectl rollout restart deployment grafana
```

#### 3. Wrong local port
```bash
# Check configured port
cpctl config show --section tunnels | grep -i prometheus

# Start with custom port
cpctl tunnel start prometheus --local-port 19090

# Then access at localhost:19090
```

---

## Performance Issues

### ✗ Everything is Slow

**Symptom**: Commands take 20+ seconds, high CPU/memory usage

**Causes**:
1. Docker daemon overloaded
2. LocalStack processing heavy workload
3. Kubernetes cluster under-resourced

**Solutions**:

```bash
# 1. Check resource usage
docker stats

# 2. Identify resource hog
docker ps --format "table {{.Names}}\t{{.CPUPerc}}\t{{.MemUsage}}"

# 3. Check K8s resource requests/limits
kubectl describe node $(kubectl get nodes -o name | head -1)

# 4. Increase Docker limits
# Docker Desktop → Settings → Resources → CPUs: 8, Memory: 12GB

# 5. Reduce LocalStack verbosity
docker exec $(docker ps -qf "name=localstack") \
  /bin/sh -c "export DEBUG=0; tail -f /dev/null"

# 6. Clean up unused containers/images
docker system prune --volumes
```

---

## Common Error Messages

| Error | Cause | Fix |
|-------|-------|-----|
| `Connection refused` | Service not running | `cpctl status` → check what's down |
| `Port already in use` | Another process bound | `lsof -i :PORT` → kill or use different port |
| `Pod pending` | Insufficient resources | Increase Docker memory/CPU |
| `Timeout` | Service hung/crashed | Restart: `cpctl env down && cpctl env up` |
| `Invalid AWS credentials` | (LocalStack ignores anyway) | Set dummy: `AWS_ACCESS_KEY_ID=testing` |
| `File not found` | Wrong path/working directory | Check: `pwd`, `ls`, use absolute paths |
| `Permission denied` | User doesn't have access | Use `sudo` or add user to docker group |

---

## Getting Help

### Still Stuck?

1. **Collect diagnostics**:
   ```bash
   cpctl status --json > status.json
   cpctl env logs > env.log
   docker ps -a > docker-ps.log
   kubectl get all -n default > k8s.log
   ```

2. **Check logs**:
   ```bash
   # Recent errors in Docker
   docker logs $(docker ps -a | grep -i error | head -1 | awk '{print $1}')
   
   # Recent K8s events
   kubectl get events
   
   # LocalStack service logs
   docker logs $(docker ps -qf "name=localstack") | tail -150
   ```

3. **Search documentation**:
   - [ARCHITECTURE.md](./ARCHITECTURE.md)
   - [LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md)
   - [CLI_REFERENCE.md](./CLI_REFERENCE.md)

4. **Open an issue** with:
   - `cpctl version`
   - `docker --version`
   - `kubectl version --client`
   - Full error message (screenshot or paste)
   - Steps to reproduce
   - `status.json` from above

---

See also:
- [LOCAL_DEVELOPMENT.md](./LOCAL_DEVELOPMENT.md) — Best practices
- [CLI_REFERENCE.md](./CLI_REFERENCE.md) — All commands
- [ARCHITECTURE.md](./ARCHITECTURE.md) — How it all works
