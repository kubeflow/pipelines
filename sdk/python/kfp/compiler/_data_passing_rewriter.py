import copy
import json
import os
import re
from typing import Any, Dict, List, Optional, Set, Tuple

from kfp.components import _components
from kfp import dsl


def fix_big_data_passing(workflow: dict) -> dict:
    '''fix_big_data_passing converts a workflow where some artifact data is passed as parameters and converts it to a workflow where this data is passed as artifacts.
    Args:
        workflow: The workflow to fix
    Returns:
        The fixed workflow

    Motivation:
    DSL compiler only supports passing Argo parameters.
    Due to the convoluted nature of the DSL compiler, the artifact consumption and passing has been implemented on top of that using parameter passing. The artifact data is passed as parameters and the consumer template creates an artifact/file out of that data.
    Due to the limitations of Kubernetes and Argo this scheme cannot pass data larger than few kilobytes preventing any serious use of artifacts.

    This function rewrites the compiled workflow so that the data consumed as artifact is passed as artifact.
    It also prunes the unused parameter outputs. This is important since if a big piece of data is ever returned through a file that is also output as parameter, the execution will fail.
    This makes is possible to pass large amounts of data.

    Implementation:
    1. Index the DAGs to understand how data is being passed and which inputs/outputs are connected to each other.
    2. Search for direct data consumers in container/resource templates and some DAG task attributes (e.g. conditions and loops) to find out which inputs are directly consumed as parameters/artifacts.
    3. Propagate the consumption information upstream to all inputs/outputs all the way up to the data producers.
    4. Convert the inputs, outputs and arguments based on how they're consumed downstream.
    '''
    workflow = copy.deepcopy(workflow)

    container_templates = [template for template in workflow['spec']['templates'] if 'container' in template]
    dag_templates = [template for template in workflow['spec']['templates'] if 'dag' in template]
    resource_templates = [template for template in workflow['spec']['templates'] if 'resource' in template]  # TODO: Handle these
    resource_template_names = set(template['name'] for template in resource_templates)

    # 1. Index the DAGs to understand how data is being passed and which inputs/outputs are connected to each other.
    template_input_to_parent_dag_inputs = {} # (task_template_name, task_input_name) -> Set[(dag_template_name, dag_input_name)]
    template_input_to_parent_task_outputs = {} # (task_template_name, task_input_name) -> Set[(upstream_template_name, upstream_output_name)]
    template_input_to_parent_constant_arguments = {} #(task_template_name, task_input_name) -> Set[argument_value] # Unused
    dag_output_to_parent_template_outputs = {} # (dag_template_name, output_name) -> Set[(upstream_template_name, upstream_output_name)]

    for template in dag_templates:
        dag_template_name = template['name']
        # Indexing task arguments
        dag_tasks = template['dag']['tasks']
        task_name_to_template_name = {task['name']: task['template'] for task in dag_tasks}
        for task in dag_tasks:
            task_template_name = task['template']
            parameter_arguments = task.get('arguments', {}).get('parameters', {})
            for parameter_argument in parameter_arguments:
                task_input_name = parameter_argument['name']
                argument_value = parameter_argument['value']

                argument_placeholder_parts = deconstruct_single_placeholder(argument_value)
                if not argument_placeholder_parts: # Argument is considered to be constant string
                    template_input_to_parent_constant_arguments.setdefault((task_template_name, task_input_name), set()).add(argument_value)

                placeholder_type = argument_placeholder_parts[0]
                if placeholder_type not in ('inputs', 'outputs', 'tasks', 'steps', 'workflow', 'pod', 'item'):
                    # Do not fail on Jinja or other double-curly-brace templates
                    continue
                if placeholder_type == 'inputs':
                    assert argument_placeholder_parts[1] == 'parameters'
                    dag_input_name = argument_placeholder_parts[2]
                    template_input_to_parent_dag_inputs.setdefault((task_template_name, task_input_name), set()).add((dag_template_name, dag_input_name))
                elif placeholder_type == 'tasks':
                    upstream_task_name = argument_placeholder_parts[1]
                    assert argument_placeholder_parts[2] == 'outputs'
                    assert argument_placeholder_parts[3] == 'parameters'
                    upstream_output_name = argument_placeholder_parts[4]
                    upstream_template_name = task_name_to_template_name[upstream_task_name]
                    template_input_to_parent_task_outputs.setdefault((task_template_name, task_input_name), set()).add((upstream_template_name, upstream_output_name))
                elif placeholder_type == 'item' or placeholder_type == 'workflow' or placeholder_type == 'pod':
                    # Treat loop variables as constant values
                    # workflow.parameters.* placeholders are not supported, but the DSL compiler does not produce those.
                    template_input_to_parent_constant_arguments.setdefault((task_template_name, task_input_name), set()).add(argument_value)
                else:
                    raise AssertionError

                dag_input_name = extract_input_parameter_name(argument_value)
                if dag_input_name:
                    template_input_to_parent_dag_inputs.setdefault((task_template_name, task_input_name), set()).add((dag_template_name, dag_input_name))
                else:
                    template_input_to_parent_constant_arguments.setdefault((task_template_name, task_input_name), set()).add(argument_value)

            # Indexing DAG outputs (Does DSL compiler produce them?)
            for dag_output in task.get('outputs', {}).get('parameters', {}):
                dag_output_name = dag_output['name']
                output_value = dag_output['value']
                argument_placeholder_parts = deconstruct_single_placeholder(output_value)
                placeholder_type = argument_placeholder_parts[0]
                if not argument_placeholder_parts: # Argument is considered to be constant string
                    raise RuntimeError('Constant DAG output values are not supported for now.')
                if placeholder_type == 'inputs':
                    raise RuntimeError('Pass-through DAG inputs/outputs are not supported')
                elif placeholder_type == 'tasks':
                    upstream_task_name = argument_placeholder_parts[1]
                    assert argument_placeholder_parts[2] == 'outputs'
                    assert argument_placeholder_parts[3] == 'parameters'
                    upstream_output_name = argument_placeholder_parts[4]
                    upstream_template_name = task_name_to_template_name[upstream_task_name]
                    dag_output_to_parent_template_outputs.setdefault((dag_template_name, dag_output_name), set()).add((upstream_template_name, upstream_output_name))
                elif placeholder_type == 'item' or placeholder_type == 'workflow' or placeholder_type == 'pod':
                    raise RuntimeError('DAG output value "{}" is not supported.'.format(output_value))
                else:
                    raise AssertionError('Unexpected placeholder type "{}".'.format(placeholder_type))
    # Finshed indexing the DAGs

    # 2. Search for direct data consumers in container/resource templates and some DAG task attributes (e.g. conditions and loops) to find out which inputs are directly consumed as parameters/artifacts.
    inputs_directly_consumed_as_parameters = set()
    inputs_directly_consumed_as_artifacts = set()
    outputs_directly_consumed_as_parameters = set()

    # Searching for artifact input consumers in container template inputs
    for template in container_templates:
        template_name = template['name']
        for input_artifact in template.get('inputs', {}).get('artifacts', {}):
            raw_data = input_artifact['raw']['data'] # The structure must exist
            # The raw data must be a single input parameter reference. Otherwise (e.g. it's a string or a string with multiple inputs) we should not do the conversion to artifact passing.
            input_name = extract_input_parameter_name(raw_data)
            if input_name:
                inputs_directly_consumed_as_artifacts.add((template_name, input_name))
                del input_artifact['raw'] # Deleting the "default value based" data passing hack so that it's replaced by the "argument based" way of data passing.
                input_artifact['name'] = input_name # The input artifact name should be the same as the original input parameter name
    
    # Searching for parameter input consumers in DAG templates (.when, .withParam, etc)
    for template in dag_templates:
        template_name = template['name']
        dag_tasks = template['dag']['tasks']
        task_name_to_template_name = {task['name']: task['template'] for task in dag_tasks}
        for task in template['dag']['tasks']:
            # We do not care about the inputs mentioned in task arguments since we will be free to switch them from parameters to artifacts
            # TODO: Handle cases where argument value is a string containing placeholders (not just consisting of a single placeholder) or the input name contains placeholder
            task_without_arguments = task.copy() # Shallow copy
            task_without_arguments.pop('arguments', None)
            placeholders = extract_all_placeholders(task_without_arguments)
            for placeholder in placeholders:
                parts = placeholder.split('.')
                placeholder_type = parts[0]
                if placeholder_type not in ('inputs', 'outputs', 'tasks', 'steps', 'workflow', 'pod', 'item'):
                    # Do not fail on Jinja or other double-curly-brace templates
                    continue
                if placeholder_type == 'inputs':
                    if parts[1] == 'parameters':
                        input_name = parts[2]
                        inputs_directly_consumed_as_parameters.add((template_name, input_name))
                    else:
                        raise AssertionError
                elif placeholder_type == 'tasks':
                    upstream_task_name = parts[1]
                    assert parts[2] == 'outputs'
                    assert parts[3] == 'parameters'
                    upstream_output_name = parts[4]
                    upstream_template_name = task_name_to_template_name[upstream_task_name]
                    outputs_directly_consumed_as_parameters.add((upstream_template_name, upstream_output_name))
                elif placeholder_type == 'workflow' or placeholder_type == 'pod':
                    pass
                elif placeholder_type == 'item':
                    raise AssertionError('The "{{item}}" placeholder is not expected outside task arguments.')
                else:
                    raise AssertionError('Unexpected placeholder type "{}".'.format(placeholder_type))

    # Searching for parameter input consumers in container and resource templates
    for template in container_templates + resource_templates:
        template_name = template['name']
        placeholders = extract_all_placeholders(template)
        for placeholder in placeholders:
            parts = placeholder.split('.')
            placeholder_type = parts[0]
            if placeholder_type not in ('inputs', 'outputs', 'tasks', 'steps', 'workflow', 'pod', 'item'):
                # Do not fail on Jinja or other double-curly-brace templates
                continue

            if placeholder_type == 'workflow' or placeholder_type == 'pod':
                pass
            elif placeholder_type == 'inputs':
                if parts[1] == 'parameters':
                    input_name = parts[2]
                    inputs_directly_consumed_as_parameters.add((template_name, input_name))
                elif parts[1] == 'artifacts':
                    raise AssertionError('Found unexpected Argo input artifact placeholder in container template: {}'.format(placeholder))
                else:
                    raise AssertionError('Found unexpected Argo input placeholder in container template: {}'.format(placeholder))
            else:
                raise AssertionError('Found unexpected Argo placeholder in container template: {}'.format(placeholder))

    # Finished indexing data consumers

    # 3. Propagate the consumption information upstream to all inputs/outputs all the way up to the data producers.
    inputs_consumed_as_parameters = set()
    inputs_consumed_as_artifacts = set()

    outputs_consumed_as_parameters = set()
    outputs_consumed_as_artifacts = set()

    def mark_upstream_ios_of_input(template_input, marked_inputs, marked_outputs):
        # Stopping if the input has already been visited to save time and handle recursive calls
        if template_input in marked_inputs:
            return
        marked_inputs.add(template_input)

        upstream_inputs = template_input_to_parent_dag_inputs.get(template_input, [])
        for upstream_input in upstream_inputs:
            mark_upstream_ios_of_input(upstream_input, marked_inputs, marked_outputs)

        upstream_outputs = template_input_to_parent_task_outputs.get(template_input, [])
        for upstream_output in upstream_outputs:
            mark_upstream_ios_of_output(upstream_output, marked_inputs, marked_outputs)

    def mark_upstream_ios_of_output(template_output, marked_inputs, marked_outputs):
        # Stopping if the output has already been visited to save time and handle recursive calls
        if template_output in marked_outputs:
            return
        marked_outputs.add(template_output)

        upstream_outputs = dag_output_to_parent_template_outputs.get(template_output, [])
        for upstream_output in upstream_outputs:
            mark_upstream_ios_of_output(upstream_output, marked_inputs, marked_outputs)

    for input in inputs_directly_consumed_as_parameters:
        mark_upstream_ios_of_input(input, inputs_consumed_as_parameters, outputs_consumed_as_parameters)
    for input in inputs_directly_consumed_as_artifacts:
        mark_upstream_ios_of_input(input, inputs_consumed_as_artifacts, outputs_consumed_as_artifacts)
    for output in outputs_directly_consumed_as_parameters:
        mark_upstream_ios_of_output(output, inputs_consumed_as_parameters, outputs_consumed_as_parameters)


    # 4. Convert the inputs, outputs and arguments based on how they're consumed downstream.

    # Container templates already output all data as artifacts, so we do not need to convert their outputs to artifacts. (But they also output data as parameters and we need to fix that.)

    # Convert DAG argument passing from parameter to artifacts as needed
    for template in dag_templates:
        # Converting DAG inputs
        inputs = template.get('inputs', {})
        input_parameters = inputs.get('parameters', [])
        input_artifacts = inputs.setdefault('artifacts', []) # Should be empty
        for input_parameter in input_parameters:
            input_name = input_parameter['name']
            if (template['name'], input_name) in inputs_consumed_as_artifacts:
                input_artifacts.append({
                    'name': input_name,
                })

        # Converting DAG outputs
        outputs = template.get('outputs', {})
        output_parameters = outputs.get('parameters', [])
        output_artifacts = outputs.setdefault('artifacts', []) # Should be empty
        for output_parameter in output_parameters:
            output_name = output_parameter['name']
            if (template['name'], output_name) in outputs_consumed_as_artifacts:
                parameter_reference_placeholder = output_parameter['valueFrom']['parameter']
                output_artifacts.append({
                    'name': output_name,
                    'from': parameter_reference_placeholder.replace('.parameters.', '.artifacts.'),
                })

        # Converting DAG task arguments
        tasks = template.get('dag', {}).get('tasks', [])
        for task in tasks:
            task_arguments = task.get('arguments', {})
            parameter_arguments = task_arguments.get('parameters', [])
            artifact_arguments = task_arguments.setdefault('artifacts', [])
            for parameter_argument in parameter_arguments:
                input_name = parameter_argument['name']
                if (task['template'], input_name) in inputs_consumed_as_artifacts:
                    argument_value = parameter_argument['value'] # argument parameters always use "value"; output parameters always use "valueFrom" (container/DAG/etc)
                    argument_placeholder_parts = deconstruct_single_placeholder(argument_value)
                    # If the argument is consumed as artifact downstream:
                    # Pass DAG inputs and DAG/container task outputs as usual;
                    # Everything else (constant strings, loop variables, resource task outputs) is passed as raw artifact data. Argo properly replaces placeholders in it.
                    if argument_placeholder_parts and argument_placeholder_parts[0] in ['inputs', 'tasks'] and not (argument_placeholder_parts[0] == 'tasks' and argument_placeholder_parts[1] in resource_template_names):
                        artifact_arguments.append({
                            'name': input_name,
                            'from': argument_value.replace('.parameters.', '.artifacts.'),
                        })
                    else:
                        artifact_arguments.append({
                            'name': input_name,
                            'raw': {
                                'data': argument_value,
                            },
                        })

    # Remove input parameters unless they're used downstream. This also removes unused container template inputs if any.
    for template in container_templates + dag_templates:
        inputs = template.get('inputs', {})
        inputs['parameters'] = [
            input_parameter
            for input_parameter in inputs.get('parameters', [])
            if (template['name'], input_parameter['name']) in inputs_consumed_as_parameters
        ]

    # Remove output parameters unless they're used downstream
    for template in container_templates + dag_templates:
        outputs = template.get('outputs', {})
        outputs['parameters'] = [
            output_parameter
            for output_parameter in outputs.get('parameters', [])
            if (template['name'], output_parameter['name']) in outputs_consumed_as_parameters
        ]

    # Remove DAG parameter arguments unless they're used downstream
    for template in dag_templates:
        tasks = template.get('dag', {}).get('tasks', [])
        for task in tasks:
            task_arguments = task.get('arguments', {})
            task_arguments['parameters'] = [
                parameter_argument
                for parameter_argument in task_arguments.get('parameters', [])
                if (task['template'], parameter_argument['name']) in inputs_consumed_as_parameters
            ]
    
    # Fix Workflow parameter arguments that are consumed as artifacts downstream
    # 
    workflow_spec = workflow['spec']
    entrypoint_template_name = workflow_spec['entrypoint']
    workflow_arguments = workflow_spec['arguments']
    parameter_arguments = workflow_arguments.get('parameters', [])
    artifact_arguments = workflow_arguments.get('artifacts', []) # Should be empty
    for parameter_argument in parameter_arguments:
        input_name = parameter_argument['name']
        if (entrypoint_template_name, input_name) in inputs_consumed_as_artifacts:
            artifact_arguments.append({
                'name': input_name,
                'raw': {
                    'data': '{{workflow.parameters.' + input_name + '}}',
                },
            })
    if artifact_arguments:
        workflow_arguments['artifacts'] = artifact_arguments

    clean_up_empty_workflow_structures(workflow)
    return workflow


