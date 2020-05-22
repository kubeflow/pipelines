__all__ = [
    'create_component_from_tfx_component',
]

import collections
import copy
import inspect
import tempfile
import textwrap
import typing

from . import create_component_from_func


def _run_tfx_executor(component_class, **kwargs):
    #Generated code
    import json
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

    # Workaround for a TFX+Beam bug to make DataflowRunner work.
    # Remove after the next release that has https://github.com/tensorflow/tfx/commit/ddb01c02426d59e8bd541e3fd3cbaaf68779b2df
    import tfx
    tfx.version.__version__ += 'dev'

    executor_context = base_executor.BaseExecutor.Context(
        beam_pipeline_args=beam_pipeline_args,
        tmp_dir=tempfile.gettempdir(),
        unique_id='tfx_component',
    )
    executor = component_class_instance.executor_spec.executor_class(executor_context)
    executor.Do(
        input_dict=input_dict,
        output_dict=output_dict,
        exec_properties=exec_properties,
    )


def _generate_tfx_component_func_source(
    tfx_component_class,
    use_uri_io: bool = True,
    parameters_to_skip: list = None,
) -> str:
    from google.protobuf import message

    parameters_to_skip = parameters_to_skip or set()
    required_parameters = collections.OrderedDict()
    optional_parameters = collections.OrderedDict()
    return_parameters = collections.OrderedDict()
    return_objects = []

    # Processing inputs
    for name, channel_parameter in tfx_component_class.SPEC_CLASS.INPUTS.items():
        if name in parameters_to_skip:
            continue
        type_name = str(channel_parameter.type.TYPE_NAME)

        if use_uri_io:
            name = name + '_uri'
            annotation_repr = repr(type_name + 'Uri')
        else:
            name = name + '_path'
            annotation_repr = 'InputPath(' + repr(type_name) + ')'

        if channel_parameter.optional:
            optional_parameters[name] = annotation_repr
        else:
            required_parameters[name] = annotation_repr

    # Processing outputs
    for name, channel_parameter in tfx_component_class.SPEC_CLASS.OUTPUTS.items():
        if name in parameters_to_skip:
            continue
        type_name = str(channel_parameter.type.TYPE_NAME)

        if use_uri_io:
            # _uri is more correct than _path here, but conversion of pipelines will be harder
            # With '' suffix, the user only needs to change the components with most of the pipeline remaining the same.
            # The types should protect from mixing incompatible components.
            # In the end I chose the '_uri' suffix, since
            # with '_path' suffix the users will still need to change the pipeline
            # and with '' suffix, the input names are not correct for the semantics (passing a URI to a "model" input looks strange).
            name = name + '_uri'
            annotation_repr = repr(type_name + 'Uri')
            return_parameters[name] = annotation_repr
            # Prepending 'output_' to the parameter names corresponding to output URIs so that the user does not confuse them with the input URIs
            required_parameters['output_' + name] = annotation_repr
            return_objects.append('output_' + name)
        else:
            name = name + '_path'
            annotation_repr = 'OutputPath(' + repr(type_name) + ')'
            # Outputs cannot be optional
            required_parameters[name] = annotation_repr

    # Processing parameters
    for name, execution_parameter in tfx_component_class.SPEC_CLASS.PARAMETERS.items():
        if name in parameters_to_skip:
            continue
        param_type = execution_parameter.type

        if isinstance(execution_parameter.type, tuple):
            union_types = set(param_type)
            if len(union_types) == 1:
                param_type = list(union_types)[0]
            else:
                raise NotImplementedError('The type of parameter {} is a union: {} which is not supported'.format(name, union_types))
        if issubclass(param_type, message.Message):
            type_spec = dict(
                JsonObject=dict(
                    data_type='proto:' + param_type.DESCRIPTOR.full_name
                )
            )
            annotation_repr = repr(type_spec)
        elif issubclass(param_type, typing.List):
            # TODO: support generic types
            annotation_repr = 'list'
        elif issubclass(param_type, typing.Dict):
            # TODO: support generic types
            annotation_repr = 'dict'
        else:
            annotation_repr = param_type.__name__

        if execution_parameter.optional:
            optional_parameters[name] = annotation_repr
        else:
            required_parameters[name] = annotation_repr

    # beam_pipeline_args usually specifies an alternative runner that is usually non-local.
    # So the feature usually required URI-based I/O.
    # It is possible to download/upload URIs, but it's not trivial.
    # Check whether DataflowRunner supports local inputs
    if use_uri_io:
        optional_parameters['beam_pipeline_args'] = 'list'

    # Generating the function code

    signature_lines = []
    signature_lines.extend(name + ': ' + annotation_repr for name, annotation_repr in required_parameters.items())
    signature_lines.extend(name + ': ' + annotation_repr + ' = None' for name, annotation_repr in optional_parameters.items())

    function_name = tfx_component_class.__name__
    function_def_lines = ['def ' + function_name + '(']

    function_def_lines.extend('    ' + signature_line + ',' for signature_line in signature_lines)
    if return_parameters:
        function_def_lines.append(""") -> NamedTuple('Outputs', [""")
        function_def_lines.extend("""    ('{}', {}),""".format(name, annotation_repr) for name, annotation_repr in return_parameters.items())
        function_def_lines.append(']):')

        return_line = '    return (' + ''.join(var + ', ' for var in return_objects) + ')'
    else:
        function_def_lines.append('):')

    imports_lines = []
    if return_parameters:
        imports_lines.append('from typing import NamedTuple')
    if not use_uri_io:
        imports_lines.append('from kfp.components import InputPath, OutputPath')

    full_source_lines = []

    if imports_lines:
        full_source_lines.extend(imports_lines)
        full_source_lines.append('')

    full_source_lines.extend(function_def_lines)

    full_source_lines.append('    ' + 'from {} import {} as component_class'.format(tfx_component_class.__module__, tfx_component_class.__name__))
    full_source_lines.append('')

    run_executor_code = '\n'.join(inspect.getsource(_run_tfx_executor).split('\n')[1:])
    run_executor_code = textwrap.indent(textwrap.dedent(run_executor_code), '    ')
    run_executor_lines = run_executor_code.rstrip().split('\n')

    full_source_lines.extend(run_executor_lines)

    if return_parameters:
        full_source_lines.append('')
        full_source_lines.append(return_line)
    
    return ''.join(line + '\n' for line in full_source_lines)


