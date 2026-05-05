# System Selection Decision Framework

A step-by-step methodology for choosing the right data system with detailed examples and decision trees.

## Overview

This framework guides you through a systematic process for selecting data systems, ensuring you choose the right tool for your workload while avoiding common pitfalls.

## The Decision Process

```
1. Understand Requirements
   ↓
2. Classify Workload
   ↓
3. Map to Archetype
   ↓
4. Select Candidate Systems
   ↓
5. Validate System Boundaries
   ↓
6. Evaluate Trade-offs
   ↓
7. Make Decision & Document
```

---

## Step 1: Understand Requirements

### Questions to Ask

**Functional Requirements:**
- What is the use case? (dashboard, report, API, etc.)
- Who are the users? (customers, analysts, engineers)
- What data do they need?
- What queries/operations will they perform?

**Non-Functional Requirements:**
- **Latency**: What's the acceptable response time? (p50, p99)
- **Freshness**: How fresh must data be? (real-time, minutes, hours, T-1)
- **Concurrency**: How many concurrent users/queries?
- **Availability**: What's the uptime requirement? (99%, 99.9%, 99.99%)
- **Consistency**: Need strong consistency or eventual OK?
- **Scale**: Data volume, query volume, growth rate

**Business Constraints:**
- Budget constraints
- Timeline (launch date)
- Team expertise
- Compliance requirements

### Example: Merchant Analytics Dashboard

**Functional:**
- Use case: Dashboard showing payment metrics for merchants
- Users: 10K merchants (external users)
- Data: Payment transactions, aggregated stats
- Queries: Pre-defined charts (volume, success rate, top payment methods)

**Non-Functional:**
- Latency: < 100ms p99 (user-facing)
- Freshness: 5-10 minutes acceptable
- Concurrency: 500 concurrent merchants (peak)
- Availability: 99.9% (not mission-critical)
- Consistency: Eventual (analytics, not transactional)
- Scale: 100M transactions/day, 1000 queries/minute

**Business:**
- Budget: $5K/month
- Timeline: 3 months
- Team: Familiar with SQL, Kafka
- Compliance: PCI-DSS (data masking required)

---

## Step 2: Classify Workload

Use the [Workload Classification Framework](workload-classification.md) to categorize your workload across six dimensions:

### Dimension Checklist

```yaml
latency:
  required_p99: "< 100ms"
  category: "Low"

freshness:
  required: "5-10 minutes"
  category: "Near real-time"

concurrency:
  peak_users: 500
  peak_qps: 1000
  category: "High"

query_shape:
  type: "Fixed"
  patterns:
    - "Payment volume by day"
    - "Success rate by hour"
    - "Top payment methods"

consistency:
  required: "Eventual"

data_quality:
  required: "High (< 0.1% error)"
```

### Decision Tree for Quick Classification

```
Q1: Do you need ACID transactions or writes?
    YES → Transactional system (Aurora, TiDB)
    NO → Continue

Q2: Is latency < 100ms required?
    YES → Is concurrency high (> 100 QPS)?
          YES → User-facing dashboard (OLAP)
          NO → Operational queries (TiDB)
    NO → Continue

Q3: Are queries fixed patterns or ad-hoc?
    FIXED → Is freshness real-time?
            YES → User-facing dashboard (OLAP)
            NO → Historical analytics (Iceberg + Trino)
    AD-HOC → Ad-hoc exploration (Trino + Iceberg)

Q4: Is this for ETL/batch processing?
    YES → Batch processing (Spark)
    NO → Real-time processing (Flink)
```

---

## Step 3: Map to Archetype

Based on classification, map to one of the common archetypes:

### Archetype Mapping

| Dimensions | Archetype | Primary System |
|-----------|-----------|----------------|
| Low latency + High concurrency + Fixed queries | User-Facing Dashboard | OLAP (Pinot) |
| Medium latency + Low concurrency + Ad-hoc queries | Ad-hoc Exploration | Trino + Iceberg |
| Low latency + Medium concurrency + Semi-fixed | Operational Reporting | TiDB or OLAP |
| High latency + Low concurrency + Variable | Historical Analytics | Iceberg + Trino/Spark |
| N/A latency + Batch | Batch Processing | Spark |
| Real-time + Stateful | Real-time Processing | Flink |
| Ultra-low latency + ACID | Transactional | Aurora, TiDB |

### Example Mapping

**Merchant Analytics Dashboard:**
- Low latency (< 100ms) ✓
- High concurrency (500 users) ✓
- Fixed queries (pre-defined charts) ✓
- Near real-time freshness (5-10 min) ✓
- Eventual consistency ✓