def clean_up_empty_workflow_structures(workflow: dict):
    templates = workflow['spec']['templates']
    for template in templates:
        inputs = template.setdefault('inputs', {})
        if not inputs.setdefault('parameters', []):
            del inputs['parameters']
        if not inputs.setdefault('artifacts', []):
            del inputs['artifacts']
        if not inputs:
            del template['inputs']
        outputs = template.setdefault('outputs', {})
        if not outputs.setdefault('parameters', []):
            del outputs['parameters']
        if not outputs.setdefault('artifacts', []):
            del outputs['artifacts']
        if not outputs:
            del template['outputs']
        if 'dag' in template:
            for task in template['dag'].get('tasks', []):
                arguments = task.setdefault('arguments', {})
                if not arguments.setdefault('parameters', []):
                    del arguments['parameters']
                if not arguments.setdefault('artifacts', []):
                    del arguments['artifacts']
                if not arguments:
                    del task['arguments']


def extract_all_placeholders(template: dict) -> Set[str]:
    template_str = json.dumps(template)
    placeholders = set(re.findall('{{([-._a-zA-Z0-9]+)}}', template_str))
    return placeholders


def extract_input_parameter_name(s: str) -> Optional[str]:
    match = re.fullmatch('{{inputs.parameters.([-_a-zA-Z0-9]+)}}', s) 
    if not match:
        return None
    (input_name,) = match.groups()
    return input_name


