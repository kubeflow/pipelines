import kfp
from kfp import components
from kfp import dsl

sagemaker_MonitoringSchedule_op = components.load_component_from_file(
    "../../MonitoringSchedule/component.yaml"
)
sagemaker_DataQualityJobDefinition_op = components.load_component_from_file(
    "../../DataQualityJobDefinition/component.yaml"
)


@dsl.pipeline(
    name="MonitoringSchedule", description="SageMaker MonitoringSchedule component"
)
def MonitoringSchedule(
    region="",
    data_quality_app_specification="",
    data_quality_baseline_config="",
    data_quality_job_input="",
    data_quality_job_output_config="",
    job_definition_name="",
    job_resources="",
    role_arn="",
    stopping_condition="",
    monitoring_schedule_name="",
    monitoring_type="",
    schedule_expression="",
):
    DataQualityJobDefinition = sagemaker_DataQualityJobDefinition_op(
        region=region,
        job_definition_name=job_definition_name,
        job_resources=job_resources,
        data_quality_app_specification=data_quality_app_specification,
        data_quality_baseline_config=data_quality_baseline_config,
        data_quality_job_input=data_quality_job_input,
        data_quality_job_output_config=data_quality_job_output_config,
        role_arn=role_arn,
        stopping_condition=stopping_condition,
    )

    monitoring_schedule_config = {
        "monitoringType": monitoring_type,
        "scheduleConfig": {"scheduleExpression": schedule_expression},
        "monitoringJobDefinitionName": DataQualityJobDefinition.outputs[
            "sagemaker_resource_name"
        ],
    }

    MonitoringSchedule = sagemaker_MonitoringSchedule_op(
        region=region,
        monitoring_schedule_config=monitoring_schedule_config,
        monitoring_schedule_name=monitoring_schedule_name,
    )


kfp.compiler.Compiler().compile(MonitoringSchedule, __file__ + ".tar.gz")
