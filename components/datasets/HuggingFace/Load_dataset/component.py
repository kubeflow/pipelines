from typing import NamedTuple

from kfp.components import create_component_from_func, OutputPath


def load_dataset_using_huggingface(
    dataset_name: str,
    dataset_dict_path: OutputPath('HuggingFaceDatasetDict'),
) -> NamedTuple('Outputs', [
    ('splits', list),
]):
    from datasets import load_dataset

    dataset_dict = load_dataset(dataset_name)
    dataset_dict.save_to_disk(dataset_dict_path)
    splits = list(dataset_dict.keys())
    return (splits,)


if __name__ == '__main__':
    load_dataset_op = create_component_from_func(
        load_dataset_using_huggingface,
        base_image='python:3.9',
        packages_to_install=['datasets==1.6.2'],
        annotations={
            'author': 'Alexey Volkov <alexey.volkov@ark-kun.com>',
            "canonical_location": "https://raw.githubusercontent.com/Ark-kun/pipeline_components/master/components/datasets/HuggingFace/Load_dataset/component.yaml",
        },
        output_component_file='component.yaml',
    )