def deconstruct_single_placeholder(s: str) -> List[str]:
    if not re.fullmatch('{{[-._a-zA-Z0-9]+}}', s):
        return None
    return s.lstrip('{').rstrip('}').split('.')


def _replace_output_dir_and_run_id(command_line: str,
    output_directory: Optional[str] = None) -> str:
    """Replaces the output directory placeholder."""
    if _components.OUTPUT_DIR_PLACEHOLDER in command_line:
        if not output_directory:
            raise ValueError('output_directory of a pipeline must be specified '
                             'when URI placeholder is used.')
        command_line = command_line.replace(
            _components.OUTPUT_DIR_PLACEHOLDER, output_directory)
    if _components.RUN_ID_PLACEHOLDER in command_line:
        command_line = command_line.replace(
            _components.RUN_ID_PLACEHOLDER, dsl.RUN_ID_PLACEHOLDER)
    return command_line


def _refactor_outputs_if_uri_placeholder(
    container_template: Dict[str, Any],
    output_to_filename: Dict[str, str]
) -> None:
    """Rewrites the output of the container in case of URI placeholder.

    Also, collects the mapping from the output names to output file name.

    Args:
        container_template: The container template structure.
        output_to_filename: The mapping from the artifact name to the actual file
            name of the content. This will be used later when reconciling the
            URIs on the consumer side.
    """

    # If there's no artifact outputs then no refactor is needed.
    if not container_template.get('outputs') or not container_template[
        'outputs'].get('artifacts'):
        return

    parameter_outputs = container_template['outputs'].get('parameters') or []
    new_artifact_outputs = []
    for artifact_output in container_template['outputs']['artifacts']:
        # Check if this is an output associated with URI placeholder based
        # on its path.
        if _components.OUTPUT_DIR_PLACEHOLDER in artifact_output['path']:
            # If so, we'll add a parameter output to output the pod name
            parameter_outputs.append(
                {
                    'name': _components.PRODUCER_POD_NAME_PARAMETER.format(
                        artifact_output['name']),
                    'value': '{{pod.name}}'
                })
            output_to_filename[artifact_output['name']] = os.path.basename(
                artifact_output['path'])
        else:
            # Otherwise, this artifact output is preserved.
            new_artifact_outputs.append(artifact_output)

    container_template['outputs']['artifacts'] = new_artifact_outputs
    container_template['outputs']['parameters'] = parameter_outputs


