#!/usr/bin/env python3
"""
Identify optimization opportunities in Trino queries.

This script analyzes queries for anti-patterns and suggests specific optimizations
to reduce cost and improve performance.
"""

import sys
import argparse
import json
import re
from typing import Dict, List
import analyze_query_plan


class QueryOptimizer:
    """Analyzes queries and suggests optimizations."""

    def __init__(self, query: str, explain_output: str = None):
        self.query = query
        self.query_lower = query.lower()
        self.explain_output = explain_output
        self.plan_features = None

        if explain_output:
            self.plan_features = analyze_query_plan.parse_query_plan(explain_output)

    def analyze(self) -> Dict:
        """
        Perform comprehensive analysis and return optimization suggestions.

        Returns:
            Dictionary with findings and recommendations
        """
        findings = {
            'anti_patterns': [],
            'optimizations': [],
            'warnings': [],
            'cost_reduction_potential': 'unknown',
            'priority': 'low'
        }

        # Run all checks
        findings['anti_patterns'].extend(self._check_select_star())
        findings['anti_patterns'].extend(self._check_cross_joins())
        findings['anti_patterns'].extend(self._check_non_partitioned_filters())
        findings['anti_patterns'].extend(self._check_multiple_aggregations())
        findings['anti_patterns'].extend(self._check_distinct_in_aggregation())
        findings['anti_patterns'].extend(self._check_or_conditions())
        findings['anti_patterns'].extend(self._check_implicit_conversions())
        findings['anti_patterns'].extend(self._check_subquery_in_select())

        findings['optimizations'].extend(self._suggest_partition_pruning())
        findings['optimizations'].extend(self._suggest_predicate_pushdown())
        findings['optimizations'].extend(self._suggest_join_reordering())
        findings['optimizations'].extend(self._suggest_materialization())
        findings['optimizations'].extend(self._suggest_limit_usage())
        findings['optimizations'].extend(self._suggest_approximate_aggregations())

        # Assess cost reduction potential
        if len(findings['anti_patterns']) > 3:
            findings['cost_reduction_potential'] = 'high'
            findings['priority'] = 'high'
        elif len(findings['anti_patterns']) > 1:
            findings['cost_reduction_potential'] = 'medium'
            findings['priority'] = 'medium'
        elif len(findings['anti_patterns']) > 0:
            findings['cost_reduction_potential'] = 'low'
            findings['priority'] = 'low'

        return findings

    def _check_select_star(self) -> List[Dict]:
        """Check for SELECT * usage."""
        issues = []
        if re.search(r'select\s+\*\s+from', self.query_lower):
            issues.append({
                'type': 'SELECT_STAR',
                'severity': 'medium',
                'description': 'Using SELECT * retrieves all columns, increasing data transfer and processing',
                'recommendation': 'Select only the columns you need: SELECT col1, col2 FROM ...',
                'potential_savings': '20-50% reduction in data processed'
            })
        return issues

    def _check_cross_joins(self) -> List[Dict]:
        """Check for cross joins."""
        issues = []
        if self.plan_features and 'CROSS' in self.plan_features.get('join_types', []):
            issues.append({
                'type': 'CROSS_JOIN',
                'severity': 'critical',
                'description': 'Cross join detected - produces cartesian product of tables',
                'recommendation': 'Add JOIN conditions: JOIN table ON condition',
                'potential_savings': '90%+ reduction in cost'
            })
        elif re.search(r'cross\s+join', self.query_lower):
            issues.append({
                'type': 'CROSS_JOIN',
                'severity': 'critical',
                'description': 'Explicit CROSS JOIN detected',
                'recommendation': 'Replace with INNER JOIN with proper conditions if possible',
                'potential_savings': '90%+ reduction in cost'
            })
        return issues

    def _check_non_partitioned_filters(self) -> List[Dict]:
        """Check if partitioning columns are used in filters."""
        issues = []
        # Common partition columns
        partition_cols = ['dt', 'date', 'year', 'month', 'day', 'partition_date']

        has_date_filter = any(col in self.query_lower for col in partition_cols)

        if not has_date_filter and 'from' in self.query_lower:
            issues.append({
                'type': 'MISSING_PARTITION_FILTER',
                'severity': 'high',
                'description': 'No partition filter detected (dt, date, etc.)',
                'recommendation': 'Add partition filters: WHERE dt >= \'2024-01-01\' to scan less data',
                'potential_savings': '70-95% reduction in data scanned'
            })
        return issues

    def _check_multiple_aggregations(self) -> List[Dict]:
        """Check for inefficient multiple aggregations."""
        issues = []
        agg_count = len(re.findall(r'\b(count|sum|avg|max|min)\s*\(', self.query_lower))

        if agg_count > 5:
            issues.append({
                'type': 'MULTIPLE_AGGREGATIONS',
                'severity': 'medium',
                'description': f'Query has {agg_count} aggregation functions',
                'recommendation': 'Consider breaking into multiple queries or using approximate aggregations',
                'potential_savings': '15-30% reduction in processing time'
            })
        return issues

    def _check_distinct_in_aggregation(self) -> List[Dict]:
        """Check for COUNT(DISTINCT) which can be expensive."""
        issues = []
        if re.search(r'count\s*\(\s*distinct', self.query_lower):
            issues.append({
                'type': 'COUNT_DISTINCT',
                'severity': 'medium',
                'description': 'COUNT(DISTINCT) can be expensive for high cardinality columns',
                'recommendation': 'Consider using approx_distinct() for approximate counts (usually within 2-3% accuracy)',
                'potential_savings': '40-60% reduction in memory and CPU'
            })
        return issues

    def _check_or_conditions(self) -> List[Dict]:
        """Check for multiple OR conditions in WHERE clause."""
        issues = []
        or_count = len(re.findall(r'\bor\b', self.query_lower))

        if or_count > 3:
            issues.append({
                'type': 'MULTIPLE_OR_CONDITIONS',
                'severity': 'low',
                'description': f'Query has {or_count} OR conditions which can prevent index usage',
                'recommendation': 'Consider using IN clause or UNION instead: WHERE col IN (val1, val2, ...)',
                'potential_savings': '10-25% performance improvement'
            })
        return issues

    def _check_implicit_conversions(self) -> List[Dict]:
        """Check for potential implicit type conversions."""
        issues = []
        # Look for numeric columns compared to strings
        if re.search(r'(id|number|count)\s*=\s*[\'"]', self.query_lower):
            issues.append({
                'type': 'IMPLICIT_CONVERSION',
                'severity': 'low',
                'description': 'Potential implicit type conversion detected (comparing number to string)',
                'recommendation': 'Ensure types match in comparisons: WHERE id = 123 (not \'123\')',
                'potential_savings': '5-15% performance improvement'
            })
        return issues

    def _check_subquery_in_select(self) -> List[Dict]:
        """Check for subqueries in SELECT clause."""
        issues = []
        # Simple pattern to detect subquery in SELECT
        if re.search(r'select[^from]+(select\s+.*?\s+from)', self.query_lower):
            issues.append({
                'type': 'SUBQUERY_IN_SELECT',
                'severity': 'medium',
                'description': 'Subquery in SELECT clause executes for each row',
                'recommendation': 'Move subquery to FROM clause with JOIN or use window functions',
                'potential_savings': '30-70% reduction in execution time'
            })
        return issues

    def _suggest_partition_pruning(self) -> List[Dict]:
        """Suggest partition pruning optimizations."""
        suggestions = []
        if self.plan_features and self.plan_features.get('scanned_tables'):
            suggestions.append({
                'type': 'PARTITION_PRUNING',
                'description': 'Ensure partition columns are used in WHERE clause',
                'example': 'WHERE dt BETWEEN \'2024-01-01\' AND \'2024-01-31\'',
                'benefit': 'Reduces amount of data scanned from S3'
            })
        return suggestions

    def _suggest_predicate_pushdown(self) -> List[Dict]:
        """Suggest predicate pushdown."""
        suggestions = []
        if self.plan_features and self.plan_features.get('join_count', 0) > 0:
            suggestions.append({
                'type': 'PREDICATE_PUSHDOWN',
                'description': 'Apply filters before joins to reduce intermediate data',
                'example': 'WITH filtered AS (SELECT * FROM table WHERE condition) SELECT * FROM filtered JOIN ...',
                'benefit': 'Reduces data size before expensive join operations'
            })
        return suggestions

    def _suggest_join_reordering(self) -> List[Dict]:
        """Suggest join order optimization."""
        suggestions = []
        if self.plan_features and self.plan_features.get('join_count', 0) > 2:
            suggestions.append({
                'type': 'JOIN_REORDERING',
                'description': 'Order joins from smallest to largest tables',
                'example': 'Join small dimension tables first, then large fact tables',
                'benefit': 'Reduces intermediate result sizes'
            })
        return suggestions

    def _suggest_materialization(self) -> List[Dict]:
        """Suggest materializing subqueries."""
        suggestions = []
        # Check for repeated complex patterns
        cte_count = len(re.findall(r'\bwith\b', self.query_lower))
        subquery_count = len(re.findall(r'select.*?from\s*\(', self.query_lower))

        if subquery_count > 2 and cte_count == 0:
            suggestions.append({
                'type': 'USE_CTE',
                'description': 'Use Common Table Expressions (CTEs) for repeated subqueries',
                'example': 'WITH cte AS (SELECT ...) SELECT * FROM cte',
                'benefit': 'Improves readability and may enable better optimization'
            })
        return suggestions

    def _suggest_limit_usage(self) -> List[Dict]:
        """Suggest adding LIMIT for testing."""
        suggestions = []
        if self.plan_features and not self.plan_features.get('has_limit'):
            if self.plan_features.get('complexity_score', 0) > 3:
                suggestions.append({
                    'type': 'ADD_LIMIT',
                    'description': 'Add LIMIT clause when testing complex queries',
                    'example': 'Add: LIMIT 1000 to the end of the query',
                    'benefit': 'Reduces cost and time during development and testing'
                })
        return suggestions

    def _suggest_approximate_aggregations(self) -> List[Dict]:
        """Suggest using approximate aggregations."""
        suggestions = []
        if 'count(distinct' in self.query_lower:
            suggestions.append({
                'type': 'APPROXIMATE_AGGREGATION',
                'description': 'Use approx_distinct() instead of COUNT(DISTINCT) for large datasets',
                'example': 'SELECT approx_distinct(user_id) FROM table',
                'benefit': '40-60% faster with <3% error rate'
            })
        return suggestions


