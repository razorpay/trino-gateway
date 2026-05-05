#!/usr/bin/env python3
"""
Report generation - Format analysis results into human-readable reports

Generates summary tables, detailed findings, and actionable recommendations.
"""

import logging
from datetime import datetime
from typing import Dict, List, Any
from collections import defaultdict

logger = logging.getLogger(__name__)


def generate_report(
    namespace: str,
    clusters: List[str],
    hours: int,
    analysis_results: Dict[str, Dict],
    start_time: datetime,
    end_time: datetime,
    pod_summary: str = None
) -> str:
    """
    Generate a comprehensive analysis report

    Args:
        namespace: K8s namespace analyzed
        clusters: List of clusters
        hours: Number of hours analyzed
        analysis_results: Results from analyze_deployment_metrics
        start_time: Analysis start time
        end_time: Analysis end time
        pod_summary: Summary string of pod categorization (e.g., "350 running (150 web + 180 workers) + 12 cronjobs = 362 total")

    Returns:
        Formatted report string
    """

    logger.info("Generating report...")

    lines = []

    # Header
    lines.append("=" * 80)
    lines.append("K8s RESOURCE ANALYZER REPORT")
    lines.append("=" * 80)
    lines.append("")

    # Metadata
    lines.append(f"Namespace: {namespace}")
    lines.append(f"Clusters: {', '.join(clusters)}")
    lines.append(f"Analyzed Deployments: {len(analysis_results)}")
    if pod_summary:
        lines.append(f"Pod Summary: {pod_summary}")
    else:
        total_pods = sum(r.get('pod_count', 0) for r in analysis_results.values())
        lines.append(f"Total Pods: {total_pods}")
    lines.append(f"Analysis Period: {hours} hours ({start_time.isoformat()} to {end_time.isoformat()})")
    lines.append(f"Peak Calculation: P95 percentile")
    lines.append("")

    # Summary table
    lines.extend(generate_summary_table(analysis_results))
    lines.append("")

    # Findings by recommendation type
    lines.extend(generate_set_limit_findings(analysis_results))
    lines.append("")

    lines.extend(generate_increase_findings(analysis_results))
    lines.append("")

    lines.extend(generate_decrease_findings(analysis_results))
    lines.append("")

    lines.extend(generate_hpa_candidates(analysis_results))
    lines.append("")

    lines.extend(generate_at_risk_findings(analysis_results))
    lines.append("")

    # Detailed breakdown
    lines.append("DETAILED BREAKDOWN:")
    lines.append("-" * 80)
    lines.append("")

    for deployment_name in sorted(analysis_results.keys()):
        analysis = analysis_results[deployment_name]
        lines.extend(generate_deployment_detail(deployment_name, analysis))
        lines.append("")

    # Summary statistics
    lines.extend(generate_summary_stats(analysis_results))
    lines.append("")

    # Memory optimization summary - MOVED TO END with prominent display
    lines.extend(generate_memory_optimization_summary(analysis_results))
    lines.append("")

    lines.append("=" * 80)
    lines.append("End of Report")
    lines.append("=" * 80)

    report = "\n".join(lines)
    logger.info(f"Report generated ({len(report)} chars)")

    return report


