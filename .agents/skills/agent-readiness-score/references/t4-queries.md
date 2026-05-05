# T4 — Devstack Runtime Health Queries

Both queries run via `mcp__coralogix-nonprod-server__metrics__range_query_v1`.
Use `start = now-7d`, `end = now`, `step = 8h`, `limit = 20`.

Replace `APP_NAME` with the service's namespace (typically the repo name).

---

## Signal A — Devstack Deployment Uptime SR

Measures what fraction of time the `*-base` deployment in `dev-serve` had ready replicas over the last 7 days.

```promql
(
  count(
    kube_deployment_status_replicas_ready{
      cluster='dev-serve',
      deployment=~'.*-base',
      namespace=~'APP_NAME|APP_NAME-.*',
      namespace!~"capital-loc|capital-lender|doc-vault|iso-connector-base24|frontend-universe-node-demo-app-01|edge-cp|backstage|capital-bnpl|capital-bnpl-ext|capital-collections|capital-es|capital-los|capital-scorecard|litellm"
    }[7d] > 0
  ) by (namespace)
  /
  count(
    kube_deployment_spec_replicas{
      cluster='dev-serve',
      deployment=~'.*-base',
      namespace=~'APP_NAME|APP_NAME-.*',
      namespace!~"capital-loc|capital-lender|doc-vault|iso-connector-base24|frontend-universe-node-demo-app-01|edge-cp|backstage|capital-bnpl|capital-bnpl-ext|capital-collections|capital-es|capital-los|capital-scorecard|litellm"
    }[7d] > 0
  ) by (namespace)
) * 100
```

Store as `DEVSTACK_UPTIME_PCT`. Empty result → service not on devstack → score 0.

---

## Signal B — E2E Test Success Rate

Measures the ratio of succeeded E2E workflow runs to total runs over the last 7 days, across PR validation, master validation, and dry-run workflow classes.

```promql
sum(
  increase(
    argo_workflows_e2e_status_total{
      status="Succeeded",
      namespace="argo-workflows",
      workflow_class=~"end_to_end_tests_service_pr_validation|end_to_end_tests_service_master_validation|end_to_end_tests_dry_run",
      service="APP_NAME"
    }[7d]
  )
)
/
sum(
  increase(
    argo_workflows_e2e_status_total{
      namespace="argo-workflows",
      workflow_class=~"end_to_end_tests_service_pr_validation|end_to_end_tests_service_master_validation|end_to_end_tests_dry_run",
      service="APP_NAME"
    }[7d]
  )
)
```

Result is a fraction (0–1). Multiply by 100 to get `E2E_SUCCESS_RATE_PCT`.
Empty result → no E2E runs in 7d → mark N/A, exclude from denominator.
Skip entirely if `T1-E2E` is in `EXCEPTIONS`.

---

## Reference Dashboard

[DevProductivity FY26 Q1 OKRs — Service Specific Uptime](https://grafana.np.razorpay.in/d/f6c01c72-3165-4008-b570-c7dea5eb177e/devproductivity-fy26-q1-okrs?orgId=1&from=now-7d&to=now)
