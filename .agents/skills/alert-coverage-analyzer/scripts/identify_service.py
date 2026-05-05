#!/usr/bin/env python3
"""
Identify service name from repository files.
Checks go.mod, package.json, and common metric files.
"""
import os
import sys
import json
import re


def identify_from_go_mod(repo_path):
    """Extract module name from go.mod"""
    go_mod = os.path.join(repo_path, "go.mod")
    if os.path.exists(go_mod):
        with open(go_mod, 'r') as f:
            first_line = f.readline()
            match = re.match(r'module\s+(\S+)', first_line)
            if match:
                return match.group(1).split('/')[-1]
    return None


def identify_from_package_json(repo_path):
    """Extract name from package.json"""
    pkg_json = os.path.join(repo_path, "package.json")
    if os.path.exists(pkg_json):
        with open(pkg_json, 'r') as f:
            data = json.load(f)
            return data.get('name', '').replace('@razorpay/', '')
    return None


def identify_from_metrics(repo_path):
    """Extract namespace from metric files"""
    patterns = [
        ('app/metric/metric.go', r'Namespace\s*=\s*"(\w+)"'),
        ('src/metrics.js', r'namespace:\s*["\'](\w+)["\']'),
        ('internal/metrics/metrics.go', r'const\s+Namespace\s*=\s*"(\w+)"'),
    ]

    for file_path, pattern in patterns:
        full_path = os.path.join(repo_path, file_path)
        if os.path.exists(full_path):
            with open(full_path, 'r') as f:
                content = f.read()
                match = re.search(pattern, content)
                if match:
                    return match.group(1)
    return None


def main():
    repo_path = sys.argv[1] if len(sys.argv) > 1 else '.'

    # Try multiple methods
    service_name = (
        identify_from_go_mod(repo_path) or
        identify_from_package_json(repo_path) or
        identify_from_metrics(repo_path)
    )

    if service_name:
        print(service_name)
        return 0
    else:
        print("unknown", file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
