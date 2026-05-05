#!/usr/bin/env python3
"""
Fetch pod definitions from Kubernetes using Friday MCP or local kubectl

This module queries pod specs, resource requests/limits from K8s clusters.
Tries local kubectl first, then falls back to Friday MCP if available.
"""

import json
import logging
import subprocess
from typing import List, Dict, Any, Optional

logger = logging.getLogger(__name__)


def fetch_pods_from_cluster(namespace: str, cluster: str) -> List[Dict[str, Any]]:
    """
    Fetch all pods in a namespace from a specific K8s cluster

    Uses Friday MCP (remote-friday-mcp-server) to query Kubernetes.

    Args:
        namespace: K8s namespace (e.g., 'settlements', 'payments-api')
        cluster: Cluster name (prod-green or prod-white)

    Returns:
        List of pod definitions with resource specs
    """

    logger.info(f"Fetching pods from {cluster}/{namespace}")

    if cluster not in ["prod-green", "prod-white"]:
        logger.error(f"Unknown cluster: {cluster}")
        return []

    try:
        # kubectl query - uses Friday MCP (remote-friday-mcp-server)
        # MCP Tool: mcp__remote-friday-mcp-server__kubectl_generic
        kubectl_query = f"kubectl get pods -n {namespace} --context={cluster} -o json"

        logger.debug(f"Executing: {kubectl_query}")

        # Call Friday MCP
        pods_data = call_friday_mcp(kubectl_query)

        if not pods_data:
            logger.warning(f"No pods found in {cluster}/{namespace}")
            return []

        # Parse response
        pods = parse_pod_list(pods_data, cluster)
        logger.info(f"Fetched {len(pods)} pods from {cluster}/{namespace}")

        return pods

    except Exception as e:
        logger.error(f"Error fetching pods from {cluster}/{namespace}: {str(e)}")
        return []


