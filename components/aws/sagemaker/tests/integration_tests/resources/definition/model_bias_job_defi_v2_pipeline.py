import kfp
from kfp import components
from kfp import dsl

sagemaker_ModelBiasJobDefinition_op = components.load_component_from_file(
    "../../ModelBiasJobDefinition/component.yaml"
)


@dsl.pipeline(
    name="ModelBiasJobDefinition",
    description="SageMaker ModelBiasJobDefinition component",
)
def ModelBiasJobDefinition(
    region="",
    model_bias_app_specification="",
    model_bias_baseline_config="",
    model_bias_job_input="",
    model_bias_job_output_config="",
    job_definition_name="",
    job_resources="",
    role_arn="",
    stopping_condition="",
):
    DataQualityJobDefinition = sagemaker_ModelBiasJobDefinition_op(
        region=region,
        model_bias_app_specification=model_bias_app_specification,
        model_bias_baseline_config=model_bias_baseline_config,
        model_bias_job_input=model_bias_job_input,
        model_bias_job_output_config=model_bias_job_output_config,
        job_definition_name=job_definition_name,
        job_resources=job_resources,
        role_arn=role_arn,
        stopping_condition=stopping_condition,
    )


kfp.compiler.Compiler().compile(ModelBiasJobDefinition, __file__ + ".tar.gz")
