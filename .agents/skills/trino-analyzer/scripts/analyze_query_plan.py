#!/usr/bin/env python3
"""
Analyze Trino query plans using EXPLAIN output.

This script gets the query plan and extracts key features for cost estimation.
"""

import sys
import argparse
import json
import re
from typing import Dict, List, Optional
import trino

# Trino connection configuration
TRINO_CONFIG = {
    'host': 'trino-querybook-coordinator.de.razorpay.com',
    'port': 443,
    'user': 'root',
    'catalog': 'hive',
    'schema': 'dbt_prod_de_metrics',
    'http_scheme': 'https'
}


def connect_trino():
    """Establish connection to Trino."""
    return trino.dbapi.connect(**TRINO_CONFIG)


def explain_query(query: str) -> str:
    """
    Get EXPLAIN output for a query.

    Args:
        query: SQL query to explain

    Returns:
        EXPLAIN output as string
    """
    conn = connect_trino()
    cursor = conn.cursor()

    try:
        cursor.execute(f"EXPLAIN {query}")
        rows = cursor.fetchall()
        explain_output = '\n'.join([row[0] for row in rows])
        return explain_output
    finally:
        cursor.close()
        conn.close()


def explain_analyze_query(query: str) -> str:
    """
    Get EXPLAIN ANALYZE output for a query (executes the query).

    Args:
        query: SQL query to explain and analyze

    Returns:
        EXPLAIN ANALYZE output with actual runtime statistics
    """
    conn = connect_trino()
    cursor = conn.cursor()

    try:
        cursor.execute(f"EXPLAIN ANALYZE {query}")
        rows = cursor.fetchall()
        explain_output = '\n'.join([row[0] for row in rows])
        return explain_output
    finally:
        cursor.close()
        conn.close()


def parse_query_plan(explain_output: str) -> Dict:
    """
    Parse EXPLAIN output to extract key features for cost estimation.

    Args:
        explain_output: Output from EXPLAIN query

    Returns:
        Dictionary with parsed plan features
    """
    features = {
        'has_join': False,
        'join_count': 0,
        'join_types': [],
        'has_aggregation': False,
        'aggregation_count': 0,
        'has_window_functions': False,
        'has_table_scan': False,
        'table_scan_count': 0,
        'scanned_tables': [],
        'has_filter': False,
        'has_sort': False,
        'has_limit': False,
        'partition_count_estimate': 0,
        'output_rows_estimate': 0,
        'cpu_estimate': 0.0,
        'memory_estimate': 0.0
    }

    lines = explain_output.split('\n')

    for line in lines:
        line_lower = line.lower()

        # Detect joins
        if 'join' in line_lower:
            features['has_join'] = True
            features['join_count'] += 1
            if 'inner join' in line_lower:
                features['join_types'].append('INNER')
            elif 'left join' in line_lower:
                features['join_types'].append('LEFT')
            elif 'right join' in line_lower:
                features['join_types'].append('RIGHT')
            elif 'cross join' in line_lower:
                features['join_types'].append('CROSS')

        # Detect aggregations
        if 'aggregate' in line_lower or 'group by' in line_lower:
            features['has_aggregation'] = True
            features['aggregation_count'] += 1

        # Detect window functions
        if 'window' in line_lower:
            features['has_window_functions'] = True

        # Detect table scans
        if 'tablescan' in line_lower or 'scan' in line_lower:
            features['has_table_scan'] = True
            features['table_scan_count'] += 1

            # Extract table name
            table_match = re.search(r'table\s*=\s*([^\s,\]]+)', line)
            if table_match:
                features['scanned_tables'].append(table_match.group(1))

        # Detect filters
        if 'filter' in line_lower or 'where' in line_lower:
            features['has_filter'] = True

        # Detect sorting
        if 'sort' in line_lower or 'order by' in line_lower:
            features['has_sort'] = True

        # Detect limit
        if 'limit' in line_lower:
            features['has_limit'] = True

        # Extract row estimates
        rows_match = re.search(r'(\d+)\s*rows', line_lower)
        if rows_match:
            row_estimate = int(rows_match.group(1))
            features['output_rows_estimate'] = max(features['output_rows_estimate'], row_estimate)

        # Extract CPU estimates (if available in EXPLAIN ANALYZE)
        cpu_match = re.search(r'cpu:\s*([\d.]+)', line_lower)
        if cpu_match:
            features['cpu_estimate'] = max(features['cpu_estimate'], float(cpu_match.group(1)))

        # Extract memory estimates
        memory_match = re.search(r'memory:\s*([\d.]+)', line_lower)
        if memory_match:
            features['memory_estimate'] = max(features['memory_estimate'], float(memory_match.group(1)))

    # Calculate complexity score (simple heuristic)
    complexity_score = (
        features['join_count'] * 2 +
        features['aggregation_count'] * 1.5 +
        (2 if features['has_window_functions'] else 0) +
        features['table_scan_count'] * 0.5 +
        (1 if features['has_sort'] else 0)
    )
    features['complexity_score'] = complexity_score

    return features


