#!/usr/bin/env python3
"""
Fetch Prometheus metrics from Grafana MCP

Queries CPU and memory usage for pods over a time range.
"""

import json
import logging
from datetime import datetime, timedelta
from typing import List, Dict, Any, Optional
from statistics import stdev, mean

logger = logging.getLogger(__name__)


def fetch_metrics_from_prometheus(
    namespace: str,
    pod_names: List[str],
    start_time: datetime,
    end_time: datetime
) -> Dict[str, Any]:
    """
    Fetch CPU and memory metrics for pods from Prometheus

    Args:
        namespace: K8s namespace
        pod_names: List of pod names
        start_time: Start time for metrics
        end_time: End time for metrics

    Returns:
        Dictionary containing CPU and memory metrics
    """

    logger.info(f"Fetching Prometheus metrics for {len(pod_names)} pods")

    metrics = {
        "namespace": namespace,
        "pod_names": pod_names,
        "time_range": {
            "start": start_time.isoformat(),
            "end": end_time.isoformat()
        },
        "pods": {}
    }

    # Fetch CPU metrics
    cpu_data = fetch_cpu_metrics(namespace, pod_names, start_time, end_time)
    if cpu_data:
        for pod_name, cpu_values in cpu_data.items():
            if pod_name not in metrics["pods"]:
                metrics["pods"][pod_name] = {}
            metrics["pods"][pod_name]["cpu"] = cpu_values

    # Fetch memory metrics
    mem_data = fetch_memory_metrics(namespace, pod_names, start_time, end_time)
    if mem_data:
        for pod_name, mem_values in mem_data.items():
            if pod_name not in metrics["pods"]:
                metrics["pods"][pod_name] = {}
            metrics["pods"][pod_name]["memory"] = mem_values

    # Aggregate to deployment level
    metrics["aggregated"] = aggregate_metrics(metrics["pods"])

    logger.info(f"Fetched metrics for {len(metrics['pods'])} pods")
    return metrics


def fetch_cpu_metrics(
    namespace: str,
    pod_names: List[str],
    start_time: datetime,
    end_time: datetime
) -> Optional[Dict[str, List[float]]]:
    """
    Fetch CPU usage metrics from Prometheus

    Query:
    sum(rate(container_cpu_usage_seconds_total{namespace="{namespace}",container!="POD"}[2m])) by (pod)
    """

    logger.info("Fetching CPU metrics...")

    # Build query
    promql = f"""
    sum(rate(container_cpu_usage_seconds_total{{
      namespace="{namespace}",
      container!="POD"
    }}[2m])) by (pod)
    """

    try:
        # Call Grafana MCP range_query
        # This would normally call: grafana_mcp.range_query(datasource_uid="6ZssswRnk", ...)
        response = call_grafana_mcp_range_query(
            datasource_uid="6ZssswRnk",  # Primary Prometheus
            query=promql,
            start_time=start_time,
            end_time=end_time,
            step=300  # 5 minute intervals
        )

        if not response:
            logger.warning("No CPU metrics returned from Prometheus")
            return None

        # Parse response
        cpu_data = parse_prometheus_response(response)

        # Convert to millicores
        for pod_name in cpu_data:
            cpu_data[pod_name] = [value * 1000 for value in cpu_data[pod_name]]

        logger.info(f"Got CPU metrics for {len(cpu_data)} pods")
        return cpu_data

    except Exception as e:
        logger.error(f"Error fetching CPU metrics: {str(e)}")
        return None


def fetch_memory_metrics(
    namespace: str,
    pod_names: List[str],
    start_time: datetime,
    end_time: datetime
) -> Optional[Dict[str, List[float]]]:
    """
    Fetch memory usage metrics from Prometheus

    Query:
    sum(container_memory_working_set_bytes{namespace="{namespace}"}) by (pod)
    """

    logger.info("Fetching memory metrics...")

    # Build query
    promql = f"""
    sum(container_memory_working_set_bytes{{
      namespace="{namespace}"
    }}) by (pod)
    """

    try:
        # Call Grafana MCP range_query
        response = call_grafana_mcp_range_query(
            datasource_uid="6ZssswRnk",  # Primary Prometheus
            query=promql,
            start_time=start_time,
            end_time=end_time,
            step=300  # 5 minute intervals
        )

        if not response:
            logger.warning("No memory metrics returned from Prometheus")
            return None

        # Parse response
        mem_data = parse_prometheus_response(response)

        # Convert bytes to MiB
        for pod_name in mem_data:
            mem_data[pod_name] = [value / (1024 ** 2) for value in mem_data[pod_name]]

        logger.info(f"Got memory metrics for {len(mem_data)} pods")
        return mem_data

    except Exception as e:
        logger.error(f"Error fetching memory metrics: {str(e)}")
        return None


