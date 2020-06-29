# flake8: noqa TODO

from kfp.components import InputPath, OutputPath


def Transform(
    examples_path: InputPath('Examples'),
    schema_path: InputPath('Schema'),

    transform_graph_path: OutputPath('TransformGraph'),
    transformed_examples_path: OutputPath('Examples'),

    module_file: str = None,
    preprocessing_fn: str = None,
    custom_config: dict = None,
):
    """A TFX component to transform the input examples.

    The Transform component wraps TensorFlow Transform (tf.Transform) to
    preprocess data in a TFX pipeline. This component will load the
    preprocessing_fn from input module file, preprocess both 'train' and 'eval'
    splits of input examples, generate the `tf.Transform` output, and save both
    transform function and transformed examples to orchestrator desired locations.

    ## Providing a preprocessing function
    The TFX executor will use the estimator provided in the `module_file` file
    to train the model.  The Transform executor will look specifically for the
    `preprocessing_fn()` function within that file.

    An example of `preprocessing_fn()` can be found in the [user-supplied
    code]((https://github.com/tensorflow/tfx/blob/master/tfx/examples/chicago_taxi_pipeline/taxi_utils.py))
    of the TFX Chicago Taxi pipeline example.

    Args:
      examples: A Channel of 'Examples' type (required). This should
        contain the two splits 'train' and 'eval'.
      schema: A Channel of 'SchemaPath' type. This should contain a single
        schema artifact.
      module_file: The file path to a python module file, from which the
        'preprocessing_fn' function will be loaded. The function must have the
        following signature.

        def preprocessing_fn(inputs: Dict[Text, Any]) -> Dict[Text, Any]:
          ...

        where the values of input and returned Dict are either tf.Tensor or
        tf.SparseTensor.  Exactly one of 'module_file' or 'preprocessing_fn'
        must be supplied.
      preprocessing_fn: The path to python function that implements a
         'preprocessing_fn'. See 'module_file' for expected signature of the
         function. Exactly one of 'module_file' or 'preprocessing_fn' must
         be supplied.

    Returns:
      transform_graph: Optional output 'TransformPath' channel for output of
        'tf.Transform', which includes an exported Tensorflow graph suitable for
        both training and serving;
      transformed_examples: Optional output 'ExamplesPath' channel for
        materialized transformed examples, which includes both 'train' and
        'eval' splits.

    Raises:
      ValueError: When both or neither of 'module_file' and 'preprocessing_fn'
        is supplied.
    """
    from tfx.components.transform.component import Transform
    component_class = Transform

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
        Transform,
        base_image='tensorflow/tfx:0.21.4',
        output_component_file='component.yaml'
    )
