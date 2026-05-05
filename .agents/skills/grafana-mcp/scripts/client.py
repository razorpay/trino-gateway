import os
import subprocess
import sys
from typing import Any, Dict, List, Optional

import requests

from auth import get_base_url, get_cookie, validate_cookie

LOGIN_SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "login.py")


def _auto_login(cell: str) -> None:
    """Trigger browser-based SAML login for a cell."""
    subprocess.run(
        [sys.executable, LOGIN_SCRIPT, cell],
        timeout=180,
    )


def _ensure_auth(cell: str) -> str:
    """Return a valid cookie for the cell, triggering login if needed."""
    cookie = get_cookie(cell)
    if cookie:
        result = validate_cookie(cell)
        if result["valid"]:
            return cookie

    _auto_login(cell)

    cookie = get_cookie(cell)
    if not cookie:
        raise ValueError(f"Login failed for cell '{cell}' — no session cookie after SAML flow")
    return cookie


class GrafanaClient:
    def __init__(self, cell: str):
        self.cell = cell.lower().strip()
        self.base_url = get_base_url(self.cell)
        cookie = _ensure_auth(self.cell)
        self.session = requests.Session()
        self.session.headers.update({"Cookie": f"grafana_session={cookie}"})
        self.session.verify = False

    def _get(self, path: str, params: Optional[Dict] = None) -> Any:
        resp = self.session.get(f"{self.base_url}{path}", params=params, timeout=30)
        if resp.status_code == 401:
            self._reauth()
            resp = self.session.get(f"{self.base_url}{path}", params=params, timeout=30)
        resp.raise_for_status()
        return resp.json()

    def _post(self, path: str, json_body: Dict) -> Any:
        resp = self.session.post(f"{self.base_url}{path}", json=json_body, timeout=30)
        if resp.status_code == 401:
            self._reauth()
            resp = self.session.post(f"{self.base_url}{path}", json=json_body, timeout=30)
        resp.raise_for_status()
        return resp.json()

    def _reauth(self) -> None:
        _auto_login(self.cell)
        cookie = get_cookie(self.cell)
        if cookie:
            self.session.headers.update({"Cookie": f"grafana_session={cookie}"})

    def list_datasources(self) -> List[Dict]:
        ds_list = self._get("/api/datasources")
        return [
            {"id": ds["id"], "uid": ds["uid"], "name": ds["name"], "type": ds["type"], "url": ds.get("url", "")}
            for ds in ds_list
            if ds.get("type") in ("prometheus", "thanos", "cortex", "mimir")
        ]

    def query_instant(self, datasource_id: int, expr: str, time: Optional[str] = None) -> Dict:
        params = {"query": expr}
        if time:
            params["time"] = time
        return self._get(f"/api/datasources/proxy/{datasource_id}/api/v1/query", params=params)

    def query_range(
        self, datasource_id: int, expr: str, start: str, end: str, step: str = "60s"
    ) -> Dict:
        params = {"query": expr, "start": start, "end": end, "step": step}
        return self._get(f"/api/datasources/proxy/{datasource_id}/api/v1/query_range", params=params)

    def list_metric_names(self, datasource_id: int, match: Optional[str] = None) -> List[str]:
        params = {}
        if match:
            params["match[]"] = match
        result = self._get(f"/api/datasources/proxy/{datasource_id}/api/v1/label/__name__/values", params=params)
        return result.get("data", [])

    def search_dashboards(self, query: str = "", limit: int = 25) -> List[Dict]:
        params = {"query": query, "limit": limit, "type": "dash-db"}
        results = self._get("/api/search", params=params)
        return [
            {"uid": d["uid"], "title": d["title"], "url": d.get("url", ""), "tags": d.get("tags", [])}
            for d in results
        ]

    def get_dashboard(self, uid: str) -> Dict:
        data = self._get(f"/api/dashboards/uid/{uid}")
        dashboard = data.get("dashboard", {})
        panels = []
        for panel in dashboard.get("panels", []):
            p = {"id": panel.get("id"), "title": panel.get("title", ""), "type": panel.get("type", "")}
            targets = panel.get("targets", [])
            if targets:
                p["queries"] = [t.get("expr", "") for t in targets if t.get("expr")]
            panels.append(p)
        return {
            "uid": dashboard.get("uid"),
            "title": dashboard.get("title"),
            "tags": dashboard.get("tags", []),
            "panels": panels,
        }
