---
name: k8s-resource-analyzer
description: "Analyze Kubernetes pod resource allocation vs actual usage and generate cost-saving recommendations. Use when optimizing pod requests/limits, finding cost-saving opportunities, or planning HPA adoption. Compares P95 usage metrics against allocated resources across prod-green and prod-white clusters."
version: "1.0.0"
category: infrastructure
tags: [kubernetes, cost-optimization, resource-management, k8s]
metadata:
  mcp_servers:
    - remote-friday-mcp-server (Friday MCP for kubectl)
    - grafana (for Prometheus metrics)
  setup: "See SETUP.md - requires Friday MCP and Grafana MCP registration"
  plan: "See PLAN.md for execution flow"
---

# K8s Resource Analyzer

Analyze Kubernetes deployments by comparing actual usage (P95 percentile) against allocated resources.

## Overview

The skill performs four steps:

1. **Fetch pod definitions** from Kubernetes (Friday MCP)
   - Get unique deployments, resource requests, and limits
   - **Filter to analyzable pods**: running pods + completed cronjobs (excludes scaled-down/pending/failed)
   - **Categorize by type**: web, worker, cronjob (for clearer reporting)

2. **Query Prometheus metrics** (Grafana MCP)
   - CPU usage: P95 percentile over configurable time window (default: 48 hours)
   - Memory usage: P95 percentile over configurable time window
   - **Configurable time ranges**: 1 hour to 30 days

3. **Analyze and compare**
   - If no limit set → **Set limit to P95 × 1.5**
   - If P95 > request → Increase resources
   - If P95 < request/2 → Decrease resources (with savings tracking)
   - If variance > 30% → Enable HPA
   - If P95 > limit → Critical: Will be OOMKilled

4. **Generate report**
   - **Organized by action**:
     - 🚨 Set resource limits (missing limits)
     - ⬆️ Increase resources (under-provisioned)
     - ⬇️ Decrease resources (over-provisioned)
     - 💾 Memory optimization summary (savings vs increases)
     - 🎯 HPA candidates
     - 🔴 At-risk pods
   - Pod summary shows breakdown: "350 running (150 web + 180 workers) + 12 cronjobs"
   - Focuses on actively executing workloads

## Quick Start

```bash
# Analyze default time window (48h) across both clusters
python main.py settlements

# Analyze single cluster with 24-hour window
python main.py settlements --cluster prod-green --hours 24

# Analyze 1 week (168 hours) of data
python main.py scrooge --hours 168

# Use environment variables
export K8S_NAMESPACE=settlements
export K8S_TIME_HOURS=72
python main.py
```

## Input Parameters

The skill supports flexible configuration:

| Parameter | Type | Default | Range |
|-----------|------|---------|-------|
| `namespace` | string | required | Any K8s namespace |
| `--cluster` / `-c` | string | prod-green,prod-white | prod-green, prod-white, or comma-separated |
| `--hours` / `-t` | int | 48 | 1-720 hours |

**Environment Variables** (alternative to CLI arguments):
- `K8S_NAMESPACE` - Kubernetes namespace (required)
- `K8S_CLUSTER` - Cluster to analyze (optional)
- `K8S_TIME_HOURS` - Time window in hours (optional)

CLI arguments take precedence over environment variables.

## Prerequisites

**Required MCPs**:
- Friday MCP (for kubectl access)
- Grafana MCP (for Prometheus)

**Setup**:
1. Run: `/rzp-discover:setup`
2. Run: `/rzp-discover:connect-mcps`

See `SETUP.md` for detailed setup instructions.

## Example Output

```json
{
  "status": "success",
  "namespace": "settlements",
  "clusters": ["prod-green", "prod-white"],
  "pod_summary": "350 running (150 web + 180 workers) + 12 cronjobs = 362 total",
  "deployments_analyzed": 8,
  "summary": {
    "total_pods": 362,
    "total_deployments": 8
  },
  "recommendations": {
    "increase_resources": [
      {
        "deployment": "payment-processor",
        "cpu": "500m → 1500m",
        "memory": "512Mi → 1536Mi"
      }
    ],
    "decrease_resources": [
      {
        "deployment": "ledger-service",
        "cpu": "1000m → 300m",
        "memory": "1024Mi → 384Mi"
      }
    ],
    "enable_hpa": [
      {
        "deployment": "fraud-detector",
        "reason": "60% CPU variance"
      }
    ]
  }
}
```

## Execution Flow

See `PLAN.md` for detailed step-by-step execution flow.

## Thresholds

| Metric | Condition | Action |
|--------|-----------|--------|
| **CPU/Memory Limit** | Not set | 🚨 Set limit to P95 × 1.5 |
| **CPU P95** | > request | ⬆️ Increase to P95 × 2 |
| **CPU P95** | < request/2 | ⬇️ Decrease to P95 × 1.5 |
| **Memory P95** | > request | ⬆️ Increase to P95 × 2 |
| **Memory P95** | < request/2 | ⬇️ Decrease to P95 × 1.5 |
| **Variance** | std_dev > 30% | 🎯 Enable HPA |
| **P95 > Limit** | Critical risk | 🔴 Will be OOMKilled |

## Files

| File | Purpose |
|------|---------|
| `PLAN.md` | Execution flow and detailed steps |
| `SETUP.md` | MCP setup and configuration |
| `scripts/main.py` | Orchestrator |
| `scripts/fetch_pods.py` | Query pods via Friday MCP |
| `scripts/fetch_metrics.py` | Query metrics via Grafana MCP |
| `scripts/analyze.py` | Compare and generate recommendations |
| `scripts/report_generator.py` | Format output |

## Notes

- Time range: Default 48 hours (customizable)
- Clusters: prod-green and prod-white
- Datasource: Primary Prometheus (UID: 6ZssswRnk)
- Output: JSON with recommendations grouped by action
