import kfp
from kfp import components
from kfp import dsl

sagemaker_model_op = components.load_component_from_file("../../model/component.yaml")
sagemaker_batch_transform_op = components.load_component_from_file(
    "../../batch_transform/component.yaml"
)


@dsl.pipeline(
    name="Batch Transform Job in SageMaker",
    description="SageMaker batch transform component test",
)
def batch_transform_pipeline(
    region="",
    image="",
    model_name="",
    job_name="",
    model_artifact_url="",
    instance_type="",
    instance_count="",
    data_input="",
    data_type="",
    content_type="",
    compression_type="",
    output_location="",
    max_concurrent="",
    max_payload="",
    batch_strategy="",
    split_type="",
    network_isolation="",
    role="",
):
    create_model = sagemaker_model_op(
        region=region,
        model_name=model_name,
        image=image,
        model_artifact_url=model_artifact_url,
        network_isolation=network_isolation,
        role=role,
    )

    sagemaker_batch_transform_op(
        region=region,
        model_name=create_model.output,
        job_name=job_name,
        instance_type=instance_type,
        instance_count=instance_count,
        max_concurrent=max_concurrent,
        max_payload=max_payload,
        batch_strategy=batch_strategy,
        input_location=data_input,
        data_type=data_type,
        content_type=content_type,
        split_type=split_type,
        compression_type=compression_type,
        output_location=output_location,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        batch_transform_pipeline, "SageMaker_batch_transform" + ".yaml"
    )
