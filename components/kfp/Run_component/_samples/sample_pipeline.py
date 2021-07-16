from kfp import components

kfp_endpoint = None


run_component_or_pipeline_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/2d13b358690c761f64e5b59b70624de8f1f52a29/components/kfp/Run_component/component.yaml')


def my_pipeline():
    run_component_or_pipeline_op(
        component_url='https://raw.githubusercontent.com/kubeflow/pipelines/68a367de3d1cc435637b0b4e78dcb42600fbbc37/components/basics/Calculate_hash/component.yaml',
        arguments=dict(
            data='Hello world',
        ),
    )


if __name__ == '__main__':
    import kfp
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(my_pipeline, arguments={})