def optimize_query(query: str, get_plan: bool = True) -> Dict:
    """
    Analyze query and return optimization suggestions.

    Args:
        query: SQL query to optimize
        get_plan: Whether to get query plan (requires Trino connection)

    Returns:
        Dictionary with optimization findings
    """
    explain_output = None

    if get_plan:
        try:
            explain_output = analyze_query_plan.explain_query(query)
        except Exception as e:
            print(f"Warning: Could not get query plan: {e}", file=sys.stderr)

    optimizer = QueryOptimizer(query, explain_output)
    findings = optimizer.analyze()

    return {
        'query': query[:200] + '...' if len(query) > 200 else query,
        'analysis': findings,
        'summary': {
            'anti_patterns_found': len(findings['anti_patterns']),
            'optimizations_suggested': len(findings['optimizations']),
            'cost_reduction_potential': findings['cost_reduction_potential'],
            'priority': findings['priority']
        }
    }


def main():
    parser = argparse.ArgumentParser(description='Identify Trino query optimizations')
    parser.add_argument('--query', help='SQL query to analyze')
    parser.add_argument('--query-file', help='File containing SQL query')
    parser.add_argument('--no-plan', action='store_true',
                       help='Skip getting query plan (faster but less accurate)')

    args = parser.parse_args()

    # Get query
    if args.query_file:
        with open(args.query_file, 'r') as f:
            query = f.read()
    elif args.query:
        query = args.query
    else:
        parser.print_help()
        return

    # Optimize
    result = optimize_query(query, get_plan=not args.no_plan)

    # Pretty print results
    print("=" * 80)
    print("TRINO QUERY OPTIMIZATION ANALYSIS")
    print("=" * 80)

    summary = result['summary']
    print(f"\nSummary:")
    print(f"  Anti-patterns found: {summary['anti_patterns_found']}")
    print(f"  Optimizations suggested: {summary['optimizations_suggested']}")
    print(f"  Cost reduction potential: {summary['cost_reduction_potential']}")
    print(f"  Priority: {summary['priority']}")

    analysis = result['analysis']

    if analysis['anti_patterns']:
        print("\n" + "=" * 80)
        print("ANTI-PATTERNS DETECTED:")
        print("=" * 80)
        for i, issue in enumerate(analysis['anti_patterns'], 1):
            print(f"\n{i}. {issue['type']} (Severity: {issue['severity']})")
            print(f"   Description: {issue['description']}")
            print(f"   Recommendation: {issue['recommendation']}")
            if 'potential_savings' in issue:
                print(f"   Potential savings: {issue['potential_savings']}")

    if analysis['optimizations']:
        print("\n" + "=" * 80)
        print("OPTIMIZATION SUGGESTIONS:")
        print("=" * 80)
        for i, opt in enumerate(analysis['optimizations'], 1):
            print(f"\n{i}. {opt['type']}")
            print(f"   {opt['description']}")
            print(f"   Example: {opt['example']}")
            print(f"   Benefit: {opt['benefit']}")


if __name__ == '__main__':
    main()
