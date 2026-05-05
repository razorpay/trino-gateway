#!/usr/bin/env python3
"""
Analysis logic - Generate recommendations from metrics

Compares actual usage (P95) vs allocated resources and generates recommendations.
"""

import logging
from typing import Dict, Any, Optional, List

logger = logging.getLogger(__name__)


def analyze_deployment_metrics(
    deployment_name: str,
    pod_count: int,
    resource_spec: Dict[str, Optional[int]],
    metrics: Dict[str, Any],
    clusters: List[str]
) -> Dict[str, Any]:
    """
    Analyze a deployment's resource allocation vs usage

    Args:
        deployment_name: Deployment name
        pod_count: Number of pods in deployment
        resource_spec: Current resource requests/limits (from pod spec)
        metrics: Usage metrics (from Prometheus)
        clusters: List of clusters this deployment runs on

    Returns:
        Analysis with recommendations
    """

    analysis = {
        "deployment": deployment_name,
        "clusters": clusters,
        "pod_count": pod_count,
        "status": "OK",
        "current_spec": format_resources(resource_spec),
        "metrics": {},
        "recommendations": {},
        "flags": []
    }

    # Extract aggregated metrics
    aggregated = metrics.get("aggregated", {})

    if not aggregated.get("cpu_m") and not aggregated.get("memory_mi"):
        analysis["status"] = "NO_METRICS"
        analysis["flags"].append("No Prometheus metrics available - pods may be too new or not running")
        return analysis

    # Analyze CPU
    cpu_p95 = aggregated.get("cpu_m", {}).get("p95")
    if cpu_p95 is not None:
        cpu_analysis = recommend_cpu(
            deployment_name=deployment_name,
            spec=resource_spec,
            metrics=aggregated.get("cpu_m", {})
        )
        analysis["metrics"]["cpu"] = aggregated.get("cpu_m", {})
        analysis["recommendations"]["cpu"] = cpu_analysis

        # Update status
        if cpu_analysis["priority"] in ["CRITICAL", "HIGH"]:
            analysis["status"] = "⚠️ NEEDS ATTENTION"

    # Analyze Memory
    mem_p95 = aggregated.get("memory_mi", {}).get("p95")
    if mem_p95 is not None:
        mem_analysis = recommend_memory(
            deployment_name=deployment_name,
            spec=resource_spec,
            metrics=aggregated.get("memory_mi", {})
        )
        analysis["metrics"]["memory"] = aggregated.get("memory_mi", {})
        analysis["recommendations"]["memory"] = mem_analysis

        # Update status
        if mem_analysis["priority"] in ["CRITICAL", "HIGH"]:
            analysis["status"] = "⚠️ NEEDS ATTENTION"

    # Analyze variance and recommend HPA
    hpa_analysis = recommend_hpa(
        aggregated.get("cpu_m", {}),
        aggregated.get("memory_mi", {}),
        resource_spec
    )
    analysis["recommendations"]["hpa"] = hpa_analysis

    # Flag any issues
    check_for_issues(analysis)

    # If no critical issues, mark as OK
    if analysis["status"] == "OK" and not analysis["flags"]:
        analysis["status"] = "✅ HEALTHY"

    logger.info(f"{deployment_name}: {analysis['status']}")

    return analysis


