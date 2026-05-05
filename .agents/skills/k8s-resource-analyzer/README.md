# K8s Resource Analyzer Skill

Simple Kubernetes resource optimization analyzer for Razorpay.

## What It Does

Analyzes pod resource allocation vs actual usage (P95 percentile) and generates cost-saving recommendations.

```
Friday MCP         Grafana MCP
    ↓                  ↓
Fetch Pods    ←→    Query Metrics
    ↓                  ↓
    └─→  ANALYZE  ←─┘
          ↓
    RECOMMENDATIONS
```

## Quick Start

### 1. Setup MCPs

Run these commands to set up Friday MCP and Grafana MCP:

```bash
/rzp-discover:setup
/rzp-discover:connect-mcps
```

See `SETUP.md` for detailed instructions.

### 2. Run Analysis

```bash
Analyze K8s resources for settlements namespace
```

### 3. Get Results

JSON output with recommendations grouped by action (increase, decrease, enable HPA).

---

## Documentation

| Document | Purpose |
|----------|---------|
| `SKILL.md` | Skill definition and examples |
| `PLAN.md` | Execution flow (3-step process) |
| `SETUP.md` | MCP configuration and troubleshooting |
| `references/prometheus-queries.md` | PromQL query reference |

---

## Simple 3-Step Flow

1. **Fetch Pods** (Friday MCP)
   - Query `kubectl get pods -n {namespace}` for both clusters
   - Extract deployment names and resource requests/limits

2. **Query Metrics** (Grafana MCP)
   - CPU usage: `container_cpu_usage_seconds_total[2m]`
   - Memory usage: `container_memory_working_set_bytes`
   - Calculate P95 percentile over 48 hours

3. **Analyze & Recommend**
   - If P95 > request → Increase
   - If P95 < request/2 → Decrease
   - If variance > 30% → Enable HPA

---

## Files

```
.
├── SKILL.md              # Skill definition
├── PLAN.md               # Execution flow
├── SETUP.md              # MCP setup
├── README.md             # This file
├── references/
│   └── prometheus-queries.md  # PromQL queries
└── scripts/
    ├── main.py           # Orchestrator
    ├── fetch_pods.py     # Friday MCP integration
    ├── fetch_metrics.py  # Grafana MCP integration
    ├── analyze.py        # Analysis logic
    └── report_generator.py  # Output formatting
```

---

## Requirements

- **Friday MCP**: `remote-friday-mcp-server` for Kubernetes access
- **Grafana MCP**: `grafana` for Prometheus metrics

Setup via:
```bash
/rzp-discover:setup
/rzp-discover:connect-mcps
```

---

## Example

**Input**:
```
Analyze K8s resources for settlements
```

**Output**:
```json
{
  "status": "success",
  "namespace": "settlements",
  "recommendations": {
    "increase_resources": [
      {"deployment": "payment-processor", "cpu": "500m → 1500m"}
    ],
    "decrease_resources": [
      {"deployment": "ledger-service", "cpu": "1000m → 300m"}
    ],
    "enable_hpa": [
      {"deployment": "fraud-detector"}
    ]
  }
}
```

---

## Notes

- Analysis window: 48 hours (customizable)
- Clusters: prod-green, prod-white
- Metrics: P95 percentile
- Output: JSON format

For detailed information, see `PLAN.md` and `SETUP.md`.
