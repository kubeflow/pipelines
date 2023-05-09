import kfp
from kfp import components
from kfp import dsl

sagemaker_MonitoringSchedule_op = components.load_component_from_file(
    "../../MonitoringSchedule/component.yaml"
)
sagemaker_ModelBiasJobDefinition_op = components.load_component_from_file(
    "../../ModelBiasJobDefinition/component.yaml"
)


@dsl.pipeline(
    name="MonitoringSchedule", description="SageMaker MonitoringSchedule component"
)
def MonitoringSchedule(
    region="",
    model_bias_app_specification="",
    model_bias_job_input="",
    model_bias_job_output_config="",
    job_definition_name="",
    job_resources="",
    role_arn="",
    stopping_condition="",
    monitoring_schedule_name="",
    monitoring_schedule_config="",
):
    ModelBiasJobDefinition = sagemaker_ModelBiasJobDefinition_op(
        region=region,
        job_definition_name=job_definition_name,
        job_resources=job_resources,
        model_bias_app_specification=model_bias_app_specification,
        model_bias_job_input=model_bias_job_input,
        model_bias_job_output_config=model_bias_job_output_config,
        role_arn=role_arn,
        stopping_condition=stopping_condition,
    )

    MonitoringSchedule = sagemaker_MonitoringSchedule_op(
        region=region,
        monitoring_schedule_config=monitoring_schedule_config,
        monitoring_schedule_name=monitoring_schedule_name,
    ).after(ModelBiasJobDefinition)


kfp.compiler.Compiler().compile(MonitoringSchedule, __file__ + ".tar.gz")