def _refactor_inputs_if_uri_placeholder(
    container_template: Dict[str, Any],
    output_to_filename: Dict[str, str],
    refactored_inputs: Dict[Tuple[str, str], str]
) -> None:
    """Rewrites the input of the container in case of URI placeholder.

    Rewrites the artifact input of the container when it's used as a URI
    placeholder. Also, collects the inputs being rewritten into a list, so that
    it can be wired to correct task outputs later. Meanwhile, the filename used
    by input URI placeholder will be reconciled with its corresponding producer.

    Args:
        container_template: The container template structure.
        output_to_filename: The mapping from output name to the file name.
        refactored_inputs: The mapping used to collect the input artifact being
            refactored from (template name, previous name) to its new name.
    """

    # If there's no artifact inputs then no refactor is needed.
    if not container_template.get('inputs') or not container_template[
        'inputs'].get('artifacts'):
        return

    parameter_inputs = container_template['inputs'].get('parameters') or []
    new_artifact_inputs = []
    for artifact_input in container_template['inputs']['artifacts']:
        # Check if this is an input artifact associated with URI placeholder,
        # according to its path.
        if _components.OUTPUT_DIR_PLACEHOLDER in artifact_input['path']:
            # If so, we'll add a parameter input to receive the producer's pod
            # name.
            # The correct input parameter name should be parsed from the
            # path field, which is given according to the component I/O
            # definition.
            m = re.match(
                r'.*/{{kfp\.run_uid}}/{{inputs\.parameters\.(?P<input_name>.*)'
                r'}}/.*',
                artifact_input['path'])
            input_name = m.group('input_name')
            parameter_inputs.append({'name': input_name})
            # Here we're using the template name + previous artifact input name
            # as key, because it will be refactored later at the DAG level.
            refactored_inputs[(container_template['name'],
                               artifact_input['name'])] = input_name

            # In the container implementation, the pod name is already connected
            # to the input parameter per the implementation in _components.
            # The only thing yet to be reconciled is the file name.

            def reconcile_filename(
                command_lines: List[str]) -> List[str]:
                new_command_lines = []
                for cmd in command_lines:
                    matched = re.match(
                        r'.*/{{kfp\.run_uid}}/{{inputs\.parameters\.'
                        + input_name + r'}}/(?P<filename>.*)', cmd)
                    if matched:
                        new_command_lines.append(
                            cmd[:-len(matched.group('filename'))] +
                            output_to_filename[artifact_input['name']])
                    else:
                        new_command_lines.append(cmd)
                return new_command_lines

            if container_template['container'].get('args'):
                container_template['container']['args'] = reconcile_filename(
                    container_template['container']['args'])
            if container_template['container'].get('command'):
                container_template['container']['command'] = reconcile_filename(
                    container_template['container']['command'])
        else:
            new_artifact_inputs.append(artifact_input)

    container_template['inputs']['artifacts'] = new_artifact_inputs
    container_template['inputs']['parameters'] = parameter_inputs


