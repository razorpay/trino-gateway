# Roles and Responsibilities

RACI matrix and responsibility definitions for data platform operations, data quality, and system ownership.

## Overview

Clear ownership and responsibilities are critical for data platform success. This document defines who is Responsible, Accountable, Consulted, and Informed (RACI) for various data platform activities.

### RACI Definitions

- **R (Responsible)**: Does the work to complete the task
- **A (Accountable)**: Ultimately answerable for completion and has authority to approve
- **C (Consulted)**: Provides input and expertise (two-way communication)
- **I (Informed)**: Kept up-to-date on progress (one-way communication)

---

## Team Roles

### Platform Team (Data Platform Engineering)

**Responsibilities:**
- Manage infrastructure (clusters, storage, compute)
- Build and maintain data pipelines
- Ensure platform reliability (SLAs, monitoring, on-call)
- Provide tooling and frameworks
- Platform-level governance (quotas, access control)

**Systems Owned:**
- Iceberg lakehouse
- Trino clusters
- Spark clusters
- Flink jobs (CDC, streaming)
- Kafka infrastructure
- Monitoring and alerting

### Application Teams (Product Engineering)

**Responsibilities:**
- Own source data (Aurora, DynamoDB)
- Define schema and data models
- Ensure data quality at source
- Implement application-level CDC
- Provide domain expertise

**Systems Owned:**
- Aurora (application databases)
- DynamoDB tables
- Application-specific APIs

### Analytics Team (Data Analysts, BI Engineers)

**Responsibilities:**
- Define analytical requirements
- Build dashboards and reports
- Query data (Trino, Metabase)
- Validate data quality (analytical checks)
- Document insights and findings

**Systems Used:**
- Trino (ad-hoc queries)
- Metabase / Looker (BI)
- Pinot (dashboard APIs)
- Iceberg (historical data)

### Data Science Team (ML Engineers, Data Scientists)

**Responsibilities:**
- Feature engineering
- ML model development and training
- Productionize ML pipelines
- Monitor model performance

**Systems Used:**
- Spark (feature engineering)
- Iceberg (training data)
- Jupyter notebooks
- ML platforms (SageMaker, Kubeflow)

### Data Governance Team (Data Stewards, Compliance)

**Responsibilities:**
- Define data policies (access, retention, privacy)
- Enforce governance (Apache Ranger)
- Manage data catalog
- Audit compliance (GDPR, PCI-DSS)
- Handle data requests (access, deletion)

**Systems Owned:**
- Apache Ranger (access control)
- Data catalog
- Compliance monitoring

---

## RACI Matrix: Data Platform Operations

| Activity | Platform Team | App Team | Analytics | Data Science | Governance |
|----------|--------------|----------|-----------|--------------|------------|
| **Infrastructure Management** |
| Provision clusters (Trino, Spark) | R/A | I | I | I | I |
| Scale infrastructure | R/A | C | I | I | I |
| Upgrade software versions | R/A | I | I | I | I |
| Monitor platform health | R/A | I | I | I | I |
| On-call rotation (platform) | R/A | I | I | I | I |
| **Data Pipelines** |
| Build CDC pipelines | R/A | C | I | I | C |
| Build batch ETL | R | C | C | C | C |
| Monitor pipeline health | R/A | I | I | I | I |
| Debug pipeline failures | R | C | C | C | I |
| Optimize pipeline performance | R/A | C | I | I | I |
| **Data Quality** |
| Define quality metrics | C | R | C | C | A |
| Implement validation checks | R | C | C | C | C |
| Monitor data quality | R | R | R | R | A |
| Fix data quality issues | C | R/A | I | I | C |
| **Schema Management** |
| Define source schema | I | R/A | C | C | C |
| Evolve schema (backwards-compatible) | C | R/A | C | C | C |
| Breaking schema changes | C | R/A | C | C | A |
| Migrate schemas | R | A | C | C | C |
| **Access Control** |
| Define access policies | C | C | C | C | R/A |
| Implement policies (Ranger) | R | I | I | I | A |
| Grant access requests | R | I | I | I | A |
| Audit access logs | R | I | I | I | A |
| **Cost Management** |
| Set budgets | C | C | C | C | A |
| Monitor costs | R/A | I | I | I | I |
| Optimize query costs | R | C | C | C | I |
| Enforce quotas | R/A | I | I | I | C |

---

## RACI Matrix: Data Lifecycle

