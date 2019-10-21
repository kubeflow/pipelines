from kfp.components import InputPath, OutputPath

def CsvExampleGen(
    # Inputs
    input_base_path: InputPath('ExternalPath'),
    #input_base_path: 'ExternalPath', # A Channel of 'ExternalPath' type, which includes one artifact whose uri is an external directory with csv files inside (required).

    # Outputs
    output_examples_path: OutputPath('ExamplesPath'),
    #output_examples_path: 'ExamplesPath',

    # Execution properties
    #input_config_splits: {'List' : {'item_type': 'ExampleGen.Input.Split'}},
    input_config: 'ExampleGen.Input' = '{"splits": []}', # JSON-serialized example_gen_pb2.Input instance, providing input configuration. If unset, the files under input_base will be treated as a single split.
    #output_config_splits: {'List' : {'item_type': 'ExampleGen.SplitConfig'}},
    output_config: 'ExampleGen.Output' = '{"splitConfig": {"splits": []}}', # JSON-serialized example_gen_pb2.Output instance, providing output configuration. If unset, default splits will be 'train' and 'eval' with size 2:1.
    #custom_config: 'ExampleGen.CustomConfig' = None,
):
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
      ??? example_artifacts: Optional channel of 'ExamplesPath' for output train and
        eval examples.
      ??? input: Forwards compatibility alias for the 'input_base' argument.
      ??? instance_name: Optional unique instance name. Necessary if multiple
        CsvExampleGen components are declared in the same pipeline.
    """

    import json
    import os
    from google.protobuf import json_format
    from tfx.components.example_gen import utils
    from tfx.components.example_gen.csv_example_gen.component import CsvExampleGen
    from tfx.proto import example_gen_pb2
    from tfx.types import standard_artifacts

    # Create input dict.
    # input_dict['input_base'] always has a single entry
    input_base = standard_artifacts.ExternalArtifact()
    input_base.uri = input_base_path
    input_dict = {
        'input_base': [input_base],
    }

    # Create output dict.
    input_config_obj = example_gen_pb2.Input()
    output_config_obj = example_gen_pb2.Output()
    json_format.Parse(input_config, input_config_obj)
    json_format.Parse(output_config, output_config_obj)
    split_names = utils.generate_output_split_names(input_config_obj, output_config_obj)
    output_dict_examples = []
    for split_name in split_names:
        output_split_examples = standard_artifacts.Examples(split=split_name)
        output_split_examples.uri = os.path.join(output_examples_path, split_name)
        output_dict_examples.append(output_split_examples)
    output_dict = {
        'examples': output_dict_examples,
    }

    # Create exec proterties.
    exec_properties = {
        'input_config': input_config,
        'output_config': output_config
    }

    executor = CsvExampleGen.EXECUTOR_SPEC.executor_class()
    executor.Do(
        input_dict=input_dict,
        output_dict=output_dict,
        exec_properties=exec_properties,
    )


if __name__ == '__main__':
    import kfp
    kfp.components.func_to_container_op(
        CsvExampleGen,
        base_image='tensorflow/tensorflow:1.14.0-py3',
        packages_to_install=['tfx==0.14', 'six>=1.12.0'],
        output_component_file='CsvExampleGen.component.yaml'
    )