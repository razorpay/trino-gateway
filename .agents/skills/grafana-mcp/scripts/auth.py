import json
import os
from typing import Optional, Dict

import requests

COOKIE_FILE = os.path.expanduser("~/.grafana-mcp/cookies.json")

CELL_URLS: Dict[str, str] = {
    "in": "https://vajra.razorpay.com",
    "sg": "https://grafana-sg.razorpay.com",
    "us": "https://grafana-us.razorpay.com",
}


def get_base_url(cell: str) -> str:
    cell = cell.lower().strip()
    if cell not in CELL_URLS:
        raise ValueError(f"Unknown cell '{cell}'. Must be one of: {', '.join(CELL_URLS)}")
    return CELL_URLS[cell]


def _load_cookies() -> Dict[str, str]:
    if not os.path.exists(COOKIE_FILE):
        return {}
    with open(COOKIE_FILE, "r") as f:
        return json.load(f)


def _save_cookies(data: Dict[str, str]) -> None:
    os.makedirs(os.path.dirname(COOKIE_FILE), exist_ok=True)
    with open(COOKIE_FILE, "w") as f:
        json.dump(data, f, indent=2)


def save_cookie(cell: str, cookie_value: str) -> None:
    cell = cell.lower().strip()
    if cell not in CELL_URLS:
        raise ValueError(f"Unknown cell '{cell}'. Must be one of: {', '.join(CELL_URLS)}")
    cookies = _load_cookies()
    cookies[cell] = cookie_value.strip()
    _save_cookies(cookies)


def get_cookie(cell: str) -> Optional[str]:
    cell = cell.lower().strip()
    return _load_cookies().get(cell)


def validate_cookie(cell: str) -> Dict:
    cell = cell.lower().strip()
    cookie = get_cookie(cell)
    if not cookie:
        return {"valid": False, "cell": cell, "error": "No cookie saved for this cell"}

    base_url = get_base_url(cell)
    try:
        resp = requests.get(
            f"{base_url}/api/user",
            headers={"Cookie": f"grafana_session={cookie}"},
            timeout=10,
            verify=False,
        )
        if resp.status_code == 200:
            user = resp.json()
            return {"valid": True, "cell": cell, "user": user.get("login", "unknown")}
        return {"valid": False, "cell": cell, "error": f"HTTP {resp.status_code}"}
    except requests.RequestException as e:
        return {"valid": False, "cell": cell, "error": str(e)}