def _refactor_dag_inputs(
    dag_template: Dict[str, Any],
    refactored_inputs: Dict[Tuple[str, str], str]
) -> None:
    """Refactors the inputs of the DAG template.

    Args:
        dag_template: The DAG template structure.
        refactored_inputs: The mapping of template and input names to be
            refactored, to its new name.
    """
    # One hacky way to do the refactoring is by looking at the name of the
    # artifact argument. If the name appears in the refactored_inputs mapping,
    # this should be changed to a parameter input regardless of the template
    # name.
    # The correctness of this approach is ensured by the data passing rewriting
    # process that changed the artifact inputs' name to be
    # '{{output_template}}-{{output_name}}', which is consistent across all
    # templates.
    if not dag_template.get('inputs', {}).get('artifacts'):
        return
    artifact_to_new_name = {k[1] : v for k, v in refactored_inputs.items()}

    parameter_inputs = dag_template['inputs'].get('parameters') or []
    new_artifact_inputs = []
    for input_artifact in dag_template['inputs']['artifacts']:
        if input_artifact['name'] in artifact_to_new_name:
            parameter_inputs.append(
                {'name': artifact_to_new_name[input_artifact['name']]})
            refactored_inputs[(dag_template['name'],
                               input_artifact['name'])] = artifact_to_new_name[
                input_artifact['name']]
        else:
            new_artifact_inputs.append(input_artifact)
    dag_template['inputs']['artifacts'] = new_artifact_inputs
    dag_template['inputs']['parameters'] = parameter_inputs


