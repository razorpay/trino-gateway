---
name: dataplatform-advisor
description: "Guide users in selecting the right data system (hot/warm/cold storage, OLAP, compute engines) and understanding overall data flow architecture. Use when someone is: (1) Building a new data feature and needs to choose where to query data, (2) Understanding how data flows from source to consumption, (3) Migrating workloads between systems, (4) Troubleshooting performance issues by validating system fit, (5) Understanding data architecture patterns, or (6) Making build vs buy decisions for data infrastructure. Triggers include 'which database should I use', 'how does data flow', 'data pipeline architecture', 'explain the data platform', 'data system selection', 'storage vs compute', 'OLAP vs lakehouse', 'CDC to lakehouse'."
user-invocable: true
disable-model-invocation: true
---

# Data Platform Advisor

A comprehensive decision framework for selecting the right data system and understanding data flow architecture. Helps engineers choose between hot/warm/cold storage, OLAP systems, and compute engines while understanding how data moves through the platform and the trade-offs involved.

## Quick Start

### Classify Your Workload

Before selecting a system, answer these key questions:

1. **Latency**: What is the required p99 latency? (milliseconds / seconds / minutes / hours)
2. **Freshness**: How fresh must the data be? (real-time / minutes / hours / T-1)
3. **Concurrency**: How many concurrent users/queries? (low / medium / high)
4. **Query Shape**: Are queries fixed patterns or ad-hoc?
5. **Consistency**: Need read-after-write guarantees?
6. **Data Quality**: What error tolerance? (exact / <1% / <5%)

### System Selection Decision Tree

```
START: What is the primary requirement?
|
+-- Need latency < 100ms?
|   |
|   +-- YES --> Transactional (read-after-write)? --> TiDB
|   |           Fixed query patterns? -----------> OLAP (Pinot/ClickHouse)
|   +-- NO --> Continue
|
+-- Need ACID transactions or writes?
|   |
|   +-- YES --> Point access + consistency? -----> TiDB
|   |           Large scale + cost-efficient? ---> ICEBERG
|   +-- NO --> Continue
|
+-- Need to join multiple data sources? -----------> TRINO
|
+-- Historical/batch analytics? -------------------> ICEBERG (Trino/Spark)
|
+-- Heavy ETL/transformation? ---------------------> SPARK
|
+-- Real-time event processing? -------------------> FLINK
|
+-- Default for analytical queries ----------------> TRINO on ICEBERG
```

## Examples

Quick use case → system recommendations:

1. **"Build a merchant dashboard showing payment success rates in real-time"**
   → **OLAP (Pinot)** - Sub-second latency, high concurrency, fixed query patterns

2. **"Run ad-hoc analysis to investigate payment failures across multiple data sources"**
   → **Trino + Iceberg** - Interactive queries, cross-source joins, flexible exploration

3. **"Generate daily ML training data from historical transactions"**
   → **Spark on Iceberg** - Batch processing, large-scale transformations, framework compatibility

4. **"Provide an API endpoint for merchant transaction lookup with < 50ms latency"**
   → **TiDB** - Point queries, strong consistency, predictable low latency

5. **"Build ETL pipeline to denormalize payment data for analytics"**
   → **Spark** - Batch ETL, complex transformations, cost-efficient at scale

