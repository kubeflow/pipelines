from kfp.components import InputPath, OutputPath


def SchemaGen(
    stats_path: InputPath('ExampleStatistics'),
    #statistics_path: InputPath('ExampleStatistics'),
    output_path: OutputPath('Schema'),
    #schema_path: InputPath('Schema') = None,
    infer_feature_shape: bool = False,
):
    """Constructs a SchemaGen component.

    Args:
      stats: A Channel of `ExampleStatistics` type (required if spec is not
        passed). This should contain at least a `train` split. Other splits are
        currently ignored.
      #  Exactly one of 'stats'/'statistics' or 'schema' is required.
      #schema: A Channel of `Schema` type that provides an instance of Schema.
      #  If provided, pass through this schema artifact as the output. Exactly
      #  one of 'stats'/'statistics' or 'schema' is required.
      infer_feature_shape: Boolean value indicating whether or not to infer the
        shape of features. If the feature shape is not inferred, downstream
        Tensorflow Transform component using the schema will parse input
        as tf.SparseTensor.
      #statistics: Future replacement of the 'stats' argument.
      #Either `statistics` or `stats` must be present in the input arguments.
    Returns:
      output: Output `Schema` channel for schema result.
    """

    import json
    import os
    from google.protobuf import json_format
    from tfx.types import standard_artifacts
    from tfx.types import channel_utils

    # Create input dict.
    input_base_path = stats_path
    input_artifact_class = standard_artifacts.ExampleStatistics
    # Recovering splits
    splits = sorted(os.listdir(input_base_path))
    input_data_artifacts = []
    for split in splits:
        artifact = input_artifact_class()
        artifact.split = split
        artifact.uri = os.path.join(input_base_path, split) + '/'
        input_data_artifacts.append(artifact)
    input_data_channel = channel_utils.as_channel(input_data_artifacts)

    from tfx.components.schema_gen.component import SchemaGen
    component_class_instance = SchemaGen(
        stats=input_data_channel,
    )

    input_dict = {name: channel.get() for name, channel in component_class_instance.inputs.get_all().items()}
    output_dict = {name: channel.get() for name, channel in component_class_instance.outputs.get_all().items()}
    exec_properties = component_class_instance.exec_properties

    # Generating paths for output artifacts
    for output_artifact in output_dict['output']:
        output_artifact.uri = os.path.join(output_path, output_artifact.split) # Default split is ''

    print('component instance: ' + str(component_class_instance))

    executor = component_class_instance.executor_spec.executor_class()
    executor.Do(
        input_dict=input_dict,
        output_dict=output_dict,
        exec_properties=exec_properties,
    )
    #return (output_path,)


if __name__ == '__main__':
    import kfp
    kfp.components.func_to_container_op(
        SchemaGen,
        base_image='tensorflow/tfx:0.15.0',
        output_component_file='component.yaml'
    )
