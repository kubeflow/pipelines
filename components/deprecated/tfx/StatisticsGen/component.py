from kfp.components import InputPath, OutputPath

def StatisticsGen(
    examples_path: InputPath('Examples'),
    statistics_path: OutputPath('ExampleStatistics'),
    schema_path: InputPath('Schema') = None,
    exclude_splits: str = None,
):
    from tfx.components.statistics_gen.component import StatisticsGen as component_class

    #Generated code
    import os
    import tempfile
    from tensorflow.io import gfile
    from google.protobuf import json_format, message
    from tfx.types import channel_utils, artifact_utils
    from tfx.components.base import base_executor

    arguments = locals().copy()

    component_class_args = {}

    for name, execution_parameter in component_class.SPEC_CLASS.PARAMETERS.items():
        argument_value = arguments.get(name, None)
        if argument_value is None:
            continue
        parameter_type = execution_parameter.type
        if isinstance(parameter_type, type) and issubclass(parameter_type, message.Message):
            argument_value_obj = parameter_type()
            json_format.Parse(argument_value, argument_value_obj)
        else:
            argument_value_obj = argument_value
        component_class_args[name] = argument_value_obj

    for name, channel_parameter in component_class.SPEC_CLASS.INPUTS.items():
        artifact_path = arguments.get(name + '_uri') or arguments.get(name + '_path')
        if artifact_path:
            artifact = channel_parameter.type()
            artifact.uri = artifact_path.rstrip('/') + '/'  # Some TFX components require that the artifact URIs end with a slash
            if channel_parameter.type.PROPERTIES and 'split_names' in channel_parameter.type.PROPERTIES:
                # Recovering splits
                subdirs = gfile.listdir(artifact_path)
                # Workaround for https://github.com/tensorflow/tensorflow/issues/39167
                subdirs = [subdir.rstrip('/') for subdir in subdirs]
                artifact.split_names = artifact_utils.encode_split_names(sorted(subdirs))
            component_class_args[name] = channel_utils.as_channel([artifact])

    component_class_instance = component_class(**component_class_args)

    input_dict = channel_utils.unwrap_channel_dict(component_class_instance.inputs.get_all())
    output_dict = channel_utils.unwrap_channel_dict(component_class_instance.outputs.get_all())
    exec_properties = component_class_instance.exec_properties

    # Generating paths for output artifacts
    for name, artifacts in output_dict.items():
        base_artifact_path = arguments.get('output_' + name + '_uri') or arguments.get(name + '_path')
        if base_artifact_path:
            # Are there still cases where output channel has multiple artifacts?
            for idx, artifact in enumerate(artifacts):
                subdir = str(idx + 1) if idx > 0 else ''
                artifact.uri = os.path.join(base_artifact_path, subdir)  # Ends with '/'

    print('component instance: ' + str(component_class_instance))

    executor_context = base_executor.BaseExecutor.Context(
        beam_pipeline_args=arguments.get('beam_pipeline_args'),
        tmp_dir=tempfile.gettempdir(),
        unique_id='tfx_component',
    )
    executor = component_class_instance.executor_spec.executor_class(executor_context)
    executor.Do(
        input_dict=input_dict,
        output_dict=output_dict,
        exec_properties=exec_properties,
    )