def create_component_from_tfx_component(
    tfx_component_class,
    output_component_file: str = None,
    base_image: str = 'tensorflow/tfx:0.21.4',
    use_uri_io: bool = True,
    output_generated_code_file: str = None,
):
    """Creates a component from a TFX component class.

    The components can be generated in two flavors - with file-based I/O and URI-based I/O.
    With file-based I/O the system takes care of storing output data and making it available to downstream components.
    With URI-based I/O, only the URIs pointing to the data are passed between components and the pipeline author is responsible for providing unique URIs for all output artifacts of the components in the pipeline.

    Args:
        tfx_component_class: TFX component class serived from tfx.components.base.BaseComponent
        output_component_file: Optional. Write a component definition to a local file. The produced component file can be loaded back by calling `load_component_from_file` or `load_component_from_uri`.
        base_image: The container image that has the TFX component installed.
        use_uri_io: Whether to use URI-based I/O. Setting use_uri_io=True also enables the `beam_pipeline_args` input.

    Example::

        from tfx.components import CsvExampleGen
        csv_example_gen_op = create_component_from_tfx_component(CsvExampleGen)
    """
    component_func_name = tfx_component_class.__name__
    component_func_code = _generate_tfx_component_func_source(
        tfx_component_class=tfx_component_class,
        use_uri_io=use_uri_io,
    )
    generated_script_code = component_func_code + textwrap.dedent('''
        if __name__ == '__main__':
            import kfp
            kfp.components.create_component_from_func(
                {},
                base_image={},
                output_component_file='component.yaml'
            )
        '''.format(component_func_name, repr(base_image))
    )
    if not output_generated_code_file:
        output_generated_code_file = tempfile.mkstemp()[1]

    with open(output_generated_code_file, 'w') as generated_code_file:
        generated_code_file.write(component_func_code)
    
    global_scope = {}
    local_scope = {}
    global_scope = copy.copy(globals())

    # Preventing the __main__ script part from executing
    global_scope['__name__'] = '<STDIN>'
    # Without compile with full file name, create_component_from_func will fail getting the function source code during exec.
    component_code = compile(source=component_func_code, filename=str(output_generated_code_file), mode='exec')
    exec(component_code, global_scope, local_scope)
    component_func = local_scope[component_func_name]

    factory_func = create_component_from_func(
        func=component_func,
        base_image=base_image,
        output_component_file=output_component_file,
    )
    return factory_func
