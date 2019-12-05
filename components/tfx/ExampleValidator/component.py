from kfp.components import InputPath, OutputPath


def ExampleValidator(
    stats_path: InputPath('ExampleStatistics'),
    #statistics_path: InputPath('ExampleStatistics'),
    schema_path: InputPath('Schema'),

    output_path: OutputPath('ExampleValidation'),
):
    """
    A TFX component to validate input examples.

    The ExampleValidator component uses [Tensorflow Data
    Validation](https://www.tensorflow.org/tfx/data_validation) to
    validate the statistics of some splits on input examples against a schema.

    The ExampleValidator component identifies anomalies in training and serving
    data. The component can be configured to detect different classes of anomalies
    in the data. It can:
        - perform validity checks by comparing data statistics against a schema that
        codifies expectations of the user.
        - detect data drift by looking at a series of data.
        - detect changes in dataset-wide data (i.e., num_examples) across spans or
        versions.

    Schema Based Example Validation
    The ExampleValidator component identifies any anomalies in the example data by
    comparing data statistics computed by the StatisticsGen component against a
    schema. The schema codifies properties which the input data is expected to
    satisfy, and is provided and maintained by the user.

    Please see https://www.tensorflow.org/tfx/data_validation for more details.

    Args:
        stats: A Channel of 'ExampleStatisticsPath` type. This should contain at
            least 'eval' split. Other splits are ignored currently.  Will be
            deprecated in the future for the `statistics` parameter.
        #statistics: Future replacement of the 'stats' argument.
        schema: A Channel of "SchemaPath' type. _required_
    Returns:
        output: Output channel of 'ExampleValidationPath' type.

    Either `stats` or `statistics` must be present in the arguments.
    """
    from tfx.components.example_validator.component import ExampleValidator
    component_class = ExampleValidator
    input_channels_with_splits = {'stats', 'statistics'}
    output_channels_with_splits = {}


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
        ExampleValidator,
        base_image='tensorflow/tfx:0.15.0',
        output_component_file='component.yaml'
    )
