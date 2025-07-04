import os


def scale_column(df, col, factor):
    """
    Multiplies the given column by 'factor'.
    """
    df[col] = df[[col]] * factor


def parse_cpu(cpu_str: str) -> float:
    """
    Parse a CPU string like '421m' into a float (e.g., 421).
    If the string has no 'm' suffix, interpret it directly as a float.
    """
    if isinstance(cpu_str, str) and cpu_str.endswith('m'):
        return float(cpu_str.replace('m', ''))
    else:
        return float(cpu_str)


def parse_mem(mem_str: str) -> float:
    """
    Parse a memory string like '1044Mi' into a float (e.g., 1044).
    If the string has no 'Mi' suffix, interpret it directly as a float.
    """
    if isinstance(mem_str, str) and 'Mi' in mem_str:
        return float(mem_str.replace('Mi', ''))
    elif isinstance(mem_str, str) and 'Gi' in mem_str:
        return float(mem_str.replace('Gi', '')) * 1024
    else:
        return float(mem_str)


def _exist_or_create_dir(dir_path):
    """
    Check if the directory exists, if not, create it.
    """
    if not os.path.exists(dir_path):
        os.makedirs(dir_path)
