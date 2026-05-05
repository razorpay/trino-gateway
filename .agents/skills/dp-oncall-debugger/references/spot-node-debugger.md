# EMR Spot Node Debugger

When the RCA points to **spot instance loss** (shuffle fetch failures, executor disconnections, instance fleet under-provisioned), run this analysis to find the problematic instance type and suggest a better alternative.

## WHEN TO TRIGGER

Call this analysis when ANY of these appear in the debugging:
- Executor logs show `Failed to fetch remote block` or `Empty buffer received for non empty block`
- Driver logs show `ERROR TransportClient: Failed to send RPC ... ClosedChannelException`
- EMR instance fleet shows `ProvisionedSpotCapacity < TargetSpotCapacity`
- Instances terminated after short runtime (< 30 min) — classic spot interruption
- Job is alive (batch duration metrics emit) but produces zero output after executor churn

## ANALYSIS STEPS

### Step 1: Identify the EMR cluster

Extract the cluster ID from the alert or Spinnaker pipeline name. Use AWS CLI:

```bash
aws emr describe-cluster --cluster-id <cluster-id> --region ap-south-1 \
  --query 'Cluster.{Name:Name,State:Status.State,Created:Status.Timeline.CreationDateTime}'
```

### Step 2: Check instance fleet health

```bash
aws emr list-instances --cluster-id <cluster-id> --region ap-south-1 \
  --instance-fleet-type TASK \
  --query 'Instances[*].{Id:Ec2InstanceId,Type:InstanceType,State:Status.State,Created:Status.Timeline.CreationDateTime,Ended:Status.Timeline.EndDateTime}'
```

Look for:
- Instances with short lifespans (created → ended < 30 min) = spot interruption
- Instances that never reached RUNNING state = spot capacity unavailable
- Provisioned < Target capacity = fleet couldn't fill

### Step 3: Get 24h spot price history for the configured instance types

```bash
aws ec2 describe-spot-price-history \
  --instance-types <all types from cluster config> \
  --product-descriptions "Linux/UNIX" \
  --region ap-south-1 \
  --start-time "<24h ago ISO8601>" \
  --query 'SpotPriceHistory[*].{Type:InstanceType,AZ:AvailabilityZone,Price:SpotPrice,Time:Timestamp}'
```

### Step 4: Compute stability metrics

For each instance type, compute across all AZs:
- **Average spot price** over 24h
- **CV% (Coefficient of Variation)** = (StdDev / Mean) * 100
  - CV < 3% = EXCELLENT (almost no interruptions)
  - CV 3-8% = GOOD (rare interruptions)
  - CV 8-15% = MODERATE (occasional interruptions)
  - CV > 15% = POOR (frequent interruptions, recommend removal)

### Step 5: Find replacement candidates

Query spot prices for similar instance types NOT in the current config:

For `r*.4xlarge` (memory-optimized, 16 vCPU, 128GB) fleets, good candidates:
- `r6g.4xlarge` — Graviton2, typically cheapest and most stable
- `r6gd.4xlarge` — Graviton2 + NVMe
- `r7i.4xlarge` — Intel, good diversification
- `r6i.4xlarge` — Intel
- `r5ad.4xlarge` — AMD + NVMe

For `m*.4xlarge` (general purpose) fleets:
- `m6g.4xlarge` — Graviton2
- `m7g.4xlarge` — Graviton3
- `m6i.4xlarge` — Intel

Run the same 24h spot price history and CV% analysis on candidates.

### Step 6: Output recommendation

```
## Spot Fleet Analysis

**Cluster**: <cluster name>
**Problem**: <instance type> has <CV%> volatility — highest interruption risk in fleet

### Current Fleet (ranked by spot stability, 24h data)

| Type | Avg Spot | CV% | Stability | Verdict |
|---|---|---|---|---|
| <best type> | $X.XX | X.X% | GOOD | KEEP |
| ... | ... | ... | ... | ... |
| <worst type> | $X.XX | XX.X% | POOR | REMOVE |

### Recommended Replacement

| Type | Avg Spot | CV% | Stability |
|---|---|---|---|
| <replacement> | $X.XX | X.X% | GOOD — <reason> |

Same spec (XX vCPU, XXGB). XX% cheaper, Xx more stable.
```

## KEY PRINCIPLES

1. **More instance types = fewer interruptions** — `PRICE_CAPACITY_OPTIMIZED` allocation picks from available pools. More types = more pools to choose from.
2. **Diversify across architectures** — Mix AMD (r5a/r6a), Intel (r6i/r7i), and Graviton (r6g/r7g) so a capacity crunch in one pool doesn't kill the fleet.
3. **Remove the worst, don't just add more** — A single high-volatility type can get allocated by the strategy and then interrupted, causing cascade failures.
4. **Graviton instances are typically cheapest and most stable** — Lower demand in ap-south-1 means less competition and fewer interruptions.
5. **Check per-AZ pricing** — An instance type can be EXCELLENT in one AZ but POOR in another. `PRICE_CAPACITY_OPTIMIZED` handles this, but it's good to know.
