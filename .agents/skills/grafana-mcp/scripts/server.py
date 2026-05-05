#!/usr/bin/env python3
"""Grafana MCP Server — multi-cell PromQL access via SAML session cookies."""

import asyncio
import logging
import os
import subprocess
import sys
from typing import Any, Dict, List, Optional

from mcp.server.fastmcp import FastMCP

from auth import save_cookie, validate_cookie, CELL_URLS
from client import GrafanaClient

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("grafana-mcp")

mcp = FastMCP("grafana")

LOGIN_SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "login.py")


@mcp.tool(name="login")
def login(cell: str = "all") -> Dict[str, Any]:
    """Login to Grafana via SAML SSO. Opens a browser — complete the login there.

    This is the recommended way to authenticate. A Chrome window opens,
    you complete your normal SSO login, and the session cookie is captured automatically.

    Args:
        cell: Which cell to login to — "in", "sg", "us", or "all" (default: "all")
    """
    try:
        result = subprocess.run(
            [sys.executable, LOGIN_SCRIPT, cell],
            capture_output=True, text=True, timeout=180,
        )
        output = result.stdout.strip()
        if result.returncode == 0:
            if cell == "all":
                return {"status": "ok", "message": output, "cells": list(CELL_URLS.keys())}
            validation = validate_cookie(cell)
            return {"status": "ok", "cell": cell, "user": validation.get("user"), "message": output}
        return {"status": "error", "message": output, "stderr": result.stderr.strip()}
    except subprocess.TimeoutExpired:
        return {"status": "timeout", "error": "Login timed out after 3 minutes"}
    except Exception as e:
        return {"error": str(e)}


@mcp.tool(name="save_session")
def save_session(cell: str, cookie: str) -> Dict[str, Any]:
    """Fallback: manually save a grafana_session cookie for a cell.

    Use the 'login' tool instead for seamless SAML auth.
    Only use this if the browser-based login doesn't work.

    Args:
        cell: Which cell — "in", "sg", or "us"
        cookie: The grafana_session cookie value from your browser
    """
    try:
        save_cookie(cell, cookie)
        result = validate_cookie(cell)
        if result["valid"]:
            return {"status": "saved", "cell": cell, "user": result["user"]}
        return {"status": "saved_but_invalid", "cell": cell, "error": result["error"]}
    except ValueError as e:
        return {"error": str(e)}


@mcp.tool(name="check_auth")
def check_auth(cell: Optional[str] = None) -> Dict[str, Any]:
    """Check if saved session cookies are still valid.

    Args:
        cell: Check a specific cell ("in", "sg", "us"), or omit to check all
    """
    if cell:
        return validate_cookie(cell)
    return {c: validate_cookie(c) for c in CELL_URLS}


@mcp.tool(name="list_datasources")
def list_datasources(cell: str) -> Dict[str, Any]:
    """List Prometheus-compatible datasources on a Grafana cell.

    Args:
        cell: Which cell — "in", "sg", or "us"
    """
    try:
        client = GrafanaClient(cell)
        return {"cell": cell, "datasources": client.list_datasources()}
    except Exception as e:
        return {"error": str(e)}


@mcp.tool(name="query_instant")
def query_instant(
    cell: str, datasource_id: int, expr: str, time: Optional[str] = None
) -> Dict[str, Any]:
    """Run an instant PromQL query against a Grafana cell.

    Args:
        cell: Which cell — "in", "sg", or "us"
        datasource_id: Numeric datasource ID (from list_datasources)
        expr: PromQL expression (e.g. "up", "rate(http_requests_total[5m])")
        time: Evaluation timestamp (RFC3339 or Unix). Defaults to now.
    """
    try:
        client = GrafanaClient(cell)
        result = client.query_instant(datasource_id, expr, time)
        return {"cell": cell, "status": result.get("status"), "data": result.get("data")}
    except Exception as e:
        return {"error": str(e)}


@mcp.tool(name="query_range")
def query_range(
    cell: str, datasource_id: int, expr: str, start: str, end: str, step: str = "60s"
) -> Dict[str, Any]:
    """Run a range PromQL query against a Grafana cell.

    Args:
        cell: Which cell — "in", "sg", or "us"
        datasource_id: Numeric datasource ID (from list_datasources)
        expr: PromQL expression
        start: Start time (RFC3339 like "2024-01-01T00:00:00Z", Unix timestamp, or relative like "now-1h")
        end: End time (same formats as start, or "now")
        step: Query resolution step (e.g. "60s", "5m", "1h"). Default "60s".
    """
    try:
        client = GrafanaClient(cell)
        result = client.query_range(datasource_id, expr, start, end, step)
        return {"cell": cell, "status": result.get("status"), "data": result.get("data")}
    except Exception as e:
        return {"error": str(e)}


@mcp.tool(name="list_metrics")
def list_metrics(cell: str, datasource_id: int, match: Optional[str] = None) -> Dict[str, Any]:
    """Discover available metric names on a Prometheus datasource.

    Args:
        cell: Which cell — "in", "sg", or "us"
        datasource_id: Numeric datasource ID (from list_datasources)
        match: Optional PromQL series selector to filter (e.g. "{job='api-server'}")
    """
    try:
        client = GrafanaClient(cell)
        names = client.list_metric_names(datasource_id, match)
        return {"cell": cell, "count": len(names), "metrics": names}
    except Exception as e:
        return {"error": str(e)}


@mcp.tool(name="search_dashboards")
def search_dashboards(cell: str, query: str = "", limit: int = 25) -> Dict[str, Any]:
    """Search dashboards by name on a Grafana cell.

    Args:
        cell: Which cell — "in", "sg", or "us"
        query: Search term (partial match on dashboard title)
        limit: Max results (default 25)
    """
    try:
        client = GrafanaClient(cell)
        return {"cell": cell, "dashboards": client.search_dashboards(query, limit)}
    except Exception as e:
        return {"error": str(e)}


@mcp.tool(name="get_dashboard")
def get_dashboard(cell: str, uid: str) -> Dict[str, Any]:
    """Get dashboard details including panels and their PromQL queries.

    Args:
        cell: Which cell — "in", "sg", or "us"
        uid: Dashboard UID (from the URL or search_dashboards)
    """
    try:
        client = GrafanaClient(cell)
        return {"cell": cell, "dashboard": client.get_dashboard(uid)}
    except Exception as e:
        return {"error": str(e)}


if __name__ == "__main__":
    logger.info("Starting Grafana MCP server (stdio)...")
    mcp.run(transport="stdio")