def generate_summary_table(analysis_results: Dict[str, Dict]) -> List[str]:
    """Generate summary table with deployment metrics and recommendations"""

    lines = ["SUMMARY TABLE:"]
    lines.append("-" * 100)
    lines.append(f"{'Deployment':<25} {'Pods':<6} {'P95 CPU/Mem':<18} {'Req CPU/Mem':<18} {'Recommendation':<20}")
    lines.append("-" * 100)

    for deployment_name in sorted(analysis_results.keys()):
        analysis = analysis_results[deployment_name]

        pod_count = analysis.get("pod_count", 0)

        # Get P95 values
        cpu_p95 = analysis.get("metrics", {}).get("cpu", {}).get("p95", "N/A")
        mem_p95 = analysis.get("metrics", {}).get("memory", {}).get("p95", "N/A")

        if isinstance(cpu_p95, (int, float)):
            cpu_p95_str = f"{int(cpu_p95)}m"
        else:
            cpu_p95_str = str(cpu_p95)

        if isinstance(mem_p95, (int, float)):
            mem_p95_str = f"{int(mem_p95)}Mi"
        else:
            mem_p95_str = str(mem_p95)

        p95_str = f"{cpu_p95_str} / {mem_p95_str}"

        # Get requested values
        spec_lines = analysis.get("current_spec", {}).get("request", "N/A").split(" / ")
        req_str = " / ".join(spec_lines[:2]) if len(spec_lines) >= 2 else "N/A"

        # Get recommendation
        cpu_rec = analysis.get("recommendations", {}).get("cpu", {}).get("type", "")
        mem_rec = analysis.get("recommendations", {}).get("memory", {}).get("type", "")
        hpa_rec = "HPA" if analysis.get("recommendations", {}).get("hpa", {}).get("enabled") else ""

        recommendation = " / ".join(filter(None, [cpu_rec, mem_rec, hpa_rec]))[:20]

        status = analysis.get("status", "UNKNOWN")
        icon = "✅" if "HEALTHY" in status else "⚠️" if "ATTENTION" in status else "❓"

        lines.append(f"{deployment_name:<25} {pod_count:<6} {p95_str:<18} {req_str:<18} {icon} {recommendation:<20}")

    lines.append("-" * 100)

    return lines


def generate_set_limit_findings(analysis_results: Dict[str, Dict]) -> List[str]:
    """Generate findings for deployments missing resource limits"""

    findings = []
    limit_deployments = []

    for dep_name, analysis in analysis_results.items():
        cpu_rec = analysis.get("recommendations", {}).get("cpu", {})
        mem_rec = analysis.get("recommendations", {}).get("memory", {})

        if cpu_rec.get("type") == "SET_LIMIT" or mem_rec.get("type") == "SET_LIMIT":
            limit_deployments.append((dep_name, analysis))

    if limit_deployments:
        findings.append("🚨 SET RESOURCE LIMITS (Pods have no limits set):")
        findings.append("-" * 80)

        for dep_name, analysis in limit_deployments:
            findings.append(f"\n{dep_name} ({analysis.get('pod_count')} pods)")

            cpu_rec = analysis.get("recommendations", {}).get("cpu", {})
            if cpu_rec.get("type") == "SET_LIMIT":
                findings.append(f"  CPU: {cpu_rec.get('reason', 'N/A')}")
                if cpu_rec.get("new_limit_m"):
                    findings.append(f"       → Set CPU limit to {int(cpu_rec.get('new_limit_m'))}m")
                findings.append(f"       Priority: {cpu_rec.get('priority', 'UNKNOWN')}")

            mem_rec = analysis.get("recommendations", {}).get("memory", {})
            if mem_rec.get("type") == "SET_LIMIT":
                findings.append(f"  Memory: {mem_rec.get('reason', 'N/A')}")
                if mem_rec.get("new_limit_mi"):
                    findings.append(f"          → Set memory limit to {int(mem_rec.get('new_limit_mi'))}Mi")
                findings.append(f"          Priority: {mem_rec.get('priority', 'UNKNOWN')}")

    return findings