def call_grafana_mcp_range_query(
    datasource_uid: str,
    query: str,
    start_time: datetime,
    end_time: datetime,
    step: int = 300
) -> Optional[Dict]:
    """
    Call Grafana MCP range_query tool

    Args:
        datasource_uid: Prometheus datasource UID
        query: PromQL query
        start_time: Start timestamp
        end_time: End timestamp
        step: Query step in seconds

    Returns:
        Prometheus response
    """

    logger.debug(f"Calling Grafana MCP: {query[:100]}...")

    try:
        # Grafana MCP is registered in .mcp.json and exposes:
        # - query_prometheus(datasource_uid: str, query: str, start: int, end: int, step: int) -> dict
        # - get_dashboard_panel_queries(dashboard_uid: str) -> dict
        # - list_datasources() -> dict
        #
        # In Claude Code with MCP support, this is called as:
        #   response = mcp_client.call_tool("mcp__grafana__query_prometheus", {
        #       "datasource_uid": "6ZssswRnk",
        #       "query": promql,
        #       "start": int(start_time.timestamp()),
        #       "end": int(end_time.timestamp()),
        #       "step": 300
        #   })
        #
        # For direct HTTP to Grafana API:
        import requests
        import os

        api_key = os.getenv("GRAFANA_API_KEY")

        # Try multiple Grafana endpoints
        endpoints = [
            ("https://grafana-mcp.razorpay.com/api/datasources/proxy/uid/{}/api/v1/query_range".format(datasource_uid), {"Authorization": f"Bearer {api_key}"} if api_key else None),
            ("https://vajra.razorpay.com/api/datasources/proxy/uid/{}/api/v1/query_range".format(datasource_uid), {"Authorization": f"Bearer {api_key}"} if api_key else None),
            ("http://localhost:3000/tools/mcp__grafana__query_prometheus", None)
        ]

        response = None
        for endpoint, headers in endpoints:
            try:
                logger.debug(f"Trying Grafana endpoint: {endpoint}")

                if headers:
                    response = requests.post(
                        endpoint,
                        headers=headers,
                        json={
                            "query": query,
                            "start": int(start_time.timestamp()),
                            "end": int(end_time.timestamp()),
                            "step": step
                        },
                        timeout=60
                    )
                else:
                    response = requests.post(
                        endpoint,
                        json={
                            "datasource_uid": datasource_uid,
                            "query": query,
                            "start": int(start_time.timestamp()),
                            "end": int(end_time.timestamp()),
                            "step": step
                        },
                        timeout=60
                    )

                if response.status_code == 200:
                    logger.info(f"Grafana endpoint success: {endpoint}")
                    break
                else:
                    logger.debug(f"Grafana endpoint failed with status {response.status_code}: {endpoint}")
            except Exception as e:
                logger.debug(f"Grafana endpoint error: {endpoint} - {str(e)}")
                continue

        if response.status_code == 200:
            result = response.json()
            logger.info(f"Grafana MCP success: Got Prometheus response")
            return result
        else:
            logger.error(f"Grafana MCP error: status {response.status_code}")
            logger.error(f"Response: {response.text[:200]}")
            return None

    except Exception as e:
        logger.error(f"Error calling Grafana MCP: {str(e)}")
        logger.info("Note: Grafana MCP should be configured in .mcp.json or GRAFANA_API_KEY env var")
        return None


def parse_prometheus_response(response: Dict) -> Dict[str, List[float]]:
    """
    Parse Prometheus range_query response

    Expected format:
    {
      "status": "success",
      "data": {
        "resultType": "matrix",
        "result": [
          {
            "metric": {"pod": "pod-name"},
            "values": [[timestamp, "value"], ...]
          }
        ]
      }
    }
    """

    data_dict = {}

    if response.get("status") != "success":
        logger.error(f"Prometheus query failed: {response.get('error', 'unknown error')}")
        return data_dict

    results = response.get("data", {}).get("result", [])

    for result in results:
        metric = result.get("metric", {})
        pod_name = metric.get("pod")

        if not pod_name:
            logger.warning("Result missing pod label, skipping")
            continue

        values = result.get("values", [])

        # Extract numeric values (convert from strings)
        numeric_values = []
        for timestamp, value_str in values:
            try:
                value = float(value_str)
                numeric_values.append(value)
            except (ValueError, TypeError):
                logger.warning(f"Invalid value for {pod_name}: {value_str}")

        if numeric_values:
            data_dict[pod_name] = numeric_values
            logger.debug(f"Pod {pod_name}: {len(numeric_values)} data points")

    return data_dict


def aggregate_metrics(pods_metrics: Dict[str, Dict]) -> Dict[str, Any]:
    """
    Aggregate metrics across pods

    Returns P95, average, std_dev, min, max for deployment
    """

    aggregated = {
        "cpu_m": {},
        "memory_mi": {}
    }

    # Collect all CPU and memory values
    all_cpu_values = []
    all_mem_values = []

    for pod_name, metrics in pods_metrics.items():
        if "cpu" in metrics:
            all_cpu_values.extend(metrics["cpu"])
        if "memory" in metrics:
            all_mem_values.extend(metrics["memory"])

    # Calculate statistics
    if all_cpu_values:
        aggregated["cpu_m"] = calculate_percentiles(all_cpu_values)
    if all_mem_values:
        aggregated["memory_mi"] = calculate_percentiles(all_mem_values)

    return aggregated


def calculate_percentiles(values: List[float]) -> Dict[str, float]:
    """
    Calculate percentiles and statistics for a list of values
    """

    if not values:
        return {}

    sorted_values = sorted(values)
    n = len(sorted_values)

    return {
        "p50": sorted_values[int(n * 0.50)],
        "p75": sorted_values[int(n * 0.75)],
        "p90": sorted_values[int(n * 0.90)],
        "p95": sorted_values[int(n * 0.95)],
        "p99": sorted_values[int(n * 0.99)],
        "min": min(values),
        "max": max(values),
        "average": mean(values),
        "std_dev": stdev(values) if len(values) > 1 else 0
    }


if __name__ == "__main__":
    # Example usage
    logging.basicConfig(level=logging.INFO)

    start = datetime.utcnow() - timedelta(hours=48)
    end = datetime.utcnow()

    metrics = fetch_metrics_from_prometheus(
        namespace="settlements",
        pod_names=["payments-api-live-1", "payments-api-live-2"],
        start_time=start,
        end_time=end
    )

    print(json.dumps(metrics, indent=2, default=str))
