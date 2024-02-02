from typing import Any, Dict, List
from kfp import compiler, dsl


@dsl.component
def add(a: int, b: int) -> int:
    return a + b


@dsl.component()
def list_dict_without_type_maker_1() -> List[Dict[str, int]]:
    return [{'a': 1, 'b': 2}, {'a': 3, 'b': 4}]


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=list_dict_without_type_maker_1,
        package_path='up_stream_component.yaml')
