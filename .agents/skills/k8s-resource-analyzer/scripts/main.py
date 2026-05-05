#!/usr/bin/env python3
"""
K8s Resource Analyzer - Main Orchestrator

Analyzes Kubernetes pod resource allocation vs actual usage and generates recommendations.
"""

import json
import sys
import argparse
import os
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional
import logging

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

from fetch_pods import fetch_pods_from_cluster, filter_pods, summarize_pod_categories
from fetch_metrics import fetch_metrics_from_prometheus
from analyze import analyze_deployment_metrics
from report_generator import generate_report


def check_mcps_available() -> tuple[bool, str]:
    """
    Check if required MCPs are available

    Returns:
        (is_available, message)
    """
    # Check for Friday MCP
    try:
        import requests
        response = requests.get("http://localhost:3000/health", timeout=2)
        friday_available = response.status_code == 200
    except:
        friday_available = False

    # Log MCP status
    if friday_available:
        logger.info("✅ MCPs detected (local Friday MCP running)")
        return True, "MCPs are available"
    else:
        logger.info("⚠️  MCPs not detected locally")
        logger.info("In Claude Code, MCPs will be automatically available")
        return False, "MCPs not available in standalone mode"


def parse_inputs(namespace: str, cluster: Optional[str] = None, hours: int = 48) -> Dict[str, Any]:
    """Parse and validate user inputs"""

    # Validate namespace
    if not namespace or not isinstance(namespace, str):
        raise ValueError("Namespace must be a non-empty string")

    # Normalize and default clusters
    if cluster is None:
        clusters = ["prod-green", "prod-white"]
    else:
        # Support comma-separated clusters
        cluster_list = [c.strip() for c in cluster.split(",")]
        clusters = []
        for c in cluster_list:
            # Normalize cluster name (remove -eks suffix if present)
            cluster_name = c.replace("-eks", "")
            if cluster_name in ["prod-green", "prod-white"]:
                if cluster_name not in clusters:  # Avoid duplicates
                    clusters.append(cluster_name)
            else:
                raise ValueError(f"Invalid cluster: {c}. Must be 'prod-green' or 'prod-white'")

    if not clusters:
        raise ValueError("At least one valid cluster must be specified")

    # Validate time range
    if not (1 <= hours <= 720):  # Max 30 days
        raise ValueError("Time range must be between 1 and 720 hours")

    return {
        "namespace": namespace.strip(),
        "clusters": clusters,
        "hours": hours,
        "start_time": datetime.utcnow() - timedelta(hours=hours),
        "end_time": datetime.utcnow()
    }


