import kfp
from kfp import components
from kfp import dsl

rlestimator_training_job_op = components.load_component_from_file(
    "../../rlestimator/component.yaml"
)


@dsl.pipeline(
    name="RLEstimator Toolkit & Framework Pipeline test",
    description="RLEstimator training job test where the AWS Docker image is auto-selected based on the Toolkit and Framework we define",
)
def rlestimator_training_toolkit_pipeline_test(
    region="",
    entry_point="",
    source_dir="",
    toolkit="",
    toolkit_version="",
    framework="",
    role="",
    instance_type="",
    instance_count="",
    model_artifact_path="",
    job_name="",
    metric_definitions="",
    max_run="",
    hyperparameters="",
):
    rlestimator_training_job_op(
        region=region,
        entry_point=entry_point,
        source_dir=source_dir,
        toolkit=toolkit,
        toolkit_version=toolkit_version,
        framework=framework,
        role=role,
        instance_type=instance_type,
        instance_count=instance_count,
        model_artifact_path=model_artifact_path,
        job_name=job_name,
        metric_definitions=metric_definitions,
        max_run=max_run,
        hyperparameters=hyperparameters,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        rlestimator_training_toolkit_pipeline_test, __file__ + ".zip"
    )
