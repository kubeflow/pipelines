from typing import Dict

from google_cloud_pipeline_components.types.artifact_types import \
    VertexEndpoint
from kfp import compiler
from kfp.components import placeholders
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Output
from kfp.dsl import OutputPath

# from kfp.dsl import Model


@container_component
def endpoint_create(
    project: str,
    display_name: str,
    gcp_resources: OutputPath(str),
    endpoint: Output[VertexEndpoint],
    location: str = 'us-central1',
    description: str = '',
    labels: Dict[str, str] = '{}',
    encryption_spec_key_name: str = '',
    network: str = '',
):
    return ContainerSpec(
        image='gcr.io/ml-pipeline/google-cloud-pipeline-components:1.0.17',
        command=[
            'python3', '-u', '-m',
            'google_cloud_pipeline_components.container.v1.gcp_launcher.launcher'
        ],
        args=[
            '--type',
            'CreateEndpoint',
            '--payload',
            placeholders.ConcatPlaceholder([
                '{', '"display_name": "', display_name, '"',
                ', "description": "', description, '"', ', "labels": ', labels,
                ', "encryption_spec": {"kms_key_name":"',
                encryption_spec_key_name, '"}', ', "network": "', network, '"',
                '}'
            ]),
            '--project',
            project,
            '--location',
            location,
            '--gcp_resources',
            gcp_resources,
            '--executor_input',
            '{{$}}',
        ])


from kfp import dsl


@dsl.container_component
def comp(x: str, input_: dsl.Input[CustomArtifact],
         output_: dsl.Output[CustomArtifact]):
    return dsl.ContainerSpec(image='python:3.7', command=['echo hello world'])


from kfp import compiler

compiler.Compiler().compile(endpoint_create, 'component.yaml')
from kfp import components

x = components.load_component_from_file('component.yaml')
print(x.component_spec)
