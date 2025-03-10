import google_cloud_pipeline_components.v1.custom_job as custom_job
from kfp import dsl


@dsl.pipeline
def pipeline(
    project: str,
    location: str,
    machine_type: str,
    accelerator_type: str,
    accelerator_count: int,
    encryption_spec_key_name: str = '',
):
    custom_job.CustomTrainingJobOp(
        display_name='add-numbers',
        worker_pool_specs=[{
            'container_spec': {
                'image_uri':
                    ('gcr.io/ml-pipeline/google-cloud-pipeline-components:2.5.0'
                    ),
                'command': ['echo'],
                'args': ['foo'],
            },
            'machine_spec': {
                'machine_type': machine_type,
                'accelerator_type': accelerator_type,
                'accelerator_count': accelerator_count,
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
