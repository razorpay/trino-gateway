#!/usr/bin/env python3
"""
Predict Trino query cost by combining query plan analysis with historical data.

This script provides the most accurate cost predictions by leveraging both
query plan features and historical execution patterns.
"""

import sys
import argparse
import json
from typing import Dict, Optional
import analyze_query_plan
import query_historical_data


def predict_cost(query: str, use_historical: bool = True) -> Dict:
    """
    Predict cost for a query.

    Args:
        query: SQL query to predict cost for
        use_historical: Whether to use historical data for better predictions

    Returns:
        Dictionary with cost prediction and breakdown
    """
    # Step 1: Get query plan and features
    explain_output = analyze_query_plan.explain_query(query)
    plan_features = analyze_query_plan.parse_query_plan(explain_output)

    # Step 2: Get historical statistics if requested
    historical_stats = None
    if use_historical:
        # Get general cost statistics for last 30 days
        historical_stats = query_historical_data.get_cost_statistics(days=30)

    # Step 3: Estimate cost
    cost_estimate = analyze_query_plan.estimate_cost_from_plan(
        plan_features,
        historical_stats
    )

    # Step 4: Build comprehensive prediction result
    prediction = {
        'query': query[:200] + '...' if len(query) > 200 else query,
        'estimated_cost_usd': cost_estimate['estimated_cost'],
        'confidence': cost_estimate['confidence'],
        'complexity_score': cost_estimate['complexity_score'],
        'cost_breakdown': cost_estimate['cost_breakdown'],
        'plan_features': {
            'joins': plan_features['join_count'],
            'join_types': plan_features['join_types'],
            'aggregations': plan_features['aggregation_count'],
            'table_scans': plan_features['table_scan_count'],
            'scanned_tables': plan_features['scanned_tables'],
            'has_window_functions': plan_features['has_window_functions'],
            'has_sort': plan_features['has_sort'],
            'has_limit': plan_features['has_limit']
        }
    }

    # Add historical context if available
    if historical_stats:
        prediction['historical_context'] = {
            'avg_cpu_time': historical_stats.get('avg_cpu_time', 0),
            'avg_peak_mem': historical_stats.get('avg_peak_mem', 0),
            'avg_input_size': historical_stats.get('avg_input_size', 0),
            'max_cpu_time': historical_stats.get('max_cpu_time', 0),
            'max_peak_mem': historical_stats.get('max_peak_mem', 0)
        }

    # Add recommendations
    recommendations = []

    if plan_features['join_count'] > 3:
        recommendations.append("Query has many joins - consider denormalizing data or materializing intermediate results")

    if 'CROSS' in plan_features['join_types']:
        recommendations.append("WARNING: Cross join detected - this can be extremely expensive. Add join conditions if possible")

    if plan_features['has_window_functions'] and not plan_features['has_limit']:
        recommendations.append("Window functions without LIMIT can be expensive - add LIMIT if you don't need all rows")

    if not plan_features['has_filter']:
        recommendations.append("No filter detected - adding WHERE clauses can significantly reduce cost")

    if plan_features['table_scan_count'] > 5:
        recommendations.append("Many table scans detected - consider using CTEs or temp tables for repeated data access")

    if not plan_features['has_limit'] and plan_features['complexity_score'] > 5:
        recommendations.append("Complex query without LIMIT - consider adding LIMIT for testing or if you don't need all results")

    prediction['recommendations'] = recommendations

    return prediction


def compare_with_historical(query_id: str) -> Dict:
    """
    Compare a past query's predicted cost with actual resource usage.

    Args:
        query_id: Historical query ID to analyze

    Returns:
        Dictionary with comparison results
    """
    # Get actual metrics
    actual = query_historical_data.get_query_metrics_by_id(query_id)

    if not actual:
        return {'error': f'Query {query_id} not found in historical data'}

    # Get prediction for the same query
    prediction = predict_cost(actual['query'], use_historical=True)

    return {
        'query_id': query_id,
        'predicted_cost': prediction['estimated_cost_usd'],
        'actual_metrics': {
            'cpu_time': actual.get('cpu_time', 0),
            'peak_mem': actual.get('peak_mem', 0),
            'input_size': actual.get('input_size', 0),
            'input_rows': actual.get('input_rows', 0),
            'execution_time': actual.get('execution_time', 0)
        },
        'prediction_details': prediction,
        'note': 'Actual cost not available in historical data. Use resource metrics (CPU, memory, data scanned) to validate prediction reasonableness.'
    }


def main():
    parser = argparse.ArgumentParser(description='Predict Trino query cost')
    parser.add_argument('--query', help='SQL query to predict cost for')
    parser.add_argument('--query-file', help='File containing SQL query')
    parser.add_argument('--compare', help='Compare prediction with historical query by ID')
    parser.add_argument('--no-historical', action='store_true',
                       help='Don\'t use historical data for prediction')

    args = parser.parse_args()

    if args.compare:
        # Compare with historical query
        result = compare_with_historical(args.compare)
        print(json.dumps(result, indent=2, default=str))

    elif args.query or args.query_file:
        # Predict cost for new query
        if args.query_file:
            with open(args.query_file, 'r') as f:
                query = f.read()
        else:
            query = args.query

        result = predict_cost(query, use_historical=not args.no_historical)

        # Pretty print results
        print("=" * 80)
        print("TRINO QUERY COST PREDICTION")
        print("=" * 80)
        print(f"\nEstimated Cost: ${result['estimated_cost_usd']:.6f}")
        print(f"Confidence: {result['confidence']}")
        print(f"Complexity Score: {result['complexity_score']:.1f}")

        print("\nCost Breakdown:")
        for component, cost in result['cost_breakdown'].items():
            print(f"  {component}: ${cost:.6f}" if isinstance(cost, float) else f"  {component}: {cost}")

        print("\nPlan Features:")
        for feature, value in result['plan_features'].items():
            if value or isinstance(value, int):
                print(f"  {feature}: {value}")

        if result.get('recommendations'):
            print("\nRecommendations:")
            for i, rec in enumerate(result['recommendations'], 1):
                print(f"  {i}. {rec}")

        if result.get('historical_context'):
            print("\nHistorical Context (Last 30 Days):")
            ctx = result['historical_context']
            print(f"  Average CPU time: {ctx['avg_cpu_time']:.2f}s")
            print(f"  Average peak memory: {ctx['avg_peak_mem'] / (1024**3):.2f} GB")
            print(f"  Average input size: {ctx['avg_input_size'] / (1024**3):.2f} GB")

    else:
        parser.print_help()


if __name__ == '__main__':
    main()
