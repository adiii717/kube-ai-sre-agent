# kube-ai-sre-agent

Lightweight, event-driven Kubernetes incident response with AI-powered root cause analysis.

## What Makes This Different

**Event-driven by design**
Watches Kubernetes events in real-time. No manual scans, no external alerting required.

**Minimal footprint**
Tiny controller (~10-20MB) runs 24/7. Heavy analysis runs as on-demand Kubernetes Jobs.

**Predictable costs**
```
Controller: ~$0.50/month (always-on, minimal resources)
Analysis Jobs: $0.001 per incident (spawn → analyze → terminate)
```

**Config-driven**
```yaml
events:
  crashLoopBackOff: true
  imagePullBackOff: true
  healthCheckFailure: false
llm:
  provider: gemini  # or claude, openai
```

## Architecture

```
Event occurs → Controller detects → Spawns Job → Analyzes with LLM → Slack notification → Job terminates
```

- **Controller**: Watches K8s events, spawns Jobs (10m CPU, 20MB RAM)
- **Analysis Job**: Fetches logs, calls LLM, sends alerts (500m CPU, 512MB RAM, 30-60s)

## Installation

### Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- LLM API key (Gemini, Claude, or OpenAI)
- Slack webhook URL (optional)

### Install from OCI Registry

```bash
# Install with Gemini
helm install kube-ai-sre-agent oci://ghcr.io/adiii717/kube-ai-sre-agent \
  --version 0.1.0 \
  --set llm.provider=gemini \
  --set llm.apiKey=YOUR_GEMINI_API_KEY \
  --set slack.webhook=YOUR_SLACK_WEBHOOK

# Install with Claude
helm install kube-ai-sre-agent oci://ghcr.io/adiii717/kube-ai-sre-agent \
  --version 0.1.0 \
  --set llm.provider=claude \
  --set llm.apiKey=YOUR_CLAUDE_API_KEY \
  --set slack.webhook=YOUR_SLACK_WEBHOOK
```

**Note for Apple Silicon (M1/M2) users:** Published images are amd64 only. For local testing on arm64, build from source (see below).

### Install from Source

```bash
git clone https://github.com/adiii717/kube-ai-sre-agent.git
cd kube-ai-sre-agent

helm install kube-ai-sre-agent ./helm/kube-ai-sre-agent \
  --set llm.provider=gemini \
  --set llm.apiKey=YOUR_API_KEY \
  --set slack.webhook=YOUR_WEBHOOK
```

### Configuration

Create a `values.yaml` file:

```yaml
# Enable/disable specific event types
events:
  crashLoopBackOff: true
  imagePullBackOff: true
  healthCheckFailure: true
  oomKilled: true

# LLM provider configuration
llm:
  provider: gemini  # gemini, claude, or openai
  apiKey: "your-api-key"

# Slack notifications
slack:
  enabled: true
  webhook: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

# Resource limits
controller:
  resources:
    requests:
      cpu: 10m
      memory: 20Mi
    limits:
      cpu: 100m
      memory: 64Mi

analyzer:
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 1000m
      memory: 1Gi
```

Install with custom values:

```bash
helm install kube-ai-sre-agent oci://ghcr.io/adiii717/kube-ai-sre-agent \
  --version 0.1.0 \
  -f values.yaml
```

### Verify Installation

```bash
# Check controller is running
kubectl get pods -l app.kubernetes.io/name=kube-ai-sre-agent

# View logs
kubectl logs -l app.kubernetes.io/component=controller -f

# Watch for analysis jobs
kubectl get jobs -w
```

## Uninstall

```bash
helm uninstall kube-ai-sre-agent
```

## Features

- [x] Real-time CrashLoopBackOff detection
- [x] ImagePullBackOff monitoring
- [x] Health check failure alerts
- [x] Multi-LLM support (Gemini, Claude, OpenAI)
- [x] Slack notifications
- [ ] PagerDuty integration
- [ ] Custom event handlers

## Development

### Build Locally (Mac/Linux)

```bash
# Clone repo
git clone https://github.com/adiii717/kube-ai-sre-agent.git
cd kube-ai-sre-agent

# Build binaries
make build

# Build Docker images (uses your local arch - arm64 on M1/M2)
docker build -t ghcr.io/adiii717/kube-ai-sre-agent-controller:local -f Dockerfile.controller .
docker build -t ghcr.io/adiii717/kube-ai-sre-agent-analyzer:local -f Dockerfile.analyzer .

# Install with local images
helm install kube-ai-sre-agent ./helm/kube-ai-sre-agent \
  --set controller.image.tag=local \
  --set analyzer.image.tag=local \
  --set llm.provider=gemini \
  --set llm.apiKey=YOUR_KEY \
  --set slack.enabled=false
```

## License

MIT