def _refactor_dag_template_uri_inputs(
    dag_template: Dict[str, Any],
    refactored_inputs: Dict[Tuple[str, str], str]
) -> None:
    """Refactors artifact inputs within the DAG template.

    An artifact input will be changed to a parameter input if it needs to be
    connected to the producer's pod name output. This is determined by whether
    the input is present in refactored_inputs list, which is generated by the
    container template refactoring process `_refactor_inputs_if_uri_placeholder`

    Args:
        dag_template: The DAG template structure.
        refactored_inputs: The mapping of template and input names to be
            refactored, to its new name.
    """
    # Traverse the tasks in the DAG, and inspect the task arguments.
    for task in dag_template['dag'].get('tasks', []):
        if not task.get('arguments') or not task['arguments'].get('artifacts'):
            continue
        artifact_args = task['arguments']['artifacts']
        new_artifact_args = []
        template_name = task['name']
        parameter_args = task.get('arguments', {}).get('parameters', [])
        for artifact_arg in artifact_args:
            assert 'name' in artifact_arg, (
                    'Illegal artifact format: %s' % artifact_arg)
            if (template_name, artifact_arg['name']) in refactored_inputs:
                # If this is an input artifact that has been refactored.
                # It will be changed to an input parameter receiving the
                # producer's pod name.
                pod_parameter_name = refactored_inputs[
                    (template_name, artifact_arg['name'])]

                # There are two cases for a DAG template.
                assert (artifact_arg.get('from', '').startswith(
                    '{{inputs.') or artifact_arg.get('from', '').startswith(
                    '{{tasks.')), (
                        "Illegal 'from' found for argument %s" % artifact_arg)
                arg_from = artifact_arg['from']
                if arg_from.startswith('{{tasks.'):
                    # 1. The argument to refactor is from another task in the same
                    # DAG.
                    task_matches = re.match(
                        r'{{tasks\.(?P<task_name>.*)\.outputs\.artifacts'
                        r'\.(?P<output_name>.*)}}',
                        arg_from)
                    task_name, output_name = task_matches.group(
                        'task_name'), task_matches.group('output_name')
                    parameter_args.append({
                        'name': pod_parameter_name,
                        'value': '{{{{tasks.{task_name}.outputs.'
                                 'parameters.{output}}}}}'.format(
                            task_name=task_name,
                            output=_components.PRODUCER_POD_NAME_PARAMETER.format(
                                output_name)
                        )})
                else:
                    # 2. The assert above ensures that the argument to refactor
                    # is from an input of the current DAG template.
                    # Then, we'll reconnect it to the DAG input, which has been
                    # renamed.
                    input_matches = re.match(
                        r'{{inputs\.artifacts\.(?P<input_name>.*)}}',
                        arg_from)
                    assert input_matches, (
                            'The matched input is expected to be artifact, '
                            'get parameter instead: %s' % arg_from)
                    # Get the corresponding refactored name of this DAG template
                    new_input = refactored_inputs[(
                        dag_template['name'],
                        input_matches.group('input_name'))]
                    parameter_args.append({
                        'name': pod_parameter_name,
                        'value': '{{{{inputs.parameters.{new_input}}}}}'.format(
                            new_input=new_input,
                        )
                    })
            else:
                # Otherwise this artifact input will be preserved
                new_artifact_args.append(artifact_arg)

        task['arguments']['artifacts'] = new_artifact_args
        task['arguments']['parameters'] = parameter_args