| Activity | Platform Team | App Team | Analytics | Data Science | Governance |
|----------|--------------|----------|-----------|--------------|------------|
| **Data Creation** |
| Create source data | I | R/A | I | I | C |
| Define data model | C | R/A | C | C | C |
| Validate at creation | C | R/A | I | I | C |
| **Data Ingestion** |
| Ingest to warm tier (TiDB) | R/A | C | I | I | C |
| Ingest to OLAP (Pinot) | R/A | C | I | I | C |
| Ingest to cold tier (Iceberg) | R/A | C | I | I | C |
| Monitor ingestion lag | R/A | I | I | I | I |
| **Data Transformation** |
| Define business logic | C | C | R | R | C |
| Implement transformations | R | C | C | C | C |
| Test transformations | R | C | R | R | C |
| Deploy transformations | R/A | C | I | I | C |
| **Data Serving** |
| Query data (ad-hoc) | C | C | R/A | R/A | I |
| Build dashboards | C | C | R/A | I | I |
| Build APIs | R | R/A | C | C | C |
| Optimize queries | R | C | C | C | I |
| **Data Archival** |
| Define retention policies | C | C | C | C | R/A |
| Implement archival | R/A | I | I | I | C |
| Delete expired data | R | C | I | I | A |
| **Data Deletion (Compliance)** |
| Receive deletion requests | I | I | I | I | R/A |
| Validate request | C | C | I | I | R/A |
| Execute deletion | R | R | I | I | A |
| Verify deletion | R | R | I | I | A |

---

## RACI Matrix: System-Specific Responsibilities

### Aurora / DynamoDB (Source Databases)

| Activity | Platform Team | App Team | Analytics | Governance |
|----------|--------------|----------|-----------|------------|
| Schema design | C | R/A | C | C |
| Write application code | I | R/A | I | I |
| Enable CDC (Debezium) | R | A | I | C |
| Monitor database health | C | R/A | I | I |
| Optimize queries | C | R/A | I | I |
| Backup and recovery | C | R/A | I | I |
| Access control | C | R/A | I | A |

### TiDB (Warm Tier)

| Activity | Platform Team | App Team | Analytics | Governance |
|----------|--------------|----------|-----------|------------|
| Provision cluster | R/A | I | I | I |
| Replicate from Aurora | R/A | C | I | C |
| Define retention policy | C | C | C | R/A |
| Archive to Iceberg | R/A | I | I | C |
| Query optimization | R | C | C | I |
| Access control | R | I | I | A |

### OLAP (Pinot / ClickHouse)

| Activity | Platform Team | App Team | Analytics | Governance |
|----------|--------------|----------|-----------|------------|
| Provision cluster | R/A | I | I | I |
| Define schema | R | C | C | C |
| Build pre-aggregation pipeline | R | C | C | I |
| Ingest data | R/A | C | I | I |
| Create indexes | R | C | C | I |
| Query API (dashboards) | R | R/A | R/A | I |
| Monitor query performance | R/A | I | I | I |
| Access control | R | I | I | A |

### Iceberg (Lakehouse)

| Activity | Platform Team | App Team | Analytics | Data Science | Governance |
|----------|--------------|----------|-----------|--------------|------------|
| Manage Iceberg catalog | R/A | I | I | I | C |
| Ingest data (batch) | R | C | C | C | C |
| Ingest data (streaming) | R/A | C | I | I | C |
| Define partitioning | R | C | C | C | C |
| Schema evolution | R | C | C | C | C |
| Compact small files | R/A | I | I | I | I |
| Set retention policies | C | C | C | C | R/A |
| Access control | R | I | I | I | A |

### Trino

| Activity | Platform Team | App Team | Analytics | Data Science | Governance |
|----------|--------------|----------|-----------|--------------|------------|
| Provision cluster | R/A | I | I | I | I |
| Configure connectors | R/A | C | C | C | C |
| Create resource groups | R/A | C | C | C | C |
| Run ad-hoc queries | C | C | R/A | R/A | I |
| Optimize queries | R | C | R | R | I |
| Monitor query performance | R/A | I | I | I | I |
| Kill long-running queries | R/A | I | I | I | I |

### Spark

| Activity | Platform Team | App Team | Analytics | Data Science | Governance |
|----------|--------------|----------|-----------|--------------|------------|
| Provide Spark clusters | R/A | I | I | I | I |
| Build ETL jobs | R | C | C | C | C |
| Build ML pipelines | C | I | C | R/A | C |
| Schedule jobs (Airflow) | R | C | C | C | I |
| Monitor job execution | R/A | I | I | I | I |
| Optimize job performance | R | C | C | C | I |
| Debug job failures | R | C | C | C | I |

### Flink

| Activity | Platform Team | App Team | Analytics | Governance |
|----------|--------------|----------|-----------|------------|
| Provision Flink cluster | R/A | I | I | I |
| Build CDC jobs | R/A | C | I | C |
| Build streaming jobs | R | C | C | C |
| Monitor job health | R/A | I | I | I |
| Handle backpressure | R/A | C | I | I |
| Savepoint and recovery | R/A | I | I | I |

---

## Incident Response

### Incident Types

| Incident Type | Primary Responder | Escalation |
|--------------|-------------------|------------|
| Platform outage (Trino down) | Platform Team | Engineering leadership |
| Data quality issue | App Team (source) | Data Governance |
| Pipeline failure | Platform Team | App Team (if source issue) |
| Query performance | Platform Team | Analytics / Data Science |
| Access control issue | Governance Team | Security team |
| Cost spike | Platform Team | Finance, Engineering leadership |

### RACI: Incident Handling