**Archetype**: User-Facing Dashboard
**Primary System**: OLAP (Pinot)

---

## Step 4: Select Candidate Systems

Based on archetype, identify candidate systems and alternatives.

### Candidate Selection Matrix

**For User-Facing Dashboard archetype:**

| System | Pros | Cons | Cost |
|--------|------|------|------|
| **Pinot** (Primary) | Sub-second latency, high concurrency, real-time ingestion | Fixed schema, no joins | Medium |
| **ClickHouse** (Alternative) | Good latency, high compression | Lower concurrency than Pinot | Medium |
| **TiDB** (Alternative) | MySQL-compatible, ACID | Higher cost, row-oriented | High |

**For Ad-hoc Exploration archetype:**

| System | Pros | Cons | Cost |
|--------|------|------|------|
| **Trino + Iceberg** (Primary) | Handles any SQL, low storage cost | Seconds latency, low concurrency | Low |
| **TiDB** (Alternative) | Low latency, high concurrency | High cost, limited retention | High |

**For Batch Processing archetype:**

| System | Pros | Cons | Cost |
|--------|------|------|------|
| **Spark** (Primary) | High throughput, fault-tolerant | High startup overhead | Low (ephemeral) |
| **Trino** (Alternative) | Interactive, no overhead | Not for heavy ETL | Medium |

### Example: Merchant Analytics Dashboard

**Candidates:**
1. **Pinot** (Primary): Best fit for fixed queries, high concurrency
2. **ClickHouse** (Alternative): Similar capabilities, slightly lower concurrency
3. **TiDB**: MySQL-compatible, but more expensive and row-oriented

**Selected Candidates**: Pinot, ClickHouse

---

## Step 5: Validate System Boundaries

Check [System Boundaries](system-boundaries.md) to ensure workload fits within system design envelope.

### Validation Checklist

**For each candidate system:**

1. **Design Envelope**:
   - Is the workload within the system's designed use case? ✓/✗
   - Are there known anti-patterns? ✓/✗

2. **Capacity Limits**:
   - Data volume within limits? ✓/✗
   - Query volume within limits? ✓/✗
   - Concurrency within limits? ✓/✗

3. **Feature Support**:
   - Supports required query patterns? ✓/✗
   - Supports required latency? ✓/✗
   - Supports required freshness? ✓/✗

### Example Validation: Pinot

**Design Envelope:**
- ✓ User-facing dashboards (designed for this)
- ✓ Fixed query patterns (star-tree indexes)
- ✓ High concurrency (1000+ QPS)
- ✗ Ad-hoc exploration (not designed for this)

**Capacity Limits:**
- ✓ Data volume: 100M events/day (Pinot handles this)
- ✓ Query volume: 1000 QPS (within limits)
- ✓ Concurrency: 500 users (well within limits)

**Feature Support:**
- ✓ Latency: 10-100ms (meets < 100ms requirement)
- ✓ Freshness: Real-time ingestion (meets 5-10 min requirement)
- ✓ Fixed queries: Pre-defined charts (perfect fit)

**Validation Result**: ✓ PASS

### Example Validation: TiDB

**Design Envelope:**
- ✓ Operational queries (designed for this)
- ✗ User-facing dashboards at scale (row-oriented, expensive)

**Capacity Limits:**
- ✓ Data volume: 100M events/day (TiDB can handle)
- ⚠ Query volume: 1000 QPS (possible but expensive vs OLAP)
- ✓ Concurrency: 500 users (within limits)

**Feature Support:**
- ✓ Latency: 10-50ms (meets requirement)
- ✓ Freshness: Real-time (better than requirement)
- ✓ Query patterns: SQL (supports all queries)

**Validation Result**: ⚠ MARGINAL (works but not optimal for high QPS)

---

## Step 6: Evaluate Trade-offs

Compare candidate systems across key dimensions.

### Trade-off Matrix

| Dimension | Pinot | ClickHouse | TiDB |
|-----------|-------|------------|------|
| **Latency** | 10-100ms ⭐⭐⭐ | 10-100ms ⭐⭐⭐ | 10-50ms ⭐⭐⭐ |
| **Concurrency** | 1000+ QPS ⭐⭐⭐ | 500 QPS ⭐⭐ | 500 QPS ⭐⭐ |
| **Freshness** | Real-time ⭐⭐⭐ | Real-time ⭐⭐⭐ | Real-time ⭐⭐⭐ |
| **Cost** | $3K/mo ⭐⭐ | $2.5K/mo ⭐⭐⭐ | $5K/mo ⭐ |
| **Operational Complexity** | Medium ⭐⭐ | Medium ⭐⭐ | Low (MySQL-compatible) ⭐⭐⭐ |
| **Query Flexibility** | Fixed ⭐ | Fixed ⭐ | Ad-hoc ⭐⭐⭐ |
| **Team Expertise** | Learning curve ⭐ | Learning curve ⭐ | Familiar (MySQL) ⭐⭐⭐ |

