import os
import re
from typing import NamedTuple

from kfp.components import create_component_from_func, InputPath, OutputPath


def upload_Tensorflow_model_to_Google_Cloud_Vertex_AI(
    model_path: InputPath("TensorflowSavedModel"),
    tensorflow_version: str = None,
    use_gpu: bool = False,

    display_name: str = None,
    description: str = None,

    instance_schema_uri: str = None,
    parameters_schema_uri: str = None,
    prediction_schema_uri: str = None,
    explanation_metadata: dict = None,  # "google.cloud.aiplatform_v1.types.explanation_metadata.ExplanationMetadata"
    explanation_parameters: dict = None,  # "google.cloud.aiplatform_v1.types.explanation.ExplanationParameters"

    project: str = None,
    location: str = None,
    labels: dict = None,
    encryption_spec_key_name: str = None,
    staging_bucket: str = None,
) -> NamedTuple("Outputs", [
    ("model_name", str),
    ("model_dict", dict),
]):
    """Imports TensorFlow model into Vertex AI Model registry.
    For more details, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/upload.

    Args:
        model_path:
            TensorFlow SavedModel directory
        tensorflow_version:
            TensorFlow version
        use_gpu:
            Whether to use GPU when serving

        display_name:
            Required. The display name of the Model. The name can be up to 128
            characters long and can be consist of any UTF-8 characters.
        description:
            The description of the model.

        instance_schema_uri:
            Points to a YAML file stored on Google Cloud
            Storage describing the format of a single instance, which
            are used in
            ``PredictRequest.instances``,
            ``ExplainRequest.instances``
            and
            ``BatchPredictionJob.input_config``.
            The schema is defined as an OpenAPI 3.0.2 `Schema
            Object <https://tinyurl.com/y538mdwt#schema-object>`__.
            AutoML Models always have this field populated by AI
            Platform. Note: The URI given on output will be immutable
            and probably different, including the URI scheme, than the
            one given on input. The output URI will point to a location
            where the user only has a read access.

            For more details on PredictionSchema, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models#predictschemata.
        parameters_schema_uri:
            Points to a YAML file stored on Google Cloud
            Storage describing the parameters of prediction and
            explanation via
            ``PredictRequest.parameters``,
            ``ExplainRequest.parameters``
            and
            ``BatchPredictionJob.model_parameters``.
            The schema is defined as an OpenAPI 3.0.2 `Schema
            Object <https://tinyurl.com/y538mdwt#schema-object>`__.
            AutoML Models always have this field populated by AI
            Platform, if no parameters are supported it is set to an
            empty string. Note: The URI given on output will be
            immutable and probably different, including the URI scheme,
            than the one given on input. The output URI will point to a
            location where the user only has a read access.

            For more details on PredictionSchema, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models#predictschemata.
        prediction_schema_uri:
            Points to a YAML file stored on Google Cloud
            Storage describing the format of a single prediction
            produced by this Model, which are returned via
            ``PredictResponse.predictions``,
            ``ExplainResponse.explanations``,
            and
            ``BatchPredictionJob.output_config``.
            The schema is defined as an OpenAPI 3.0.2 `Schema
            Object <https://tinyurl.com/y538mdwt#schema-object>`__.
            AutoML Models always have this field populated by AI
            Platform. Note: The URI given on output will be immutable
            and probably different, including the URI scheme, than the
            one given on input. The output URI will point to a location
            where the user only has a read access.

            For more details on PredictionSchema, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models#predictschemata
        explanation_metadata (Optional[dict]):
            Metadata describing the Model's input and output for explanation.
            Both `explanation_metadata` and `explanation_parameters` must be
            passed together when used.

            For more details, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#explanationmetadata.
        explanation_parameters (Optional[dict]):
            Parameters to configure explaining for Model's predictions.

            For more details, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#explanationmetadata.

        project:
            Required. Project to upload this model to.
        location:
            Optional location to upload this model to. If not set,
            default to us-central1.
        labels:
            The labels with user-defined metadata to organize your model.

            Label keys and values can be no longer than
            64 characters (Unicode codepoints), can only contain lowercase
            letters, numeric characters, underscores and dashes.
            International characters are allowed.

            See https://goo.gl/xmQnxf for more information and examples of labels.
        encryption_spec_key_name:
            Customer-managed encryption key spec for a Model.
            If set, this Model and all sub-resources of this Model will
            be secured by this key.

            Has the form:
            ``projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key``.
            The key needs to be in the same region as where the compute
            resource is created.

    Returns:
        model_name:
            Vertex Model resource name.
        model_dict:
            Representation of the created Vertex Model.
    """
    import json
    import os
    from google.cloud import aiplatform

    if not location:
        location = os.environ.get("CLOUD_ML_REGION")

    if not labels:
        labels = {}
    labels["google-ready-to-go-vertex"] = "1"

    model = aiplatform.Model.upload_tensorflow_saved_model(
        saved_model_dir=model_path,
        tensorflow_version=tensorflow_version,
        use_gpu=use_gpu,

        display_name=display_name,
        description=description,

        instance_schema_uri=instance_schema_uri,
        parameters_schema_uri=parameters_schema_uri,
        prediction_schema_uri=prediction_schema_uri,
        explanation_metadata=explanation_metadata,
        explanation_parameters=explanation_parameters,

        project=project,
        location=location,
        labels=labels,
        encryption_spec_key_name=encryption_spec_key_name,
        staging_bucket=staging_bucket,
    )
    model_json = json.dumps(model.to_dict(), indent=2)
    print(model_json)
    return (model.resource_name, model_json)


if __name__ == "__main__":
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    upload_Tensorflow_model_to_Google_Cloud_Vertex_AI_op = create_component_from_func(
        func=upload_Tensorflow_model_to_Google_Cloud_Vertex_AI,
        base_image="gcr.io/ml-pipeline/google-cloud-pipeline-components:latest",
        packages_to_install=[],
        output_component_file="component.yaml",
    )
