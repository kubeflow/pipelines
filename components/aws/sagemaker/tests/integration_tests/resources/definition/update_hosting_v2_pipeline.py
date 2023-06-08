import kfp
from kfp import components
from kfp import dsl

sagemaker_Model_op = components.load_component_from_file("../../Modelv2/component.yaml")

sagemaker_EndpointConfig_op = components.load_component_from_file(
    "../../EndpointConfig/component.yaml"
)

sagemaker_Endpoint_op = components.load_component_from_file(
    "../../Endpoint/component.yaml"
)


@dsl.pipeline(name="Update Hosting", description="SageMaker Hosting")
def UpdateHosting(
    region="",
    execution_role_arn="",
    model_name="",
    primary_container="",
    endpoint_config_name="",
    production_variants="",
    endpoint_name="",
    second_endpoint_config_name="",
    second_production_variants="",
):
    Model = sagemaker_Model_op(
        region=region,
        execution_role_arn=execution_role_arn,
        model_name=model_name,
        primary_container=primary_container,
    )
    EndpointConfig = sagemaker_EndpointConfig_op(
        region=region,
        endpoint_config_name=endpoint_config_name,
        production_variants=production_variants,
    ).after(Model)

    Endpoint = sagemaker_Endpoint_op(
        region=region,
        endpoint_config_name=endpoint_config_name,
        endpoint_name=endpoint_name,
    ).after(EndpointConfig)

    SecondEndpointConfig = sagemaker_EndpointConfig_op(
        region=region,
        endpoint_config_name=second_endpoint_config_name,
        production_variants=second_production_variants,
    ).after(Model)

    EndpointUpdate = sagemaker_Endpoint_op(
        region=region,
        endpoint_config_name=second_endpoint_config_name,
        endpoint_name=endpoint_name,
    ).after(Endpoint)


kfp.compiler.Compiler().compile(UpdateHosting, __file__ + ".tar.gz")
