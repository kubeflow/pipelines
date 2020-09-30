from typing import NamedTuple
from kfp.components import create_component_from_func


def automl_export_model_to_gcs(
    model_path: str,
    gcs_output_uri_prefix: str,
    model_format: str = 'tf_saved_model',
) -> NamedTuple('Outputs', [
    ('model_directory', 'Uri'),
]):
    """Exports a trained model to a user specified Google Cloud Storage location.

    Args:
        model_path: The resource name of the model to export. Format: 'projects/<project>/locations/<location>/models/<model>'
        gcs_output_uri_prefix: The Google Cloud Storage directory where the model should be written to. Must be in the same location as AutoML. Required location: us-central1.
        model_format: The format in which the model must be exported. The available, and default, formats depend on the problem and model type. Possible formats: tf_saved_model, tf_js, tflite, core_ml, edgetpu_tflite. See https://cloud.google.com/automl/docs/reference/rest/v1/projects.locations.models/export?hl=en#modelexportoutputconfig

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    """
    from google.cloud import automl

    client = automl.AutoMlClient()
    response = client.export_model(
        name=model_path,
        output_config=automl.ModelExportOutputConfig(
            model_format=model_format,
            gcs_destination=automl.GcsDestination(
                output_uri_prefix=gcs_output_uri_prefix,
            ),
        ),
    )

    print('Operation started:')
    print(response.operation)
    result = response.result()
    metadata = response.metadata
    print('Operation finished:')
    print(metadata)
    return (metadata.export_model_details.output_info.gcs_output_directory, )


if __name__ == '__main__':
    automl_export_model_to_gcs_op = create_component_from_func(
        automl_export_model_to_gcs,
        output_component_file='component.yaml',
        base_image='python:3.8',
        packages_to_install=[
            'google-cloud-automl==2.0.0',
        ]
    )
