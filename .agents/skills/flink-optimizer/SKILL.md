---
name: flink-optimizer
description: Optimize Flink job code and configuration based on runtime metrics and execution flow graph. Use when user asks to "optimize Flink job", "analyze Flink job performance", "check for backpressure", "review Flink DAG", "find Flink configuration issues", or provides a Flink cluster URL with job ID. Analyzes running jobs from Flink UI (via REST API), reviews YAML jobspec configuration, examines Java DAG construction code, and identifies optimization opportunities including backpressure, data skew, dead code/wiring, incorrect stream connections, checkpoint issues, memory configuration, and inefficient aggregations.
---

# Flink Job Optimizer

Comprehensive Flink job optimization skill for analyzing and improving Apache Flink jobs built with the jobspec-based runtime.

## Overview

This skill helps optimize Flink jobs by:

1. **Fetching runtime metrics** from Flink cluster REST API (backpressure, checkpoints, parallelism)
2. **Analyzing YAML jobspec** for configuration issues (dead streams, missing wiring, memory settings)
3. **Reviewing Java code** for DAG construction problems (operator chaining, inefficient wiring)
4. **Generating actionable recommendations** with specific fixes

## Typical Usage

**User starts Claude session in the Flink job codebase and says:**

- "Optimize the payment-processing Flink job using cluster https://flink.de.razorpay.com"
- "Analyze backpressure issues in job abc123 on https://flink.de.razorpay.com"
- "Optimize job {job_id} on cluster {url}, config at configs/local/jobs/sample-jobspec.yaml"
- "Review my Flink job for performance issues"

## Workflow

### Step 1: Gather Information

Ask the user for:

1. **Flink cluster URL** (e.g., `https://flink.de.razorpay.com`)
2. **Job ID** (optional - can list running jobs if not provided)
3. **Jobspec YAML path** (e.g., `configs/local/jobs/sample-jobspec_eventtime.yaml`)

**If job ID not provided:** Run the analysis script with `--list-jobs` to show running jobs, then ask user which job to analyze.

### Step 2: Run Analysis Script

Activate Pyhton Virtual Environment if not already active.

Execute `scripts/analyze_flink_job.py`:

```bash
python3 ~/.claude/skills/flink-optimizer/scripts/analyze_flink_job.py \
  --cluster-url https://flink.de.razorpay.com \
  --job-id <job-id> \
  --jobspec-path configs/local/jobs/sample-jobspec.yaml
```

**Script Output:** Comprehensive report with:
- Runtime issues (backpressure, checkpoint failures)
- Configuration issues (dead streams, missing wiring, low parallelism)
- DAG summary (sources, operators, streams)

### Step 3: Deep Dive Code Analysis

Based on issues found, examine the Java code:

1. **For wiring issues:** Check `Main.java` `buildJobDAG()` method
   - Look for how `inputStreams` and `outputStream` are wired in the `streams` HashMap
   - Verify all outputStreams are consumed by downstream operators or sinks

2. **For parallelism issues:** Check operator configurations
   - Review `applyOperatorParallelism()` logic in Main.java
   - Verify parallelism settings in YAML for heavy operators

3. **For aggregation issues:** Review operator implementations
   - Check `GenericWindowAggregateFunction` for efficiency
   - Look for repeated CASE WHEN patterns in YAML aggregations

### Step 4: Generate Recommendations

Provide specific, actionable fixes with file paths and line numbers.

**Example format:**

```markdown
## Optimization Recommendations

### 1. Fix Dead Output Stream (CRITICAL)
**Issue:** Operator 'merchant-filter' produces 'filtered-payments' but no operator consumes it.

**Fix in YAML (configs/local/jobs/sample-jobspec.yaml:82):**
```yaml
operators:
  - name: "window-aggregator"
    inputStreams: ["filtered-payments"]  # ← Add this
    outputStream: "aggregated-payments"
```

**Expected Impact:** Fixes broken DAG, enables proper data flow.
```

### Step 5: Validate Fixes (Optional)

If user requests, help validate the fixes:

1. Review modified YAML for syntax correctness
2. Check that all stream references are now valid
3. Verify parallelism settings are reasonable
4. Run ConfigValidator logic manually if needed

