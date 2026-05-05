# Query DAG Failure Debugger

**Triages and investigates Airflow DAG failures across ~3000 DAGs. Give it an error message or log snippet — it classifies, explains, and deep-dives with live data. Zero credential setup required.**

---

## How It Works

```mermaid
flowchart LR
    IN(["📥 DAG name\n+ error / log"]) --> CL["🔍 Classify\n→ bucket + confidence"]

    CL --> B1["⚡ Spark/EMR"]
    CL --> B2["🗄️ Trino/Presto\n⚠️ no Trino MCP yet\nuses Athena instead"]
    CL --> B3["🔄 Airflow"]
    CL --> B4["🏗️ Infra"]

    subgraph MCPs["🔌 MCPs"]
        direction TB
        CX["📜 Coralogix\ntask logs · timeline\nscheduler warnings"]
        AWS["☁️ Friday AWS MCP\nEMR · S3 logs\nAthena metadata"]
        K8S["🖥️ Friday K8s MCP\nkubectl · pod health"]
        GF["📊 Grafana\nnode/pod metrics"]
    end

    B1 --> CX & AWS
    B2 --> CX & AWS
    B3 --> CX & K8S
    B4 --> CX & K8S & GF

    CX & AWS & K8S & GF --> REPORT(["📋 Report"])

    style CL fill:#6366F1,color:#fff,stroke:#6366F1
    style IN fill:#1E293B,color:#fff,stroke:#1E293B
    style REPORT fill:#22C55E,color:#fff,stroke:#22C55E
    style MCPs fill:#FFF7ED,stroke:#F97316,stroke-width:2px
    style B2 fill:#FEF9C3,stroke:#EAB308
```

---

## Features at a Glance

| | |
|---|---|
| **Buckets** | 9 — across Spark/EMR, Trino/Presto, Airflow, Infra |
| **Confidence** | HIGH / MEDIUM / LOW, evidence-grounded |
| **Regression check** | Proactive `git log` + last successful run — no asking |
| **MCP-powered** | Friday AWS MCP (EMR/S3/Athena) · Coralogix · Grafana · Friday K8s MCP |
| **Output** | Slack triage block — bucket, confidence, root cause, owners |
| **Scope** | ~3000 DAGs · airflow-dags monorepo · AWS ap-south-1 |
