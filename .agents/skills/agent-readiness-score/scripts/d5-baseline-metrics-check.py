#!/usr/bin/env python3
import argparse
import json
import os
import re
from pathlib import Path

TEXT_EXTENSIONS = {
    '.go', '.py', '.js', '.ts', '.tsx', '.jsx', '.java', '.kt', '.scala', '.rb', '.php', '.cs', '.rs',
    '.yaml', '.yml', '.json', '.toml', '.ini', '.conf', '.txt', '.md', '.proto', '.sh', '.xml'
}

EXCLUDED_DIRS = {
    '.git', 'node_modules', 'vendor', 'dist', 'build', 'target', '.next', '.turbo', '.idea', '.vscode'
}

SURFACES = {
    'http': {
        'display_name': 'HTTP',
        'detection_patterns': [
            r'\bgin\.Default\(',
            r'\bgin\.New\(',
            r'\becho\.New\(',
            r'\bfiber\.New\(',
            r'\bhttp\.Handle(Func)?\(',
            r'\bhttp\.NewServeMux\(',
            r'\bListenAndServe\(',
            r'\bHandleFunc\(',
            r'\brouter\.(GET|POST|PUT|DELETE|PATCH|Any)\(',
            r'\bServeHTTP\(',
        ],
        'required_metrics': [
            'http_requests_total',
            'http_responses_total',
            'http_durations_ms_histogram_bucket',
            'http_request_duration_ms_bucket',
        ],
    },
    'grpc': {
        'display_name': 'gRPC',
        'detection_patterns': [
            r'\bgrpc\.NewServer\(',
            r'\bgrpcserver\b',
            r'Register[A-Za-z0-9_]+Server\(',
        ],
        'required_metrics': [
            'server_requests_total',
            'grpc_server_handling_seconds_bucket',
            'grpc_server_handled_total',
            'server_error_response_total',
        ],
        'satisfied_by_libraries': [
            {
                'name': 'github.com/razorpay/goutils/grpcserver',
                'import_pattern': r'github\.com/razorpay/goutils/grpcserver',
                'extra_patterns': [
                    r'grpc_prometheus',
                    r'go-grpc-prometheus',
                    r'grpcprometheus',
                    r'EnableHandlingTimeHistogram\(',
                ],
            }
        ],
    },
    'worker': {
        'display_name': 'worker',
        'detection_patterns': [
            r'github\.com/razorpay/goutils/worker',
            r'\bworker\.(New|Register|Start|Process)',
            r'\bconsumer\.(Register|Start)',
            r'\bqueue\b',
            r'\bjob handler\b',
        ],
        'required_metrics': [
            'job_processed_count',
            'job_processing_time_bucket',
            'invalid_message_count',
        ],
        'satisfied_by_libraries': [
            {
                'name': 'github.com/razorpay/goutils/worker',
                'import_pattern': r'github\.com/razorpay/goutils/worker',
            }
        ],
    },
    'egress': {
        'display_name': 'egress',
        'detection_patterns': [
            r'github\.com/razorpay/goutils/request/httpclient',
            r'\bhttp\.Client\b',
            r'\bhttp\.NewRequest\(',
            r'\bresty\.New\(',
            r'\bNewRequestWithContext\(',
            r'\bDo\(req\)',
        ],
        'required_metrics': [
            'httpclient_http_requests_count',
            'httpclient_http_request_duration_ms_hist_bucket',
        ],
        'satisfied_by_libraries': [
            {
                'name': 'github.com/razorpay/goutils/request/httpclient',
                'import_pattern': r'github\.com/razorpay/goutils/request/httpclient',
            }
        ],
    },
    'outbox': {
        'display_name': 'outbox',
        'detection_patterns': [
            r'github\.com/razorpay/goutils/outbox',
            r'\boutbox\.(New|Start|Process)',
            r'\boutbox_store_',
            r'\boutbox_job_',
        ],
        'required_metrics': [
            'outbox_store_create_job_invoked_total',
            'outbox_store_create_job_invoked_seconds_bucket',
            'outbox_store_update_job_invoked_total',
            'outbox_store_update_job_invoked_seconds_bucket',
            'outbox_store_delete_job_invoked_total',
            'outbox_store_delete_job_invoked_seconds_bucket',
            'outbox_store_find_pending_jobs_invoked_total',
            'outbox_store_find_pending_jobs_invoked_seconds_bucket',
            'outbox_store_find_failed_jobs_invoked_total',
            'outbox_store_find_failed_jobs_invoked_seconds_bucket',
            'outbox_job_handler_processed_count',
            'outbox_job_handler_processing_time_ms_bucket',
            'outbox_age_of_oldest_pending_job_seconds',
            'outbox_store_fetched_pending_jobs_total_bucket',
            'outbox_store_fetched_failed_jobs_total_bucket',
            'outbox_job_process_duration_seconds_bucket',
        ],
        'satisfied_by_libraries': [
            {
                'name': 'github.com/razorpay/goutils/outbox',
                'import_pattern': r'github\.com/razorpay/goutils/outbox',
            }
        ],
    },
}

