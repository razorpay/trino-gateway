# Canary Metrics

Read this file when the task involves canary analysis, progressive delivery, or baseline-versus-canary comparison.

These are infra- or rollout-owned comparison metrics rather than service-specific implementation details.

## Metric Families

| Metric family | Metric or form | Description |
|---|---|---|
| Canary versus baseline comparison | Custom canary and baseline metrics | Comparison of performance between canary and stable paths |
| Error rate comparison | Error rate comparison metric | Relative error percentage between canary and stable |
| Latency comparison | Histogram bucket-based latency | Relative response-time comparison between canary and stable |
| Traffic distribution | Traffic shaping metric | Percentage split of traffic between canary and baseline |
| Rollback trigger signals | Custom SLO signal metric | Signals used to determine rollback conditions |
| Success rate analysis | `http_responses_total` | Stability and success-rate comparison for the rollout |
