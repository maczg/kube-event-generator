import argparse

from analyzer import Analyzer


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--data-dir",
        type=str,
        default="data",
        help="Directory containing the data files.",
    )
    args = parser.parse_args()
    if not args.data_dir:
        raise ValueError("Data directory is required.")
    analyzer = Analyzer(args.data_dir)
    analyzer.fragmentation_index()


if __name__ == "__main__":
    main()
