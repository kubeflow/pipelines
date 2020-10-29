import kfp
from kfp import components
from kfp import dsl

sagemaker_process_op = components.load_component_from_file(
    "../../process/component.yaml"
)


@dsl.pipeline(name="SageMaker Processing", description="SageMaker processing job test")
def processing_pipeline(
    region="",
    job_name="",
    image="",
    instance_type="",
    instance_count="",
    volume_size="",
    max_run_time="",
    environment={},
    container_entrypoint=[],
    container_arguments=[],
    input_config={},
    output_config={},
    network_isolation=False,
    role="",
    assume_role="",
):
    sagemaker_process_op(
        region=region,
        job_name=job_name,
        image=image,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_run_time=max_run_time,
        environment=environment,
        container_entrypoint=container_entrypoint,
        container_arguments=container_arguments,
        input_config=input_config,
        output_config=output_config,
        network_isolation=network_isolation,
        role=role,
        assume_role=assume_role,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        processing_pipeline, "SageMaker_processing_pipeline" + ".yaml"
    )
