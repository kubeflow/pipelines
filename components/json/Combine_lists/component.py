from kfp.components import create_component_from_func


def combine_lists(
    list_1: list = None,
    list_2: list = None,
    list_3: list = None,
    list_4: list = None,
    list_5: list = None,
) -> list:
    """Combines multiple JSON arrays into one.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    """
    result = []
    for list in [list_1, list_2, list_3, list_4, list_5]:
        if list is not None:
            result.extend(list)
    return result


if __name__ == '__main__':
    combine_lists_op = create_component_from_func(
        combine_lists,
        base_image='python:3.8',
        output_component_file='component.yaml',
    )
