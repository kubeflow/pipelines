# flake8: noqa TODO

from kfp.components import InputPath, OutputPath


def Evaluator(
    evaluation_path: OutputPath('ModelEvaluation'),

    examples_path: InputPath('Examples'),
    model_path: InputPath('Model'),
    baseline_model_path: InputPath('Model') = None,
    schema_path: InputPath('Schema') = None,

    feature_slicing_spec: {'JsonObject': {'data_type': 'proto:tfx.components.evaluator.FeatureSlicingSpec'}} = None,  # TODO: Replace feature_slicing_spec with eval_config
    eval_config: {'JsonObject': {'data_type': 'proto:tensorflow_model_analysis.EvalConfig'}} = None,
    fairness_indicator_thresholds: list = None, # List[str]

    #blessing_path: OutputPath('ModelBlessing') = None,  # Optional outputs are not supported yet
):
    """
    A TFX component to evaluate models trained by a TFX Trainer component.

    The Evaluator component performs model evaluations in the TFX pipeline and
    the resultant metrics can be viewed in a Jupyter notebook.  It uses the
    input examples generated from the
    [ExampleGen](https://www.tensorflow.org/tfx/guide/examplegen)
    component to evaluate the models.

    Specifically, it can provide:
        - metrics computed on entire training and eval dataset
        - tracking metrics over time
        - model quality performance on different feature slices

    ## Exporting the EvalSavedModel in Trainer

    In order to setup Evaluator in a TFX pipeline, an EvalSavedModel needs to be
    exported during training, which is a special SavedModel containing
    annotations for the metrics, features, labels, and so on in your model.
    Evaluator uses this EvalSavedModel to compute metrics.

    As part of this, the Trainer component creates eval_input_receiver_fn,
    analogous to the serving_input_receiver_fn, which will extract the features
    and labels from the input data. As with serving_input_receiver_fn, there are
    utility functions to help with this.

    Please see https://www.tensorflow.org/tfx/model_analysis for more details.

    Args:
        examples: A Channel of 'Examples' type, usually produced by ExampleGen
            component. @Ark-kun: Must have the eval split. _required_
        model: A Channel of 'Model' type, usually produced by
            Trainer component.
        feature_slicing_spec:
            [evaluator_pb2.FeatureSlicingSpec](https://github.com/tensorflow/tfx/blob/master/tfx/proto/evaluator.proto)
            instance that describes how Evaluator should slice the data.
    Returns:
        evaluation: Channel of `ModelEvaluation` to store the evaluation results.

    Either `model_exports` or `model` must be present in the input arguments.

    """
    from tfx.components.evaluator.component import Evaluator as component_class

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
        base_artifact_path = arguments.get(name + '_path', None)
        if base_artifact_path:
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
        Evaluator,
        base_image='tensorflow/tfx:0.21.4',
        output_component_file='component.yaml'
    )
