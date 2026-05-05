# EMR and DAG Investigation via AWS CLI

How to investigate a Spark/EMR DAG run by reading the DAG definition, finding the right EMR cluster, and pulling the highest-signal logs from S3 using the AWS CLI.

---

## Prerequisites

Use **Friday AWS MCP** (`mcp__aws-api__call_aws`) for all AWS commands. Auth is handled automatically — **no `--profile` flag, no credentials, no access keys needed from the user.** Commands run against the `prod` account in `ap-south-1` by default.

Use the **Fast path** first: DAG read → cluster lookup → cluster status → failed step → step `stderr.gz`.
Switch to **Deep mode** only when step logs are inconclusive and you need container-level evidence.

Repo-specific pattern to remember: `CLUSTER_CONFIG_FILE` values like `de/datasink_common.json` are often the quickest way to derive the EMR cluster name pattern, and `log_uri` in the cluster config / executor setup determines the final S3 log location.

Before downloading large logs, make sure you already have the minimum Spark artifacts: DAG name, execution date, cluster ID or cluster pattern, and step ID if available.

---

## Step 0 — Read the DAG Definition First

Before touching AWS, inspect the DAG file and extract:
- `CLUSTER_CONFIG_FILE`
- `spark_submit_params`
- `python_packages`
- EMR step names or job entrypoint

Common DAG locations:
- `dags/spark/`
- `dags/python/`
- `dags/data-quality/`

The cluster config name without `.json` is often the best cluster-name search pattern.

Example:
- `CLUSTER_CONFIG_FILE = "de/datasink_common.json"`
- cluster name pattern to search = `datasink_common`

---

## Step 1 — Get the Cluster ID

Preferred order:

1. If the Airflow task log already contains the cluster ID, use that.
2. If the user gave the cluster ID, use that.
3. Otherwise derive a cluster-name pattern from `CLUSTER_CONFIG_FILE` and search for clusters.

The cluster ID (`j-XXXXXXXXXX`) is often in the Airflow task log. Look for lines like:

```
Cluster created: j-3ABCDEF123456
Waiting for step to complete on cluster: j-3ABCDEF123456
```

If you need to search by cluster name pattern:

```bash
# Active clusters
aws emr list-clusters --active --output json | jq -r '.Clusters[] | select(.Name | test("<cluster-pattern>"; "i")) | [.Id, .Name, .Status.State] | @tsv'

# Terminated / failed clusters
aws emr list-clusters --cluster-states TERMINATED TERMINATED_WITH_ERRORS --output json | jq -r '.Clusters[] | select(.Name | test("<cluster-pattern>"; "i")) | [.Id, .Name, .Status.State] | @tsv'
```

Or check the EMR console: AWS Console → EMR → Clusters → filter by creation time or cluster name.

---

## Step 2 — Check Cluster Status

```bash
aws emr describe-cluster --cluster-id j-XXXXXXXXXX \
  --query 'Cluster.{Status:Status.State,Reason:Status.StateChangeReason.Message}'
```

Key states:
- `WAITING` — cluster is up, no active steps
- `RUNNING` — steps currently executing
- `TERMINATED` — cluster was shut down (check reason: auto-termination? manual? spot reclaim?)
- `TERMINATED_WITH_ERRORS` — a step failed; see Step 3

---

## Step 3 — Find the Failed Step

```bash
# List all steps and their status
aws emr list-steps --cluster-id j-XXXXXXXXXX \
  --query 'Steps[*].{Name:Name,State:Status.State,Reason:Status.FailureDetails.Reason}' \
  --output table
```

Note the Step ID (`s-XXXXXXXXXX`) of the failed step.

```bash
# Get full failure details for a specific step
aws emr describe-step --cluster-id j-XXXXXXXXXX --step-id s-XXXXXXXXXX \
  --query 'Step.Status.FailureDetails'
```

Also pull the cluster log URI before fetching S3 logs:

```bash
aws emr describe-cluster --cluster-id j-XXXXXXXXXX \
  --query 'Cluster.LogUri' --output text
```

---

## Step 4 — Pull Step Logs from S3

EMR logs are structured in S3 as:

