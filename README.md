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

## Quick Start

```bash
helm install sre-agent ./helm/kube-ai-sre-agent \
  --set llm.apiKey=YOUR_KEY \
  --set llm.provider=gemini \
  --set slack.webhook=YOUR_WEBHOOK
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

```bash
# Build
make build

# Run locally
make run

# Build images
make docker-build

# Deploy to cluster
make deploy
```

## License

MIT
