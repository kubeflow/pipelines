# flake8: noqa TODO

from kfp.components import InputPath, OutputPath


def Trainer(
    examples_path: InputPath('Examples'),
    schema_path: InputPath('Schema'),

    model_path: OutputPath('Model'),

    train_args: {'JsonObject': {'data_type': 'proto:tfx.components.trainer.TrainArgs'}},
    eval_args: {'JsonObject': {'data_type': 'proto:tfx.components.trainer.EvalArgs'}},
    module_file: str = None,
    trainer_fn: str = None,
    custom_config: dict = None,

    transform_graph_path: InputPath('TransformGraph') = None,
    base_model_path: InputPath('Model') = None,
    hyperparameters_path: InputPath('HyperParameters') = None,
):
    """
    A TFX component to train a TensorFlow model.

    The Trainer component is used to train and eval a model using given inputs and
    a user-supplied estimator.  This component includes a custom driver to
    optionally grab previous model to warm start from.

    ## Providing an estimator
    The TFX executor will use the estimator provided in the `module_file` file
    to train the model.  The Trainer executor will look specifically for the
    `trainer_fn()` function within that file.  Before training, the executor will
    call that function expecting the following returned as a dictionary:

        - estimator: The
        [estimator](https://www.tensorflow.org/api_docs/python/tf/estimator/Estimator)
        to be used by TensorFlow to train the model.
        - train_spec: The
        [configuration](https://www.tensorflow.org/api_docs/python/tf/estimator/TrainSpec)
        to be used by the "train" part of the TensorFlow `train_and_evaluate()`
        call.
        - eval_spec: The
        [configuration](https://www.tensorflow.org/api_docs/python/tf/estimator/EvalSpec)
        to be used by the "eval" part of the TensorFlow `train_and_evaluate()` call.
        - eval_input_receiver_fn: The
        [configuration](https://www.tensorflow.org/tfx/model_analysis/get_started#modify_an_existing_model)
        to be used
        by the [ModelValidator](https://www.tensorflow.org/tfx/guide/modelval)
        component when validating the model.

    An example of `trainer_fn()` can be found in the [user-supplied
    code]((https://github.com/tensorflow/tfx/blob/master/tfx/examples/chicago_taxi_pipeline/taxi_utils.py))
    of the TFX Chicago Taxi pipeline example.


    Args:
      examples: A Channel of 'Examples' type, serving as the source of
        examples that are used in training (required). May be raw or
        transformed.
      transform_graph: An optional Channel of 'TransformGraph' type, serving as
        the input transform graph if present.
      schema:  A Channel of 'SchemaPath' type, serving as the schema of training
        and eval data.
      module_file: A path to python module file containing UDF model definition.
        The module_file must implement a function named `trainer_fn` at its
        top level. The function must have the following signature.

        def trainer_fn(tf.contrib.training.HParams,
                       tensorflow_metadata.proto.v0.schema_pb2) -> Dict:
          ...

        where the returned Dict has the following key-values.
          'estimator': an instance of tf.estimator.Estimator
          'train_spec': an instance of tf.estimator.TrainSpec
          'eval_spec': an instance of tf.estimator.EvalSpec
          'eval_input_receiver_fn': an instance of tfma.export.EvalInputReceiver

        Exactly one of 'module_file' or 'trainer_fn' must be supplied.
      trainer_fn:  A python path to UDF model definition function. See
        'module_file' for the required signature of the UDF.
        Exactly one of 'module_file' or 'trainer_fn' must be supplied.
      train_args: A trainer_pb2.TrainArgs instance, containing args used for
        training. Current only num_steps is available.
      eval_args: A trainer_pb2.EvalArgs instance, containing args used for eval.
        Current only num_steps is available.
      custom_config: A dict which contains the training job parameters to be
        passed to Google Cloud ML Engine.  For the full set of parameters
        supported by Google Cloud ML Engine, refer to
        https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#Job
    Returns:
      model: Optional 'Model' channel for result of exported models.
    Raises:
      ValueError:
        - When both or neither of 'module_file' and 'trainer_fn' is supplied.
    """
    from tfx.components.trainer.component import Trainer as component_class

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
        Trainer,
        base_image='tensorflow/tfx:0.21.4',
        output_component_file='component.yaml'
    )
