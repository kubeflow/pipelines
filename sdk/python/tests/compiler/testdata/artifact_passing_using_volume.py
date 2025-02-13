from pathlib import Path
from typing import NamedTuple

import kfp.deprecated as kfp
from kfp.deprecated.components import create_component_from_func
from kfp.deprecated.components import load_component_from_file

test_data_dir = Path(__file__).parent / 'test_data'
producer_op = load_component_from_file(
    str(test_data_dir / 'produce_2.component.yaml'))
processor_op = load_component_from_file(
    str(test_data_dir / 'process_2_2.component.yaml'))
consumer_op = load_component_from_file(
    str(test_data_dir / 'consume_2.component.yaml'))


def metadata_and_metrics() -> NamedTuple(
    "Outputs",
    [("mlpipeline_ui_metadata", "UI_metadata"), ("mlpipeline_metrics", "Metrics"
                                                )],
):
    metadata = {
        "outputs": [{
            "storage": "inline",
            "source": "*this should be bold*",
            "type": "markdown"
        }]
    }
    metrics = {
        "metrics": [
            {
                "name": "train-accuracy",
                "numberValue": 0.9,
            },
            {
                "name": "test-accuracy",
                "numberValue": 0.7,
            },
        ]
    }
    from collections import namedtuple
    import json

    return namedtuple("output",
                      ["mlpipeline_ui_metadata", "mlpipeline_metrics"])(
                          json.dumps(metadata), json.dumps(metrics))


@kfp.dsl.pipeline()
def artifact_passing_pipeline():
    producer_task = producer_op()
    processor_task = processor_op(producer_task.outputs['output_1'],
                                  producer_task.outputs['output_2'])
    consumer_task = consumer_op(processor_task.outputs['output_1'],
                                processor_task.outputs['output_2'])

    markdown_task = create_component_from_func(func=metadata_and_metrics)()
    # This line is only needed for compiling using dsl-compile to work
    kfp.dsl.get_pipeline_conf(
    ).data_passing_method = volume_based_data_passing_method


from kfp.deprecated.dsl import data_passing_methods
from kubernetes.client.models import V1PersistentVolumeClaimVolumeSource
from kubernetes.client.models import V1Volume

volume_based_data_passing_method = data_passing_methods.KubernetesVolume(
    volume=V1Volume(
        name='data',
        persistent_volume_claim=V1PersistentVolumeClaimVolumeSource(
            claim_name='data-volume',),
    ),
    path_prefix='artifact_data/',
)

if __name__ == '__main__':
    pipeline_conf = kfp.dsl.PipelineConf()
    pipeline_conf.data_passing_method = volume_based_data_passing_method
    kfp.compiler.Compiler().compile(
        artifact_passing_pipeline,
        __file__ + '.yaml',
        pipeline_conf=pipeline_conf)