def analyze_namespace(
    namespace: str,
    cluster: Optional[str] = None,
    hours: int = 48
) -> Dict[str, Any]:
    """
    Main analysis function

    Args:
        namespace: K8s namespace to analyze
        cluster: Optional cluster (prod-green or prod-white), defaults to both
        hours: Number of hours to analyze (default 48)

    Returns:
        Dictionary containing analysis results
    """

    try:
        logger.info(f"Starting K8s resource analysis for namespace: {namespace}")

        # Check MCPs availability
        mcps_available, mcp_message = check_mcps_available()
        logger.info(f"MCP Status: {mcp_message}")

        # Parse and validate inputs
        config = parse_inputs(namespace, cluster, hours)
        logger.info(f"Configuration: {json.dumps({k: v for k, v in config.items() if k != 'start_time' and k != 'end_time'}, indent=2)}")

        # Fetch pod definitions from all clusters
        all_pods = {}
        all_pods_by_deployment = {}
        total_pods_all = []

        for cluster_name in config["clusters"]:
            logger.info(f"Fetching pod definitions from {cluster_name}...")
            pods = fetch_pods_from_cluster(config["namespace"], cluster_name)

            if not pods:
                logger.warning(f"No pods found in {cluster_name}/{config['namespace']}")
                continue

            # Filter to only analyzable pods (running + completed cronjobs)
            pods = filter_pods(pods, exclude_failed=False, only_ready=True)

            if not pods:
                logger.warning(f"No analyzable pods found in {cluster_name}/{config['namespace']} after filtering")
                continue

            all_pods[cluster_name] = pods
            total_pods_all.extend(pods)
            pod_summary = summarize_pod_categories(pods)
            logger.info(f"Found {pod_summary} in {cluster_name}")

            # Group by deployment
            for pod in pods:
                dep_name = extract_deployment_name(pod["name"])
                if dep_name not in all_pods_by_deployment:
                    all_pods_by_deployment[dep_name] = {
                        "clusters": set(),
                        "pods": [],
                        "spec": {}
                    }
                all_pods_by_deployment[dep_name]["clusters"].add(cluster_name)
                all_pods_by_deployment[dep_name]["pods"].append(pod)
                if not all_pods_by_deployment[dep_name]["spec"]:
                    all_pods_by_deployment[dep_name]["spec"] = extract_resource_spec(pod)

        if not all_pods_by_deployment:
            logger.warning("No pods found in any cluster")
            return {
                "status": "no_data",
                "message": f"No pods found in namespace {config['namespace']}",
                "namespace": config["namespace"],
                "clusters": config["clusters"]
            }

        logger.info(f"Grouped {len(all_pods_by_deployment)} deployments")

        # Fetch metrics for each deployment
        logger.info(f"Fetching Prometheus metrics for {config['hours']}h...")
        all_metrics = {}

        for dep_name, dep_data in all_pods_by_deployment.items():
            pod_names = [p["name"] for p in dep_data["pods"]]

            # Fetch metrics
            metrics = fetch_metrics_from_prometheus(
                namespace=config["namespace"],
                pod_names=pod_names,
                start_time=config["start_time"],
                end_time=config["end_time"]
            )

            if metrics:
                all_metrics[dep_name] = metrics
                logger.info(f"  {dep_name}: Got metrics for {len(metrics.get('pods', {}))} pods")
            else:
                logger.warning(f"  {dep_name}: No metrics available")

        # Analyze each deployment
        logger.info("Analyzing deployments...")
        analysis_results = {}

        for dep_name, dep_data in all_pods_by_deployment.items():
            metrics = all_metrics.get(dep_name, {})

            analysis = analyze_deployment_metrics(
                deployment_name=dep_name,
                pod_count=len(dep_data["pods"]),
                resource_spec=dep_data["spec"],
                metrics=metrics,
                clusters=list(dep_data["clusters"])
            )

            analysis_results[dep_name] = analysis
            logger.info(f"  {dep_name}: {analysis.get('status', 'unknown')}")

        # Generate report
        logger.info("Generating report...")
        pod_summary = summarize_pod_categories(total_pods_all) if total_pods_all else "0 pods"
        report = generate_report(
            namespace=config["namespace"],
            clusters=config["clusters"],
            hours=config["hours"],
            analysis_results=analysis_results,
            start_time=config["start_time"],
            end_time=config["end_time"],
            pod_summary=pod_summary
        )

        logger.info("Analysis complete")

        return {
            "status": "success",
            "namespace": config["namespace"],
            "clusters": config["clusters"],
            "hours": config["hours"],
            "pod_summary": pod_summary,
            "deployments_analyzed": len(analysis_results),
            "report": report,
            "details": analysis_results
        }

    except Exception as e:
        logger.error(f"Error during analysis: {str(e)}", exc_info=True)
        return {
            "status": "error",
            "error": str(e),
            "namespace": namespace,
            "clusters": [cluster] if cluster else ["prod-green", "prod-white"]
        }


def extract_deployment_name(pod_name: str) -> str:
    """
    Extract deployment name from pod name

    Examples:
        payments-api-live-1 → payments-api
        ledger-service-abc123 → ledger-service
        fraud-detector → fraud-detector
    """
    # Remove trailing hash/id
    parts = pod_name.rsplit("-", 1)
    if len(parts) == 2 and (len(parts[1]) == 5 or parts[1].isdigit()):
        # Looks like pod hash or replica number
        return parts[0]
    return pod_name