See [Workflows](#workflows) section below for detailed decision-making process.

## Data Platform Architecture Overview

### Storage Tiers

| Tier | System | Freshness | Primary Use |
|------|--------|-----------|-------------|
| **Hot Storage** | Aurora/DynamoDB | Real-time | Transactional, application DB |
| **Warm DB** | TiDB | ~10 minutes | Operational queries, debugging |
| **OLAP** | Pinot/ClickHouse | Minutes | User-facing dashboards |
| **Cold Storage** | Iceberg Lakehouse | Hours to T-1 | Historical analytics, ML |

### Compute Engines

| Engine | Optimized For | Latency | Use Case |
|--------|---------------|---------|----------|
| **Spark** | Throughput, batch | Minutes-Hours | ETL, ML, backfills |
| **Trino** | Interactive queries | Seconds-Minutes | Ad-hoc analysis, BI |
| **Flink** | Stream processing | Real-time | CDC, real-time ingestion |

### Data Flow Architecture

The platform follows a multi-tier architecture where data flows from hot sources through various pipelines to serving layers:

```
┌─────────────────┐
│  Application    │ (Aurora, DynamoDB)
│  Services       │
│  (Hot Storage)  │ Source of Truth
└────────┬────────┘
         │
         │ CDC (Debezium)
         ├──────────────┐
         │              │
         ▼              ▼
    ┌────────┐    ┌─────────┐
    │ Kafka  │    │  Kafka  │ Event Streaming
    └───┬────┘    └────┬────┘
        │              │
        │              │
        ├──────────────┼─────────────┐
        │              │             │
        ▼              ▼             ▼
   ┌─────────┐   ┌──────────┐  ┌─────────────┐
   │  TiDB   │   │   Pinot  │  │   Iceberg   │
   │ (Warm)  │   │  (OLAP)  │  │  (Lakehouse)│
   └─────────┘   └──────────┘  └──────┬──────┘
        │             │                │
        │             │                │
        ▼             ▼                ▼
   Operational    Dashboard      Analytics/ML
     Queries       Serving        Processing
                                      │
                                      ▼
                              ┌──────────────┐
                              │ Trino/Spark  │
                              │   Compute    │
                              └──────────────┘
```

**Key Data Flow Patterns:**

1. **CDC Pattern** (Hot → Kafka → Downstream)
   - Aurora/DynamoDB → Debezium → Kafka → TiDB/Iceberg/Pinot
   - Real-time change capture from source systems
   - Preserves event ordering and schema evolution

2. **Batch Ingestion Pattern** (Hot → Cold)
   - Aurora → S3 snapshots → Iceberg (for historical backfills)
   - Scheduled full/incremental loads
   - Used for initial data migrations

3. **Stream Processing Pattern** (Kafka → OLAP)
   - Kafka → Spark/Flink → Pinot (pre-joined, denormalized)
   - Real-time transformations and enrichment
   - Optimized for dashboard serving

4. **Lakehouse Pattern** (Kafka → Iceberg → Compute)
   - Kafka → Spark → Iceberg (MERGE for upserts)
   - Iceberg → Trino (interactive queries)
   - Iceberg → Spark (batch processing, ML)

5. **Federation Pattern** (Trino → Multiple Sources)
   - Trino can query across Iceberg, MySQL, Pinot in single SQL
   - Useful for cross-system joins and exploration

**Data Governance Layer:**
- Apache Ranger for centralized RBAC/ABAC policies
- Data masking and column-level security
- Audit logging across all systems

See [references/data-flow-patterns.md](references/data-flow-patterns.md) for detailed examples and implementation guidance.

## Workflows

### Workflow 0: Understanding Data Flow Architecture

Use this when you need to understand how data moves through the platform.

**Steps:**
1. **Identify the data source**:
   - Which application service owns this data?
   - Is it in Aurora, DynamoDB, or another source system?
   - What's the schema and update frequency?

2. **Trace the data pipeline**:
   - Is CDC enabled? (Check Kafka topics)
   - Which downstream systems consume this data?
   - What transformations happen in flight?

3. **Understand serving layers**:
   - Where can this data be queried?
   - What's the freshness in each system?
   - What are the access patterns?

4. **Map data lineage**:
   - Source → CDC → Kafka topic(s) → Downstream tables
   - Document ownership (source team, data platform)
   - Identify data quality checkpoints

**Example Questions:**
- "How does payment transaction data flow from Aurora to the merchant dashboard?"
  - Aurora `payments` table → Debezium CDC → `payments` Kafka topic → Spark transformation → Pinot `payments_enriched` table → Dashboard API
  - Freshness: ~5 min end-to-end
  - Transformations: Joined with merchant metadata, pre-aggregated

- "Where can I query historical settlement data?"
  - Primary: Iceberg lakehouse (via Trino or Spark)
  - Freshness: T-1 (updated daily)
  - Retention: Unlimited
  - Alternative: TiDB for last 90 days with ~10 min freshness

- "What systems have customer email data and what are the access controls?"
  - Source: Aurora `customers` table
  - Downstream: TiDB (masked for most users), Iceberg (governed by Ranger)
  - Access: Column-level masking via Apache Ranger
  - Who can see unmasked: Security team, compliance team

### Workflow 1: New Feature - Choose Data System

Use this when building a new data-driven feature.

**Steps:**
1. **Classify the workload** using the questionnaire (see references/workload-classification.md)
2. **Map to archetype**:
   - User-facing dashboard? → OLAP
   - Ad-hoc exploration? → Trino + Iceberg
   - Operational reporting? → TiDB or OLAP
   - Historical analytics? → Iceberg + Trino/Spark
   - Batch processing? → Spark
   - Real-time alerting? → OLAP or Flink
   - Transactional access? → TiDB or Aurora

3. **Validate system boundaries** (see references/system-boundaries.md):
   - Check the system's capabilities match requirements
   - Verify workload fits within design constraints
   - Review limitations and anti-patterns

4. **Consider alternatives**:
   - Is there a simpler approach?
   - Can existing systems handle this?
   - What's the operational overhead?

5. **Document decision**:
   - Workload characteristics
   - System chosen and why
   - Trade-offs accepted
   - Alternative systems considered

**Example:**
```
User need: "Build a merchant analytics dashboard showing payment trends"

Classification:
- Latency: < 100ms (user-facing)
- Freshness: 5-10 minutes acceptable
- Concurrency: High (100s of merchants)
- Query Shape: Fixed patterns (pre-defined charts)
- Consistency: Eventual OK
- Quality: High accuracy required

Recommendation: OLAP (Pinot)
Why:
✓ Sub-second latency for fixed queries
✓ High concurrency support
✓ Star-tree indexes for dashboard patterns
✓ Real-time ingestion from Kafka

Not suitable:
✗ Trino: Variable latency, not for user-facing
✗ Iceberg: Too slow (seconds), batch-oriented
✗ TiDB: Expensive at scale, row-oriented
```

### Workflow 2: Performance Troubleshooting - System Fit Analysis

Use this when a workload is experiencing performance issues.

**Steps:**
1. **Identify symptoms**:
   - What's slow? (queries, writes, ingestion)
   - When did it start?
   - Is it consistent or intermittent?

2. **Review workload classification**:
   - Has the workload changed?
   - Are requirements within system's design envelope?
   - Check system boundaries violations (see references/system-boundaries.md)

3. **Common mismatches**:
   - **OLAP used for ad-hoc exploration** → Queries fail due to schema rigidity
   - **Trino used for user-facing API** → Unpredictable latency
   - **TiDB used for large analytical scans** → Row-oriented storage is slow
   - **Iceberg used for low-latency queries** → Minimum latency is seconds

4. **Recommendation**:
   - If system mismatch: Migrate to appropriate system
   - If within envelope: Optimize query or scale infrastructure
   - Document the root cause analysis

**Example:**
```
Problem: "Dashboard queries timing out in Trino"

Analysis:
- Current: Trino querying Iceberg
- Workload: User-facing dashboard, < 1s latency, 100+ QPS
- Issue: Trino designed for interactive analytics (seconds), not serving (milliseconds)

Root cause: System mismatch
- Trino is for ad-hoc queries, not high-QPS serving
- Shared cluster means no latency SLA

Recommendation: Migrate to OLAP (Pinot)
- Pre-compute and denormalize data
- Ingest to Pinot with star-tree indexes
- Achieve < 100ms p99 latency
```

### Workflow 3: System Selection for Different Personas

#### Analytics & Reporting (Business Analysts)

**Primary needs:**
- Query speed for interactive exploration
- Concurrency for many simultaneous users
- Cost efficiency for complex queries

**Recommended systems:**
- **Ad-hoc analysis**: Trino querying Iceberg
- **BI dashboards (variable)**: Trino with Metabase/Looker
- **Customer dashboards (fixed)**: OLAP (Pinot/ClickHouse)

**Anti-patterns:**
- Using OLAP for ad-hoc exploration (schema rigidity)
- Using Aurora directly (impacts production)

#### Data Engineering (Data Engineers)

**Primary needs:**
- Cost-performance balance
- Fault tolerance for pipelines
- Rich orchestration ecosystem

**Recommended systems:**
- **Batch ETL**: Spark
- **Pre-join for OLAP**: Spark
- **Real-time CDC**: Flink
- **Backfills**: Spark

**Anti-patterns:**
- Using Trino for heavy transformations (resource contention)
- Using OLAP for processing (not designed for it)

#### Data Science & ML (Data Scientists)

**Primary needs:**
- Framework compatibility (Pandas, PyTorch)
- Scalability for large datasets
- Easy iteration and experimentation

**Recommended systems:**
- **Feature engineering**: Spark on Iceberg
- **Training data prep**: Spark
- **Feature exploration**: Trino on Iceberg

**Anti-patterns:**
- Using OLAP/TiDB for ML features (limited retention, expensive)

#### Application Development (Backend Engineers)

**Primary needs:**
- Sub-second latency for APIs
- Predictable performance under load
- Strong consistency for operations

**Recommended systems:**
- **Analytics API**: OLAP (Pinot/ClickHouse)
- **Operational queries**: TiDB
- **Application backend**: TiDB

**Anti-patterns:**
- Using Trino for APIs (unpredictable latency)
- Using Iceberg for serving (too slow)

### Workflow 4: Migration Between Systems

Use this when moving workloads from one system to another.

**Steps:**
1. **Document current state**:
   - Current system and why it was chosen
   - Workload characteristics
   - Pain points and limitations

2. **Justify migration**:
   - What requirements changed?
   - Why is current system insufficient?
   - What are the costs of NOT migrating?

3. **Choose target system**:
   - Use classification framework
   - Validate system boundaries
   - Consider operational overhead

4. **Migration plan**:
   - Data migration strategy (batch vs streaming)
   - Dual-write period for validation
   - Rollback plan
   - Performance testing

5. **Post-migration validation**:
   - Compare latency, cost, reliability
   - Monitor for regressions
   - Document lessons learned

## Reference Documentation

For deeper understanding, consult these references:

### Core Concepts
- **[data-flow-patterns.md](references/data-flow-patterns.md)**: Complete data flow patterns (CDC, batch, streaming) with examples and implementation guidance
- **[workload-classification.md](references/workload-classification.md)**: Complete framework for classifying workloads with examples and questionnaire
- **[storage-systems.md](references/storage-systems.md)**: Deep dive into each storage tier (hot/warm/OLAP/cold) with internal architectures
- **[compute-engines.md](references/compute-engines.md)**: Spark, Trino, Flink capabilities and when to use each
- **[system-boundaries.md](references/system-boundaries.md)**: What each system should and should NOT be used for

### Decision Support
- **[decision-framework.md](references/decision-framework.md)**: Step-by-step methodology for system selection with examples
- **[architecture-patterns.md](references/architecture-patterns.md)**: Common patterns and anti-patterns in data architecture
- **[roles-responsibilities.md](references/roles-responsibilities.md)**: RACI matrix for data quality and system operations

## Quick Reference Tables

### System Capabilities Matrix

| Capability | Trino | Iceberg | OLAP | TiDB |
|------------|-------|---------|------|------|
| **Latency** | Seconds | Seconds-Min | Milliseconds | Milliseconds |
| **Concurrency** | Low-Med | Low | Very High | High |
| **Freshness** | Source-dependent | Hours | Seconds | Real-time |
| **Joins** | Excellent | Good | None | Good |
| **ACID** | Read-only | Yes | No | Yes |
| **Cost** | On-demand | Low (S3) | Med-High | High |

### Use Case to System Mapping

| Use Case | Primary System | Alternative | Avoid |
|----------|---------------|-------------|-------|
| User-facing dashboard | OLAP | TiDB | Trino, Iceberg |
| Ad-hoc exploration | Trino | — | OLAP |
| Historical analysis | Iceberg + Trino | — | OLAP |
| Transactional + reporting | TiDB | — | OLAP |
| ML feature engineering | Iceberg + Spark | — | OLAP, TiDB |
| API serving | TiDB | OLAP | Trino, Iceberg |

### Workload Archetypes

| Archetype | Latency | Freshness | Concurrency | Query Shape | Recommended System |
|-----------|---------|-----------|-------------|-------------|-------------------|
| User-Facing Dashboard | Sub-sec | Minutes | High | Fixed | OLAP |
| Ad-hoc Exploration | Sec-min | Hours | Low | Variable | Trino + Iceberg |
| Operational Reporting | Seconds | Minutes | Medium | Semi-fixed | TiDB or OLAP |
| Historical Analytics | Min-hours | Hours | Low | Variable | Iceberg + Trino/Spark |
| Batch Processing | Hours | N/A | N/A | Fixed | Spark |
| Real-time Alerting | Millis | Real-time | Low | Fixed | OLAP or Flink |
| Transactional Access | Millis | Real-time | High | Point lookup | TiDB or Aurora |

### Cost Thresholds

Use these guidelines for evaluating storage/compute costs:

**Storage:**
- Hot (Aurora): ~$0.10/GB/month (expensive, optimized for OLTP)
- Warm (TiDB): ~$0.08/GB/month (expensive, distributed SQL)
- OLAP (Pinot): ~$0.05/GB/month (medium, SSD-backed)
- Cold (Iceberg/S3): ~$0.023/GB/month (cheap, object storage)

**Compute:**
- Trino: On-demand, pay per query second
- Spark: Ephemeral clusters, pay per job
- OLAP: Fixed cluster cost, high concurrency

## Common Pitfalls

### 1. Using Wrong System for Workload

**Problem**: System chosen based on familiarity, not fit.

**Example**: Using Aurora for analytics → Kills production performance

**Solution**: Always classify workload first, then select system.

### 2. Extending System Beyond Design Envelope

**Problem**: Adding features to make system handle unintended workloads.

**Example**: Adding caching layer to Trino for user-facing API → Still unpredictable

**Solution**: Use purpose-built system (OLAP for serving).

### 3. Ignoring Data Freshness Requirements

**Problem**: Choosing system without considering ingestion lag.

**Example**: Using Iceberg (T-1) for operational debugging → Data too stale

**Solution**: Map freshness requirement to appropriate tier.

### 4. Not Validating System Boundaries

**Problem**: Assuming system can handle any workload.

**Example**: Using OLAP for complex joins → Not supported

**Solution**: Review system limitations before committing.

### 5. Optimizing Before Right-Sizing

**Problem**: Trying to optimize queries in wrong system.

**Example**: Tuning Trino queries for < 100ms → Never achievable

**Solution**: Validate system selection first, then optimize.

## Tips for Success

### When Starting a New Project

1. **Start with requirements, not technology**
   - What are the latency, freshness, concurrency needs?
   - How will the workload scale?
   - What's the error tolerance?

2. **Use the classification framework**
   - Map to workload archetype
   - Identify non-negotiable requirements
   - Understand trade-offs

3. **Choose the simplest system that fits**
   - Don't add new systems unless necessary
   - Consider operational overhead
   - Think about team expertise

### When Evaluating Performance

1. **Check system fit first**
   - Is workload within design envelope?
   - Are there boundary violations?
   - Has workload evolved?

2. **Optimize within system capabilities**
   - Query optimization
   - Index tuning
   - Resource scaling

3. **Consider migration if mismatch**
   - Document why current system is insufficient
   - Validate target system fit
   - Plan migration carefully

### When Making Architecture Decisions

1. **Make trade-offs explicit**
   - What are we optimizing for?
   - What are we sacrificing?
   - Is this reversible?

2. **Document decisions**
   - Workload characteristics
   - System chosen and alternatives
   - Expected performance and cost
   - Review date

3. **Plan for evolution**
   - How might requirements change?
   - What are migration paths?
   - What's the exit strategy?

## Limitations

1. **Context-specific**: This framework is based on Razorpay's data platform. Your organization may have different systems or constraints.

2. **Technology evolution**: Data systems evolve rapidly. Capabilities and limitations change with new versions.

3. **Data-dependent**: Actual performance depends on data volume, distribution, and skew.

4. **Cost variation**: Costs vary by cloud provider, region, and usage patterns.

5. **Organizational factors**: Team expertise, operational maturity, and existing investments matter.

## Getting Help

When using this skill:

1. **For workload classification**: Provide context about your use case, expected scale, and requirements.

2. **For system selection**: Share workload characteristics and constraints.

3. **For performance issues**: Describe symptoms, current system, and workload evolution.

4. **For migration planning**: Explain current pain points and future requirements.

The skill will guide you through the decision framework and provide specific recommendations with rationale.