## Common Issues & Fixes

### Issue 1: Dead Output Streams

**Detection:** Script reports "Dead output stream 'X' from operator 'Y'"

**Root Cause:** Operator produces an outputStream that no downstream operator consumes.

**Fix:**
- Either wire it to a downstream operator's inputStreams
- Or remove the operator if it's not needed

**Files to check:**
- YAML jobspec: Look for operators with matching outputStream
- `Main.java`: Check `buildJobDAG()` - outputStream stored but never retrieved

### Issue 2: Missing Input Streams

**Detection:** Script reports "Missing input stream 'X' required by operators: Y"

**Root Cause:** Operator references an inputStream that doesn't exist.

**Fix:**
- Correct the inputStream name to match existing outputStream or source
- Or add the missing upstream operator/source

**Files to check:**
- YAML jobspec: Verify source names and operator outputStreams
- `Main.java`: Check if source name matches YAML definition

### Issue 3: Backpressure from Low Parallelism

**Detection:**
- Script reports "Low parallelism (1) for heavy operator"
- Flink UI shows high backpressure on specific vertices

**Root Cause:** Expensive operators (window aggregation, joins) run with parallelism=1.

**Fix:**
- Add explicit parallelism to the operator in YAML
- Or increase default job parallelism

**Recommended Values:**
- Window aggregators: 4-16
- Rule evaluators: 2-8
- RCA analyzers: 4-8
- Filters: 1-2 (lightweight)

**Files to modify:**
- YAML jobspec: Add `parallelism: N` to operator config

### Issue 4: Data Skew

**Detection:**
- Some subtasks much busier than others
- Per-subtask metrics show large variance (check Flink UI metrics)

**Root Cause:** Uneven key distribution in keyBy operations.

**Fix Options:**

1. **Add key salting** (requires code change):
```java
// In operator implementation
String saltedKey = merchantId + "-" + (hash(merchantId) % numSalts);
```

2. **Two-phase aggregation** (YAML):
```yaml
# Pre-aggregate with distribution, then final aggregate
operators:
  - name: "pre-agg"
    config:
      keyField: "salted_key"
  - name: "final-agg"
    config:
      keyField: "original_key"
```

3. **Increase parallelism** (partial fix):
- Higher parallelism spreads skew across more instances

**Files to check:**
- YAML: Look for keyField in WINDOW_AGGREGATOR config
- Java: Check WindowAggregateMetricsWrapper keyBy logic

### Issue 5: Inefficient Aggregations

**Detection:** Script reports "Operator has 50+ aggregations" or "similar CASE WHEN conditions"

**Root Cause:** Too many aggregation expressions or repeated logic.

**Fix Options:**

1. **Split into multiple operators:**
```yaml
operators:
  - name: "upi-aggregator"
    config:
      aggregations:
        # Only UPI metrics (10-15 expressions)

  - name: "card-aggregator"
    config:
      aggregations:
        # Only card metrics (10-15 expressions)
```

2. **Simplify expressions:**
- Pre-compute complex fields before aggregation
- Reduce repeated CASE WHEN patterns

**Files to modify:**
- YAML jobspec: Split aggregations config
- May need to add intermediate operators

### Issue 6: Checkpoint Issues

**Detection:**
- Script reports "High checkpoint failure rate" or "Long checkpoint duration"
- Flink UI shows frequent checkpoint timeouts

**Root Causes & Fixes:**

**High Failure Rate:**
```yaml
checkpointing:
  interval: 60000
  checkpointTimeout: 300000  # ← Increase from 60s to 5min
```

**Long Duration:**
- Reduce state size (simplify aggregations, shorter TTL)
- Increase managed memory
- Enable incremental checkpointing

```yaml
state:
  config:
    rocksdb:
      execution.checkpointing.incremental: true
```

**Interval Too Frequent:**
```yaml
checkpointing:
  interval: 60000  # ← Change from 5000ms to 60000ms
```

**Files to modify:**
- YAML jobspec: Update checkpointing config

### Issue 7: Memory Configuration

**Detection:** Script warns about low memory allocation

**Root Cause:** Insufficient memory for state-heavy operators (RocksDB).