def generate_memory_optimization_summary(analysis_results: Dict[str, Dict]) -> List[str]:
    """Generate resource optimization summary at end of report

    Shows per-deployment breakdown:
    - Memory savings (over-provisioned deployments)
    - Memory increases needed (under-provisioned deployments)
    - CPU savings and increases
    - Net impact
    - Calculation: per-pod change × actual pod count
    """

    findings = []

    # Separate memory savings and increases
    mem_savings_total_mi = 0
    cpu_savings_total_m = 0
    mem_increase_total_mi = 0
    cpu_increase_total_m = 0
    mem_savings_deployments = []
    mem_increase_deployments = []
    cpu_savings_deployments = []
    cpu_increase_deployments = []

    for dep_name, analysis in analysis_results.items():
        mem_rec = analysis.get("recommendations", {}).get("memory", {})
        cpu_rec = analysis.get("recommendations", {}).get("cpu", {})
        requested = analysis.get("current_spec", {})
        pod_count = analysis.get("pod_count", 1)

        # Parse memory from spec: "XXXX m CPU / XXXXMi Memory"
        spec_request = requested.get("request", "")
        spec_parts = spec_request.split("/")
        cpu_str = spec_parts[0].strip() if len(spec_parts) > 0 else "0m"
        mem_str = spec_parts[1].strip() if len(spec_parts) > 1 else "0Mi"

        try:
            current_cpu_m = int(cpu_str.replace("m", "").strip())
        except:
            current_cpu_m = 0

        try:
            current_mem_mi = int(mem_str.replace("Mi", "").strip())
        except:
            current_mem_mi = 0

        # Calculate memory changes
        if mem_rec.get("type") == "DECREASE_REQUEST":
            new_mem_mi = mem_rec.get("new_request_mi", 0)
            per_pod_mem_savings = current_mem_mi - new_mem_mi
            if per_pod_mem_savings > 0:
                total_mem_savings = per_pod_mem_savings * pod_count
                mem_savings_total_mi += total_mem_savings
                mem_savings_deployments.append({
                    "name": dep_name,
                    "pods": pod_count,
                    "current": current_mem_mi,
                    "new": new_mem_mi,
                    "per_pod": per_pod_mem_savings,
                    "total": total_mem_savings
                })

        elif mem_rec.get("type") == "INCREASE_REQUEST":
            new_mem_mi = mem_rec.get("new_request_mi", 0)
            per_pod_mem_increase = new_mem_mi - current_mem_mi
            if per_pod_mem_increase > 0:
                total_mem_increase = per_pod_mem_increase * pod_count
                mem_increase_total_mi += total_mem_increase
                mem_increase_deployments.append({
                    "name": dep_name,
                    "pods": pod_count,
                    "current": current_mem_mi,
                    "new": new_mem_mi,
                    "per_pod": per_pod_mem_increase,
                    "total": total_mem_increase
                })

        # Calculate CPU changes (same logic)
        if cpu_rec.get("type") == "DECREASE_REQUEST":
            new_cpu_m = cpu_rec.get("new_request_m", 0)
            per_pod_cpu_savings = current_cpu_m - new_cpu_m
            if per_pod_cpu_savings > 0:
                total_cpu_savings = per_pod_cpu_savings * pod_count
                cpu_savings_total_m += total_cpu_savings
                cpu_savings_deployments.append({
                    "name": dep_name,
                    "pods": pod_count,
                    "current": current_cpu_m,
                    "new": new_cpu_m,
                    "per_pod": per_pod_cpu_savings,
                    "total": total_cpu_savings
                })

        elif cpu_rec.get("type") == "INCREASE_REQUEST":
            new_cpu_m = cpu_rec.get("new_request_m", 0)
            per_pod_cpu_increase = new_cpu_m - current_cpu_m
            if per_pod_cpu_increase > 0:
                total_cpu_increase = per_pod_cpu_increase * pod_count
                cpu_increase_total_m += total_cpu_increase
                cpu_increase_deployments.append({
                    "name": dep_name,
                    "pods": pod_count,
                    "current": current_cpu_m,
                    "new": new_cpu_m,
                    "per_pod": per_pod_cpu_increase,
                    "total": total_cpu_increase
                })

    if mem_savings_deployments or mem_increase_deployments or cpu_savings_deployments or cpu_increase_deployments:
        # Header - make it VERY prominent
        findings.append("")
        findings.append("=" * 80)
        findings.append("💾 RESOURCE OPTIMIZATION SUMMARY")
        findings.append("=" * 80)
        findings.append("")

        # Memory Savings - HIGHLIGHT
        if mem_savings_deployments:
            mem_savings_gib = mem_savings_total_mi / 1024
            findings.append("✅ MEMORY THAT CAN BE FREED:")
            findings.append("-" * 80)
            findings.append(f"💾 TOTAL: {mem_savings_gib:.2f} GiB")
            findings.append("")

            # Per-deployment breakdown
            for item in sorted(mem_savings_deployments, key=lambda x: x["total"], reverse=True):
                findings.append(f"{item['name']} ({item['pods']} pods)")
                findings.append(f"  {item['current']}Mi → {item['new']}Mi per pod")
                findings.append(f"  Savings: {item['per_pod']}Mi/pod × {item['pods']} pods = {item['total']}Mi")
                findings.append("")

        # CPU Savings
        if cpu_savings_deployments:
            cpu_savings_cores = cpu_savings_total_m / 1000
            findings.append("✅ CPU THAT CAN BE FREED:")
            findings.append("-" * 80)
            findings.append(f"⚙️  TOTAL: {cpu_savings_cores:.2f} cores")
            findings.append("")

            for item in sorted(cpu_savings_deployments, key=lambda x: x["total"], reverse=True):
                findings.append(f"{item['name']} ({item['pods']} pods)")
                findings.append(f"  {item['current']}m → {item['new']}m per pod")
                findings.append(f"  Savings: {item['per_pod']}m/pod × {item['pods']} pods = {item['total']}m")
                findings.append("")

        # Memory Increases
        if mem_increase_deployments:
            mem_increase_gib = mem_increase_total_mi / 1024
            findings.append("⚠️  MEMORY INCREASES NEEDED (for stability):")
            findings.append("-" * 80)
            findings.append(f"TOTAL: {mem_increase_gib:.2f} GiB")
            findings.append("")

            for item in sorted(mem_increase_deployments, key=lambda x: x["total"], reverse=True):
                findings.append(f"{item['name']} ({item['pods']} pods)")
                findings.append(f"  {item['current']}Mi → {item['new']}Mi per pod")
                findings.append(f"  Increase: {item['per_pod']}Mi/pod × {item['pods']} pods = {item['total']}Mi")
                findings.append("")

        # CPU Increases
        if cpu_increase_deployments:
            cpu_increase_cores = cpu_increase_total_m / 1000
            findings.append("⚠️  CPU INCREASES NEEDED (for stability):")
            findings.append("-" * 80)
            findings.append(f"TOTAL: {cpu_increase_cores:.2f} cores")
            findings.append("")

            for item in sorted(cpu_increase_deployments, key=lambda x: x["total"], reverse=True):
                findings.append(f"{item['name']} ({item['pods']} pods)")
                findings.append(f"  {item['current']}m → {item['new']}m per pod")
                findings.append(f"  Increase: {item['per_pod']}m/pod × {item['pods']} pods = {item['total']}m")
                findings.append("")

        # NET IMPACT
        findings.append("📊 NET RESOURCE IMPACT:")
        findings.append("-" * 80)
        net_mem_mi = mem_savings_total_mi - mem_increase_total_mi
        net_cpu_m = cpu_savings_total_m - cpu_increase_total_m
        net_mem_gib = net_mem_mi / 1024
        net_cpu_cores = net_cpu_m / 1000

        findings.append(f"Memory: {net_mem_gib:+.2f} GiB (can be freed)")
        findings.append(f"CPU: {net_cpu_cores:+.2f} cores (can be freed)")
        findings.append("")

        if net_mem_gib > 0 or net_cpu_cores > 0:
            findings.append(f"✅ By right-sizing resources, you optimize allocation while improving pod stability")
            findings.append(f"   and preventing OOM kills and CPU throttling.")
        else:
            findings.append(f"ℹ️  Resource increases needed for stability will offset some optimization gains.")

        findings.append("=" * 80)
        findings.append("")

    return findings