def recommend_cpu(
    deployment_name: str,
    spec: Dict[str, Optional[int]],
    metrics: Dict[str, Any]
) -> Dict[str, Any]:
    """
    Generate CPU recommendation

    Returns: type, new_request, new_limit, priority, reason
    """

    recommendation = {
        "type": "OK",
        "new_request_m": None,
        "new_limit_m": None,
        "priority": "NONE",
        "reason": "CPU allocation is healthy",
        "action": None
    }

    p95 = metrics.get("p95")
    requested = spec.get("requested_cpu_m")
    limit = spec.get("limit_cpu_m")

    if p95 is None or requested is None:
        return recommendation

    variance = calculate_variance(metrics.get("std_dev", 0), metrics.get("average", p95))

    # CHECK 0: Missing limit - always recommend setting limit
    if limit is None:
        recommendation["type"] = "SET_LIMIT"
        recommendation["reason"] = f"No CPU limit set - Pod has unlimited CPU access"
        recommendation["new_limit_m"] = int(p95 * 1.5)  # 1.5x P95 for safety
        recommendation["action"] = f"Set CPU limit to {int(p95 * 1.5)}m"
        recommendation["priority"] = "MEDIUM"
        return recommendation

    # CHECK 1: Risk - P95 exceeds request
    if p95 > requested:
        recommendation["type"] = "INCREASE_REQUEST"
        recommendation["reason"] = f"P95 usage ({int(p95)}m) exceeds request ({requested}m)"
        recommendation["new_request_m"] = int(p95 * 2)  # 2x for safety margin
        recommendation["action"] = f"Increase CPU request to {int(p95 * 2)}m"

        # Check if exceeds limit too
        if limit and p95 > limit:
            recommendation["priority"] = "CRITICAL"
            recommendation["reason"] += f" and limit ({limit}m) - Pod at risk of throttling!"
            recommendation["new_limit_m"] = int(p95 * 1.5)  # Also increase limit
            recommendation["action"] += f" and limit to {int(p95 * 1.5)}m"
        else:
            recommendation["priority"] = "HIGH"

    # CHECK 2: Over-provisioning
    elif requested and p95 < (requested * 0.5):
        recommendation["type"] = "DECREASE_REQUEST"
        recommendation["reason"] = f"P95 usage ({int(p95)}m) is only {(p95/requested)*100:.0f}% of request ({requested}m)"
        recommendation["new_request_m"] = int(p95 * 1.5)  # 1.5x for safety margin
        recommendation["action"] = f"Decrease CPU request to {int(p95 * 1.5)}m"
        recommendation["priority"] = "LOW"

    # CHECK 3: High variance - consider HPA
    elif variance > 30 and limit and p95 < (limit * 0.8):
        # Only suggest if not at risk
        recommendation["type"] = "MONITOR"
        recommendation["reason"] = f"High variance ({variance:.0f}%) indicates bursty workload"
        recommendation["priority"] = "LOW"

    return recommendation


def recommend_memory(
    deployment_name: str,
    spec: Dict[str, Optional[int]],
    metrics: Dict[str, Any]
) -> Dict[str, Any]:
    """
    Generate memory recommendation

    Same logic as CPU but for memory
    """

    recommendation = {
        "type": "OK",
        "new_request_mi": None,
        "new_limit_mi": None,
        "priority": "NONE",
        "reason": "Memory allocation is healthy",
        "action": None
    }

    p95 = metrics.get("p95")
    requested = spec.get("requested_memory_mi")
    limit = spec.get("limit_memory_mi")

    if p95 is None or requested is None:
        return recommendation

    variance = calculate_variance(metrics.get("std_dev", 0), metrics.get("average", p95))

    # CHECK 0: Missing limit - always recommend setting limit
    if limit is None:
        recommendation["type"] = "SET_LIMIT"
        recommendation["reason"] = f"No memory limit set - Pod could consume all node memory"
        recommendation["new_limit_mi"] = int(p95 * 1.5)  # 1.5x P95 for safety
        recommendation["action"] = f"Set memory limit to {int(p95 * 1.5)}Mi"
        recommendation["priority"] = "MEDIUM"
        return recommendation

    # CHECK 1: Risk - P95 exceeds request
    if p95 > requested:
        recommendation["type"] = "INCREASE_REQUEST"
        recommendation["reason"] = f"P95 usage ({int(p95)}Mi) exceeds request ({requested}Mi)"
        recommendation["new_request_mi"] = int(p95 * 2)  # 2x for safety margin
        recommendation["action"] = f"Increase memory request to {int(p95 * 2)}Mi"

        # Check if exceeds limit (critical for memory - causes OOMKill)
        if limit and p95 > limit:
            recommendation["priority"] = "CRITICAL"
            recommendation["reason"] += f" and limit ({limit}Mi) - Pod will be OOM killed!"
            recommendation["new_limit_mi"] = int(p95 * 1.5)  # Also increase limit
            recommendation["action"] += f" and limit to {int(p95 * 1.5)}Mi"
        else:
            recommendation["priority"] = "HIGH"

    # CHECK 2: Over-provisioning
    elif requested and p95 < (requested * 0.5):
        recommendation["type"] = "DECREASE_REQUEST"
        recommendation["reason"] = f"P95 usage ({int(p95)}Mi) is only {(p95/requested)*100:.0f}% of request ({requested}Mi)"
        recommendation["new_request_mi"] = int(p95 * 1.5)  # 1.5x for safety margin
        recommendation["action"] = f"Decrease memory request to {int(p95 * 1.5)}Mi"
        recommendation["priority"] = "LOW"

    # CHECK 3: High variance
    elif variance > 30 and limit and p95 < (limit * 0.8):
        recommendation["type"] = "MONITOR"
        recommendation["reason"] = f"High variance ({variance:.0f}%) indicates bursty workload"
        recommendation["priority"] = "LOW"

    return recommendation


