#!/usr/bin/env python3
"""
Extract API routes from flow documentation.

Parses flow markdown files to find route definitions in various formats:
- POST /v1/offers
- GET /api/v2/transactions/{id}
- Route: /offers/validate
- Endpoint: POST /v1/settlements
"""

import sys
import re
from pathlib import Path
from typing import List, Set


def extract_routes_from_content(content: str) -> Set[str]:
    """
    Extract unique routes from markdown content.

    Args:
        content: Markdown file content

    Returns:
        Set of unique routes found
    """
    routes = set()

    # Pattern 1: HTTP method followed by route
    # Example: POST /v1/offers, GET /api/transactions/{id}
    method_route_pattern = r'\b(GET|POST|PUT|PATCH|DELETE)\s+(/[^\s\)\]`]+)'
    for match in re.finditer(method_route_pattern, content, re.IGNORECASE):
        method = match.group(1).upper()
        route = match.group(2)
        routes.add(f"{method} {route}")

    # Pattern 2: Route: or Endpoint: prefix
    # Example: Route: /offers/validate
    prefix_pattern = r'(?:Route|Endpoint|Path):\s*([A-Z]+\s+)?(/[^\s\)\]`]+)'
    for match in re.finditer(prefix_pattern, content, re.IGNORECASE):
        method = match.group(1).strip().upper() if match.group(1) else ""
        route = match.group(2)
        if method:
            routes.add(f"{method} {route}")
        else:
            routes.add(route)

    # Pattern 3: Code blocks with route definitions
    # Example: `POST /v1/offers`
    code_route_pattern = r'`((?:GET|POST|PUT|PATCH|DELETE)\s+/[^`]+)`'
    for match in re.finditer(code_route_pattern, content, re.IGNORECASE):
        routes.add(match.group(1))

    return routes


def extract_routes_from_file(filepath: Path) -> Set[str]:
    """
    Extract routes from a single markdown file.

    Args:
        filepath: Path to markdown file

    Returns:
        Set of unique routes
    """
    try:
        content = filepath.read_text()
        return extract_routes_from_content(content)
    except Exception as e:
        print(f"Warning: Could not read {filepath}: {e}", file=sys.stderr)
        return set()


def extract_routes_from_directory(directory: Path) -> Set[str]:
    """
    Extract routes from all markdown files in directory recursively.

    Args:
        directory: Path to directory

    Returns:
        Set of all unique routes found
    """
    all_routes = set()

    for md_file in directory.rglob("*.md"):
        routes = extract_routes_from_file(md_file)
        if routes:
            all_routes.update(routes)

    return all_routes


def main():
    if len(sys.argv) != 2:
        print("Usage: extract_routes.py <file_or_directory>", file=sys.stderr)
        print("Example: extract_routes.py domain/offer/flows.md", file=sys.stderr)
        print("Example: extract_routes.py .claude/skills/offers-engine-skill/", file=sys.stderr)
        sys.exit(1)

    path = Path(sys.argv[1])

    if not path.exists():
        print(f"Error: Path does not exist: {path}", file=sys.stderr)
        sys.exit(1)

    if path.is_file():
        routes = extract_routes_from_file(path)
    elif path.is_directory():
        routes = extract_routes_from_directory(path)
    else:
        print(f"Error: Path is neither file nor directory: {path}", file=sys.stderr)
        sys.exit(1)

    if routes:
        for route in sorted(routes):
            print(route)
    else:
        print("No routes found", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