def generate_increase_findings(analysis_results: Dict[str, Dict]) -> List[str]:
    """Generate findings for deployments that need increased resources"""

    findings = []
    increase_deployments = []

    for dep_name, analysis in analysis_results.items():
        cpu_rec = analysis.get("recommendations", {}).get("cpu", {})
        mem_rec = analysis.get("recommendations", {}).get("memory", {})

        if cpu_rec.get("type") == "INCREASE_REQUEST" or mem_rec.get("type") == "INCREASE_REQUEST":
            increase_deployments.append((dep_name, analysis))

    if increase_deployments:
        findings.append("⬆️  INCREASE RESOURCES (Pods may need more capacity):")
        findings.append("-" * 80)

        for dep_name, analysis in increase_deployments:
            findings.append(f"\n{dep_name} ({analysis.get('pod_count')} pods)")

            cpu_rec = analysis.get("recommendations", {}).get("cpu", {})
            if cpu_rec.get("type") == "INCREASE_REQUEST":
                findings.append(f"  CPU: {cpu_rec.get('reason', 'N/A')}")
                if cpu_rec.get("new_request_m"):
                    findings.append(f"       → Increase request to {int(cpu_rec.get('new_request_m'))}m")
                findings.append(f"       Priority: {cpu_rec.get('priority', 'UNKNOWN')}")

            mem_rec = analysis.get("recommendations", {}).get("memory", {})
            if mem_rec.get("type") == "INCREASE_REQUEST":
                findings.append(f"  Memory: {mem_rec.get('reason', 'N/A')}")
                if mem_rec.get("new_request_mi"):
                    findings.append(f"          → Increase request to {int(mem_rec.get('new_request_mi'))}Mi")
                findings.append(f"          Priority: {mem_rec.get('priority', 'UNKNOWN')}")

    if findings and findings[0] != "⬆️  INCREASE RESOURCES (Pods may need more capacity):":
        return []

    return findings


