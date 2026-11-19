# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Argo DAG scopes, conditional branches and parameter fan-out/fan-in."""

import graphlib
import hashlib
import re

from kfp.compiler.argo_compiler import _json, _kind, _ref
from kfp.dsl import for_loop, pipeline_channel, structures, tasks_group


def _key(channel):
    return (channel.task_name or '', channel.name)


def _port(channel):
    return 'ch-' + hashlib.sha256(repr(_key(channel)).encode()).hexdigest()[:16]


class _Scope:
    def __init__(self, compiler, group, parent=None):
        self.compiler = compiler
        self.group = group
        self.parent = parent
        self.template = {
            'name': compiler._template_name(getattr(group, 'name', 'root')),
            'dag': {'tasks': []},
        }
        compiler.templates.append(self.template)
        self.inputs = {}
        self.exports = {}
        self.children = [
            _Scope(compiler, child, self) for child in group.groups
        ]
        self.owners = {task.name: task for task in group.tasks}
        for child in self.children:
            self.owners.update({name: child for name in child.owners})
        self.loop = isinstance(group, tasks_group.ParallelFor)
        self.exit_handler = isinstance(group, tasks_group.ExitHandler)
        self.cleanup = None
        self.wrapper = None
        if self.loop:
            self.wrapper = {
                'name': compiler._template_name(group.name + '-fanout'),
                'dag': {'tasks': []},
            }
            if group.parallelism_limit:
                self.wrapper['parallelism'] = group.parallelism_limit
            compiler.templates.append(self.wrapper)
        elif self.exit_handler:
            # Both are targets: otherwise a successful cleanup masks a failed
            # body because Argo assesses only leaf targets by default.
            self.wrapper = {
                'name': compiler._template_name(group.name + '-exit'),
                'dag': {'failFast': False, 'target': 'body exit', 'tasks': []},
            }
            compiler.templates.append(self.wrapper)

    @property
    def name(self):
        return (self.wrapper or self.template)['name']

    def input(self, channel, field, prefix='', suffix=''):
        name = (
            channel.name
            if self.parent is None and not channel.task_name
            else _port(channel)
        )
        if self.loop and isinstance(
            channel,
            (for_loop.LoopParameterArgument, for_loop.LoopArgumentVariable),
        ):
            argument = (
                channel.loop_argument
                if isinstance(channel, for_loop.LoopArgumentVariable)
                else channel
            )
            if argument.name == self.group.loop_argument.name:
                self.inputs[_key(channel)] = channel
                name = _port(channel)
        self.inputs[_key(channel)] = channel
        return _ref('inputs.' + field, prefix + name + suffix)

    def resolve(
        self, value, field='parameters', prefix='', suffix='', collected=False
    ):
        if isinstance(value, for_loop.Collected):
            if value.is_artifact_channel:
                raise ValueError(
                    'Collected artifact lists need native multi-artifact staging; use parameter collections for now.'
                )
            return self.resolve(
                value.output, field, prefix, suffix, collected=True
            )
        if not isinstance(value, pipeline_channel.PipelineChannel):
            if isinstance(value, str):
                for (
                    channel
                ) in pipeline_channel.extract_pipeline_channels_from_string(
                    value
                ):
                    value = value.replace(str(channel), self.resolve(channel))
                return value
            return _json(value)
        owner = self.owners.get(value.task_name)
        if owner is not None:
            if isinstance(owner, _Scope):
                if owner.loop and not collected:
                    raise ValueError(
                        'A loop output used outside its scope must be wrapped in dsl.Collected().'
                    )
                if isinstance(owner.group, tasks_group._ConditionBase):
                    raise ValueError(
                        'Conditional output escapes its branch; move its consumer inside the branch. Conditional output merges are not yet supported by this prototype.'
                    )
                owner.exports[_key(value)] = (
                    for_loop.Collected(value)
                    if collected and not owner.loop
                    else value
                )
                name = _port(value)
                target = owner.group.name
            else:
                name = value.name
                target = owner.name
            return _ref(
                f'tasks.{target}.outputs.' + field, prefix + name + suffix
            )
        if value.task_name and self.parent is None:
            raise ValueError(
                f'Channel {value.full_name!r} references a task outside the pipeline scope.'
            )
        return self.input(value, field, prefix, suffix)

    def expression(self, value):
        if not isinstance(value, pipeline_channel.PipelineChannel):
            return _json(value)
        reference = self.resolve(value)[2:-2]
        match = re.fullmatch(
            r'tasks\.([^.]+)\.outputs\.parameters\.(.+)', reference
        )
        if match:
            reference = f'tasks[{_json(match[1])}].outputs.parameters[{_json(match[2])}]'
        else:
            reference = (
                'inputs.parameters['
                + _json(reference.removeprefix('inputs.parameters.'))
                + ']'
            )
        if _kind(structures.InputSpec(type=value.channel_type)) != 'String':
            reference = 'fromJSON(' + reference + ')'
        return reference

    def conditions(self, group):
        expressions = []
        for condition in group.conditions:
            expression = f'({self.expression(condition.left_operand)} {condition.operator} {self.expression(condition.right_operand)})'
            expressions.append(
                '!' + expression if condition.negate else expression
            )
        return '{{=' + ' && '.join(expressions) + '}}'

    def arguments(self, task, final_status=None):
        arguments = {'parameters': [], 'artifacts': []}
        for name, port in (task.component_spec.inputs or {}).items():
            if name not in task.inputs:
                if (
                    port.type == 'PipelineTaskFinalStatus'
                    and final_status is not None
                ):
                    arguments['parameters'].append(
                        {'name': name, 'value': final_status}
                    )
                    continue
                if port.optional:
                    continue
                if port.type == 'PipelineTaskFinalStatus':
                    raise ValueError(
                        'Task-final-status must be supplied by a native exit hook.'
                    )
                raise ValueError(f'Missing input {name!r} on {task.name!r}.')
            value = task.inputs[name]
            if structures.spec_type_is_parameter(port.type):
                arguments['parameters'].append(
                    {'name': name, 'value': self.resolve(value)}
                )
            else:
                arguments['artifacts'].extend(
                    [
                        {
                            'name': name,
                            'from': self.resolve(value, 'artifacts'),
                        },
                        {
                            'name': name + '-metadata',
                            'from': self.resolve(
                                value, 'artifacts', suffix='-metadata'
                            ),
                            'optional': True,
                        },
                    ]
                )
                for prefix in ('kfp-uri-', 'kfp-meta-'):
                    arguments['parameters'].append(
                        {
                            'name': prefix + name,
                            'value': self.resolve(value, prefix=prefix),
                        }
                    )
        return {key: value for key, value in arguments.items() if value}

    def build(self):
        for child in self.children:
            child.build()
        for task in self.group.tasks:
            target = self.compiler.compile_component(task.component_spec, task)
            item = {
                'name': task.name,
                'template': target,
                'arguments': self.arguments(task),
            }
            if task._ignore_upstream_failure_tag:
                raise ValueError(
                    'ignore_upstream_failure requires explicit native continuation semantics; use an exit handler.'
                )
            if task.dependent_tasks:
                item['dependencies'] = [
                    self.owners[name].group.name
                    if isinstance(self.owners.get(name), _Scope)
                    else name
                    for name in task.dependent_tasks
                ]
            self.template['dag']['tasks'].append(item)
        for child in self.children:
            item = {'name': child.group.name, 'template': child.name}
            if isinstance(child.group, tasks_group._ConditionBase):
                item['when'] = self.conditions(child.group)
            self.template['dag']['tasks'].append(item)
        if self.exit_handler:
            task = self.group.exit_task
            target = self.compiler.compile_component(task.component_spec, task)
            status = (
                "{{=toJson({'state': tasks['body'].status == 'Succeeded' ? 'SUCCEEDED' : 'FAILED', 'pipeline_job_resource_name': workflow.uid, 'pipeline_task_name': "
                + _json(self.group.name)
                + '})}}'
            )
            self.cleanup = {
                'name': 'exit',
                'template': target,
                'depends': 'body.Succeeded || body.Failed || body.Errored',
                'arguments': self.arguments(task, status),
            }

    def propagate(self):
        for channel in list(self.exports.values()):
            self.resolve(channel)
        for child in self.children:
            child.propagate()
            for channel in child.inputs.values():
                if not child.is_loop_item(channel):
                    self.resolve(channel)
            if child.loop:
                self.resolve(
                    child.group.loop_argument.items_or_pipeline_channel
                )
        return (
            len(self.inputs)
            + len(self.exports)
            + sum(child.size() for child in self.children)
        )

    def size(self):
        return (
            len(self.inputs)
            + len(self.exports)
            + sum(child.size() for child in self.children)
        )

    def finish(self):
        # Exports may be discovered by a sibling while lowering its arguments.
        for channel in list(self.exports.values()):
            self.resolve(channel)
        for child in self.children:
            child.finish()
        for child in self.children:
            item = next(
                item
                for item in self.template['dag']['tasks']
                if item['name'] == child.group.name
            )
            arguments = {'parameters': [], 'artifacts': []}
            for channel in child.inputs.values():
                if child.is_loop_item(channel):
                    continue
                name = _port(channel)
                if isinstance(
                    channel, pipeline_channel.PipelineArtifactChannel
                ):
                    arguments['artifacts'].extend(
                        [
                            {
                                'name': name,
                                'from': self.resolve(channel, 'artifacts'),
                            },
                            {
                                'name': name + '-metadata',
                                'from': self.resolve(
                                    channel, 'artifacts', suffix='-metadata'
                                ),
                                'optional': True,
                            },
                        ]
                    )
                    for prefix in ('kfp-uri-', 'kfp-meta-'):
                        arguments['parameters'].append(
                            {
                                'name': prefix + name,
                                'value': self.resolve(channel, prefix=prefix),
                            }
                        )
                else:
                    arguments['parameters'].append(
                        {'name': name, 'value': self.resolve(channel)}
                    )
            if child.loop:
                arguments['parameters'].append(
                    {
                        'name': 'kfp-items',
                        'value': self.resolve(
                            child.group.loop_argument.items_or_pipeline_channel
                        ),
                    }
                )
            item['arguments'] = {
                key: value for key, value in arguments.items() if value
            }
        parameters, artifacts = [], []
        for channel in self.inputs.values():
            name = (
                channel.name
                if self.parent is None and not channel.task_name
                else _port(channel)
            )
            if isinstance(channel, pipeline_channel.PipelineArtifactChannel):
                artifacts.extend(
                    [
                        {'name': name},
                        {'name': name + '-metadata', 'optional': True},
                    ]
                )
                parameters.extend(
                    [{'name': 'kfp-uri-' + name}, {'name': 'kfp-meta-' + name}]
                )
            else:
                parameters.append({'name': name})
        self.template['inputs'] = {
            key: value
            for key, value in [
                ('parameters', parameters),
                ('artifacts', artifacts),
            ]
            if value
        }
        self.output_declarations(
            self.template, lambda channel: self.resolve(channel)
        )
        if self.loop:
            wrapper_inputs = {
                'parameters': [{'name': 'kfp-items'}],
                'artifacts': [],
            }
            arguments = {'parameters': [], 'artifacts': []}
            for channel in self.inputs.values():
                name = _port(channel)
                if self.is_loop_item(channel):
                    item = '{{item}}'
                    if isinstance(channel, for_loop.LoopArgumentVariable):
                        item = '{{item.' + channel.subvar_name + '}}'
                    arguments['parameters'].append(
                        {'name': name, 'value': item}
                    )
                elif isinstance(
                    channel, pipeline_channel.PipelineArtifactChannel
                ):
                    for suffix in ('', '-metadata'):
                        wrapper_inputs['artifacts'].append(
                            {'name': name + suffix, 'optional': bool(suffix)}
                        )
                        arguments['artifacts'].append(
                            {
                                'name': name + suffix,
                                'from': _ref('inputs.artifacts', name + suffix),
                                'optional': bool(suffix),
                            }
                        )
                    for prefix in ('kfp-uri-', 'kfp-meta-'):
                        wrapper_inputs['parameters'].append(
                            {'name': prefix + name}
                        )
                        arguments['parameters'].append(
                            {
                                'name': prefix + name,
                                'value': _ref(
                                    'inputs.parameters', prefix + name
                                ),
                            }
                        )
                else:
                    wrapper_inputs['parameters'].append({'name': name})
                    arguments['parameters'].append(
                        {'name': name, 'value': _ref('inputs.parameters', name)}
                    )
            self.wrapper['inputs'] = {
                key: value for key, value in wrapper_inputs.items() if value
            }
            self.wrapper['dag']['tasks'] = [
                {
                    'name': 'iteration',
                    'template': self.template['name'],
                    'withParam': '{{inputs.parameters.kfp-items}}',
                    'arguments': {
                        key: value for key, value in arguments.items() if value
                    },
                }
            ]
            self.output_declarations(
                self.wrapper,
                lambda channel: _ref(
                    'tasks.iteration.outputs.parameters', _port(channel)
                ),
            )
        if self.exit_handler:
            self.wrapper['inputs'] = self.template['inputs']
            arguments = {'parameters': [], 'artifacts': []}
            for field, ports in self.template['inputs'].items():
                for port in ports:
                    arguments[field].append(
                        {
                            'name': port['name'],
                            'value' if field == 'parameters' else 'from': _ref(
                                'inputs.' + field, port['name']
                            ),
                            **(
                                {'optional': True}
                                if port.get('optional')
                                else {}
                            ),
                        }
                    )
            self.wrapper['dag']['tasks'] = [
                {
                    'name': 'body',
                    'template': self.template['name'],
                    'arguments': {
                        key: value for key, value in arguments.items() if value
                    },
                },
                self.cleanup,
            ]
            self.output_declarations(
                self.wrapper,
                lambda channel: _ref(
                    'tasks.body.outputs.parameters', _port(channel)
                ),
            )
        dependencies = {}
        for item in self.template['dag']['tasks']:
            references = re.findall(
                r'tasks\.([a-zA-Z0-9_-]+)\.', _json(item)
            ) + re.findall(r'tasks\[\\?"([^"\\]+)\\?"\]', _json(item))
            deps = set(item.get('dependencies', [])) | set(references)
            deps.discard(item['name'])
            if deps:
                item['dependencies'] = sorted(deps)
            dependencies[item['name']] = deps
        missing = (
            set().union(*dependencies.values()) - set(dependencies)
            if dependencies
            else set()
        )
        if missing:
            raise ValueError(
                f'Dependency crosses a group boundary: {sorted(missing)}.'
            )
        try:
            tuple(graphlib.TopologicalSorter(dependencies).static_order())
        except graphlib.CycleError as error:
            raise ValueError(
                'Pipeline contains a cyclic dependency.'
            ) from error

    def is_loop_item(self, channel):
        if not self.loop:
            return False
        if isinstance(channel, for_loop.LoopArgumentVariable):
            channel = channel.loop_argument
        return (
            isinstance(channel, for_loop.LoopParameterArgument)
            and channel.name == self.group.loop_argument.name
        )

    def output_declarations(self, template, reference):
        parameters, artifacts = [], []
        for channel in self.exports.values():
            name = _port(channel)
            if isinstance(channel, pipeline_channel.PipelineArtifactChannel):
                if self.loop:
                    raise ValueError(
                        'Argo does not aggregate output artifacts; use a supported parameter collection.'
                    )
                wrapped = self.exit_handler and template is self.wrapper
                source = (
                    _ref('tasks.body.outputs.artifacts', name)
                    if wrapped
                    else self.resolve(channel, 'artifacts')
                )
                artifacts.append({'name': name, 'from': source})
                metadata_source = (
                    _ref('tasks.body.outputs.artifacts', name + '-metadata')
                    if wrapped
                    else self.resolve(channel, 'artifacts', suffix='-metadata')
                )
                artifacts.append(
                    {
                        'name': name + '-metadata',
                        'from': metadata_source,
                        'optional': True,
                    }
                )
                for prefix in ('kfp-uri-', 'kfp-meta-'):
                    source = (
                        _ref('tasks.body.outputs.parameters', prefix + name)
                        if wrapped
                        else self.resolve(channel, prefix=prefix)
                    )
                    parameters.append(
                        {
                            'name': prefix + name,
                            'valueFrom': {'parameter': source},
                        }
                    )
            else:
                parameters.append(
                    {
                        'name': name,
                        'valueFrom': {'parameter': reference(channel)},
                    }
                )
        if parameters or artifacts:
            template['outputs'] = {
                key: value
                for key, value in [
                    ('parameters', parameters),
                    ('artifacts', artifacts),
                ]
                if value
            }