**Fix:**
```yaml
resources:
  memory: "8gb"      # ← Increase from 2gb
  managedMemory: "2gb"  # ← 25-40% for RocksDB
```

**Best Practices:**
- Total memory: 4-16GB per task slot
- Managed memory: 25-40% of total for RocksDB jobs

**Files to modify:**
- YAML jobspec: Update resources config

## Reference Materials

### Flink REST API Endpoints

See `references/flink-rest-api.md` for detailed API documentation.

**Quick Reference:**
- List jobs: `GET /jobs`
- Job details: `GET /jobs/:jobid`
- Checkpoint stats: `GET /jobs/:jobid/checkpoints`
- Metrics: `GET /jobs/:jobid/metrics`

### Optimization Patterns

See `references/optimization-patterns.md` for comprehensive patterns and anti-patterns.

**Key Sections:**
- DAG Wiring Issues
- Parallelism & Backpressure
- Data Skew
- Checkpoint Configuration
- Memory Management
- Window Aggregation Optimization
- Operator Chaining

## Code Structure Reference

### Main.java Key Methods

**`buildJobDAG(env, jobSpec)`** - Constructs the DAG:
```java
Map<String, DataStream<?>> streams = new HashMap<>();

// 1. Build sources -> stored in streams map
for (SourceConfig source : jobSpec.getSources()) {
    streams.put(source.getName(), buildSource(env, source));
}

// 2. Build operators -> read from streams, produce new streams
for (OperatorConfig operator : jobSpec.getOperators()) {
    DataStream<?> result = buildOperator(streams, operator, jobSpec, env);
    streams.put(operator.getOutputStream(), result);
}

// 3. Build sinks -> consume from streams
for (SinkConfig sink : jobSpec.getSinks()) {
    buildSink(streams, sink);
}
```

**`buildOperator(streams, operatorConfig, jobSpec, env)`** - Builds specific operator:
```java
// Get input stream from map
String inputStreamName = operatorConfig.getInputStreams().get(0);
DataStream<?> inputStream = streams.get(inputStreamName);

// Apply operator logic
DataStream<?> result = inputStream.filter(...).map(...);

// Store output stream
return result;
```

### Jobspec YAML Structure

```yaml
jobName: "payment-detection"
parallelism: 2  # Default parallelism

sources:
  - name: "payments-kafka-source"  # ← Produces stream "payments-kafka-source"
    type: "KAFKA"
    config:
      topic: "payments"

operators:
  - name: "merchant-filter"
    type: "FILTER"
    inputStreams: ["payments-kafka-source"]  # ← Consumes source
    outputStream: "filtered-payments"        # ← Produces stream
    parallelism: 2  # Override default

  - name: "window-aggregator"
    type: "WINDOW_AGGREGATOR"
    inputStreams: ["filtered-payments"]  # ← Consumes previous operator
    outputStream: "aggregated-payments"
    parallelism: 8
    config:
      windowType: "TUMBLING"
      keyField: "merchant_id"
      aggregations:
        total_count: "COUNT(*)"

  - name: "kafka-sink"
    type: "ANOMALY_SINK"
    inputStreams: ["aggregated-payments"]  # ← Final consumer
    config:
      sinks:
        - type: "KAFKA"
          topic: "alerts"
```

## Tips for Effective Optimization

1. **Start with the script** - Don't manually inspect without data
2. **Fix critical issues first** - Dead streams and missing wiring before performance tuning
3. **Measure before and after** - Note current backpressure, throughput metrics
4. **Test incrementally** - Fix one issue at a time, validate
5. **Use Flink UI** - Visual representation helps understand flow
6. **Check job logs** - ConfigValidator errors provide specific line numbers
7. **Validate YAML syntax** - Use YAML linter before deploying

## When to Read Reference Files

- **Before first optimization:** Skim `optimization-patterns.md` for overview
- **For API questions:** Check `flink-rest-api.md` for endpoint details
- **For specific issues:** Jump to relevant section in `optimization-patterns.md`
  - DAG issues → "DAG Wiring Issues"
  - Slow processing → "Parallelism & Backpressure"
  - Uneven load → "Data Skew"
  - Checkpoint failures → "Checkpoint Configuration"
  - OOM errors → "Memory Management"
  - Slow aggregations → "Window Aggregation Optimization"