def call_friday_mcp(query: str) -> Optional[Dict]:
    """
    Execute kubectl query using local kubectl or Friday MCP fallback

    Tries local kubectl first (if available), then falls back to Friday MCP.

    MCP Tool (fallback): mcp__remote-friday-mcp-server__kubectl_generic

    Args:
        query: kubectl command (e.g., "kubectl get pods -n settlements -o json")

    Returns:
        Parsed JSON response from kubectl
    """

    logger.debug(f"Executing: {query}")

    # Try local kubectl first
    try:
        logger.info("Attempting local kubectl execution...")
        result = subprocess.run(
            query,
            shell=True,
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode == 0:
            try:
                data = json.loads(result.stdout)
                logger.info("Local kubectl: Success")
                return data
            except json.JSONDecodeError as e:
                logger.error(f"Failed to parse kubectl output as JSON: {e}")
                return None
        else:
            logger.warning(f"Local kubectl failed: {result.stderr[:200]}")

    except Exception as e:
        logger.debug(f"Local kubectl not available: {str(e)}")

    # Fallback to Friday MCP via HTTP
    try:
        logger.info("Falling back to Friday MCP...")
        import requests

        # Try Razorpay's Friday MCP endpoint first
        endpoints = [
            "https://friday-mcp.razorpay.com/mcp/tools/kubectl_generic",
            "http://localhost:3000/tools/kubectl_generic"
        ]

        for endpoint in endpoints:
            try:
                logger.debug(f"Trying endpoint: {endpoint}")
                response = requests.post(
                    endpoint,
                    json={"query": query},
                    timeout=30
                )

                if response.status_code == 200:
                    result = response.json()
                    logger.info(f"Friday MCP: Success ({endpoint})")
                    return result
            except Exception as e:
                logger.debug(f"Endpoint failed: {endpoint} - {str(e)}")
                continue

        logger.error("All Friday MCP endpoints failed")
        return None

    except Exception as e:
        logger.error(f"Error calling Friday MCP: {str(e)}")

    logger.error("Could not fetch pods: kubectl and Friday MCP both unavailable")
    logger.info("In Claude Code, Friday MCP tool will be available: mcp__remote-friday-mcp-server__kubectl_generic")
    return None


def parse_pod_list(response: Dict, cluster: str) -> List[Dict[str, Any]]:
    """
    Parse kubectl get pods response

    Args:
        response: kubectl get pods -o json response
        cluster: Cluster name for reference

    Returns:
        List of processed pod definitions
    """

    pods = []

    if not isinstance(response, dict):
        logger.error("Invalid response format from Friday MCP")
        return pods

    items = response.get("items", [])
    logger.info(f"Processing {len(items)} items from kubectl response")

    for item in items:
        try:
            pod = parse_pod_item(item, cluster)
            if pod:
                pods.append(pod)
        except Exception as e:
            pod_name = item.get("metadata", {}).get("name", "unknown")
            logger.warning(f"Error parsing pod {pod_name}: {str(e)}")

    return pods


def parse_pod_item(item: Dict, cluster: str) -> Optional[Dict[str, Any]]:
    """
    Parse a single pod from kubectl response

    Args:
        item: Single pod item from kubectl get pods response
        cluster: Cluster name

    Returns:
        Processed pod definition
    """

    metadata = item.get("metadata", {})
    spec = item.get("spec", {})
    status = item.get("status", {})

    pod_name = metadata.get("name")
    namespace = metadata.get("namespace")

    if not pod_name or not namespace:
        return None

    # Extract labels for deployment detection
    labels = metadata.get("labels", {})

    pod_info = {
        "name": pod_name,
        "namespace": namespace,
        "cluster": cluster,
        "labels": labels,
        "spec": {
            "containers": [],
            "restart_policy": spec.get("restartPolicy", "Always"),
            "termination_grace_period": spec.get("terminationGracePeriodSeconds", 30)
        },
        "status": {
            "phase": status.get("phase"),  # Running, Pending, Failed, etc.
            "pod_ip": status.get("podIP"),
            "host_ip": status.get("hostIP"),
            "start_time": status.get("startTime"),
            "container_statuses": status.get("containerStatuses", [])
        }
    }

    # Parse containers for resource specs
    containers = spec.get("containers", [])
    for container in containers:
        container_info = {
            "name": container.get("name"),
            "image": container.get("image"),
            "resources": container.get("resources", {})
        }
        pod_info["spec"]["containers"].append(container_info)

    return pod_info


def filter_pods(
    pods: List[Dict[str, Any]],
    exclude_failed: bool = False,
    min_age_hours: int = 1,
    only_ready: bool = True
) -> List[Dict[str, Any]]:
    """
    Filter pods for analysis - includes all container-based pods with ready containers

    Includes:
    - Running pods with all containers ready
    - Completed cronjobs (they executed and consumed resources)

    Excludes:
    - Pods with unready containers
    - Pods that are too new (< min_age_hours)
    - Pending/Failed pods (can't get metrics from these)

    Args:
        pods: List of all pods
        exclude_failed: Exclude Pending/Failed pods (default: False - they're excluded anyway via readiness check)
        min_age_hours: Minimum pod age in hours to have metrics (default: 1)
        only_ready: Only include pods where all containers are ready/completed (default: True)

    Returns:
        Filtered list of analyzable pods (running + completed cronjobs)
    """

    from datetime import datetime, timedelta

    filtered = []
    now = datetime.utcnow()
    min_age = timedelta(hours=min_age_hours)
    excluded_count = {
        "not_ready": 0,
        "too_new": 0,
    }

    for pod in pods:
        pod_name = pod.get("name", "unknown")

        # Check container readiness (applies to both Running and Completed)
        if only_ready:
            container_statuses = pod["status"].get("container_statuses", [])
            if not container_statuses:
                logger.debug(f"Excluding {pod_name}: no container status found")
                excluded_count["not_ready"] += 1
                continue

            # All containers must have ready: true (for Running) or completed (for Completed)
            # For Completed cronjobs, containers may show ready: false but state: {"terminated": {...}}
            all_ready = all(cs.get("ready", False) or cs.get("state", {}).get("terminated") for cs in container_statuses)
            if not all_ready:
                ready_count = sum(1 for cs in container_statuses if cs.get("ready", False))
                logger.debug(f"Excluding {pod_name}: {ready_count}/{len(container_statuses)} containers ready")
                excluded_count["not_ready"] += 1
                continue

        # Check pod age
        start_time_str = pod["status"]["start_time"]
        if start_time_str:
            try:
                start_time = datetime.fromisoformat(start_time_str.replace('Z', '+00:00'))
                age = now - start_time
                if age < min_age:
                    logger.debug(f"Excluding {pod_name}: too new (age={age})")
                    excluded_count["too_new"] += 1
                    continue
            except Exception as e:
                logger.warning(f"Could not parse start_time for {pod_name}: {e}")

        filtered.append(pod)

    logger.info(
        f"Filtered: {len(filtered)} analyzable pods remain (from {len(pods)} original) - "
        f"Excluded: not_ready={excluded_count['not_ready']}, "
        f"too_new={excluded_count['too_new']}"
    )
    return filtered


def categorize_pods(pods: List[Dict[str, Any]]) -> Dict[str, List[str]]:
    """
    Categorize pods by type based on naming patterns and characteristics

    Categories:
    - cronjob: pods from cronjob deployments
    - worker: pods with 'worker' in name (background tasks)
    - web: pods without worker/cronjob designation (main services)
    - other: unclassified pods

    Args:
        pods: List of pod dictionaries

    Returns:
        Dictionary with pod names grouped by category
    """

    categories = {
        "cronjob": [],
        "worker": [],
        "web": [],
        "other": []
    }

    for pod in pods:
        pod_name = pod.get("name", "").lower()

        # Identify category based on pod name
        if "cronjob" in pod_name:
            categories["cronjob"].append(pod.get("name", "unknown"))
        elif "worker" in pod_name:
            categories["worker"].append(pod.get("name", "unknown"))
        elif any(x in pod_name for x in ["live", "test", "baseline", "canary"]):
            # Main service deployments
            categories["web"].append(pod.get("name", "unknown"))
        else:
            categories["other"].append(pod.get("name", "unknown"))

    return categories


def summarize_pod_categories(pods: List[Dict[str, Any]]) -> str:
    """
    Generate a summary string of pod categories

    Example:
        "350 pods (150 web + 180 workers) + 12 cronjobs = 362 total"

    Args:
        pods: List of pod dictionaries

    Returns:
        Formatted summary string
    """

    categories = categorize_pods(pods)

    # Build summary
    active_pods = len(categories["web"]) + len(categories["worker"]) + len(categories["other"])
    cronjobs = len(categories["cronjob"])
    total = active_pods + cronjobs

    parts = []
    if active_pods > 0:
        active_breakdown = []
        if categories["web"]:
            active_breakdown.append(f"{len(categories['web'])} web")
        if categories["worker"]:
            active_breakdown.append(f"{len(categories['worker'])} workers")
        if categories["other"]:
            active_breakdown.append(f"{len(categories['other'])} other")

        parts.append(f"{active_pods} running ({' + '.join(active_breakdown)})")

    if cronjobs > 0:
        parts.append(f"{cronjobs} cronjobs")

    summary = " + ".join(parts)
    if total > 0:
        summary += f" = {total} total"

    return summary


if __name__ == "__main__":
    # Example usage
    logging.basicConfig(level=logging.INFO)

    pods = fetch_pods_from_cluster("settlements", "prod-green")
    print(json.dumps(pods, indent=2, default=str))
