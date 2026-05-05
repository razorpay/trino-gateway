#!/usr/bin/env python3
"""
Verify that the repository belongs to Razorpay by checking git remote.
"""
import subprocess
import sys


def is_razorpay_repo(repo_path='.'):
    """Check if git remote URL contains razorpay"""
    try:
        result = subprocess.run(
            ['git', 'remote', 'get-url', 'origin'],
            cwd=repo_path,
            capture_output=True,
            text=True,
            check=True
        )
        remote_url = result.stdout.strip().lower()
        return 'razorpay' in remote_url
    except subprocess.CalledProcessError:
        return False


def main():
    repo_path = sys.argv[1] if len(sys.argv) > 1 else '.'

    if is_razorpay_repo(repo_path):
        print("true")
        return 0
    else:
        print("false")
        return 1


if __name__ == '__main__':
    sys.exit(main())
