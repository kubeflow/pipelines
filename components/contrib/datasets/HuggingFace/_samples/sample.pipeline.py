from kfp import components
from kfp import dsl


load_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/d0e14a1dad4b851ad2a60a0c1a8201493f3d931c/components/datasets/HuggingFace/Load_dataset/component.yaml')
split_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/d0e14a1dad4b851ad2a60a0c1a8201493f3d931c/components/datasets/HuggingFace/Split_dataset/component.yaml')


def huggingface_pipeline():
    dataset_dict_task = load_dataset_op(dataset_name='imdb')
    with dsl.ParallelFor(dataset_dict_task.outputs['splits']) as split_name:
        deataset_task = split_dataset_op(
            dataset_dict=dataset_dict_task.outputs['dataset_dict'],
            split_name=split_name,
        )


if __name__ == '__main__':
    import kfp
    kfp_endpoint = None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
        huggingface_pipeline,
        arguments={}
    )