for surface in SURFACES.values():
    surface['detection_patterns'] = [re.compile(p, re.IGNORECASE) for p in surface['detection_patterns']]
    for rule in surface.get('satisfied_by_libraries', []):
        rule['import_pattern'] = re.compile(rule['import_pattern'])
        rule['extra_patterns'] = [re.compile(p, re.IGNORECASE) for p in rule.get('extra_patterns', [])]


def is_text_file(path: Path) -> bool:
    return path.suffix.lower() in TEXT_EXTENSIONS or path.suffix == ''


def walk(repo: Path):
    files = []
    for root, dirs, names in os.walk(repo):
        dirs[:] = [d for d in dirs if d not in EXCLUDED_DIRS]
        for name in names:
            full = Path(root) / name
            if is_text_file(full):
                files.append(full)
    return files


def safe_read(path: Path):
    try:
        if path.stat().st_size > 1024 * 1024:
            return None
        return path.read_text(encoding='utf-8', errors='ignore')
    except Exception:
        return None


def rel(repo: Path, path: Path) -> str:
    return str(path.relative_to(repo))


def first_match_evidence(repo: Path, files, patterns):
    evidence = []
    for file in files:
        content = safe_read(file)
        if not content:
            continue
        for pattern in patterns:
            match = pattern.search(content)
            if match:
                evidence.append({
                    'file': rel(repo, file),
                    'pattern': pattern.pattern,
                    'match': match.group(0),
                })
                if len(evidence) >= 5:
                    return evidence
    return evidence


def has_any_pattern(content: str, patterns):
    return any(pattern.search(content) for pattern in patterns)


def detect_surfaces(repo: Path, files):
    detected = []
    for name, config in SURFACES.items():
        evidence = first_match_evidence(repo, files, config['detection_patterns'])
        if evidence:
            detected.append({'name': name, 'config': config, 'evidence': evidence})
    return detected


def check_library_satisfaction(repo: Path, surface, files):
    for rule in surface['config'].get('satisfied_by_libraries', []):
        evidence = []
        import_seen = False
        extra_seen = len(rule.get('extra_patterns', [])) == 0
        for file in files:
            content = safe_read(file)
            if not content:
                continue
            if not import_seen and rule['import_pattern'].search(content):
                import_seen = True
                evidence.append(f"{rel(repo, file)} matched {rule['import_pattern'].pattern}")
            if not extra_seen and has_any_pattern(content, rule.get('extra_patterns', [])):
                extra_seen = True
                evidence.append(f"{rel(repo, file)} matched one of library extra patterns")
            if import_seen and extra_seen:
                return {'satisfied': True, 'rule': rule['name'], 'evidence': evidence}
    return {'satisfied': False, 'rule': None, 'evidence': []}


def check_metric_presence(repo: Path, metric_name: str, files):
    evidence = []
    for file in files:
        content = safe_read(file)
        if not content:
            continue
        if metric_name in content:
            evidence.append(rel(repo, file))
            if len(evidence) >= 5:
                break
    return evidence


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--repo', required=True)
    parser.add_argument('--output', required=True)
    args = parser.parse_args()

    repo = Path(args.repo).resolve()
    files = walk(repo)
    detected_surfaces = detect_surfaces(repo, files)

    result = {
        'repo': str(repo),
        'scanned_text_files_count': len(files),
        'detected_surfaces': [],
        'library_satisfied': {},
        'missing_by_surface': {},
        'surface_evidence': {},
        'metric_evidence': {},
        'summary': {
            'detected_surface_count': 0,
            'library_satisfied_count': 0,
            'surfaces_with_gaps_count': 0,
            'no_relevant_surfaces_detected': len(detected_surfaces) == 0,
        },
    }

    for surface in detected_surfaces:
        name = surface['name']
        result['detected_surfaces'].append(name)
        result['surface_evidence'][name] = surface['evidence']

        library_result = check_library_satisfaction(repo, surface, files)
        if library_result['satisfied']:
            result['library_satisfied'][name] = {
                'rule': library_result['rule'],
                'evidence': library_result['evidence'],
            }
            continue

        missing_metrics = []
        metric_evidence = {}
        for metric_name in surface['config']['required_metrics']:
            evidence = check_metric_presence(repo, metric_name, files)
            if evidence:
                metric_evidence[metric_name] = evidence
            else:
                missing_metrics.append(metric_name)

        result['metric_evidence'][name] = metric_evidence
        if missing_metrics:
            result['missing_by_surface'][name] = {
                'display_name': surface['config']['display_name'],
                'missing_metrics': missing_metrics,
            }

    result['summary']['detected_surface_count'] = len(result['detected_surfaces'])
    result['summary']['library_satisfied_count'] = len(result['library_satisfied'])
    result['summary']['surfaces_with_gaps_count'] = len(result['missing_by_surface'])

    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')


if __name__ == '__main__':
    main()
