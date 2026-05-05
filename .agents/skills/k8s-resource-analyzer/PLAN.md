# K8s Resource Analyzer - Execution Plan

## Overview

This skill analyzes Kubernetes pod resource allocation vs actual usage metrics and generates recommendations for:
- ✅ **Setting missing resource limits** - Detect pods with no CPU/Memory limits and recommend safe limits
- ⬆️ **Increasing under-provisioned resources** - P95 usage exceeds requests
- ⬇️ **Decreasing over-provisioned resources** - P95 usage is < 50% of requests, with savings tracking
- 🎯 **Enabling HPA** - For variable workloads (variance > 30%)
- 💾 **Memory optimization** - Separate tracking of memory savings vs increases needed

## Simple 4-Step Flow

```
┌─────────────────────────────────────────────────────────────┐
│ Input: Namespace (e.g., "settlements")                      │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
     ┌────────────────────────────────┐
     │ STEP 1: FETCH POD DEFINITIONS   │
     │ via Friday MCP                  │
     └────────────────────────────────┘
                      │
         ┌────────────┴────────────┐
         ▼                         ▼
   prod-green                   prod-white
   Cluster A                    Cluster B
         │                         │
         └────────────┬────────────┘
                      ▼
     ┌────────────────────────────────┐
     │ EXTRACT:                        │
     │ - Unique deployments            │
     │ - Resource requests (CPU/Mem)   │
     │ - Resource limits (CPU/Mem)     │
     └────────────────────────────────┘
                      │
                      ▼
     ┌────────────────────────────────┐
     │ STEP 2: FETCH METRICS           │
     │ via Grafana MCP                 │
     │ (Prometheus queries)            │
     └────────────────────────────────┘
                      │
         ┌────────────┴────────────┐
         ▼                         ▼
     CPU Usage               Memory Usage
     (rate 2m)              (working set)
     P95 percentile         P95 percentile
         │                         │
         └────────────┬────────────┘
                      ▼
     ┌────────────────────────────────┐
     │ STEP 3: ANALYZE & RECOMMEND     │
     │                                 │
     │ Compare:                        │
     │ P95 Usage vs Requests/Limits    │
     │                                 │
     │ Apply thresholds:               │
     │ - If P95 > Request → Increase   │
     │ - If P95 < Request/2 → Decrease │
     │ - If variance > 30% → Enable HPA│
     └────────────────────────────────┘
                      │
                      ▼
     ┌────────────────────────────────┐
     │ Output: JSON Report with:       │
     │ - Summary table                 │
     │ - Recommendations per action    │
     │ - HPA candidates                │
     └────────────────────────────────┘
```

---

## Detailed Steps

### STEP 1: Fetch Pod Definitions (Friday MCP)

**MCP Tool**: `mcp__remote-friday-mcp-server__kubectl_generic`

**Query**:
```bash
kubectl get pods -n {namespace} --context={cluster} -o json
```

**For each pod, extract**:
- Pod name
- Namespace
- Cluster (prod-green or prod-white)
- Container name
- Resource requests: `cpu`, `memory`
- Resource limits: `cpu`, `memory`

**Filtering** (UPDATED):
- Include pods with **all containers ready or completed**
- Includes:
  - Running workloads (phase = "Running", ready = true)
  - Completed cronjobs (phase = "Completed", containers terminated)
- Excludes:
  - Scaled-down deployments (0/0 READY status)
  - Pending pods (no metrics available)
  - Failed pods (can't analyze)
  - Pods too new (< 1 hour old, insufficient metrics)

**Output**: Analyzable pod list grouped by deployment, with categorization:
- Web pods: main service deployments
- Worker pods: background task workers
- Cronjobs: scheduled job executions
- Summary format: "350 running (150 web + 180 workers) + 12 cronjobs = 362 total"

---

### STEP 2: Fetch Prometheus Metrics (Grafana MCP)

**MCP Tool**: `mcp__grafana__query_prometheus`

**Configuration**:
- Datasource UID: `6ZssswRnk` (Primary Prometheus)
- Time range: Last 48 hours (default)
- Step: 300 seconds (5 minutes)
- Query type: `range`

**Queries**:

```promql
# CPU Usage
sum(rate(container_cpu_usage_seconds_total{
  namespace="{namespace}",
  container!="POD"
}[2m])) by (pod)
```

```promql
# Memory Usage
sum(container_memory_working_set_bytes{
  namespace="{namespace}"
}) by (pod)
```

**Extract**: P95 percentile for each pod

---

### STEP 3: Analyze & Compare

**Thresholds**:

| Metric | Condition | Action |
|--------|-----------|--------|
| **CPU/Memory** | No limit set | 🚨 Set limit to (P95 × 1.5) |
| **CPU** | P95 > request | ⬆️ Increase to (P95 × 2) |
| **CPU** | P95 < request/2 | ⬇️ Decrease to (P95 × 1.5) |
| **Memory** | P95 > request | ⬆️ Increase to (P95 × 2) |
| **Memory** | P95 < request/2 | ⬇️ Decrease to (P95 × 1.5) |
| **Variance** | std_dev > 30% | 🎯 Enable HPA |
| **Risk** | P95 > limit | 🔴 Will be OOMKilled |

---

## Input Parameters

The skill supports flexible input configuration via CLI arguments or environment variables:

### CLI Arguments
```bash
python main.py <namespace> [--cluster CLUSTER] [--hours HOURS]

# Examples:
python main.py settlements
python main.py settlements --cluster prod-green
python main.py settlements --cluster prod-green,prod-white --hours 24
python main.py scrooge -c prod-white -t 168  # 1 week analysis
```

### Environment Variables
Alternative way to configure without CLI arguments:
```bash
export K8S_NAMESPACE=settlements
export K8S_CLUSTER=prod-green,prod-white
export K8S_TIME_HOURS=48
python main.py  # Uses env vars
```

### Parameter Details
- **namespace** (required): Kubernetes namespace to analyze
- **--cluster / -c** (optional): Cluster to analyze
  - Values: `prod-green`, `prod-white`, or comma-separated list (default: both)
  - Examples: `prod-green`, `prod-green,prod-white`
- **--hours / -t** (optional): Time window in hours (default: 48)
  - Valid range: 1-720 hours (30 days max)
  - Examples: `24` (1 day), `168` (1 week), `720` (30 days)

---

## Prerequisites

Before running the skill:

1. **Friday MCP** must be registered in Claude Code
   - Provides: `mcp__remote-friday-mcp-server__kubectl_generic`
   - Access to: prod-green and prod-white clusters

2. **Grafana MCP** must be registered in Claude Code
   - Provides: `mcp__grafana__query_prometheus`
   - Access to: Prometheus via Grafana API

See `SETUP.md` for setup instructions.

---

## Files

| File | Purpose |
|------|---------|
| `scripts/main.py` | Orchestrator (check MCPs, coordinate flow) |
| `scripts/fetch_pods.py` | Step 1: Query Friday MCP for pods |
| `scripts/fetch_metrics.py` | Step 2: Query Grafana MCP for metrics |
| `scripts/analyze.py` | Step 3: Compare and generate recommendations |
| `scripts/report_generator.py` | Format output as JSON report |

---