```
s3://<log-bucket>/<log-prefix>/<cluster-id>/steps/<step-id>/
  ├── stderr.gz   ← Java/Python exception, stack trace
  ├── stdout.gz   ← job output, record counts, print statements
  └── controller  ← YARN step execution log
```

```bash
# List available log files for the step
aws s3 ls s3://<log-bucket>/<log-prefix>/<cluster-id>/steps/<step-id>/

# Check exact object size before downloading large files
aws s3api head-object --bucket <bucket> --key "<path-to-file>" --query 'ContentLength' --output text

# Download and decompress stderr (most useful for failures)
aws s3 cp s3://<log-bucket>/<log-prefix>/<cluster-id>/steps/<step-id>/stderr.gz - | gunzip | grep -i "exception\|error\|failed\|killed" | tail -100

# Download stdout (useful for record counts, job progress)
aws s3 cp s3://<log-bucket>/<log-prefix>/<cluster-id>/steps/<step-id>/stdout.gz - | gunzip | grep -i "processing\|records\|rows\|written" | tail -50
```

If any log file is larger than 100 MB, tell the user the size and ask before downloading it.

---

## Step 5 — Pull Container Logs (for OOM / executor details)

If stderr doesn't have the root cause (common with executor OOM), pull container logs:

```bash
# List container log directories
aws s3 ls s3://<log-bucket>/<log-prefix>/<cluster-id>/containers/

# Structure: containers/<app-id>/<container-id>/
# The driver container is usually container_XXXXXXXXXX_0001_01_000001

# Check the driver container contents first
aws s3 ls "s3://<log-bucket>/<log-prefix>/<cluster-id>/containers/<app-id>/<container-id>/"

# Pull driver container stderr
aws s3 cp "s3://<log-bucket>/<log-prefix>/<cluster-id>/containers/<app-id>/<container-id>/stderr.gz" - | gunzip | grep -i "OutOfMemory\|GC overhead\|YARN\|killed"

# Pull driver container stdout first when you want application progress / record counts
aws s3 cp "s3://<log-bucket>/<log-prefix>/<cluster-id>/containers/<app-id>/<container-id>/stdout.gz" - | gunzip
```

Prefer container `stdout.gz` first for application-level progress, then `stderr.gz` for Spark runtime/framework exceptions.

---

## Common Log Patterns to Search For

| Failure type | What to grep |
|---|---|
| YARN OOM kill | `Container killed by YARN for exceeding memory` |
| Heap OOM | `java.lang.OutOfMemoryError` |
| Executor lost | `ExecutorLostFailure` |
| Bootstrap failure | `BOOTSTRAP_FAILURE`, `bootstrap action` |
| Spark logic error | `Exception in thread\|Caused by:` |
| Data skew | `Fetching big block\|task.*running.*[0-9]{4,}s` |
| Livy session death | `Session.*unexpectedly reached final status` |
| Ivy dependency fail | `unresolved dependency\|not found.*ivy\|ERROR.*resolving` |

---

## If Friday AWS MCP Is Unavailable

If `mcp__aws-api__call_aws` is not available in the session:

1. Note the gap in `[Data Gaps]` — do not stop investigation; proceed with DAG code and any logs the user already shared
2. Ask the user to check the EMR console: AWS Console → EMR → Clusters → `j-XXXXXXXXXX` → Steps → click the failed step → View logs
3. Check if the Airflow task log contains a truncated version of the error (it often includes the first few lines of `stderr`)

---

## Recommended Investigation Order

1. Read the DAG file and extract `CLUSTER_CONFIG_FILE` plus Spark params
2. Find the cluster ID from Airflow logs or by cluster name pattern
3. Describe the cluster and get `LogUri`
4. List steps and identify the failed step
5. Read step `stderr.gz` for the quickest failure signal
6. If the root cause is still unclear, read driver container `stdout.gz` and `stderr.gz`
7. Map the findings back to the skill's failure buckets

This order keeps the investigation fast and reduces unnecessary large downloads.

---

## Log Bucket Names

These vary by environment — find the actual bucket name from the DAG's cluster config or from a data platform team member:

| Environment | Typical bucket pattern |
|---|---|
| Production | `s3://razorpay-emr-logs-prod/` |
| Stage | `s3://razorpay-emr-logs-stage/` |

The exact prefix within the bucket is set in the DAG's `EmrStepsExecutor` config as `log_uri`.
