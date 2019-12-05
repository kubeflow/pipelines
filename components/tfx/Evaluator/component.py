# flake8: noqa TODO

from kfp.components import InputPath, OutputPath


def Evaluator(
    examples_path: InputPath('Examples'),
    model_exports_path: InputPath('Model'),
    #model_path: InputPath('Model'),

    output_path: OutputPath('ModelEval'),

    feature_slicing_spec: 'JsonObject: evaluator_pb2.FeatureSlicingSpec' = None,
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
        examples: A Channel of 'ExamplesPath' type, usually produced by ExampleGen
            component. @Ark-kun: Must have the eval split. _required_
        model_exports: A Channel of 'ModelExportPath' type, usually produced by
            Trainer component.  Will be deprecated in the future for the `model`
            parameter.
        #model: Future replacement of the `model_exports` argument.
        feature_slicing_spec:
            [evaluator_pb2.FeatureSlicingSpec](https://github.com/tensorflow/tfx/blob/master/tfx/proto/evaluator.proto)
            instance that describes how Evaluator should slice the data.
    Returns:
        output: Channel of `ModelEvalPath` to store the evaluation results.

    Either `model_exports` or `model` must be present in the input arguments.

    """
    from tfx.components.evaluator.component import Evaluator
    component_class = Evaluator
    input_channels_with_splits = {'examples'}
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
        Evaluator,
        base_image='tensorflow/tfx:0.15.0',
        output_component_file='component.yaml'
    )