def compile_graph(compiler, component_spec, graph):
    scope = _Scope(compiler, graph.pipeline.groups[0])
    scope.build()
    outputs = graph.outputs
    if outputs is not None:
        outputs = (
            outputs._asdict()
            if hasattr(outputs, '_asdict')
            else outputs
            if isinstance(outputs, dict)
            else {'Output': outputs}
        )
        for channel in outputs.values():
            scope.resolve(channel)
    previous = -1
    for _ in range(len(compiler.templates) + 1):
        size = scope.propagate()
        if size == previous:
            break
        previous = size
    scope.finish()
    # Root parameters must keep declarations even if unused, including defaults.
    ports = {'parameters': [], 'artifacts': []}
    for name, port in (component_spec.inputs or {}).items():
        compiler.input_port(ports, name, port)
    scope.template['inputs'] = {
        key: value for key, value in ports.items() if value
    }
    if outputs:
        parameters = []
        for name, channel in outputs.items():
            if not structures.spec_type_is_parameter(
                component_spec.outputs[name].type
            ):
                raise ValueError(
                    'Grouped pipeline artifact outputs require explicit merge semantics.'
                )
            parameters.append(
                {
                    'name': name,
                    'valueFrom': {'parameter': scope.resolve(channel)},
                }
            )
        scope.template['outputs'] = {'parameters': parameters}
    return scope.name
