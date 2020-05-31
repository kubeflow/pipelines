from pathlib import Path

import kfp
from kfp.components import load_component_from_file

test_data_dir = Path(__file__).parent / 'test_data'
producer_op = load_component_from_file(str(test_data_dir / 'produce_2.component.yaml'))
processor_op = load_component_from_file(str(test_data_dir / 'process_2_2.component.yaml'))
consumer_op = load_component_from_file(str(test_data_dir / 'consume_2.component.yaml'))


@kfp.dsl.pipeline()
def artifact_passing_pipeline():
    producer_task = producer_op()
    processor_task = processor_op(producer_task.outputs['output_1'], producer_task.outputs['output_2'])
    consumer_task = consumer_op(processor_task.outputs['output_1'], processor_task.outputs['output_2'])

    # This line is only needed for compiling using dsl-compile to work
    kfp.dsl.get_pipeline_conf().data_passing_method = volume_based_data_passing_method


from kubernetes.client.models import V1Volume, V1PersistentVolumeClaimVolumeSource
from kfp.dsl import data_passing_methods


volume_based_data_passing_method = data_passing_methods.KubernetesVolume(
    volume=V1Volume(
        name='data',
        persistent_volume_claim=V1PersistentVolumeClaimVolumeSource(
            claim_name='data-volume',
        ),
    ),
    path_prefix='artifact_data/',
)


if __name__ == '__main__':
    pipeline_conf = kfp.dsl.PipelineConf()
    pipeline_conf.data_passing_method = volume_based_data_passing_method
    kfp.compiler.Compiler().compile(artifact_passing_pipeline, __file__ + '.yaml', pipeline_conf=pipeline_conf)
