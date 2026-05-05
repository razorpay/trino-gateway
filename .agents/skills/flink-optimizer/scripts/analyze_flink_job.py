#!/usr/bin/env python3
"""
Flink Job Analyzer - Analyzes running Flink jobs for optimization opportunities

This script:
1. Fetches job details from Flink REST API
2. Analyzes YAML jobspec for configuration issues
3. Identifies optimization opportunities (backpressure, data skew, dead code, etc.)
4. Generates actionable recommendations

Usage:
    python analyze_flink_job.py --cluster-url https://flink.de.razorpay.com \
                                 --job-id <job-id> \
                                 --jobspec-path configs/local/jobs/sample-jobspec.yaml
"""

import argparse
import json
import sys
from typing import Dict, List, Any, Optional, Set
import requests
import yaml


class FlinkJobAnalyzer:
    def __init__(self, cluster_url: str, job_id: Optional[str] = None):
        self.cluster_url = cluster_url.rstrip('/')
        self.job_id = job_id
        self.issues = []
        self.recommendations = []

    def fetch_json(self, endpoint: str) -> Dict[str, Any]:
        """Fetch JSON from Flink REST API endpoint"""
        url = f"{self.cluster_url}/{endpoint.lstrip('/')}"
        try:
            response = requests.get(url, timeout=30)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"Error fetching {url}: {e}", file=sys.stderr)
            return {}

    def list_running_jobs(self) -> List[Dict[str, Any]]:
        """List all running jobs from the cluster"""
        data = self.fetch_json('/jobs')
        jobs = data.get('jobs', [])
        running_jobs = [j for j in jobs if j.get('status') == 'RUNNING']
        return running_jobs

    def get_job_details(self, job_id: str) -> Dict[str, Any]:
        """Get detailed information about a specific job"""
        return self.fetch_json(f'/jobs/{job_id}')

    def get_job_metrics(self, job_id: str) -> Dict[str, Any]:
        """Get metrics for a specific job"""
        return self.fetch_json(f'/jobs/{job_id}/metrics')

    def get_job_config(self, job_id: str) -> Dict[str, Any]:
        """Get configuration for a specific job"""
        return self.fetch_json(f'/jobs/{job_id}/config')

    def get_checkpoint_stats(self, job_id: str) -> Dict[str, Any]:
        """Get checkpoint statistics"""
        return self.fetch_json(f'/jobs/{job_id}/checkpoints')

    def get_vertices(self, job_id: str) -> List[Dict[str, Any]]:
        """Get job vertices (operators) from job details"""
        job_details = self.get_job_details(job_id)
        return job_details.get('vertices', [])

    def analyze_backpressure(self, job_id: str) -> List[str]:
        """Analyze backpressure issues from job metrics"""
        issues = []
        vertices = self.get_vertices(job_id)

        for vertex in vertices:
            vertex_id = vertex.get('id')
            vertex_name = vertex.get('name', 'Unknown')
            parallelism = vertex.get('parallelism', 1)

            # Check if parallelism is low for heavy operators
            if parallelism == 1:
                if any(keyword in vertex_name.lower() for keyword in ['aggregat', 'window', 'join', 'reduce']):
                    issues.append(
                        f"Low parallelism (1) for heavy operator '{vertex_name}' (ID: {vertex_id}). "
                        f"Consider increasing parallelism to improve throughput."
                    )

        return issues

    def analyze_checkpoint_performance(self, job_id: str) -> List[str]:
        """Analyze checkpoint performance issues"""
        issues = []
        checkpoint_stats = self.get_checkpoint_stats(job_id)

        if not checkpoint_stats:
            return issues

        # Check for checkpoint failures
        counts = checkpoint_stats.get('counts', {})
        failed = counts.get('failed', 0)
        completed = counts.get('completed', 0)

        if failed > 0 and completed > 0:
            failure_rate = failed / (failed + completed)
            if failure_rate > 0.1:  # More than 10% failure rate
                issues.append(
                    f"High checkpoint failure rate: {failure_rate*100:.1f}% "
                    f"({failed} failed, {completed} completed). "
                    f"Check checkpoint timeout settings and operator state sizes."
                )

        # Check checkpoint duration
        latest = checkpoint_stats.get('latest', {})
        if latest:
            completed_checkpoint = latest.get('completed')
            if completed_checkpoint:
                duration = completed_checkpoint.get('duration', 0)
                if duration > 60000:  # Over 1 minute
                    issues.append(
                        f"Long checkpoint duration: {duration/1000:.1f}s. "
                        f"Consider reducing state size or increasing checkpoint timeout."
                    )

        return issues

    def analyze_jobspec_yaml(self, jobspec_path: str) -> Dict[str, Any]:
        """Analyze the YAML jobspec for configuration issues"""
        try:
            with open(jobspec_path, 'r') as f:
                jobspec = yaml.safe_load(f)
        except Exception as e:
            print(f"Error loading jobspec: {e}", file=sys.stderr)
            return {'issues': [], 'warnings': [], 'dag_info': {}}

        issues = []
        warnings = []

        # Build stream dependency graph
        stream_producers = {}  # stream_name -> operator/source that produces it
        stream_consumers = {}  # stream_name -> list of operators that consume it
        all_streams = set()

        # Collect sources
        sources = jobspec.get('sources', [])
        for source in sources:
            source_name = source.get('name')
            if source_name:
                stream_producers[source_name] = {'type': 'SOURCE', 'config': source}
                all_streams.add(source_name)

        # Collect operators
        operators = jobspec.get('operators', [])
        for operator in operators:
            operator_name = operator.get('name')
            output_stream = operator.get('outputStream')
            input_streams = operator.get('inputStreams', [])

            # Track producer
            if output_stream:
                stream_producers[output_stream] = {
                    'type': 'OPERATOR',
                    'name': operator_name,
                    'operator_type': operator.get('type'),
                    'config': operator
                }
                all_streams.add(output_stream)

            # Track consumers
            for input_stream in input_streams:
                if input_stream not in stream_consumers:
                    stream_consumers[input_stream] = []
                stream_consumers[input_stream].append({
                    'name': operator_name,
                    'type': operator.get('type')
                })
                all_streams.add(input_stream)

        # Check for dead streams (produced but never consumed)
        dead_streams = []
        for stream_name in stream_producers.keys():
            if stream_name not in stream_consumers:
                producer = stream_producers[stream_name]
                if producer['type'] == 'OPERATOR':
                    dead_streams.append({
                        'stream': stream_name,
                        'producer': producer['name'],
                        'operator_type': producer.get('operator_type')
                    })

        if dead_streams:
            for dead in dead_streams:
                issues.append(
                    f"Dead output stream '{dead['stream']}' from operator '{dead['producer']}' "
                    f"(type: {dead['operator_type']}). This stream is produced but never consumed. "
                    f"Remove the operator or wire it to a downstream operator/sink."
                )

        # Check for missing input streams (consumed but never produced)
        missing_streams = []
        for stream_name in stream_consumers.keys():
            if stream_name not in stream_producers:
                consumers = stream_consumers[stream_name]
                missing_streams.append({
                    'stream': stream_name,
                    'consumers': [c['name'] for c in consumers]
                })

        if missing_streams:
            for missing in missing_streams:
                issues.append(
                    f"Missing input stream '{missing['stream']}' required by operators: "
                    f"{', '.join(missing['consumers'])}. No operator or source produces this stream."
                )

        # Check for low parallelism on aggregation operators
        for operator in operators:
            operator_type = operator.get('type', '').upper()
            parallelism = operator.get('parallelism')
            operator_name = operator.get('name')

            if operator_type in ['WINDOW_AGGREGATOR', 'RULE_EVALUATOR', 'RCA_ANALYZER']:
                if parallelism is None:
                    # Will use default job parallelism
                    default_parallelism = jobspec.get('parallelism', 1)
                    if default_parallelism == 1:
                        warnings.append(
                            f"Operator '{operator_name}' (type: {operator_type}) will use default "
                            f"parallelism of 1. Consider setting explicit parallelism > 1 for better throughput."
                        )
                elif parallelism == 1:
                    warnings.append(
                        f"Low parallelism (1) for operator '{operator_name}' (type: {operator_type}). "
                        f"Consider increasing parallelism for better throughput."
                    )

        # Check checkpoint configuration
        checkpointing = jobspec.get('checkpointing', {})
        if checkpointing.get('enabled'):
            interval = checkpointing.get('interval', 60000)
            timeout = checkpointing.get('checkpointTimeout', 300000)

            if interval < 10000:  # Less than 10 seconds
                warnings.append(
                    f"Very frequent checkpoint interval ({interval}ms). "
                    f"This may impact performance. Consider increasing to 30-60 seconds."
                )

            if timeout < interval * 3:
                warnings.append(
                    f"Checkpoint timeout ({timeout}ms) is less than 3x the interval ({interval}ms). "
                    f"This may cause checkpoint failures under load."
                )

        # Check memory configuration
        resources = jobspec.get('resources', {})
        memory = resources.get('memory', '2gb')
        managed_memory = resources.get('managedMemory', '512mb')

        # Parse memory sizes (simple parsing for common formats)
        def parse_memory(mem_str):
            if isinstance(mem_str, int):
                return mem_str
            mem_str = str(mem_str).lower()
            if 'gb' in mem_str:
                return int(mem_str.replace('gb', '')) * 1024
            elif 'mb' in mem_str:
                return int(mem_str.replace('mb', ''))
            return 0

        total_mb = parse_memory(memory)
        managed_mb = parse_memory(managed_memory)

        if total_mb < 2048:  # Less than 2GB
            warnings.append(
                f"Low total memory allocation ({memory}). "
                f"For production workloads, consider at least 4GB."
            )

        if managed_mb < total_mb * 0.2:  # Less than 20% of total
            warnings.append(
                f"Managed memory ({managed_memory}) is less than 20% of total memory ({memory}). "
                f"For state-heavy operators (window aggregations, RocksDB), consider increasing managed memory."
            )

        # Check for inefficient aggregation expressions
        for operator in operators:
            if operator.get('type', '').upper() == 'WINDOW_AGGREGATOR':
                config = operator.get('config', {})
                aggregations = config.get('aggregations', {})

                # Count number of aggregations
                if len(aggregations) > 50:
                    warnings.append(
                        f"Operator '{operator.get('name')}' has {len(aggregations)} aggregations. "
                        f"Large numbers of aggregations may impact performance. "
                        f"Consider splitting into multiple operators or simplifying logic."
                    )

                # Check for repeated CASE WHEN patterns (potential for optimization)
                repeated_patterns = {}
                for agg_name, agg_expr in aggregations.items():
                    if isinstance(agg_expr, str):
                        # Simple check for repeated CASE WHEN conditions
                        if 'CASE WHEN' in agg_expr.upper():
                            base_condition = agg_expr.split('THEN')[0] if 'THEN' in agg_expr else agg_expr
                            if base_condition not in repeated_patterns:
                                repeated_patterns[base_condition] = []
                            repeated_patterns[base_condition].append(agg_name)

                for condition, agg_names in repeated_patterns.items():
                    if len(agg_names) > 3:
                        warnings.append(
                            f"Operator '{operator.get('name')}' has {len(agg_names)} aggregations with "
                            f"similar CASE WHEN conditions. Consider refactoring for efficiency."
                        )

        return {
            'issues': issues,
            'warnings': warnings,
            'dag_info': {
                'stream_producers': stream_producers,
                'stream_consumers': stream_consumers,
                'dead_streams': dead_streams,
                'missing_streams': missing_streams,
                'total_streams': len(all_streams),
                'total_operators': len(operators),
                'total_sources': len(sources)
            }
        }

    def generate_report(self, jobspec_analysis: Optional[Dict[str, Any]] = None) -> str:
        """Generate a comprehensive optimization report"""
        report_lines = []
        report_lines.append("=" * 80)
        report_lines.append("FLINK JOB OPTIMIZATION REPORT")
        report_lines.append("=" * 80)
        report_lines.append("")

        if self.job_id:
            report_lines.append(f"Cluster: {self.cluster_url}")
            report_lines.append(f"Job ID: {self.job_id}")
            report_lines.append("")

            # Runtime analysis
            report_lines.append("## RUNTIME ANALYSIS (from Flink UI)")
            report_lines.append("")

            # Backpressure issues
            backpressure_issues = self.analyze_backpressure(self.job_id)
            if backpressure_issues:
                report_lines.append("### Backpressure & Parallelism Issues:")
                for issue in backpressure_issues:
                    report_lines.append(f"  ⚠ {issue}")
                report_lines.append("")

            # Checkpoint issues
            checkpoint_issues = self.analyze_checkpoint_performance(self.job_id)
            if checkpoint_issues:
                report_lines.append("### Checkpoint Performance Issues:")
                for issue in checkpoint_issues:
                    report_lines.append(f"  ⚠ {issue}")
                report_lines.append("")

            if not backpressure_issues and not checkpoint_issues:
                report_lines.append("  ✓ No runtime issues detected")
                report_lines.append("")

        # Jobspec analysis
        if jobspec_analysis:
            report_lines.append("## JOBSPEC CONFIGURATION ANALYSIS")
            report_lines.append("")

            issues = jobspec_analysis.get('issues', [])
            warnings = jobspec_analysis.get('warnings', [])
            dag_info = jobspec_analysis.get('dag_info', {})

            if issues:
                report_lines.append("### Critical Issues (MUST FIX):")
                for issue in issues:
                    report_lines.append(f"  ❌ {issue}")
                report_lines.append("")

            if warnings:
                report_lines.append("### Warnings (Recommended to Review):")
                for warning in warnings:
                    report_lines.append(f"  ⚠ {warning}")
                report_lines.append("")

            if not issues and not warnings:
                report_lines.append("  ✓ No configuration issues detected")
                report_lines.append("")

            # DAG summary
            report_lines.append("### DAG Summary:")
            report_lines.append(f"  - Total Sources: {dag_info.get('total_sources', 0)}")
            report_lines.append(f"  - Total Operators: {dag_info.get('total_operators', 0)}")
            report_lines.append(f"  - Total Streams: {dag_info.get('total_streams', 0)}")
            report_lines.append(f"  - Dead Streams: {len(dag_info.get('dead_streams', []))}")
            report_lines.append(f"  - Missing Streams: {len(dag_info.get('missing_streams', []))}")
            report_lines.append("")

        report_lines.append("=" * 80)
        report_lines.append("END OF REPORT")
        report_lines.append("=" * 80)

        return "\n".join(report_lines)


