"""
Trino Query Cost Analyzer - Scripts Package

This package provides tools for analyzing Trino query costs and optimizations.
"""

__version__ = "1.0.0"

# Import main functions for easier access
try:
    from .predict_query_cost import predict_cost
    from .identify_optimizations import analyze_query
    from .analyze_query_plan import explain_query, parse_query_plan
    from .query_historical_data import get_cost_statistics, get_expensive_queries

    __all__ = [
        'predict_cost',
        'analyze_query',
        'explain_query',
        'parse_query_plan',
        'get_cost_statistics',
        'get_expensive_queries',
    ]
except ImportError:
    # Allow package to be imported even if dependencies aren't installed
    pass
