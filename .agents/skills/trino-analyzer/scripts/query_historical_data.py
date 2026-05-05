#!/usr/bin/env python3
"""
Query historical Trino query metrics from trino_showback table.

This script fetches historical query data to support cost analysis and prediction.
"""

import sys
import argparse
import json
from typing import Optional, Dict, List
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


def get_similar_queries(query_text: str, limit: int = 10) -> List[Dict]:
    """
    Find similar historical queries based on text similarity.

    Args:
        query_text: The query to find similar queries for
        limit: Maximum number of similar queries to return

    Returns:
        List of dictionaries containing query metrics
    """
    conn = connect_trino()
    cursor = conn.cursor()

    # Extract key patterns from query
    query_lower = query_text.lower()

    # Simple pattern matching - can be enhanced with more sophisticated similarity
    cursor.execute("""
        SELECT
            query_id,
            query,
            cpu_time,
            execution_time,
            elapsed_time,
            input_size,
            input_rows,
            peak_mem,
            cumulative_user_memory,
            physical_written_size,
            state
        FROM hive.dbt_prod_de_metrics.trino_showback
        WHERE state = 'FINISHED'
        ORDER BY query_created_at DESC
        LIMIT ?
    """, (limit * 10,))

    results = []
    for row in cursor.fetchall():
        results.append({
            'query_id': row[0],
            'query': row[1],
            'cpu_time': row[2],
            'execution_time': row[3],
            'elapsed_time': row[4],
            'input_size': row[5],
            'input_rows': row[6],
            'peak_mem': row[7],
            'cumulative_user_memory': row[8],
            'physical_written_size': row[9],
            'state': row[10]
        })

    cursor.close()
    conn.close()

    return results[:limit]


def get_query_metrics_by_id(query_id: str) -> Optional[Dict]:
    """
    Fetch metrics for a specific query by ID.

    Args:
        query_id: Trino query ID

    Returns:
        Dictionary containing query metrics or None if not found
    """
    conn = connect_trino()
    cursor = conn.cursor()

    cursor.execute("""
        SELECT
            query_id,
            query,
            cpu_time,
            scheduled_time,
            execution_time,
            elapsed_time,
            input_size,
            input_rows,
            peak_mem,
            cumulative_user_memory,
            output_rows,
            output_size,
            physical_written_rows,
            physical_written_size,
            state,
            error_type,
            error_name,
            user,
            source,
            query_type,
            query_created_at,
            cluster_label,
            cluster_group
        FROM hive.dbt_prod_de_metrics.trino_showback
        WHERE query_id = ?
    """, (query_id,))

    row = cursor.fetchone()
    cursor.close()
    conn.close()

    if row:
        return {
            'query_id': row[0],
            'query': row[1],
            'cpu_time': row[2],
            'scheduled_time': row[3],
            'execution_time': row[4],
            'elapsed_time': row[5],
            'input_size': row[6],
            'input_rows': row[7],
            'peak_mem': row[8],
            'cumulative_user_memory': row[9],
            'output_rows': row[10],
            'output_size': row[11],
            'physical_written_rows': row[12],
            'physical_written_size': row[13],
            'state': row[14],
            'error_type': row[15],
            'error_name': row[16],
            'user': row[17],
            'source': row[18],
            'query_type': row[19],
            'query_created_at': row[20],
            'cluster_label': row[21],
            'cluster_group': row[22]
        }
    return None


def get_cost_statistics(user: Optional[str] = None,
                       days: int = 30,
                       query_type: Optional[str] = None) -> Dict:
    """
    Get resource usage statistics for queries over a time period.

    Args:
        user: Filter by specific user (optional)
        days: Number of days to look back
        query_type: Filter by query type (SELECT, INSERT, etc.)

    Returns:
        Dictionary containing resource usage statistics
    """
    conn = connect_trino()
    cursor = conn.cursor()

    where_clauses = [f"CAST(query_date AS DATE) >= date_add('day', -{days}, current_date)"]
    if user:
        where_clauses.append(f"user = '{user}'")
    if query_type:
        where_clauses.append(f"query_type = '{query_type}'")

    where_clause = " AND ".join(where_clauses)

    cursor.execute(f"""
        SELECT
            COUNT(*) as total_queries,
            COUNT(DISTINCT user) as unique_users,
            AVG(cpu_time) as avg_cpu_time,
            AVG(peak_mem) as avg_peak_mem,
            AVG(input_size) as avg_input_size,
            SUM(input_size) as total_data_scanned,
            MAX(cpu_time) as max_cpu_time,
            MAX(peak_mem) as max_peak_mem
        FROM hive.dbt_prod_de_metrics.trino_showback
        WHERE {where_clause}
          AND state = 'FINISHED'
    """)

    row = cursor.fetchone()
    cursor.close()
    conn.close()

    if row:
        return {
            'total_queries': row[0],
            'unique_users': row[1],
            'avg_cpu_time': float(row[2]) if row[2] else 0,
            'avg_peak_mem': float(row[3]) if row[3] else 0,
            'avg_input_size': float(row[4]) if row[4] else 0,
            'total_data_scanned': float(row[5]) if row[5] else 0,
            'max_cpu_time': float(row[6]) if row[6] else 0,
            'max_peak_mem': float(row[7]) if row[7] else 0,
            'period_days': days
        }
    return {}


def get_expensive_queries(limit: int = 20, days: int = 7) -> List[Dict]:
    """
    Get the most resource-intensive queries by CPU time and data scanned.

    Args:
        limit: Number of queries to return
        days: Number of days to look back

    Returns:
        List of expensive queries with metrics
    """
    conn = connect_trino()
    cursor = conn.cursor()

    cursor.execute(f"""
        SELECT
            query_id,
            query,
            cpu_time,
            peak_mem,
            input_size,
            input_rows,
            execution_time,
            user,
            query_type,
            query_created_at
        FROM hive.dbt_prod_de_metrics.trino_showback
        WHERE CAST(query_date AS DATE) >= date_add('day', -{days}, current_date)
          AND state = 'FINISHED'
        ORDER BY cpu_time DESC, input_size DESC
        LIMIT {limit}
    """)

    results = []
    for row in cursor.fetchall():
        results.append({
            'query_id': row[0],
            'query': row[1],
            'cpu_time': row[2],
            'peak_mem': row[3],
            'input_size': row[4],
            'input_rows': row[5],
            'execution_time': row[6],
            'user': row[7],
            'query_type': row[8],
            'query_created_at': row[9]
        })

    cursor.close()
    conn.close()

    return results


def main():
    parser = argparse.ArgumentParser(description='Query historical Trino metrics')
    parser.add_argument('--query-id', help='Get metrics for specific query ID')
    parser.add_argument('--expensive', action='store_true', help='Get most expensive queries')
    parser.add_argument('--stats', action='store_true', help='Get cost statistics')
    parser.add_argument('--user', help='Filter by user')
    parser.add_argument('--days', type=int, default=7, help='Days to look back (default: 7)')
    parser.add_argument('--limit', type=int, default=10, help='Limit results (default: 10)')

    args = parser.parse_args()

    if args.query_id:
        result = get_query_metrics_by_id(args.query_id)
        print(json.dumps(result, indent=2, default=str))
    elif args.expensive:
        result = get_expensive_queries(limit=args.limit, days=args.days)
        print(json.dumps(result, indent=2, default=str))
    elif args.stats:
        result = get_cost_statistics(user=args.user, days=args.days)
        print(json.dumps(result, indent=2, default=str))
    else:
        parser.print_help()


if __name__ == '__main__':
    main()
