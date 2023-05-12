import kfp
from kfp import components
from kfp import dsl

sagemaker_ModelExplainabilityJobDefinition_op = components.load_component_from_file(
    "../../ModelExplainabilityJobDefinition/component.yaml"
)


@dsl.pipeline(
    name="ModelExplainabilityJobDefinition",
    description="SageMaker ModelExplainabilityJobDefinition component",
)
def ModelExplainabilityJobDefinition(
    region="",
    model_explainability_app_specification="",
    model_explainability_job_input="",
    model_explainability_job_output_config="",
    job_definition_name="",
    job_resources="",
    role_arn="",
    stopping_condition="",
):
    DataQualityJobDefinition = sagemaker_ModelExplainabilityJobDefinition_op(
        region=region,
        model_explainability_app_specification=model_explainability_app_specification,
        model_explainability_job_input=model_explainability_job_input,
        model_explainability_job_output_config=model_explainability_job_output_config,
        job_definition_name=job_definition_name,
        job_resources=job_resources,
        role_arn=role_arn,
        stopping_condition=stopping_condition,
    )


kfp.compiler.Compiler().compile(ModelExplainabilityJobDefinition, __file__ + ".tar.gz")
