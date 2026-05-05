---
name: alert-coverage-analyzer
description: Analyze alert coverage for Razorpay services by discovering multi-source monitoring (Prometheus, CloudWatch RDS, Performance Insights, Coralogix logs), scanning application metrics, and identifying missing business-critical metrics. Leverages repo skill Observability/Monitoring sections (when available) and verifies existing coverage with user before recommendations. Use when the user asks to analyze alert coverage, check for missing alerts, audit monitoring completeness, add metrics and alerts for a service, or improve observability. Only works with Razorpay repositories.
---

# Alert Coverage Analyzer

Comprehensive workflow to analyze, identify, and add missing metrics and alerts for Razorpay services.

## Prerequisites

- Must be run in a Razorpay repository (verified via git remote)
- Requires GitHub CLI (`gh`) for cloning repos and creating PRs
- **Strongly Recommended:** Repo skill for the service with an **Observability/Monitoring section** (format: `{service-name}-skill`)
  - Provides documented SLIs/SLOs to validate metric coverage
  - Lists known monitoring gaps (pre-validated blind spots)
  - Defines service-specific metric naming conventions
  - Documents critical flows requiring monitoring
  - Significantly improves analysis accuracy and completeness (can identify 2-3x more relevant gaps)

## Workflow Overview

**Discovery Phase:**
1. Verify Razorpay repository
2. Identify service name (auto-detect or manual)
3. **Select target regions** - Ask user which regions to deploy alerts to (prod-rules, prod-us-rules, prod-sg-rules)
4. **Check repo skill** (MANDATORY - read observability docs first)
5. Setup GitHub CLI
6. Locate alert-rules repository
7. **Multi-source discovery (across all selected regions):**
   - Application alerts (Prometheus)
   - Infrastructure alerts (RDS, EKS, Lambda)
   - ASK: Log-based alerts (Coralogix)
   - ASK: Cloud monitoring (Performance Insights)
8. Scan repository for existing metrics
9. **Verify understanding with user** (prevent false positives)
10. Identify actual missing metrics

**Implementation Phase** (after user confirmation):
11. Add metrics to application code
12. Create alert rules (for all selected regions)
13. Push branches
14. Create PRs (or provide manual fallback)

## Examples

### Basic Usage

**User:** "Analyze alert coverage for this service"

**Workflow:**
1. Verifies Razorpay repository and identifies service name
2. Asks user to select target regions (e.g., "All regions")
3. Checks for repo skill with observability documentation
4. Scans existing alerts across all regions (Prometheus + CloudWatch infrastructure)
5. Verifies coverage with user (histogram _count, Performance Insights, log alerts)
6. Identifies actual gaps (2-5 missing metrics typically)
7. Creates branches, adds metrics, creates alert rules for all selected regions, opens PRs

**Typical Output:**
```
✅ Target regions: prod-rules, prod-us-rules, prod-sg-rules
✅ Found 403 alerts in prod-rules, 58 in prod-us-rules, 85 in prod-sg-rules
✅ ~600 metrics, 5 RDS instances monitored
✅ Verified: Histogram _count, HandleHelper tracking, Performance Insights, Coralogix panics
⚠️  Found 2 actual gaps: External service latency, Broken tracing pipeline
✅ PRs created: /pull/1234 (metrics), /pull/5678 (alerts for all regions)
```

**For detailed examples, see:**
- [Detailed Examples](references/examples.md) - Complete workflow, multi-source verification, first-time setup, troubleshooting scenarios

## Workflow

**See [Detailed Workflow Steps](references/workflow-steps.md) for complete implementation instructions.**

**Quick reference:**

1. **Verify Razorpay repository** - Run verification script
2. **Identify service name** - Auto-detect or ask user
3. **Select target regions** - Ask which regions to deploy alerts to:
   - All regions (prod-rules, prod-us-rules, prod-sg-rules)
   - India only (prod-rules)
   - US only (prod-us-rules)
   - Singapore only (prod-sg-rules)
   - Custom selection
4. **Check repo skill** (MANDATORY) - Read observability docs first to understand existing monitoring
5. **Setup GitHub CLI** - Install and authenticate if needed
6. **Locate alert-rules repo** - Ask user for path or clone to `/tmp`
7. **Multi-source discovery (across all selected regions):**
   - Search application alerts (Prometheus) in prod-rules, prod-us-rules, prod-sg-rules
   - Search infrastructure alerts (RDS, EKS, Lambda, ElastiCache)
   - ASK user about log-based alerts (Coralogix)
   - ASK user about cloud monitoring (Performance Insights)
   - Display which regions have existing alerts
8. **Scan repository for metrics** - Find existing Prometheus metrics in code
9. **Verify with user** (CRITICAL) - Confirm understanding about histogram _count, HandleHelper, Performance Insights, log alerts
10. **Identify actual gaps** - Compare against repo skill SLIs/SLOs and business flows
11. **Add metrics to code** - Create branch, define metrics, instrument code, commit
12. **Create alert rules** - Create branch in alert-rules, add YAML rules for each selected region, commit
13. **Push and create PRs** - Push both branches, create PRs with gh CLI (PR title includes regions)
14. **Handle failures** - See [Troubleshooting Guide](references/troubleshooting.md) if PR creation fails

## References

- **Alert Patterns:** See `references/alert-patterns.md` for common alert patterns and thresholds
- **Metric Examples:** See `references/metric-examples.md` for business-critical metric patterns
- **Monitoring Sources:** See `references/monitoring-sources.md` for comprehensive checklist of all monitoring sources to verify (Prometheus, CloudWatch, Performance Insights, Coralogix, etc.)

## Notes

- **Multi-Source Monitoring:** Most services use 3-5 monitoring tools (Prometheus, CloudWatch, Coralogix, Performance Insights). Always verify coverage across ALL sources before recommending gaps. See `references/monitoring-sources.md` for complete checklist.
- **Repo Skills:** If a repo skill exists for the service, its **Observability/Monitoring section is the most authoritative source** for:
  - What metrics should exist (documented SLIs/SLOs)
  - Known monitoring blind spots (documented gaps)
  - Service-specific metric naming conventions
  - Critical flows that require monitoring
  - Alert severity and escalation policies
  Always start with the observability section when identifying gaps
- Metrics follow Prometheus naming conventions: `<namespace>_<metric>_<unit>_<type>`
- Alert files are organized by environment and region:
  - `prod-rules/` - India (default production)
  - `prod-us-rules/` - US region
  - `prod-sg-rules/` - Singapore region
  - `nonprod-rules/` - Non-production environments
- **Regional deployment:** Services running in multiple regions should have alerts in all regional folders. Always ask user which regions to target.
- All alerts require: severity, bu, pod, service, slack_channel labels
- All alerts require: identifier, description, Runbook, vajra_link annotations
- Thresholds should be based on production traffic patterns, not arbitrary values
- **CRITICAL:** Never use high cardinality labels (merchant_id, terminal_id, user_id, payment_id, etc.) - causes Prometheus memory exhaustion
- **CRITICAL:** External service alerts are owned by those services - do not add alerts for downstream API calls. **Exception:** Always add latency metrics (p99, p95, p50) and 5XX error metrics for downstream service calls to track dependency health and SLA compliance
- Label cardinality must be < 100 unique values - use logs for per-entity debugging
