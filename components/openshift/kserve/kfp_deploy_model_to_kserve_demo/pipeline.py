"""Example of a simple pipeline creating a kserve inference service"""

from kfp import dsl
import os


@dsl.pipeline(
    name='KServe Pipeline',
    description='A pipeline for creating a KServe inference service.'
)
def kserve_pipeline():
    from kfp import components

    namespace = os.getenv('NAMESPACE')

    kserve_op = components.load_component_from_url(
        'https://raw.githubusercontent.com/hbelmiro/kfp_deploy_model_to_kserve_demo/refs/heads/main/component.yaml')
    kserve_op(
        action='apply',
        namespace=namespace,
        model_name='example',
        model_uri='gs://kfserving-examples/models/sklearn/1.0/model',
        framework='sklearn'
    )
