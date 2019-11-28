# flake8: noqa TODO

from typing import NamedTuple

def CsvExampleGen_GCS( #
    # Inputs
    #input_base_path: InputPath('ExternalPath'),
    input_base_path: 'ExternalPath', # A Channel of 'ExternalPath' type, which includes one artifact whose uri is an external directory with csv files inside (required).

    # Outputs
    #example_artifacts_path: OutputPath('ExamplesPath'),
    example_artifacts_path: 'ExamplesPath',

    # Execution properties
    #input_config_splits: {'List' : {'item_type': 'ExampleGen.Input.Split'}},
    input_config: 'ExampleGen.Input' = None, # = '{"splits": []}', # JSON-serialized example_gen_pb2.Input instance, providing input configuration. If unset, the files under input_base will be treated as a single split.
    #output_config_splits: {'List' : {'item_type': 'ExampleGen.SplitConfig'}},
    output_config: 'ExampleGen.Output' = None, # = '{"splitConfig": {"splits": []}}', # JSON-serialized example_gen_pb2.Output instance, providing output configuration. If unset, default splits will be 'train' and 'eval' with size 2:1.
    #custom_config: 'ExampleGen.CustomConfig' = None,
) -> NamedTuple('Outputs', [
    ('example_artifacts', 'ExamplesPath'),
]):
    """Executes the CsvExampleGen component.

    Args:
      input_base: A Channel of 'ExternalPath' type, which includes one artifact
        whose uri is an external directory with csv files inside (required).
      input_config: An example_gen_pb2.Input instance, providing input
        configuration. If unset, the files under input_base will be treated as a
        single split.
      output_config: An example_gen_pb2.Output instance, providing output
        configuration. If unset, default splits will be 'train' and 'eval' with
        size 2:1.
      ??? input: Forwards compatibility alias for the 'input_base' argument.
    Returns:
      example_artifacts: Artifact of type 'ExamplesPath' for output train and
        eval examples.
    """

    import json
    import os
    from google.protobuf import json_format
    from tfx.components.example_gen.csv_example_gen.component import CsvExampleGen
    from tfx.proto import example_gen_pb2
    from tfx.types import standard_artifacts
    from tfx.types import channel_utils

    # Create input dict.
    input_base = standard_artifacts.ExternalArtifact()
    input_base.uri = input_base_path
    input_base_channel = channel_utils.as_channel([input_base])

    input_config_obj = None
    if input_config:
        input_config_obj = example_gen_pb2.Input()
        json_format.Parse(input_config, input_config_obj)

    output_config_obj = None
    if output_config:
        output_config_obj = example_gen_pb2.Output()
        json_format.Parse(output_config, output_config_obj)

    component_class_instance = CsvExampleGen(
        input=input_base_channel,
        input_config=input_config_obj,
        output_config=output_config_obj,
    )

    # component_class_instance.inputs/outputs are wrappers that do not behave like real dictionaries. The underlying dict can be accessed using .get_all()
    # Channel artifacts can be accessed by calling .get()
    input_dict = {name: channel.get() for name, channel in component_class_instance.inputs.get_all().items()}
    output_dict = {name: channel.get() for name, channel in component_class_instance.outputs.get_all().items()}
    exec_properties = component_class_instance.exec_properties

    # Generating paths for output artifacts
    for output_artifact in output_dict['examples']:
        output_artifact.uri = example_artifacts_path
        if output_artifact.split:
            output_artifact.uri = os.path.join(output_artifact.uri, output_artifact.split)

    print('component instance: ' + str(component_class_instance))

    executor = CsvExampleGen.EXECUTOR_SPEC.executor_class()
    executor.Do(
        input_dict=input_dict,
        output_dict=output_dict,
        exec_properties=exec_properties,
    )

    return (example_artifacts_path,)

if __name__ == '__main__':
    import kfp
    kfp.components.func_to_container_op(
        CsvExampleGen_GCS,
        base_image='tensorflow/tfx:0.15.0',
        output_component_file='component.yaml'
    )
