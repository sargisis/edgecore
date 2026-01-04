# ⚡️ EdgeCore

> **Simple and Powerful Load Balancer for Your Servers**

EdgeCore automatically distributes traffic across your backend servers, protects against overload, and monitors the health of each server.

**Why do you need a Load Balancer?**
- You have 3 API servers → EdgeCore will distribute the load between them
- One server crashes → EdgeCore automatically redirects traffic to healthy ones
- Too many requests → EdgeCore blocks excess traffic (Rate Limiting)

---

## 🚀 Quick Start (5 minutes)

### Step 1: Installation

**Option A: Download Pre-built Binary** _(coming soon)_
```bash
# Linux/Mac
curl -L https://github.com/username/edgecore/releases/latest/download/edgecore -o edgecore
chmod +x edgecore
sudo mv edgecore /usr/local/bin/
```

**Option B: Install via Go**
```bash
go install github.com/username/edgecore/cmd/edgecore@latest
```

**Option C: Build from Source**
```bash
git clone https://github.com/username/edgecore
cd edgecore
go install ./cmd/edgecore
```

### Step 2: Try in Development Mode

Start EdgeCore with automatic test servers:
```bash
edgecore --dev
```

You'll see:
```
🛠️  Dev Mode: Starting test backends...
🟢 Dev Backend started on :8081
🟢 Dev Backend started on :8082
🟢 Dev Backend started on :8083
🚀 EdgeCore LB started on :8080 [Rate Limit: 100/s]
💡 Dev Mode is ON. Press Ctrl+C to stop everything.
```

Now test it:
```bash
curl http://localhost:8080
# Response: Hello from Backend :8081

curl http://localhost:8080
# Response: Hello from Backend :8082 (next server!)
```

**Congratulations!** EdgeCore is working. Now let's configure it for your real servers.

---

## ⚙️ Configuration for Your Servers

### Step 1: Create `config.json` File

In the directory where you run EdgeCore, create a `config.json` file:

```json
{
  "backends": [
    "http://your-server-1.com:8080",
    "http://your-server-2.com:8080",
    "http://192.168.1.50:3000"
  ],
  "port": 8080,
  "rate_limit": 1000,
  "burst": 100
}
```

**Parameters:**
- `backends` — list of your servers (can be IPs or domains)
- `port` — port on which EdgeCore will listen for incoming traffic
- `rate_limit` — maximum requests per second (overload protection)
- `burst` — how many requests can "burst" above the limit

### Step 2: Start EdgeCore

```bash
edgecore
```

You'll see:
```
Registered backend: http://your-server-1.com:8080
Registered backend: http://your-server-2.com:8080
🚀 EdgeCore LB started on :8080 [Rate Limit: 1000/s]
📊 Metrics endpoint: http://localhost:8080/metrics
💚 Health endpoint: http://localhost:8080/health
```

Done! EdgeCore is now working with your servers.

### Step 3: Verify It Works

```bash
# Your users now connect to EdgeCore, not directly to servers
curl http://localhost:8080/api/users
```

EdgeCore will automatically select the least loaded server.

---

## 🔄 Update Configuration Without Downtime

If you need to add/remove a server:

1. Edit `config.json`
2. Run:
   ```bash
   # Linux/Mac
   pkill -HUP edgecore
   
   # Or find PID and send signal
   ps aux | grep edgecore
   kill -HUP <PID>
   ```

EdgeCore will reload the config **without dropping connections**.

---

## 📊 Monitoring

### Health Check
```bash
curl http://localhost:8080/health
# Response: OK (if EdgeCore is running)
```

### Metrics (for Prometheus/Grafana)
```bash
curl http://localhost:8080/metrics
```

Example response:
```
edgecore_requests_total 15234
edgecore_rate_limited_total 42
```

**What the metrics show:**
- `requests_total` — total requests processed by EdgeCore
- `rate_limited_total` — requests blocked due to rate limit

---

## 🏭 Production Deployment

### Docker
Build and run:
```bash
docker build -t edgecore .
docker run -d \
  --name edgecore \
  -p 8080:8080 \
  -v $(pwd)/config.json:/app/config.json \
  --restart unless-stopped \
  edgecore
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: edgecore
spec:
  replicas: 3
  selector:
    matchLabels:
      app: edgecore
  template:
    metadata:
      labels:
        app: edgecore
    spec:
      containers:
      - name: edgecore
        image: edgecore:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
```

### Systemd (Linux Servers)
Full instructions in **[DEPLOYMENT.md](DEPLOYMENT.md)**

```bash
# Short version:
sudo cp edgecore /usr/local/bin/
sudo cp edgecore.service /etc/systemd/system/
sudo systemctl start edgecore
sudo systemctl enable edgecore
```

---

## ❓ FAQ

**Q: What if one of the backend servers crashes?**  
A: EdgeCore will automatically detect this (health check every 30 seconds) and stop sending traffic there.

**Q: How to increase the request limit?**  
A: Change `rate_limit` and `burst` in `config.json`, then reload config (`pkill -HUP edgecore`).

**Q: Can I use EdgeCore instead of Nginx?**  
A: Yes, for HTTP load balancing EdgeCore works great. Nginx is more versatile (static files, SSL), but EdgeCore is simpler to configure.

**Q: Does EdgeCore support HTTPS?**  
A: Not yet. It's recommended to use EdgeCore behind Cloudflare or nginx with SSL termination.

---

## 🛠 Architecture

```
Internet → [EdgeCore :8080] → Backend Servers
              ↓
         - Rate Limiter
         - Health Checks
         - Least Connections Balancing
         - Logging & Metrics
```

---

## 📝 License

MIT — free to use in commercial and personal projects.