def generate_decrease_findings(analysis_results: Dict[str, Dict]) -> List[str]:
    """Generate findings for over-provisioned deployments"""

    findings = []
    decrease_deployments = []

    for dep_name, analysis in analysis_results.items():
        cpu_rec = analysis.get("recommendations", {}).get("cpu", {})
        mem_rec = analysis.get("recommendations", {}).get("memory", {})

        if cpu_rec.get("type") == "DECREASE_REQUEST" or mem_rec.get("type") == "DECREASE_REQUEST":
            decrease_deployments.append((dep_name, analysis))

    if decrease_deployments:
        findings.append("⬇️  DECREASE RESOURCES (Over-provisioned, can save costs):")
        findings.append("-" * 80)

        for dep_name, analysis in decrease_deployments:
            findings.append(f"\n{dep_name} ({analysis.get('pod_count')} pods)")

            cpu_rec = analysis.get("recommendations", {}).get("cpu", {})
            if cpu_rec.get("type") == "DECREASE_REQUEST":
                findings.append(f"  CPU: {cpu_rec.get('reason', 'N/A')}")
                if cpu_rec.get("new_request_m"):
                    findings.append(f"       → Decrease request to {int(cpu_rec.get('new_request_m'))}m")

            mem_rec = analysis.get("recommendations", {}).get("memory", {})
            if mem_rec.get("type") == "DECREASE_REQUEST":
                findings.append(f"  Memory: {mem_rec.get('reason', 'N/A')}")
                if mem_rec.get("new_request_mi"):
                    findings.append(f"          → Decrease request to {int(mem_rec.get('new_request_mi'))}Mi")

    if findings and findings[0] != "⬇️  DECREASE RESOURCES (Over-provisioned, can save costs):":
        return []

    return findings


