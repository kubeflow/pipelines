import kfp
from kfp import components
from kfp import dsl

sagemaker_model_op = components.load_component_from_file("../../model/component.yaml")
sagemaker_deploy_op = components.load_component_from_file("../../deploy/component.yaml")


@dsl.pipeline(
    name="Create Hosting Endpoint in SageMaker",
    description="SageMaker deploy component test",
)
def update_endpoint_pipeline(
    region="",
    endpoint_url="",
    image="",
    model_name="",
    endpoint_config_name="",
    endpoint_name="",
    model_artifact_url="",
    variant_name_1="",
    instance_type_1="",
    instance_type_2="",
    initial_instance_count_1="",
    initial_variant_weight_1="",
    network_isolation="",
    role="",
    update_endpoint="",
):
    create_model = sagemaker_model_op(
        region=region,
        endpoint_url=endpoint_url,
        model_name=model_name,
        image=image,
        model_artifact_url=model_artifact_url,
        network_isolation=network_isolation,
        role=role,
    )

    deploy_model = sagemaker_deploy_op(
        region=region,
        endpoint_url=endpoint_url,
        endpoint_config_name=endpoint_config_name,
        endpoint_name=endpoint_name,
        model_name_1=create_model.output,
        variant_name_1=variant_name_1,
        instance_type_1=instance_type_1,
        initial_instance_count_1=initial_instance_count_1,
        initial_variant_weight_1=initial_variant_weight_1,
    )

    update_model = sagemaker_deploy_op(
        region=region,
        endpoint_url=endpoint_url,
        endpoint_config_name=endpoint_config_name,
        endpoint_name=deploy_model.output,
        model_name_1=create_model.output,
        variant_name_1=variant_name_1,
        instance_type_1=instance_type_2,
        initial_instance_count_1=initial_instance_count_1,
        initial_variant_weight_1=initial_variant_weight_1,
        update_endpoint=update_endpoint,
    )

    dsl.get_pipeline_conf().set_image_pull_policy(policy="Always")


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        create_endpoint_pipeline, "SageMaker_update_hosting_pipeline" + ".yaml"
    )
