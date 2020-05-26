# flake8: noqa TODO

from kfp.components import InputPath, OutputPath


def BigQueryExampleGen(
    examples_path: OutputPath('Examples'),

    query: str = None,
    input_config: {'JsonObject': {'data_type': 'proto:tfx.components.example_gen.Input'}} = None,
    output_config: {'JsonObject': {'data_type': 'proto:tfx.components.example_gen.Output'}} = None,
    custom_config: {'JsonObject': {'data_type': 'proto:tfx.components.example_gen.CustomConfig'}} = None,
):
    """
    Official TFX BigQueryExampleGen component.

    The BigQuery examplegen component takes a query, and generates train
    and eval examples for downsteam components.


    Args:
        query: BigQuery sql string, query result will be treated as a single
            split, can be overwritten by input_config.
        input_config: An example_gen_pb2.Input instance with Split.pattern as
            BigQuery sql string. If set, it overwrites the 'query' arg, and allows
            different queries per split.
        output_config: An example_gen_pb2.Output instance, providing output
            configuration. If unset, default splits will be 'train' and 'eval' with
            size 2:1.
    Returns:
        examples: Optional channel of 'ExamplesPath' for output train and
            eval examples.

    Raises:
        RuntimeError: Only one of query and input_config should be set.
    """
    from tfx.components.example_gen.csv_example_gen.component import BigQueryExampleGen as component_class

    #Generated code
    import json
    import os
    import tensorflow
    from google.protobuf import json_format, message
    from tfx.types import Artifact, channel_utils, artifact_utils

    arguments = locals().copy()

    component_class_args = {}

    for name, execution_parameter in component_class.SPEC_CLASS.PARAMETERS.items():
        argument_value_obj = argument_value = arguments.get(name, None)
        if argument_value is None:
            continue
        parameter_type = execution_parameter.type
        if isinstance(parameter_type, type) and issubclass(parameter_type, message.Message): # Maybe FIX: execution_parameter.type can also be a tuple
            argument_value_obj = parameter_type()
            json_format.Parse(argument_value, argument_value_obj)
        component_class_args[name] = argument_value_obj

    for name, channel_parameter in component_class.SPEC_CLASS.INPUTS.items():
        artifact_path = arguments[name + '_path']
        if artifact_path:
            artifact = channel_parameter.type()
            artifact.uri = artifact_path + '/' # ?
            if channel_parameter.type.PROPERTIES and 'split_names' in channel_parameter.type.PROPERTIES:
                # Recovering splits
                subdirs = tensorflow.io.gfile.listdir(artifact_path)
                artifact.split_names = artifact_utils.encode_split_names(sorted(subdirs))
            component_class_args[name] = channel_utils.as_channel([artifact])

    component_class_instance = component_class(**component_class_args)

    input_dict = {name: channel.get() for name, channel in component_class_instance.inputs.get_all().items()}
    output_dict = {name: channel.get() for name, channel in component_class_instance.outputs.get_all().items()}
    exec_properties = component_class_instance.exec_properties

    # Generating paths for output artifacts
    for name, artifacts in output_dict.items():
        base_artifact_path = arguments[name + '_path']
        # Are there still cases where output channel has multiple artifacts?
        for idx, artifact in enumerate(artifacts):
            subdir = str(idx + 1) if idx > 0 else ''
            artifact.uri = os.path.join(base_artifact_path, subdir)  # Ends with '/'

    print('component instance: ' + str(component_class_instance))

    #executor = component_class.EXECUTOR_SPEC.executor_class() # Same
    executor = component_class_instance.executor_spec.executor_class()
    executor.Do(
        input_dict=input_dict,
        output_dict=output_dict,
        exec_properties=exec_properties,
    )


if __name__ == '__main__':
    import kfp
    kfp.components.func_to_container_op(
        BigQueryExampleGen,
        base_image='tensorflow/tfx:0.21.4',
        output_component_file='component.yaml'
    )