def main():
    parser = argparse.ArgumentParser(description='Analyze Flink job for optimization opportunities')
    parser.add_argument('--cluster-url', required=True, help='Flink cluster URL (e.g., https://flink.de.razorpay.com)')
    parser.add_argument('--job-id', help='Specific job ID to analyze (optional, will list running jobs if not provided)')
    parser.add_argument('--jobspec-path', help='Path to jobspec YAML file')
    parser.add_argument('--list-jobs', action='store_true', help='List all running jobs')

    args = parser.parse_args()

    analyzer = FlinkJobAnalyzer(args.cluster_url, args.job_id)

    # List jobs if requested or if no job ID provided
    if args.list_jobs or not args.job_id:
        print("Fetching running jobs from cluster...")
        running_jobs = analyzer.list_running_jobs()
        if running_jobs:
            print(f"\nFound {len(running_jobs)} running job(s):")
            for job in running_jobs:
                print(f"  - {job.get('id')} (Name: {job.get('name', 'Unknown')})")
        else:
            print("No running jobs found")

        if not args.job_id:
            return
        print("")

    # Analyze jobspec if provided
    jobspec_analysis = None
    if args.jobspec_path:
        print(f"Analyzing jobspec: {args.jobspec_path}")
        jobspec_analysis = analyzer.analyze_jobspec_yaml(args.jobspec_path)

    # Generate and print report
    report = analyzer.generate_report(jobspec_analysis)
    print(report)


if __name__ == '__main__':
    main()