def extract_resource_spec(pod: Dict) -> Dict[str, Any]:
    """Extract resource requests and limits from pod spec"""

    spec = {
        "requested_cpu_m": None,
        "requested_memory_mi": None,
        "limit_cpu_m": None,
        "limit_memory_mi": None
    }

    containers = pod.get("spec", {}).get("containers", [])

    if not containers:
        return spec

    # Get resources from first container (most representative)
    # Could be enhanced to sum all containers
    container = containers[0]
    resources = container.get("resources", {})

    # Parse requests
    requests = resources.get("requests", {})
    spec["requested_cpu_m"] = parse_resource_value(requests.get("cpu", "100m"), "cpu")
    spec["requested_memory_mi"] = parse_resource_value(requests.get("memory", "128Mi"), "memory")

    # Parse limits
    limits = resources.get("limits", {})
    spec["limit_cpu_m"] = parse_resource_value(limits.get("cpu"), "cpu")
    spec["limit_memory_mi"] = parse_resource_value(limits.get("memory"), "memory")

    return spec


def parse_resource_value(value: Optional[str], resource_type: str) -> Optional[int]:
    """
    Parse K8s resource values to standard units

    CPU: convert to millicores (m)
    Memory: convert to mebibytes (Mi)
    """

    if not value:
        return None

    value = str(value).strip()

    if resource_type == "cpu":
        if value.endswith("m"):
            return int(value[:-1])
        elif value.endswith("n"):
            return int(value[:-1]) // 1_000_000
        else:
            # Assume cores
            return int(float(value) * 1000)

    elif resource_type == "memory":
        multipliers = {
            "Ki": 1,
            "Mi": 1,
            "Gi": 1024,
            "Ti": 1024 * 1024,
            "K": 1,
            "M": 1,
            "G": 1024,
            "T": 1024 * 1024,
        }

        for suffix, mult in multipliers.items():
            if value.endswith(suffix):
                num = float(value[:-len(suffix)])
                return int(num * mult)

        # Default to Mi
        return int(float(value))

    return None


if __name__ == "__main__":
    # Parse command line arguments with support for environment variables
    parser = argparse.ArgumentParser(
        description="Analyze Kubernetes pod resource allocation vs actual usage",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python main.py settlements
  python main.py settlements --cluster prod-green
  python main.py settlements --cluster prod-green,prod-white --hours 24

Environment Variables:
  K8S_NAMESPACE     - Kubernetes namespace to analyze (required if not passed as argument)
  K8S_CLUSTER       - Cluster to analyze (optional, defaults to prod-green,prod-white)
  K8S_TIME_HOURS    - Time window in hours (optional, defaults to 48)
        """
    )

    parser.add_argument(
        "namespace",
        nargs="?",
        help="Kubernetes namespace to analyze"
    )
    parser.add_argument(
        "--cluster", "-c",
        help="Cluster to analyze (prod-green or prod-white, comma-separated for multiple)"
    )
    parser.add_argument(
        "--hours", "-t",
        type=int,
        help="Time window in hours (1-720, default: 48)"
    )

    args = parser.parse_args()

    # Get values from arguments or environment variables
    namespace = args.namespace or os.getenv("K8S_NAMESPACE")
    cluster = args.cluster or os.getenv("K8S_CLUSTER")
    hours = args.hours or (int(os.getenv("K8S_TIME_HOURS", 48)) if os.getenv("K8S_TIME_HOURS") else 48)

    # Validate required namespace
    if not namespace:
        parser.print_help()
        print("\nError: namespace is required (pass as argument or K8S_NAMESPACE env var)")
        sys.exit(1)

    try:
        result = analyze_namespace(namespace, cluster, hours)
        print(json.dumps(result, indent=2, default=str))
    except ValueError as e:
        logger.error(f"Validation error: {str(e)}")
        print(json.dumps({"status": "error", "error": str(e)}, indent=2))
        sys.exit(1)
    except Exception as e:
        logger.error(f"Unexpected error: {str(e)}", exc_info=True)
        print(json.dumps({"status": "error", "error": str(e)}, indent=2))
        sys.exit(1)
