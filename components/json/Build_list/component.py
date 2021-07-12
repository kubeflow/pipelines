from kfp.components import create_component_from_func


def build_list(
    item_1: dict = None,
    item_2: dict = None,
    item_3: dict = None,
    item_4: dict = None,
    item_5: dict = None,
) -> list:
    """Creates a JSON array from multiple items.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    """
    result = []
    for item in [item_1, item_2, item_3, item_4, item_5]:
        if item is not None:
            result.append(item)
    return result


if __name__ == '__main__':
    build_list_op = create_component_from_func(
        build_list,
        base_image='python:3.8',
        output_component_file='component.yaml',
    )