### Decision Criteria

**Must-Have (Non-Negotiable):**
- Latency < 100ms ✓ All pass
- Concurrency 1000 QPS ✓ Pinot only
- Freshness 5-10 min ✓ All pass

**Nice-to-Have (Trade-offs):**
- Low cost: ClickHouse > Pinot > TiDB
- Operational simplicity: TiDB > Pinot = ClickHouse
- Query flexibility: TiDB > Pinot = ClickHouse

### Example Decision

**Pinot vs ClickHouse:**
- Pinot: Higher concurrency (1000+ QPS), slightly higher cost
- ClickHouse: Lower cost, but max ~500 QPS

**Decision**: Choose **Pinot** because:
1. Meets must-have (1000 QPS)
2. Cost difference is acceptable ($500/mo)
3. Better long-term headroom for growth

**Eliminated TiDB** because:
- Higher cost ($5K vs $3K)
- Row-oriented (not optimal for analytics)
- Better suited for operational queries

---

## Step 7: Make Decision & Document

### Decision Template

```yaml
decision:
  workload: "Merchant Analytics Dashboard"
  date: "2024-01-15"
  owner: "Platform Team"

requirements:
  latency_p99: "< 100ms"
  freshness: "5-10 minutes"
  concurrency: "1000 QPS"
  query_patterns: "Fixed (pre-defined charts)"

classification:
  archetype: "User-Facing Dashboard"

system_selected:
  primary: "Apache Pinot"
  rationale:
    - "Designed for user-facing dashboards (high concurrency)"
    - "Sub-100ms latency at 1000+ QPS"
    - "Real-time ingestion from Kafka (5-10 min freshness)"
    - "Star-tree indexes for pre-defined query patterns"
    - "Cost-efficient at scale ($3K/mo for 1000 QPS)"

alternatives_considered:
  - system: "ClickHouse"
    rejected_reason: "Lower concurrency (500 QPS), growth constraint"
  - system: "TiDB"
    rejected_reason: "Higher cost ($5K), row-oriented (slower for analytics)"

trade_offs:
  accepted:
    - "Fixed schema (must pre-define charts)"
    - "No joins (denormalize data in Spark)"
    - "Learning curve for team (new system)"
  mitigated:
    - "Operational complexity → Managed service (cloud offering)"

architecture:
  data_flow: "Aurora → CDC → Kafka → Spark (pre-join) → Pinot"
  freshness_achieved: "~5 minutes end-to-end"
  retention: "90 days (configurable)"

risks:
  - risk: "Schema evolution requires reindexing"
    mitigation: "Plan schema changes carefully, test in staging"
  - risk: "New system for team"
    mitigation: "Training, documentation, support from vendor"

success_criteria:
  - "p99 latency < 100ms (dashboard queries)"
  - "99.9% availability"
  - "Support 1000 QPS at peak"
  - "Data freshness < 10 minutes"

review_date: "2024-07-15 (6 months)"
```

---

## Common Decision Scenarios

### Scenario 1: New Dashboard Feature

**Context**: Product team wants to add a new merchant dashboard showing payment trends.

**Process**:
1. **Understand**: User-facing, 5K merchants, pre-defined charts, < 100ms latency
2. **Classify**: Low latency, high concurrency, fixed queries
3. **Map**: User-Facing Dashboard archetype
4. **Candidates**: Pinot, ClickHouse
5. **Validate**: Both fit design envelope
6. **Trade-offs**: Pinot has higher concurrency, ClickHouse lower cost
7. **Decision**: Pinot (future growth, concurrency headroom)

### Scenario 2: Analyst Exploration Tool

**Context**: Data analysts need to explore historical payment data for insights.

**Process**:
1. **Understand**: Internal users (10 analysts), ad-hoc queries, 2+ years data
2. **Classify**: Medium latency OK, low concurrency, ad-hoc queries
3. **Map**: Ad-hoc Exploration archetype
4. **Candidates**: Trino + Iceberg
5. **Validate**: Fits design envelope (handles any SQL, low storage cost)
6. **Trade-offs**: Seconds latency (acceptable for analysts)
7. **Decision**: Trino + Iceberg (cost-efficient, flexible)

### Scenario 3: Real-time CDC Pipeline

**Context**: Replicate Aurora transactions to TiDB for operational queries.

