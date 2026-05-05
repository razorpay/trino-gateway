#!/usr/bin/env python3
"""
Parse commit SHA from Kubernetes image tag.

Handles common image tag formats:
- app:v1.2.3-abc123def -> abc123def
- app:abc123def -> abc123def
- registry.com/app:v1.2.3-abc123def -> abc123def
- app:latest-abc123def -> abc123def
"""

import sys
import re


def parse_commit_from_image(image_tag: str) -> str:
    """
    Extract commit SHA from image tag.

    Args:
        image_tag: Full image tag (e.g., "registry.com/app:v1.2.3-abc123")

    Returns:
        Commit SHA or empty string if not found
    """
    # Remove registry prefix if present (everything before last /)
    if '/' in image_tag:
        image_tag = image_tag.split('/')[-1]

    # Split on : to get tag part
    if ':' not in image_tag:
        return ""

    tag = image_tag.split(':')[-1]

    # Try to extract commit SHA (7-40 hex chars)
    # Pattern: ends with commit SHA, optionally after version or 'latest'
    patterns = [
        r'-([a-f0-9]{7,40})$',  # v1.2.3-abc123def
        r'^([a-f0-9]{7,40})$',  # abc123def
        r'latest-([a-f0-9]{7,40})$',  # latest-abc123def
    ]

    for pattern in patterns:
        match = re.search(pattern, tag)
        if match:
            return match.group(1)

    return ""


def main():
    if len(sys.argv) != 2:
        print("Usage: parse_image_tag.py <image_tag>", file=sys.stderr)
        print("Example: parse_image_tag.py 'registry.com/app:v1.2.3-abc123'", file=sys.stderr)
        sys.exit(1)

    image_tag = sys.argv[1]
    commit = parse_commit_from_image(image_tag)

    if commit:
        print(commit)
    else:
        print(f"Error: Could not extract commit SHA from: {image_tag}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
