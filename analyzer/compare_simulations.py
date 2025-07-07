#!/usr/bin/env python3
"""
Compare multiple simulation runs to analyze scheduler weight impacts.

Usage:
    python compare_simulations.py --simulations sim1 sim2 sim3 --output report/
"""

import argparse
import sys
import os
import logging
from typing import List
import pandas as pd

# Add parent directory to path for imports
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from scheduler_weight_analyzer import SchedulerWeightAnalyzer

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


def main():
    parser = argparse.ArgumentParser(
        description="Compare multiple KEG simulation runs to analyze scheduler weight impacts"
    )
    parser.add_argument(
        "--simulations",
        nargs="+",
        required=True,
        help="List of simulation directory names (e.g., sim1 sim2 sim3)"
    )
    parser.add_argument(
        "--base-dir",
        type=str,
        default="results",
        help="Base directory containing simulation results (default: results)"
    )
    parser.add_argument(
        "--output",
        type=str,
        default="analysis_report",
        help="Output directory for analysis report (default: analysis_report)"
    )
    parser.add_argument(
        "--target-metric",
        type=str,
        default="scheduling_efficiency",
        choices=[
            "scheduling_efficiency",
            "fragmentation_index",
            "avg_pending_time",
            "cpu_utilization",
            "memory_utilization",
            "resource_balance_score"
        ],
        help="Target metric to optimize for (default: scheduling_efficiency)"
    )
    parser.add_argument(
        "--export-format",
        type=str,
        default="all",
        choices=["csv", "json", "html", "all"],
        help="Export format for comparison data (default: all)"
    )
    
    args = parser.parse_args()
    
    # Initialize analyzer
    analyzer = SchedulerWeightAnalyzer(args.base_dir)
    
    # Load simulation results
    logger.info(f"Loading {len(args.simulations)} simulations...")
    analyzer.load_simulation_results(args.simulations)
    
    if not analyzer.simulations:
        logger.error("No valid simulations found!")
        sys.exit(1)
        
    logger.info(f"Successfully loaded {len(analyzer.simulations)} simulations")
    
    # Create comparison DataFrame
    comparison_df = analyzer.create_comparison_dataframe()
    logger.info(f"Created comparison DataFrame with {len(comparison_df)} rows")
    
    # Find optimal weights
    optimal_weights = analyzer.find_optimal_weights(args.target_metric)
    logger.info(f"Optimal weights for {args.target_metric}: {optimal_weights}")
    
    # Generate comprehensive report
    logger.info(f"Generating analysis report in {args.output}...")
    analyzer.generate_report(args.output)
    
    # Export comparison data in requested formats
    if args.export_format in ["csv", "all"]:
        csv_path = os.path.join(args.output, "simulation_comparison.csv")
        comparison_df.to_csv(csv_path, index=False)
        logger.info(f"Exported comparison data to {csv_path}")
        
    if args.export_format in ["json", "all"]:
        json_path = os.path.join(args.output, "simulation_comparison.json")
        comparison_df.to_json(json_path, orient="records", indent=2)
        logger.info(f"Exported comparison data to {json_path}")
        
    if args.export_format in ["html", "all"]:
        html_path = os.path.join(args.output, "simulation_comparison.html")
        comparison_df.to_html(html_path, index=False)
        logger.info(f"Exported comparison data to {html_path}")
    
    # Print summary
    print("\n" + "="*60)
    print("ANALYSIS SUMMARY")
    print("="*60)
    print(f"Simulations analyzed: {len(analyzer.simulations)}")
    print(f"Target metric: {args.target_metric}")
    print(f"Optimal weights: {optimal_weights}")
    print(f"\nTop 3 configurations by {args.target_metric}:")
    
    # Sort by target metric
    if args.target_metric == "avg_pending_time":
        top_configs = comparison_df.nsmallest(3, args.target_metric)
    else:
        top_configs = comparison_df.nlargest(3, args.target_metric)
        
    for idx, row in top_configs.iterrows():
        print(f"\n{idx+1}. {row['run_id']}")
        print(f"   {args.target_metric}: {row[args.target_metric]:.4f}")
        weight_cols = [col for col in row.index if col.startswith('weight_')]
        for col in weight_cols:
            if pd.notna(row[col]):
                print(f"   {col}: {row[col]}")
    
    print(f"\nFull report generated in: {args.output}/")
    print("="*60)


if __name__ == "__main__":
    main()