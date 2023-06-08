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


@dsl.pipeline(name="CreateHosting", description="SageMaker Hosting")
def Hosting(
    region="",
    execution_role_arn="",
    model_name="",
    primary_container="",
    endpoint_config_name="",
    production_variants="",
    endpoint_name="",
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


kfp.compiler.Compiler().compile(Hosting, __file__ + ".tar.gz")
