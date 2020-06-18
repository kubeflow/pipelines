import kfp
from kfp import components
from kfp import dsl

sagemaker_model_op = components.load_component_from_file("../../model/component.yaml")


@dsl.pipeline(
    name="Create Model in SageMaker", description="SageMaker model component test"
)
def create_model_pipeline(
    region="",
    endpoint_url="",
    image="",
    model_name="",
    model_artifact_url="",
    network_isolation="",
    role="",
):
    sagemaker_model_op(
        region=region,
        endpoint_url=endpoint_url,
        model_name=model_name,
        image=image,
        model_artifact_url=model_artifact_url,
        network_isolation=network_isolation,
        role=role,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        create_model_pipeline, "SageMaker_create_model_pipeline" + ".yaml"
    )
