# K8s Resource Analyzer - Setup Guide

## Prerequisites

This skill requires two MCP servers to be registered in Claude Code:

1. **Friday MCP** (remote-friday-mcp-server) - For Kubernetes access
2. **Grafana MCP** - For Prometheus metrics

---

## Automatic Setup (Recommended)

### Via rzp-discover Plugin

The easiest way to set up all MCPs at once:

1. **Open Claude Code** → Settings/Plugins

2. **Add Marketplace**:
   ```
   Repository: https://github.com/razorpay/claude-plugins.git
   ```

3. **Install the `rzp-discover` plugin**

4. **Run setup command**:
   ```
   /rzp-discover:setup
   ```
   This will:
   - Clone Razorpay repositories
   - Configure GitHub CLI
   - Set up all MCPs

5. **Connect all MCPs**:
   ```
   /rzp-discover:connect-mcps
   ```
   This will:
   - Register Friday MCP
   - Register Grafana MCP
   - Register all other Razorpay MCPs
   - Prompt for credentials if needed

---

## Manual Setup (If Preferred)

### 1. Friday MCP (Kubernetes)

**Status**: Automatically configured by `/rzp-discover:setup`

**Manual Configuration** (if needed):
```json
{
  "mcpServers": {
    "remote-friday-mcp-server": {
      "type": "streamable-http",
      "url": "https://friday-mcp.razorpay.com",
      "streamable": true
    }
  }
}
```

**Provides**:
- Tool: `mcp__remote-friday-mcp-server__kubectl_generic`
- Access to: prod-green, prod-white clusters

---

### 2. Grafana MCP (Prometheus)

**Status**: Automatically configured by `/rzp-discover:setup`

**Manual Configuration** (if needed):
```json
{
  "mcpServers": {
    "grafana": {
      "type": "sse",
      "url": "https://grafana-mcp.razorpay.com/sse"
    }
  }
}
```

**Provides**:
- Tool: `mcp__grafana__query_prometheus`
- Datasource UID: `6ZssswRnk` (Primary Prometheus)

**To add custom Grafana dashboard for monitoring**:
1. Go to Grafana: https://vajra.razorpay.com
2. Create a new dashboard
3. Add panels with these queries:

**Panel 1: CPU Usage (settlements)**
```promql
sum(rate(container_cpu_usage_seconds_total{
  namespace="settlements",
  container!="POD"
}[2m])) by (pod)
```

**Panel 2: Memory Usage (settlements)**
```promql
sum(container_memory_working_set_bytes{
  namespace="settlements"
}) by (pod)
```

**Panel 3: CPU Requests**
```promql
sum(kube_pod_container_resource_requests{
  namespace="settlements",
  resource="cpu"
}) by (pod)
```

**Panel 4: Memory Requests**
```promql
sum(kube_pod_container_resource_requests{
  namespace="settlements",
  resource="memory"
}) by (pod)
```

---

## Verify Setup

### Check if MCPs are registered:

```bash
# In Claude Code, run:
/mcp list
```

Expected output should show:
```
✅ remote-friday-mcp-server
✅ grafana
```

### Test Friday MCP:

Try a simple kubectl query to verify it works:
```bash
# In Claude Code, this would be executed via the MCP:
kubectl get pods -n settlements -o json
```

### Test Grafana MCP:

Query a simple metric:
```
Show me the CPU usage for the settlements namespace over the last 2 hours
```

---

## Running the Skill

Once MCPs are set up, run the skill:

```bash
Analyze K8s resources for settlements namespace
```

Or directly:
```bash
python3 scripts/main.py settlements
```

---

## Troubleshooting

### Friday MCP not found

**Symptom**: "Cannot find mcp__remote-friday-mcp-server__kubectl_generic"

**Solution**:
1. Run `/rzp-discover:connect-mcps` again
2. Verify VPN connection (if required)
3. Check firewall settings

### Grafana MCP not found

**Symptom**: "Cannot find mcp__grafana__query_prometheus"

**Solution**:
1. Run `/rzp-discover:connect-mcps` again
2. Verify you have Grafana API access
3. Check datasource UID in Grafana (should be `6ZssswRnk`)

### No pods found in namespace

**Symptom**: Returns "no_data"

**Possible causes**:
1. Namespace doesn't exist → check spelling
2. No pods in namespace → check if services are running
3. Friday MCP not working → test with `/mcp list`

---

## Support

- **Friday MCP**: https://github.com/razorpay/friday-mcp
- **Grafana**: https://vajra.razorpay.com
- **Prometheus Docs**: https://prometheus.io/docs/prometheus/latest/querying/basics/

---