| Activity | Platform Team | App Team | Analytics | Governance | Leadership |
|----------|--------------|----------|-----------|------------|------------|
| Detect incident | R/A | R | R | R | I |
| Triage severity | R/A | C | C | C | I |
| Communicate status | R | I | I | I | I |
| Investigate root cause | R | C | C | C | I |
| Implement fix | R/A | C | I | I | I |
| Verify resolution | R/A | C | C | C | I |
| Post-mortem | R/A | C | C | C | I |
| Implement prevention | R/A | C | I | I | A |

---

## Decision Authority

### System Selection

| Decision | Authority | Consulted | Informed |
|----------|-----------|-----------|----------|
| Add new data system | Platform Lead | Platform Team, App Teams, Governance | Engineering leadership |
| Upgrade major version | Platform Lead | Platform Team | All users |
| Deprecate system | Engineering Leadership | Platform Team, All users, Governance | Company-wide |

### Data Policies

| Decision | Authority | Consulted | Informed |
|----------|-----------|-----------|----------|
| Define access policy | Data Governance Lead | Legal, Security, App Teams | Platform Team |
| Define retention policy | Data Governance Lead | Legal, Compliance, App Teams | Platform Team |
| Grant PII access | Data Governance Lead | Legal, Security | Platform Team |
| Data deletion (compliance) | Data Governance Lead | Legal | Platform Team, App Team |

### Cost and Budgets

| Decision | Authority | Consulted | Informed |
|----------|-----------|-----------|----------|
| Set platform budget | Engineering Leadership | Finance, Platform Lead | Platform Team |
| Enforce user quotas | Platform Lead | Platform Team | All users |
| Approve cost spike | Engineering Leadership | Platform Lead, Finance | Platform Team |

---

## Service Level Agreements (SLAs)

### Platform Team Commitments

| Service | SLA | Measurement |
|---------|-----|-------------|
| Trino availability | 99% uptime | Monthly uptime % |
| Spark job success rate | 95% | Successful jobs / Total jobs |
| Iceberg query latency | p99 < 60s | Query performance metrics |
| Pipeline freshness | 95% on-time | On-time arrivals / Total |
| Incident response time | < 1 hour (critical) | Time to acknowledge |

### Application Team Commitments

| Service | SLA | Measurement |
|---------|-----|-------------|
| Source data quality | 99% valid records | Data quality checks |
| Schema change notice | 2 weeks advance | Change notifications |
| CDC uptime | 99% | Debezium uptime |

### Analytics Team Commitments

| Service | SLA | Measurement |
|---------|-----|-------------|
| Dashboard accuracy | 99.9% | Validation checks |
| Report delivery | On-time | Delivery vs schedule |

---

## Communication Channels

### Routine Communication

| Topic | Channel | Frequency | Participants |
|-------|---------|-----------|--------------|
| Platform updates | Email, Slack | Weekly | All users |
| Planned maintenance | Email, Slack | 1 week notice | All users |
| New features | Slack, Wiki | As released | All users |
| Cost reports | Email | Monthly | Platform Team, Leadership |

### Incident Communication

| Severity | Channel | Update Frequency | Audience |
|----------|---------|------------------|----------|
| P0 (Critical) | Slack incident channel, Status page | Every 30 min | All users, Leadership |
| P1 (Major) | Slack incident channel | Every 1 hour | Affected users, Platform Team |
| P2 (Minor) | Slack, Ticket | Daily | Affected users |

### Escalation Path

```
L1: Platform Team (on-call engineer)
    ↓ (if not resolved in 1 hour)
L2: Platform Lead
    ↓ (if not resolved in 4 hours)
L3: Engineering Director
    ↓ (if business-critical)
L4: CTO / VP Engineering
```

---

## Onboarding and Offboarding

### RACI: User Onboarding

| Activity | Platform Team | App Team | Governance | New User |
|----------|--------------|----------|------------|----------|
| Request access | I | I | I | R |
| Approve access | C | C | A | I |
| Provision accounts | R | I | I | I |
| Assign to resource groups | R | I | C | I |
| Training (platform basics) | R/A | I | I | C |
| Domain-specific training | C | R/A | I | C |

### RACI: User Offboarding

| Activity | Platform Team | App Team | Governance | HR |
|----------|--------------|----------|------------|-----|
| Notify offboarding | I | I | I | R/A |
| Revoke access | R | I | C | I |
| Transfer ownership | C | R/A | C | I |
| Archive user data | R | C | C | I |

---

## Key Principles

1. **Ownership**: Every system and dataset has a clear owner
2. **Accountability**: One person accountable for each decision
3. **Collaboration**: Consult stakeholders before major decisions
4. **Communication**: Keep stakeholders informed
5. **Escalation**: Clear escalation path for incidents
6. **Documentation**: Document decisions and rationale

---

## Review and Updates

This RACI matrix should be reviewed:
- **Quarterly**: Adjust based on org changes
- **When new systems are added**: Define ownership
- **After major incidents**: Improve response process

**Owner**: Platform Lead
**Last Updated**: 2024-01-15
**Next Review**: 2024-04-15
