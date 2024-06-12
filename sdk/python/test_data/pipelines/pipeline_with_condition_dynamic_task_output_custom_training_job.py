import google_cloud_pipeline_components.v1.custom_job as custom_job
from kfp import dsl


@dsl.component
def flip_biased_coin_op() -> str:
    """Flip a coin and output heads."""
    return 'heads'


@dsl.component
def machine_type() -> str:
    return 'n1-standard-4'


@dsl.component
def accelerator_type() -> str:
    return 'NVIDIA_TESLA_P4'


@dsl.component
def accelerator_count() -> int:
    return 1


@dsl.pipeline
def pipeline(
    project: str,
    location: str,
    encryption_spec_key_name: str = '',
):
    flip1 = flip_biased_coin_op().set_caching_options(False)
    machine_type_task = machine_type()
    accelerator_type_task = accelerator_type()
    accelerator_count_task = accelerator_count()

    with dsl.Condition(flip1.output == 'heads'):
        custom_job.CustomTrainingJobOp(
            display_name='add-numbers',
            worker_pool_specs=[{
                'container_spec': {
                    'image_uri': (
                        'gcr.io/ml-pipeline/google-cloud-pipeline-components:2.5.0'
                    ),
                    'command': ['echo'],
                    'args': ['foo'],
                },
                'machine_spec': {
                    'machine_type': machine_type_task.output,
                    'accelerator_type': accelerator_type_task.output,
                    'accelerator_count': accelerator_count_task.output,
                },
                'replica_count': 1,
            }],
            project=project,
            location=location,
            encryption_spec_key_name=encryption_spec_key_name,
        )


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.yaml'))
