import kfp
from kfp import components
from kfp import dsl

sagemaker_ModelQualityJobDefinition_op = components.load_component_from_file(
    "../../ModelQualityJobDefinition/component.yaml"
)


@dsl.pipeline(
    name="ModelQualityJobDefinition",
    description="SageMaker ModelQualityJobDefinition component",
)
def ModelQualityJobDefinition(
    region="",
    job_definition_name="",
    job_resources="",
    model_quality_app_specification="",
    model_quality_baseline_config="",
    model_quality_job_input="",
    model_quality_job_output_config="",
    network_config="",
    role_arn="",
    stopping_condition="",
):
    DataQualityJobDefinition = sagemaker_ModelQualityJobDefinition_op(
        region=region,
        job_definition_name=job_definition_name,
        job_resources=job_resources,
        model_quality_app_specification=model_quality_app_specification,
        model_quality_baseline_config=model_quality_baseline_config,
        model_quality_job_input=model_quality_job_input,
        model_quality_job_output_config=model_quality_job_output_config,
        network_config=network_config,
        role_arn=role_arn,
        stopping_condition=stopping_condition,
    )


kfp.compiler.Compiler().compile(ModelQualityJobDefinition, __file__ + ".tar.gz")