def generate_hpa_candidates(analysis_results: Dict[str, Dict]) -> List[str]:
    """Generate HPA candidate recommendations"""

    findings = []
    hpa_candidates = []

    for dep_name, analysis in analysis_results.items():
        hpa_rec = analysis.get("recommendations", {}).get("hpa", {})
        if hpa_rec.get("enabled"):
            hpa_candidates.append((dep_name, analysis, hpa_rec))

    if hpa_candidates:
        findings.append("🎯 ENABLE HPA (Variable workload - auto-scaling recommended):")
        findings.append("-" * 80)

        for dep_name, analysis, hpa_rec in hpa_candidates:
            findings.append(f"\n{dep_name}")
            findings.append(f"  Reason: {hpa_rec.get('reason', 'N/A')}")
            findings.append(f"  Metric for scaling: {hpa_rec.get('metric_for_scaling', 'cpu').upper()}")
            findings.append(f"  Suggested config:")
            findings.append(f"    - Min replicas: {hpa_rec.get('suggested_min_replicas', 1)}")
            findings.append(f"    - Max replicas: {hpa_rec.get('suggested_max_replicas', 3)}")
            findings.append(f"    - Target utilization: {hpa_rec.get('target_utilization', 70)}%")

    return findings


def generate_at_risk_findings(analysis_results: Dict[str, Dict]) -> List[str]:
    """Generate critical findings for pods at risk"""

    findings = []
    at_risk = []

    for dep_name, analysis in analysis_results.items():
        cpu_rec = analysis.get("recommendations", {}).get("cpu", {})
        mem_rec = analysis.get("recommendations", {}).get("memory", {})

        if cpu_rec.get("priority") == "CRITICAL" or mem_rec.get("priority") == "CRITICAL":
            at_risk.append((dep_name, analysis))

    if at_risk:
        findings.append("🔴 AT RISK (Pods may be throttled or OOM killed):")
        findings.append("-" * 80)

        for dep_name, analysis in at_risk:
            findings.append(f"\n{dep_name}")

            cpu_rec = analysis.get("recommendations", {}).get("cpu", {})
            if cpu_rec.get("priority") == "CRITICAL":
                findings.append(f"  ⚠️  {cpu_rec.get('reason', 'N/A')}")

            mem_rec = analysis.get("recommendations", {}).get("memory", {})
            if mem_rec.get("priority") == "CRITICAL":
                findings.append(f"  ⚠️  {mem_rec.get('reason', 'N/A')}")

            findings.append(f"  ACTION REQUIRED: Apply resource increases immediately")

    return findings


def generate_deployment_detail(deployment_name: str, analysis: Dict[str, Any]) -> List[str]:
    """Generate detailed breakdown for a single deployment"""

    lines = []

    lines.append(f"{deployment_name} ({analysis.get('pod_count')} pods, on {', '.join(analysis.get('clusters', []))})")
    lines.append(f"  Status: {analysis.get('status', 'UNKNOWN')}")
    lines.append("")

    # Current spec
    current = analysis.get("current_spec", {})
    lines.append(f"  Current Resources:")
    lines.append(f"    Request: {current.get('request', 'N/A')}")
    lines.append(f"    Limit:   {current.get('limit', 'N/A')}")
    lines.append("")

    # Metrics
    cpu_metrics = analysis.get("metrics", {}).get("cpu", {})
    mem_metrics = analysis.get("metrics", {}).get("memory", {})

    if cpu_metrics:
        lines.append(f"  CPU Metrics (P95 over 48h):")
        lines.append(f"    P95:     {int(cpu_metrics.get('p95', 0))}m")
        lines.append(f"    Average: {int(cpu_metrics.get('average', 0))}m")
        lines.append(f"    Std Dev: {int(cpu_metrics.get('std_dev', 0))}m")
        lines.append(f"    Range:   {int(cpu_metrics.get('min', 0))}m - {int(cpu_metrics.get('max', 0))}m")
        lines.append("")

    if mem_metrics:
        lines.append(f"  Memory Metrics (P95 over 48h):")
        lines.append(f"    P95:     {int(mem_metrics.get('p95', 0))}Mi")
        lines.append(f"    Average: {int(mem_metrics.get('average', 0))}Mi")
        lines.append(f"    Std Dev: {int(mem_metrics.get('std_dev', 0))}Mi")
        lines.append(f"    Range:   {int(mem_metrics.get('min', 0))}Mi - {int(mem_metrics.get('max', 0))}Mi")
        lines.append("")

    # Recommendations
    recs = analysis.get("recommendations", {})

    if recs.get("cpu"):
        lines.append(f"  CPU Recommendation: {recs['cpu'].get('type', 'N/A')}")
        if recs['cpu'].get('action'):
            lines.append(f"    {recs['cpu'].get('action')}")

    if recs.get("memory"):
        lines.append(f"  Memory Recommendation: {recs['memory'].get('type', 'N/A')}")
        if recs['memory'].get('action'):
            lines.append(f"    {recs['memory'].get('action')}")

    if recs.get("hpa", {}).get("enabled"):
        lines.append(f"  HPA Recommendation: ENABLE")

    # Flags
    flags = analysis.get("flags", [])
    if flags:
        lines.append("")
        lines.append(f"  Flags:")
        for flag in flags:
            lines.append(f"    {flag}")

    return lines