def estimate_cost_from_plan(plan_features: Dict, historical_stats: Optional[Dict] = None) -> Dict:
    """
    Estimate query cost based on plan features and optional historical statistics.

    Args:
        plan_features: Parsed query plan features
        historical_stats: Optional historical statistics for similar queries

    Returns:
        Dictionary with cost estimates
    """
    # Base cost factors (these can be tuned based on historical data)
    BASE_COST = 0.001  # Base cost in dollars

    # Cost multipliers
    cost = BASE_COST
    cost_breakdown = {'base': BASE_COST}

    # Join cost (cross joins are very expensive)
    if plan_features['has_join']:
        if 'CROSS' in plan_features['join_types']:
            join_cost = 10.0 * plan_features['join_count']
        else:
            join_cost = 0.5 * plan_features['join_count']
        cost += join_cost
        cost_breakdown['joins'] = join_cost

    # Aggregation cost
    if plan_features['has_aggregation']:
        agg_cost = 0.2 * plan_features['aggregation_count']
        cost += agg_cost
        cost_breakdown['aggregations'] = agg_cost

    # Window function cost (expensive)
    if plan_features['has_window_functions']:
        window_cost = 2.0
        cost += window_cost
        cost_breakdown['window_functions'] = window_cost

    # Table scan cost
    if plan_features['has_table_scan']:
        scan_cost = 0.1 * plan_features['table_scan_count']
        cost += scan_cost
        cost_breakdown['table_scans'] = scan_cost

    # Sort cost
    if plan_features['has_sort']:
        sort_cost = 0.3
        cost += sort_cost
        cost_breakdown['sort'] = sort_cost

    # Apply historical statistics if available
    if historical_stats:
        # Adjust based on average cost of similar queries
        if 'avg_cost' in historical_stats and historical_stats['avg_cost'] > 0:
            historical_multiplier = historical_stats['avg_cost'] / cost
            cost *= historical_multiplier
            cost_breakdown['historical_adjustment'] = historical_multiplier

    return {
        'estimated_cost': round(cost, 6),
        'cost_breakdown': cost_breakdown,
        'confidence': 'medium' if historical_stats else 'low',
        'complexity_score': plan_features['complexity_score']
    }


def main():
    parser = argparse.ArgumentParser(description='Analyze Trino query plans')
    parser.add_argument('--query', help='SQL query to analyze')
    parser.add_argument('--query-file', help='File containing SQL query')
    parser.add_argument('--analyze', action='store_true',
                       help='Use EXPLAIN ANALYZE (executes query)')
    parser.add_argument('--estimate-cost', action='store_true',
                       help='Estimate cost from plan')

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

    # Get EXPLAIN output
    if args.analyze:
        print("Getting EXPLAIN ANALYZE output (query will execute)...")
        explain_output = explain_analyze_query(query)
    else:
        print("Getting EXPLAIN output...")
        explain_output = explain_query(query)

    print("\n" + "=" * 80)
    print("EXPLAIN OUTPUT:")
    print("=" * 80)
    print(explain_output)

    # Parse plan
    print("\n" + "=" * 80)
    print("PARSED PLAN FEATURES:")
    print("=" * 80)
    features = parse_query_plan(explain_output)
    print(json.dumps(features, indent=2))

    # Estimate cost if requested
    if args.estimate_cost:
        print("\n" + "=" * 80)
        print("COST ESTIMATE:")
        print("=" * 80)
        cost_estimate = estimate_cost_from_plan(features)
        print(json.dumps(cost_estimate, indent=2))


if __name__ == '__main__':
    main()
