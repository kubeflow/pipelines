import kfp
from kfp import components
from kfp import dsl

sagemaker_DataQualityJobDefinition_op = components.load_component_from_file(
    "../../DataQualityJobDefinition/component.yaml"
)


@dsl.pipeline(
    name="DataQualityJobDefinition",
    description="SageMaker DataQualityJobDefinition component",
)
def DataQualityJobDefinition(
    region="",
    data_quality_app_specification="",
    data_quality_baseline_config="",
    data_quality_job_input="",
    data_quality_job_output_config="",
    job_definition_name="",
    job_resources="",
    role_arn="",
    stopping_condition="",
):
    DataQualityJobDefinition = sagemaker_DataQualityJobDefinition_op(
        region=region,
        data_quality_app_specification=data_quality_app_specification,
        data_quality_baseline_config=data_quality_baseline_config,
        data_quality_job_input=data_quality_job_input,
        data_quality_job_output_config=data_quality_job_output_config,
        job_definition_name=job_definition_name,
        job_resources=job_resources,
        role_arn=role_arn,
        stopping_condition=stopping_condition,
    )


kfp.compiler.Compiler().compile(DataQualityJobDefinition, __file__ + ".tar.gz")
