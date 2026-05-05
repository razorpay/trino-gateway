# Flink REST API Reference

Quick reference for Flink REST API endpoints used for job analysis and optimization.

Base URL: `{cluster_url}` (e.g., `https://flink.de.razorpay.com`)

## Core Job Monitoring Endpoints

### List All Jobs
**Endpoint:** `GET /jobs`

**Returns:** Overview of all jobs with their current states (RUNNING, FAILED, FINISHED, etc.)

**Example Response:**
```json
{
  "jobs": [
    {
      "id": "abc123...",
      "status": "RUNNING"
    }
  ]
}
```

### Get Job Details
**Endpoint:** `GET /jobs/:jobid`

**Returns:** Comprehensive job information including:
- Execution plan with vertices (operators)
- Vertex details (name, parallelism, status)
- Timestamps (start time, duration)
- Task status counts

**Example Response:**
```json
{
  "jid": "abc123...",
  "name": "My Flink Job",
  "state": "RUNNING",
  "start-time": 1234567890,
  "vertices": [
    {
      "id": "vertex-1",
      "name": "Source: payments-kafka-source",
      "parallelism": 2,
      "status": "RUNNING"
    },
    {
      "id": "vertex-2",
      "name": "merchant-filter-filter",
      "parallelism": 1,
      "status": "RUNNING"
    }
  ]
}
```

### Job Overview
**Endpoint:** `GET /jobs/overview`

**Returns:** High-level summary of all jobs with duration, start/end times, and task breakdowns

## Performance & Monitoring

### Job Metrics
**Endpoint:** `GET /jobs/:jobid/metrics`

**Returns:** Aggregated metrics across job with support for min, max, sum, avg, and skew calculations

**Query Parameters:**
- `get` - Comma-separated list of metric names to fetch

**Common Metrics:**
- `numRecordsIn` - Records received
- `numRecordsOut` - Records emitted
- `numBytesIn` - Bytes received
- `backPressureLevel` - Backpressure indication

### Checkpoint Statistics
**Endpoint:** `GET /jobs/:jobid/checkpoints`

**Returns:** Checkpointing data including:
- Counts (completed, failed, in-progress)
- Historical checkpoint records
- Summary statistics (min/max/avg duration)

**Example Response:**
```json
{
  "counts": {
    "completed": 150,
    "failed": 5,
    "in_progress": 1
  },
  "latest": {
    "completed": {
      "id": 150,
      "status": "COMPLETED",
      "duration": 45000,
      "external_path": "s3://..."
    }
  }
}
```

### Checkpoint Details
**Endpoint:** `GET /jobs/:jobid/checkpoints/details/:checkpointid`

**Returns:** Per-task checkpoint metrics and per-subtask statistics

### Checkpoint Configuration
**Endpoint:** `GET /jobs/:jobid/checkpoints/config`

**Returns:** Job's checkpoint settings:
- Checkpoint interval
- Checkpoint timeout
- Externalization options
- Mode (EXACTLY_ONCE, AT_LEAST_ONCE)

## Job Execution Information

### Job Configuration
**Endpoint:** `GET /jobs/:jobid/config`

**Returns:** Job-specific configuration settings including parallelism, restart strategy, etc.

### Job Exceptions
**Endpoint:** `GET /jobs/:jobid/exceptions`

**Returns:** Most recent exceptions that have been handled by Flink for this job

**Example Response:**
```json
{
  "all-exceptions": [
    {
      "exception": "java.lang.NullPointerException: ...",
      "task": "Source: payments-kafka-source",
      "location": "TaskManager-1",
      "timestamp": 1234567890
    }
  ],
  "truncated": false
}
```

### Job Accumulators
**Endpoint:** `GET /jobs/:jobid/accumulators`

**Returns:** Aggregated accumulator values across subtasks

## Vertex (Operator) Details

### Vertex Metrics
**Endpoint:** `GET /jobs/:jobid/vertices/:vertexid/metrics`

**Returns:** Metrics for a specific operator (vertex)

**Query Parameters:**
- `get` - Metric names to fetch

**Per-Operator Metrics:**
- `numRecordsInPerSecond` - Input rate
- `numRecordsOutPerSecond` - Output rate
- `busyTimeMsPerSecond` - Operator busy time
- `backPressuredTimeMsPerSecond` - Time spent backpressured

### Vertex Backpressure (Deprecated in newer versions)
**Endpoint:** `GET /jobs/:jobid/vertices/:vertexid/backpressure`

**Returns:** Backpressure status for operator

**Note:** In Flink 1.13+, use metrics API with `backPressuredTimeMsPerSecond` metric instead

## Usage Tips

1. **Identifying Bottlenecks:**
   - Check vertex parallelism in job details
   - Look at `backPressuredTimeMsPerSecond` metric per vertex
   - High backpressure + low parallelism = increase parallelism

2. **Checkpoint Health:**
   - Monitor failure rate: `failed / (failed + completed)`
   - Check checkpoint duration vs interval
   - Alert if duration > interval (checkpoints pile up)

3. **Performance Analysis:**
   - Compare `numRecordsInPerSecond` across operators
   - Identify data skew by checking per-subtask metrics
   - Monitor `busyTimeMsPerSecond` to find expensive operators