def recommend_hpa(
    cpu_metrics: Dict[str, Any],
    mem_metrics: Dict[str, Any],
    resource_spec: Dict[str, Optional[int]]
) -> Dict[str, Any]:
    """
    Recommend if HPA should be enabled
    """

    recommendation = {
        "enabled": False,
        "reason": "Workload is stable",
        "metric_for_scaling": None,
        "target_utilization": 70,
        "suggested_min_replicas": 1,
        "suggested_max_replicas": 1
    }

    cpu_variance = calculate_variance(cpu_metrics.get("std_dev", 0), cpu_metrics.get("average", 0))
    mem_variance = calculate_variance(mem_metrics.get("std_dev", 0), mem_metrics.get("average", 0))

    avg_variance = (cpu_variance + mem_variance) / 2 if (cpu_variance and mem_variance) else (cpu_variance or mem_variance)

    # Recommend HPA if high variance
    if avg_variance > 30:
        recommendation["enabled"] = True
        recommendation["reason"] = f"High variance ({avg_variance:.0f}%) indicates bursty/variable workload"

        # Determine which metric to scale on
        if cpu_variance > mem_variance:
            recommendation["metric_for_scaling"] = "cpu"
        else:
            recommendation["metric_for_scaling"] = "memory"

        # Suggest reasonable replica range
        # Could be enhanced with actual pod requirements
        recommendation["suggested_min_replicas"] = 1
        recommendation["suggested_max_replicas"] = 3  # Conservative default
        recommendation["target_utilization"] = 70

    return recommendation


def check_for_issues(analysis: Dict[str, Any]) -> None:
    """
    Check for configuration issues and add flags
    """

    spec = analysis.get("current_spec", {})
    cpu_req = spec.get("requested_cpu_m")
    cpu_lim = spec.get("limit_cpu_m")
    mem_req = spec.get("requested_memory_mi")
    mem_lim = spec.get("limit_memory_mi")

    # Check for misconfigurations
    if cpu_req and cpu_lim and cpu_req > cpu_lim:
        analysis["flags"].append(f"⚠️ CPU request ({cpu_req}m) > limit ({cpu_lim}m) - Invalid configuration!")

    if mem_req and mem_lim and mem_req > mem_lim:
        analysis["flags"].append(f"⚠️ Memory request ({mem_req}Mi) > limit ({mem_lim}Mi) - Invalid configuration!")

    if not cpu_req or not cpu_lim:
        analysis["flags"].append("⚠️ Missing CPU limits - Pod has unlimited CPU access")

    if not mem_req or not mem_lim:
        analysis["flags"].append("⚠️ Missing memory limits - Pod could consume all node memory")


def calculate_variance(std_dev: float, average: float) -> float:
    """
    Calculate coefficient of variation as percentage

    Returns variance percentage (0-100+)
    """

    if average == 0:
        return 0

    return (std_dev / average) * 100


def format_resources(spec: Dict[str, Optional[int]]) -> Dict[str, str]:
    """
    Format resource values as human-readable strings
    """

    return {
        "request": f"{spec.get('requested_cpu_m', 0)}m CPU / {spec.get('requested_memory_mi', 0)}Mi Memory",
        "limit": f"{spec.get('limit_cpu_m', 0)}m CPU / {spec.get('limit_memory_mi', 0)}Mi Memory"
    }


if __name__ == "__main__":
    # Example usage
    logging.basicConfig(level=logging.INFO)

    # Mock data
    resource_spec = {
        "requested_cpu_m": 500,
        "requested_memory_mi": 512,
        "limit_cpu_m": 2000,
        "limit_memory_mi": 2048
    }

    metrics = {
        "aggregated": {
            "cpu_m": {
                "p95": 800,
                "average": 600,
                "std_dev": 150
            },
            "memory_mi": {
                "p95": 1200,
                "average": 1100,
                "std_dev": 80
            }
        }
    }

    result = analyze_deployment_metrics(
        deployment_name="payments-api",
        pod_count=3,
        resource_spec=resource_spec,
        metrics=metrics,
        clusters=["prod-green"]
    )

    import json
    print(json.dumps(result, indent=2))
