# flake8: noqa TODO

from kfp.components import InputPath, OutputPath


def ImportExampleGen(
    input_base_path: InputPath('ExternalPath'),
    #input_path: InputPath('ExternalPath'),

    example_artifacts_path: OutputPath('Examples'),

    input_config: 'JsonObject: example_gen_pb2.Input' = None,
    output_config: 'JsonObject: example_gen_pb2.Output' = None,
):
    """
    Official TFX ImportExampleGen component.

    The ImportExampleGen component takes TFRecord files with TF Example data
    format, and generates train and eval examples for downsteam components.
    This component provides consistent and configurable partition, and it also
    shuffle the dataset for ML best practice.

    Args:
        input_base: A Channel of 'ExternalPath' type, which includes one artifact
            whose uri is an external directory with TFRecord files inside
            (required).
        #input: Forwards compatibility alias for the 'input_base' argument.
        input_config: An example_gen_pb2.Input instance, providing input
            configuration. If unset, the files under input_base will be treated as a
            single split.
        output_config: An example_gen_pb2.Output instance, providing output
            configuration. If unset, default splits will be 'train' and 'eval' with
            size 2:1.
    Returns:
        example_artifacts: Optional channel of 'ExamplesPath' for output train and
            eval examples.

    Raises:
        RuntimeError: Only one of query and input_config should be set.
    """
    from tfx.components.example_gen.import_example_gen.component import ImportExampleGen
    component_class = ImportExampleGen
    input_channels_with_splits = {}
    output_channels_with_splits = {'example_artifacts'}


    import json
    import os
    from google.protobuf import json_format, message
    from tfx.types import Artifact, channel_utils

    arguments = locals().copy()

    component_class_args = {}

    for name, execution_parameter in component_class.SPEC_CLASS.PARAMETERS.items():
        argument_value_obj = argument_value = arguments.get(name, None)
        if argument_value is None:
            continue
        parameter_type = execution_parameter.type
        if isinstance(parameter_type, type) and issubclass(parameter_type, message.Message): # execution_parameter.type can also be a tuple
            argument_value_obj = parameter_type()
            json_format.Parse(argument_value, argument_value_obj)
        component_class_args[name] = argument_value_obj

    for name, channel_parameter in component_class.SPEC_CLASS.INPUTS.items():
        artifact_path = arguments[name + '_path']
        artifacts = []
        if name in input_channels_with_splits:
            # Recovering splits
            splits = sorted(os.listdir(artifact_path))
            for split in splits:
                artifact = Artifact(type_name=channel_parameter.type_name)
                artifact.split = split
                artifact.uri = os.path.join(artifact_path, split) + '/'
                artifacts.append(artifact)
        else:
            artifact = Artifact(type_name=channel_parameter.type_name)
            artifact.uri = artifact_path + '/' # ?
            artifacts.append(artifact)
        component_class_args[name] = channel_utils.as_channel(artifacts)

    component_class_instance = component_class(**component_class_args)

    input_dict = {name: channel.get() for name, channel in component_class_instance.inputs.get_all().items()}
    output_dict = {name: channel.get() for name, channel in component_class_instance.outputs.get_all().items()}
    exec_properties = component_class_instance.exec_properties

    # Generating paths for output artifacts
    for name, artifacts in output_dict.items():
        base_artifact_path = arguments[name + '_path']
        for artifact in artifacts:
            artifact.uri = os.path.join(base_artifact_path, artifact.split) # Default split is ''

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
        ImportExampleGen,
        base_image='tensorflow/tfx:0.15.0',
        output_component_file='component.yaml'
    )
