from kfp.components import create_component_from_func, InputPath, OutputPath


def split_dataset_huggingface(
    dataset_dict_path: InputPath('HuggingFaceDatasetDict'),
    dataset_split_path: OutputPath('HuggingFaceDataset'),
    dataset_path: OutputPath('HuggingFaceArrowDataset'),
    # dataset_indices_path: OutputPath('HuggingFaceArrowDataset'),
    dataset_info_path: OutputPath(dict),
    dataset_state_path: OutputPath(dict),
    split_name: str = None,
):
    import os
    import shutil
    from datasets import config as datasets_config

    print(f'DatasetDict contents: {os.listdir(dataset_dict_path)}')
    shutil.copytree(os.path.join(dataset_dict_path, split_name), dataset_split_path)
    print(f'Dataset contents: {os.listdir(os.path.join(dataset_dict_path, split_name))}')
    shutil.copy(os.path.join(dataset_dict_path, split_name, datasets_config.DATASET_ARROW_FILENAME), dataset_path)
    # shutil.copy(os.path.join(dataset_dict_path, split_name, datasets_config.DATASET_INDICES_FILENAME), dataset_indices_path)
    shutil.copy(os.path.join(dataset_dict_path, split_name, datasets_config.DATASET_INFO_FILENAME), dataset_info_path)
    shutil.copy(os.path.join(dataset_dict_path, split_name, datasets_config.DATASET_STATE_JSON_FILENAME), dataset_state_path)


if __name__ == '__main__':
    split_dataset_op = create_component_from_func(
        split_dataset_huggingface,
        base_image='python:3.9',
        packages_to_install=['datasets==1.6.2'],
        annotations={
            'author': 'Alexey Volkov <alexey.volkov@ark-kun.com>',
            "canonical_location": "https://raw.githubusercontent.com/Ark-kun/pipeline_components/master/components/datasets/HuggingFace/Split_dataset/component.yaml",
        },
        output_component_file='component.yaml',
    )
