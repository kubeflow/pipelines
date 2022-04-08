from typing import Dict, Any
import warnings
import json
import yaml


def _write_ir_to_file(ir_dict: Dict[str, Any], output_file: str) -> None:
    """Writes the IR JSON to a file.

    Args:
        ir_dict (Dict[str, Any]): IR JSON.
        output_file (str): Output file path.

    Raises:
        ValueError: If output file path is not JSON or YAML.
    """

    if output_file.endswith(".json"):
        warnings.warn(
            ("Compiling to JSON is deprecated and will be "
             "removed in a future version. Please compile to a YAML file by "
             "providing a file path with a .yaml extension instead."),
            category=DeprecationWarning,
            stacklevel=2,
        )
        ir_json = json.dumps(ir_dict, sort_keys=True)
        with open(output_file, 'w') as json_file:
            json_file.write(ir_json)
    elif output_file.endswith((".yaml", ".yml")):
        with open(output_file, 'w') as yaml_file:
            yaml.dump(ir_dict, yaml_file, sort_keys=True)
    else:
        raise ValueError(
            f'The output path {output_file} should end with ".yaml".')