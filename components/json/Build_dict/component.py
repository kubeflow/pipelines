from kfp.components import create_component_from_func


def build_dict(
    key_1: str = None,
    value_1: dict = None,
    key_2: str = None,
    value_2: dict = None,
    key_3: str = None,
    value_3: dict = None,
    key_4: str = None,
    value_4: dict = None,
    key_5: str = None,
    value_5: dict = None,
) -> dict:
    """Creates a JSON object from multiple key and value pairs.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    """
    result = dict([
        (key_1, value_1),
        (key_2, value_2),
        (key_3, value_3),
        (key_4, value_4),
        (key_5, value_5),
    ])
    if None in result:
        del result[None]
    return result


if __name__ == '__main__':
    build_dict_op = create_component_from_func(
        build_dict,
        base_image='python:3.8',
        output_component_file='component.yaml',
    )
