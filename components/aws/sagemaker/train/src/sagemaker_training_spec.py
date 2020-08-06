from common.sagemaker_component_spec import SageMakerComponentSpec


class SageMakerTrainingSpec(SageMakerComponentSpec):
    INPUTS = {
        **SageMakerComponentSpec.INPUTS,
        **{
            "job_name": dict(
                type=str,
                required=False,
                help="The name of the training job.",
                default="",
            )
        },
    }

    OUTPUTS = {
        **SageMakerComponentSpec.OUTPUTS,
        **{
            "model_artifact_url_output_path": dict(
                type=str,
                default="/tmp/model-artifact-url",
                help="Local output path for the file containing the model artifacts URL.",
            )
        },
    }

