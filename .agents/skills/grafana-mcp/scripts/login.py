#!/usr/bin/env python3
"""Browser-based SAML login for Grafana. Opens a real browser, user completes SSO, cookie is captured."""

import asyncio
import sys

from playwright.async_api import async_playwright

from auth import save_cookie, get_base_url, validate_cookie, CELL_URLS


def _is_post_login_url(url: str, base_url: str) -> bool:
    """True when the browser has landed back on Grafana after SAML, not on /login."""
    if not url.startswith(base_url):
        return False
    path = url[len(base_url):]
    return "/login" not in path


async def login(cell: str) -> dict:
    base_url = get_base_url(cell)

    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=False, channel="chrome")
        context = await browser.new_context()
        page = await context.new_page()

        await page.goto(f"{base_url}/login")
        print(f"Browser opened — complete SAML login for {cell.upper()} ({base_url})")
        print("Waiting for SSO to complete (up to 3 minutes)...")

        try:
            await page.wait_for_url(
                lambda url: _is_post_login_url(url, base_url),
                timeout=180_000,
            )
            await page.wait_for_load_state("networkidle", timeout=10_000)
        except Exception:
            pass

        cookies = await context.cookies(base_url)
        await browser.close()

    grafana_cookie = next((c["value"] for c in cookies if c["name"] == "grafana_session"), None)
    if not grafana_cookie:
        grafana_cookie = next((c["value"] for c in cookies if "session" in c["name"].lower()), None)

    if not grafana_cookie:
        return {"status": "error", "cell": cell, "error": "No session cookie found after login"}

    save_cookie(cell, grafana_cookie)
    result = validate_cookie(cell)
    if result["valid"]:
        return {"status": "ok", "cell": cell, "user": result["user"]}
    return {"status": "cookie_saved_but_invalid", "cell": cell, "error": result.get("error")}


async def login_all() -> list:
    results = []
    for cell in CELL_URLS:
        print(f"\n--- Logging into {cell.upper()} ---")
        results.append(await login(cell))
    return results


if __name__ == "__main__":
    cell = sys.argv[1] if len(sys.argv) > 1 else None
    if cell and cell != "all":
        result = asyncio.run(login(cell))
    elif cell == "all":
        results = asyncio.run(login_all())
        for r in results:
            status = "ok" if r.get("status") == "ok" else "FAILED"
            print(f"  {r['cell'].upper()}: {status} — {r.get('user', r.get('error', ''))}")
        sys.exit(0)
    else:
        print(f"Usage: python login.py <cell|all>")
        print(f"  Cells: {', '.join(CELL_URLS.keys())}")
        print(f"  Example: python login.py in")
        print(f"  Example: python login.py all")
        sys.exit(1)

    if result["status"] == "ok":
        print(f"Logged in to {cell.upper()} as {result['user']}")
    else:
        print(f"Login failed: {result.get('error')}")
        sys.exit(1)
