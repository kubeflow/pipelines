import kfp
from kfp import components
from kfp import dsl


sagemaker_hpo_op = components.load_component_from_file(
    "../../hyperparameter_tuning/component.yaml"
)


@dsl.pipeline(
    name="SageMaker HyperParameter Tuning", description="SageMaker HPO job test"
)
def hpo_pipeline(
    region="",
    job_name="",
    algorithm_name="",
    training_input_mode="",
    static_parameters="",
    integer_parameters="",
    channels="",
    categorical_parameters="",
    early_stopping_type="",
    max_parallel_jobs="",
    max_num_jobs="",
    metric_name="",
    metric_type="",
    hpo_strategy="",
    instance_type="",
    instance_count="",
    volume_size="",
    max_run_time="",
    output_location="",
    network_isolation="",
    max_wait_time="",
    role="",
):
    sagemaker_hpo_op(
        region=region,
        job_name=job_name,
        algorithm_name=algorithm_name,
        training_input_mode=training_input_mode,
        static_parameters=static_parameters,
        integer_parameters=integer_parameters,
        channels=channels,
        categorical_parameters=categorical_parameters,
        early_stopping_type=early_stopping_type,
        max_parallel_jobs=max_parallel_jobs,
        max_num_jobs=max_num_jobs,
        metric_name=metric_name,
        metric_type=metric_type,
        strategy=hpo_strategy,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_run_time=max_run_time,
        output_location=output_location,
        network_isolation=network_isolation,
        max_wait_time=max_wait_time,
        role=role,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        hpo_pipeline, "SageMaker_hyperparameter_tuning_pipeline" + ".yaml"
    )
