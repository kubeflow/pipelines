import kfp
from kfp import components
from kfp import dsl

sagemaker_TrainingJob_op = components.load_component_from_file(
    "../../TrainingJob/component.yaml"
)


@dsl.pipeline(name="TrainingJob", description="SageMaker TrainingJob component")
def TrainingJob(
    region="",
    algorithm_specification="",
    enable_inter_container_traffic_encryption="",
    enable_managed_spot_training="",
    enable_network_isolation="",
    hyper_parameters="",
    input_data_config="",
    output_data_config="",
    resource_config="",
    role_arn="",
    stopping_condition="",
    training_job_name="",
):
    TrainingJob = sagemaker_TrainingJob_op(
        region=region,
        algorithm_specification=algorithm_specification,
        enable_inter_container_traffic_encryption=enable_inter_container_traffic_encryption,
        enable_managed_spot_training=enable_managed_spot_training,
        enable_network_isolation=enable_network_isolation,
        hyper_parameters=hyper_parameters,
        input_data_config=input_data_config,
        output_data_config=output_data_config,
        resource_config=resource_config,
        role_arn=role_arn,
        stopping_condition=stopping_condition,
        training_job_name=training_job_name,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        TrainingJob, "SageMaker_trainingJob_pipeline" + ".yaml"
    )