**Process**:
1. **Understand**: Real-time replication, millions of events/day, exactly-once
2. **Classify**: Real-time, stateful, continuous processing
3. **Map**: Real-time Processing archetype
4. **Candidates**: Flink, Spark Streaming
5. **Validate**: Flink designed for CDC (exactly-once, low latency)
6. **Trade-offs**: Flink requires operational overhead vs Spark Streaming
7. **Decision**: Flink (exactly-once guarantees, true streaming)

### Scenario 4: ML Feature Engineering

**Context**: Data scientists need to create features from historical transactions for ML models.

**Process**:
1. **Understand**: Batch processing, large-scale joins, 3 years data
2. **Classify**: Batch, high throughput, complex transformations
3. **Map**: Batch Processing archetype
4. **Candidates**: Spark, Trino
5. **Validate**: Spark designed for batch (fault-tolerant, high throughput)
6. **Trade-offs**: Spark has startup overhead but better for batch
7. **Decision**: Spark (ephemeral clusters, cost-efficient for batch)

---

## Decision Anti-Patterns

### Anti-Pattern 1: Technology-First Decision

**Wrong Approach**:
> "Let's use Pinot because it's new and exciting."

**Right Approach**:
> "Based on workload classification (high concurrency, fixed queries), Pinot is the best fit."

**Lesson**: Start with requirements, not technology.

### Anti-Pattern 2: Ignoring System Boundaries

**Wrong Approach**:
> "Let's use Trino for the dashboard because we already have it."

**Right Approach**:
> "Trino isn't designed for high-concurrency dashboards. Use Pinot or TiDB."

**Lesson**: Don't force systems outside their design envelope.

### Anti-Pattern 3: Over-Optimizing for Current State

**Wrong Approach**:
> "We only have 10 users now, so let's use the cheapest option."

**Right Approach**:
> "We expect 1000 users in 6 months, so let's choose a system that scales."

**Lesson**: Consider growth trajectory, not just current state.

### Anti-Pattern 4: Ignoring Operational Overhead

**Wrong Approach**:
> "Let's build a custom solution with multiple systems to save cost."

**Right Approach**:
> "The operational overhead of managing 5 systems outweighs the cost savings."

**Lesson**: Factor in operational complexity and team expertise.

### Anti-Pattern 5: Analysis Paralysis

**Wrong Approach**:
> "Let's evaluate 10 more systems before deciding."

**Right Approach**:
> "Pinot and ClickHouse both fit. Let's pick one and validate with PoC."

**Lesson**: Good enough is better than perfect. Validate with PoC.

---

## Validation with Proof of Concept (PoC)

Before committing to a system, validate with a PoC.

### PoC Checklist

**Objectives:**
- Validate latency meets requirements
- Test concurrency under load
- Verify data freshness
- Assess operational complexity

**Metrics to Measure:**
- Query latency (p50, p99)
- Throughput (QPS)
- Ingestion lag (freshness)
- Resource usage (CPU, memory)
- Operational effort (setup, monitoring)

**Example PoC Plan:**

```yaml
poc:
  system: "Apache Pinot"
  duration: "2 weeks"
  scope:
    - "Ingest 1 month of historical data"
    - "Build 5 key dashboard queries"
    - "Load test at 1000 QPS"
    - "Measure latency (p99)"

  success_criteria:
    - "p99 latency < 100ms"
    - "Sustained 1000 QPS"
    - "Ingestion lag < 5 minutes"
    - "Setup time < 1 week"

  results:
    latency_p99: "85ms ✓"
    qps_sustained: "1200 QPS ✓"
    ingestion_lag: "3 minutes ✓"
    setup_time: "4 days ✓"

  decision: "Proceed with Pinot"
```

---

## Review and Iteration

Systems decisions should be reviewed periodically.

### Review Checklist

**Every 6 months:**
1. Are requirements still the same?
2. Has workload grown as expected?
3. Is the system still meeting SLAs?
4. Are there better alternatives now?
5. What's the operational overhead?

**Triggers for Re-evaluation:**
- SLA misses (latency, availability)
- Cost exceeds budget
- Workload characteristics changed
- New requirements (features, scale)
- Team expertise changed

---

## Key Takeaways

1. **Follow the process**: Don't skip steps (understand → classify → select → validate)
2. **Start with requirements**: Technology should follow workload, not vice versa
3. **Validate boundaries**: Ensure system fits design envelope
4. **Document decisions**: Record rationale, trade-offs, and alternatives
5. **Validate with PoC**: Test before committing
6. **Review periodically**: Re-evaluate as requirements change

**Remember**: There is no perfect system. Choose the system that best fits your requirements while accepting trade-offs.
