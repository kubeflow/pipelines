from typing import NamedTuple
from kfp.components import create_component_from_func


def automl_deploy_model(
    model_path: str,
) -> NamedTuple('Outputs', [
    ('model_path', str),
]):
    """Deploys a trained model.

    Args:
        model_path: The resource name of the model to export. Format: 'projects/<project>/locations/<location>/models/<model>'

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    """
    from google.cloud import automl
    client = automl.AutoMlClient()
    response = client.deploy_model(
        name=model_path,
    )
    print('Operation started:')
    print(response.operation)
    result = response.result()
    metadata = response.metadata
    print('Operation finished:')
    print(metadata)
    return (model_path, )


if __name__ == '__main__':
    automl_deploy_model_op = create_component_from_func(
        automl_deploy_model,
        output_component_file='component.yaml',
        base_image='python:3.8',
        packages_to_install=[
            'google-cloud-automl==2.0.0',
        ]
    )