def add_pod_name_passing(
    workflow: Dict[str, Any],
    output_directory: Optional[str] = None) -> Dict[str, Any]:
    """Refactors the workflow structure to pass pod names when needded.

    Args:
        workflow: The workflow structure.
        output_directory: The specified output path.

    Returns:
        Modified workflow structure.

    Raises:
        ValueError: when uri placeholder is used in the workflow but no output
            directory was provided.
    """
    workflow = copy.deepcopy(workflow)

    # Sets of templates representing a container task.
    container_templates = [template for template in
                           workflow['spec']['templates'] if
                           'container' in template]

    # Sets of templates representing a (sub)DAG.
    dag_templates = [template for template in
                     workflow['spec']['templates'] if
                     'dag' in template]

    # 1. If there's an output using outputUri placeholder, then this container
    # template needs to declare an output to pass its pod name to the downstream
    # consumer. The name of the added output will be
    # {{output-name}}-producer-pod-id.
    # Also, eliminate the existing file artifact.
    output_to_filename = {}
    for idx, template in enumerate(container_templates):
        _refactor_outputs_if_uri_placeholder(template, output_to_filename)

    # 2. If there's an input using inputUri placeholder, then this container
    # template needs to declare an input to receive the pod name of the producer
    # task.
    refactored_inputs = {}
    for template in container_templates:
        _refactor_inputs_if_uri_placeholder(
            template, output_to_filename, refactored_inputs)

    # For DAG templates, we need to figure out all the inputs that are
    # eventually being refactored down to the container template level.
    for template in dag_templates:
        _refactor_dag_inputs(template, refactored_inputs)

    # 3. In the DAG templates, wire the pod name inputs/outputs together.
    for template in dag_templates:
        _refactor_dag_template_uri_inputs(template, refactored_inputs)

    # 4. For all the container command/args, replace {{kfp.pipeline_root}}
    # placeholders with the actual output directory specified.
    # Also, the file names need to be reconciled in the consumer to keep
    # consistent with producer.
    for template in container_templates:
        # Process {{kfp.pipeline_root}} placeholders.
        args = template['container'].get('args') or []
        if args:
            new_args = [_replace_output_dir_and_run_id(arg, output_directory)
                        for arg in args]
            template['container']['args'] = new_args

        cmds = template['container'].get('command') or []
        if cmds:
            new_cmds = [_replace_output_dir_and_run_id(cmd, output_directory)
                        for cmd in cmds]
            template['container']['command'] = new_cmds

    clean_up_empty_workflow_structures(workflow)

    return workflow