def generate_summary_stats(analysis_results: Dict[str, Dict]) -> List[str]:
    """Generate summary statistics"""

    lines = ["SUMMARY STATISTICS:"]
    lines.append("-" * 80)

    increase_count = sum(1 for a in analysis_results.values()
                         if a.get("recommendations", {}).get("cpu", {}).get("type") == "INCREASE_REQUEST"
                         or a.get("recommendations", {}).get("memory", {}).get("type") == "INCREASE_REQUEST")

    decrease_count = sum(1 for a in analysis_results.values()
                         if a.get("recommendations", {}).get("cpu", {}).get("type") == "DECREASE_REQUEST"
                         or a.get("recommendations", {}).get("memory", {}).get("type") == "DECREASE_REQUEST")

    hpa_count = sum(1 for a in analysis_results.values()
                    if a.get("recommendations", {}).get("hpa", {}).get("enabled"))

    critical_count = sum(1 for a in analysis_results.values()
                         if a.get("recommendations", {}).get("cpu", {}).get("priority") == "CRITICAL"
                         or a.get("recommendations", {}).get("memory", {}).get("priority") == "CRITICAL")

    lines.append(f"Total deployments analyzed: {len(analysis_results)}")
    lines.append(f"Deployments needing more resources: {increase_count}")
    lines.append(f"Deployments that are over-provisioned: {decrease_count}")
    lines.append(f"Deployments recommended for HPA: {hpa_count}")
    lines.append(f"Deployments at critical risk: {critical_count}")

    return lines


if __name__ == "__main__":
    # Example usage
    logging.basicConfig(level=logging.INFO)

    from datetime import datetime, timedelta

    # Mock analysis results
    analysis_results = {
        "payments-api": {
            "pod_count": 3,
            "clusters": ["prod-green"],
            "status": "⚠️ NEEDS ATTENTION",
            "current_spec": {
                "request": "500m CPU / 512Mi Memory",
                "limit": "2000m CPU / 2Gi Memory"
            },
            "metrics": {
                "cpu": {"p95": 800, "average": 600, "std_dev": 150, "min": 400, "max": 950},
                "memory": {"p95": 1200, "average": 1100, "std_dev": 80, "min": 1000, "max": 1350}
            },
            "recommendations": {
                "cpu": {"type": "INCREASE_REQUEST", "priority": "HIGH", "action": "Increase CPU request to 1600m", "reason": "P95 exceeds request"},
                "memory": {"type": "INCREASE_REQUEST", "priority": "CRITICAL", "action": "Increase memory request to 2400Mi", "reason": "P95 exceeds limit"},
                "hpa": {"enabled": False}
            },
            "flags": []
        }
    }

    start = datetime.utcnow() - timedelta(hours=48)
    end = datetime.utcnow()

    report = generate_report(
        namespace="settlements",
        clusters=["prod-green"],
        hours=48,
        analysis_results=analysis_results,
        start_time=start,
        end_time=end
    )

    print(report)
