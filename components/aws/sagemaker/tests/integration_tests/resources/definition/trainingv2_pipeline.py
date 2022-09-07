import kfp
from kfp import components
from kfp import dsl

sagemaker_TrainingJob_op = components.load_component_from_file("../../TrainingJob/component.yaml")

@dsl.pipeline(name="TrainingJob", description="SageMaker TrainingJob component")
def TrainingJob(
    region="",
    algorithm_specification="",
    checkpoint_config="",
    debug_hook_config="",
    debug_rule_configurations="",
    enable_inter_container_traffic_encryption="",
    enable_managed_spot_training="",
    enable_network_isolation="",
    environment="",
    experiment_config="",
    hyper_parameters="",
    input_data_config="",
    output_data_config="",
    profiler_config="",
    profiler_rule_configurations="",
    resource_config="",
    role_arn="",
    stopping_condition="",
    tags="",
    tensor_board_output_config="",
    training_job_name="",
    vpc_config="",
):
    TrainingJob = sagemaker_TrainingJob_op(
        region=region,
        algorithm_specification=algorithm_specification,
        checkpoint_config=checkpoint_config,
        debug_hook_config=debug_hook_config,
        debug_rule_configurations=debug_rule_configurations,
        enable_inter_container_traffic_encryption=enable_inter_container_traffic_encryption,
        enable_managed_spot_training=enable_managed_spot_training,
        enable_network_isolation=enable_network_isolation,
        environment=environment,
        experiment_config=experiment_config,
        hyper_parameters=hyper_parameters,
        input_data_config=input_data_config,
        output_data_config=output_data_config,
        profiler_config=profiler_config,
        profiler_rule_configurations=profiler_rule_configurations,
        resource_config=resource_config,
        role_arn=role_arn,
        stopping_condition=stopping_condition,
        tags=tags,
        tensor_board_output_config=tensor_board_output_config,
        training_job_name=training_job_name,
        vpc_config=vpc_config,
    )
if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        TrainingJob, "SageMaker_trainingJob_pipeline" + ".yaml"
    )