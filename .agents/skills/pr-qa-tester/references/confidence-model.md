# Confidence Model

Score each dimension 0–100, then compute the weighted total.

## Dimensions

### Flow Understanding (40% weight)

| Score | Criteria |
|-------|----------|
| 90-100 | Full call chain known, all request/response shapes understood, error paths identified |
| 70-89 | Main flow known, some edge cases unclear, entry/exit points identified |
| 50-69 | General flow direction known but details missing, dependencies partially mapped |
| 30-49 | Only surface-level understanding from PR diff, flow inferred but not confirmed |
| 0-29 | Cannot determine what the code does or how it integrates |

**Boosters**: Repo skill available (+10), discover agent available (+10), PR description is detailed (+5)
**Penalties**: No tests in PR (-10), unfamiliar service/language (-15), multi-repo change (-10)

### Service Mapping (30% weight)

| Score | Criteria |
|-------|----------|
| 90-100 | All impacted services identified, deployment configs known, dependencies mapped |
| 70-89 | Primary services identified, deployment method clear, some dependencies inferred |
| 50-69 | Main service known, unsure about downstream dependencies |
| 30-49 | Service identified from repo name only, no dependency knowledge |
| 0-29 | Cannot determine which services are impacted |

**Boosters**: Service has devspace.yaml (+10), helm chart exists (+10), discover plugin has agent (+5)
**Penalties**: New service not in devstack (-20), cross-cluster dependency (-10)

### Deployment Clarity (30% weight)

| Score | Criteria |
|-------|----------|
| 90-100 | Helm chart exists, devspace configured, image build pipeline known, custom pod available |
| 70-89 | Helm chart exists, image can be built, deployment path is clear |
| 50-69 | Service deployable but needs first-time pod setup (CI/CD path) |
| 30-49 | Deployment method unclear, may need manual helm chart creation |
| 0-29 | No deployment infrastructure exists for this service |

**Boosters**: Custom pod already exists (+15), devspace hot-reload works (+10)
**Penalties**: No helm chart (-20), no CI image pipeline (-15)

## Decision Thresholds

| Total Score | Action |
|-------------|--------|
| >= 70 | Proceed autonomously, inform user of plan |
| 50-69 | Present findings, ask user to confirm or fill gaps |
| < 50 | Stop and ask user for guidance — too many unknowns |

## Scoring Formula

```
total = (flow_understanding * 0.4) + (service_mapping * 0.3) + (deployment_clarity * 0.3)
```
