# EdgeCore Production Deployment Guide

## Prerequisites
- Go 1.21+ (for building from source)
- Docker (for containerized deployment)
- Linux server with systemd (for service deployment)

---

## Deployment Options

### 1. Docker Deployment (Recommended)

**Build Image:**
```bash
docker build -t your-registry.com/edgecore:v1.0 .
docker push your-registry.com/edgecore:v1.0
```

**Run Container:**
```bash
docker run -d \
  --name edgecore \
  -p 8080:8080 \
  -v $(pwd)/config.json:/app/config.json \
  --restart unless-stopped \
  your-registry.com/edgecore:v1.0
```

**Check Health:**
```bash
curl http://localhost:8080/health
```

---

### 2. Kubernetes Deployment

**Full Example:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: edgecore-config
data:
  config.json: |
    {
      "backends": [
        "http://backend-svc-1:8080",
        "http://backend-svc-2:8080"
      ],
      "port": 8080,
      "rate_limit": 1000,
      "burst": 100
    }
---
apiVersion: apps/v1
kind:Deployment
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
        image: your-registry.com/edgecore:v1.0
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /app/config.json
          subPath: config.json
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
      volumes:
      - name: config
        configMap:
          name: edgecore-config
---
apiVersion: v1
kind: Service
metadata:
  name: edgecore
spec:
  selector:
    app: edgecore
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

**Deploy:**
```bash
kubectl apply -f edgecore-k8s.yaml
```

---

### 3. Systemd Service (Bare Metal)

**Setup:**
```bash
# 1. Build binary
go build -o edgecore ./cmd/edgecore

# 2. Install
sudo cp edgecore /usr/local/bin/
sudo mkdir -p /opt/edgecore
sudo cp config.json /opt/edgecore/

# 3. Create user
sudo useradd -r -s /bin/false edgecore

# 4. Install service
sudo cp edgecore.service /etc/systemd/system/
sudo systemctl daemon-reload

# 5. Start
sudo systemctl start edgecore
sudo systemctl enable edgecore

# 6. Check status
sudo systemctl status edgecore
sudo journalctl -u edgecore -f
```

**Hot-Reload Config:**
```bash
sudo systemctl reload edgecore
```

---

## Monitoring & Observability

### Prometheus Integration

**Scrape Configuration:**
```yaml
# /etc/prometheus/prometheus.yml
scrape_configs:
  - job_name: 'edgecore'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

**Grafana Dashboard Queries:**
```promql
# Request Rate
rate(edgecore_requests_total[5m])

# Rate Limit Hit Rate
rate(edgecore_rate_limited_total[5m]) / rate(edgecore_requests_total[5m])
```

### Logging

EdgeCore logs to stdout. For production:
```bash
# Docker: Use log driver
docker run --log-driver=json-file --log-opt max-size=10m ...

# Systemd: Logs to journald
sudo journalctl -u edgecore --since today
```

---

## Performance Tuning

### System Limits
```bash
# /etc/security/limits.conf
edgecore soft nofile 65536
edgecore hard nofile 65536
```

### Config Tuning
```json
{
  "rate_limit": 10000,
  "burst": 1000
}
```

For 10k+ RPS, consider:
- Multiple EdgeCore instances behind L4 LB
- Increase file descriptors
- Use keepalive connections

---

## Troubleshooting

**Service won't start:**
```bash
sudo journalctl -u edgecore -n 50
```

**High latency:**
- Check backend health: `curl http://localhost:8080/health`
- Check metrics: `curl http://localhost:8080/metrics`
- Verify network connectivity to backends

**Rate limiting too aggressive:**
Adjust `rate_limit` and `burst` in `config.json`, then reload:
```bash
sudo systemctl reload edgecore
```
