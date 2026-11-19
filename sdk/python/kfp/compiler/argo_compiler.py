# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Lower captured DSL objects directly to Argo templates (no PipelineSpec
IR)."""

import copy
import dataclasses
import graphlib
import hashlib
import json
import re
from urllib.parse import urlparse

from kfp.dsl import (
    for_loop,
    pipeline_channel,
    pipeline_task,
    structures,
    tasks_group,
)
from kfp.dsl.argo_runtime import SCHEMA, serialize
from kfp.dsl.types import type_utils


def _json(value):
    return json.dumps(
        value, sort_keys=True, separators=(',', ':'), allow_nan=False
    )


def _kind(port):
    if not structures.spec_type_is_parameter(port.type):
        return port.type
    name = type_utils.get_canonical_name_for_outer_generic(port.type).lower()
    kinds = {
        'str': 'String',
        'string': 'String',
        'int': 'Integer',
        'integer': 'Integer',
        'float': 'Float',
        'double': 'Float',
        'bool': 'Boolean',
        'boolean': 'Boolean',
        'list': 'List',
        'dict': 'Dict',
        'pipelinetaskfinalstatus': 'Dict',
    }
    if name not in kinds:
        raise ValueError(
            f'Unsupported parameter type {port.type!r}; use a supported scalar, list or dict.'
        )
    return kinds[name]


def _ref(scope, name):
    return '{{' + scope + '.' + name + '}}'


def local_image_reference(image: str, digest: str) -> str:
    """Name a local image snapshot by its verified Docker build identity."""
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', digest):
        raise ValueError('Local image identity must be a full sha256 digest.')
    repository = image.split('@')[0]
    if repository.rfind(':') > repository.rfind('/'):
        repository = repository[: repository.rfind(':')]
    return repository + ':kfp-' + digest.removeprefix('sha256:')


def _name(value):
    return re.sub('[^a-z0-9-]', '-', value.lower()).strip('-') or 'pipeline'


class _Compiler:
    def __init__(self, options):
        self.options = options
        self.templates = []
        self.disable_caching = options.disable_caching
        repository = urlparse(options.artifact_repository)
        if (
            repository.scheme != 's3'
            or not repository.netloc
            or repository.query
            or repository.fragment
        ):
            raise ValueError(
                'The native prototype requires an s3://bucket[/prefix] artifact_repository.'
            )
        self.bucket = repository.netloc
        self.prefix = repository.path.strip('/')
        self.uri = options.artifact_repository.rstrip('/')

    def _template_name(self, name):
        return f't{len(self.templates)}-{_name(name)[:45]}'

    def value(self, value, field='parameters', suffix=''):
        if isinstance(value, pipeline_channel.PipelineChannel):
            scope = (
                f'tasks.{value.task_name}.outputs'
                if value.task_name
                else 'inputs'
            )
            return _ref(scope + '.' + field, value.name + suffix)
        if isinstance(value, str):
            for (
                channel
            ) in pipeline_channel.extract_pipeline_channels_from_string(value):
                value = value.replace(str(channel), self.value(channel))
            return value
        return _json(value)

    def compile_component(self, component_spec, task=None):
        graph = component_spec.implementation.graph
        if graph is not None:
            if not isinstance(graph, structures.CapturedGraph):
                raise ValueError(
                    'Precompiled KFP IR graphs are not supported; compile the original Python pipeline.'
                )
            if task and (
                task._task_spec.retry_policy
                or task._task_spec.cache_key
                or task.platform_config
            ):
                raise ValueError(
                    'Nested DAG retry/custom-cache/platform settings need native DAG lowering; configure the inner tasks instead.'
                )
            inherited = self.disable_caching
            self.disable_caching = inherited or bool(
                task and not task._task_spec.enable_caching
            )
            try:
                name = self.graph(component_spec, graph)
                graph_display_name = (
                    task._task_spec.display_name if task else None
                ) or graph.display_name
                if graph_display_name:
                    target = next(
                        template
                        for template in self.templates
                        if template['name'] == name
                    )
                    target.setdefault('metadata', {}).setdefault(
                        'annotations', {}
                    )['workflows.argoproj.io/display-name'] = graph_display_name
                return name
            finally:
                self.disable_caching = inherited
        if component_spec.implementation.importer is not None:
            raise ValueError(
                'Importer components need a versioned native artifact source; replace the importer before compiling.'
            )
        if task is None:
            task = pipeline_task.PipelineTask(component_spec, {})
        return self.leaf(task)

    def graph(self, component_spec, graph):
        root = graph.pipeline.groups[0]
        if root.groups:
            from kfp.compiler.argo_groups import compile_graph

            return compile_graph(self, component_spec, graph)
        template = {
            'name': self._template_name(component_spec.name),
            'dag': {'tasks': []},
        }
        self.templates.append(template)
        inputs = {'parameters': [], 'artifacts': []}
        for name, port in (component_spec.inputs or {}).items():
            self.input_port(inputs, name, port)
        if any(inputs.values()):
            template['inputs'] = {
                key: value for key, value in inputs.items() if value
            }
        dependencies = {}
        for task in root.tasks:
            target = self.compile_component(task.component_spec, task)
            arguments = {'parameters': [], 'artifacts': []}
            for name, port in (task.component_spec.inputs or {}).items():
                if name in task.inputs:
                    value = task.inputs[name]
                elif port.optional and port.default is not None:
                    value = port.default
                elif port.optional:
                    continue
                else:
                    raise ValueError(
                        f'Task {task.name!r} is missing input {name!r}.'
                    )
                if structures.spec_type_is_parameter(port.type):
                    arguments['parameters'].append(
                        {
                            'name': name,
                            'value': self.value(value)
                            if isinstance(
                                value, (str, pipeline_channel.PipelineChannel)
                            )
                            else serialize(value, _kind(port)),
                        }
                    )
                else:
                    if not isinstance(
                        value, pipeline_channel.PipelineArtifactChannel
                    ):
                        raise ValueError(
                            f'Artifact {name!r} must come from an upstream output or typed pipeline input.'
                        )
                    scope = (
                        f'tasks.{value.task_name}.outputs.parameters'
                        if value.task_name
                        else 'inputs.parameters'
                    )
                    arguments['artifacts'].extend(
                        [
                            {
                                'name': name,
                                'from': self.value(value, 'artifacts'),
                            },
                            {
                                'name': name + '-metadata',
                                'from': self.value(
                                    value, 'artifacts', '-metadata'
                                ),
                                'optional': True,
                            },
                        ]
                    )
                    for prefix in ('kfp-uri-', 'kfp-meta-'):
                        arguments['parameters'].append(
                            {
                                'name': prefix + name,
                                'value': _ref(scope, prefix + value.name),
                            }
                        )
            deps = set(task.dependent_tasks)
            deps.update(
                channel.task_name
                for channel in task.channel_inputs
                if channel.task_name
            )
            dependencies[task.name] = deps
            dag_task = {'name': task.name, 'template': target}
            if deps:
                dag_task['dependencies'] = sorted(deps)
            if any(arguments.values()):
                dag_task['arguments'] = {
                    key: value for key, value in arguments.items() if value
                }
            if task._ignore_upstream_failure_tag:
                raise ValueError(
                    f'Task {task.name}: ignore_upstream_failure requires native exit/continuation lowering.'
                )
            template['dag']['tasks'].append(dag_task)
        missing = (
            set().union(*dependencies.values()) - set(dependencies)
            if dependencies
            else set()
        )
        if missing:
            raise ValueError(
                f'Task dependencies cross an unsupported scope: {sorted(missing)}.'
            )
        try:
            tuple(graphlib.TopologicalSorter(dependencies).static_order())
        except graphlib.CycleError as error:
            raise ValueError(
                'Pipeline has a dependency cycle; remove the cyclic .after() or data dependency.'
            ) from error
        outputs = graph.outputs
        if outputs is not None:
            if hasattr(outputs, '_asdict'):
                outputs = outputs._asdict()
            elif not isinstance(outputs, dict):
                outputs = {'Output': outputs}
            template['outputs'] = {'parameters': [], 'artifacts': []}
            for name, value in outputs.items():
                port = component_spec.outputs[name]
                if not isinstance(value, pipeline_channel.PipelineChannel):
                    raise ValueError(
                        'Pipeline outputs must reference task output channels.'
                    )
                if structures.spec_type_is_parameter(port.type):
                    template['outputs']['parameters'].append(
                        {
                            'name': name,
                            'valueFrom': {'parameter': self.value(value)},
                        }
                    )
                else:
                    template['outputs']['artifacts'].extend(
                        [
                            {
                                'name': name,
                                'from': self.value(value, 'artifacts'),
                            },
                            {
                                'name': name + '-metadata',
                                'from': self.value(
                                    value, 'artifacts', '-metadata'
                                ),
                                'optional': True,
                            },
                        ]
                    )
                    for prefix in ('kfp-uri-', 'kfp-meta-'):
                        template['outputs']['parameters'].append(
                            {
                                'name': prefix + name,
                                'valueFrom': {
                                    'parameter': _ref(
                                        f'tasks.{value.task_name}.outputs.parameters',
                                        prefix + value.name,
                                    )
                                },
                            }
                        )
            template['outputs'] = {
                key: value
                for key, value in template['outputs'].items()
                if value
            }
        return template['name']

    def input_port(self, inputs, name, port):
        if port.is_artifact_list:
            raise ValueError(
                f'Artifact list {name!r} needs native collection lowering; use individual typed artifacts.'
            )
        if structures.spec_type_is_parameter(port.type):
            item = {'name': name}
            if port.optional and port.default is not None:
                item['default'] = serialize(port.default, _kind(port))
            inputs['parameters'].append(item)
        else:
            inputs['artifacts'].extend(
                [
                    {'name': name, 'path': '/tmp/kfp/inputs/' + name},
                    {
                        'name': name + '-metadata',
                        'path': '/tmp/kfp/inputs/' + name + '.json',
                        'optional': True,
                    },
                ]
            )
            inputs['parameters'].extend(
                [{'name': 'kfp-uri-' + name}, {'name': 'kfp-meta-' + name}]
            )

    def leaf(self, task):
        container = task.container_spec
        if container is None:
            raise ValueError(
                f'Component {task.name!r} has no supported container implementation.'
            )
        if pipeline_channel.extract_pipeline_channels_from_string(
            container.image
        ):
            raise ValueError(
                'Dynamic images need native input binding; use a fixed image for this prototype.'
            )
        if any(
            name.startswith('KFP_DEBUG_PAUSE_')
            for name in (container.env or {})
        ):
            raise ValueError(
                'KFP launcher debug pauses are unavailable; add a native Argo suspend template instead.'
            )
        template = {'name': self._template_name(task.name)}
        self.templates.append(template)
        spec = task.component_spec
        inputs = {'parameters': [], 'artifacts': []}
        outputs = {'parameters': [], 'artifacts': []}
        contract = {
            'schema': SCHEMA,
            'outputDirectory': '/tmp/kfp/outputs',
            'inputs': {},
            'outputs': {},
        }
        replacements = {}
        for name, port in (spec.inputs or {}).items():
            if (
                port.optional
                and port.default is None
                and name not in task.inputs
                and port.type != 'PipelineTaskFinalStatus'
            ):
                continue
            self.input_port(inputs, name, port)
            declaration = {'type': _kind(port)}
            if structures.spec_type_is_parameter(port.type):
                declaration['optional'] = port.optional
                supplied = task.inputs.get(name)
                if isinstance(supplied, for_loop.Collected):
                    if supplied.is_artifact_channel:
                        raise ValueError(
                            'Collected artifact lists require native multi-artifact staging.'
                        )
                    declaration['collectionElementType'] = _kind(
                        structures.InputSpec(type=supplied.output.channel_type)
                    )
                    consumer_groups = set()
                    parent = task.parent_task_group
                    while parent is not None:
                        consumer_groups.add(id(parent))
                        parent = parent.parent_task_group
                    depth = 0
                    parent = supplied.output.task.parent_task_group
                    while (
                        parent is not None and id(parent) not in consumer_groups
                    ):
                        depth += isinstance(parent, tasks_group.ParallelFor)
                        parent = parent.parent_task_group
                    declaration['collectionDepth'] = depth
                replacements[f"{{{{$.inputs.parameters['{name}']}}}}"] = _ref(
                    'inputs.parameters', name
                )
            else:
                declaration.update(
                    path='/tmp/kfp/inputs/' + name,
                    metadataPath='/tmp/kfp/inputs/' + name + '.json',
                )
                replacements[f"{{{{$.inputs.artifacts['{name}'].path}}}}"] = (
                    declaration['path']
                )
                replacements[f"{{{{$.inputs.artifacts['{name}'].uri}}}}"] = (
                    _ref('inputs.parameters', 'kfp-uri-' + name)
                )
            contract['inputs'][name] = declaration
        for name, port in (spec.outputs or {}).items():
            path = '/tmp/kfp/outputs/' + name
            declaration = {'type': _kind(port), 'path': path}
            if structures.spec_type_is_parameter(port.type):
                outputs['parameters'].append(
                    {'name': name, 'valueFrom': {'path': path}}
                )
                replacements[
                    f"{{{{$.outputs.parameters['{name}'].output_file}}}}"
                ] = path
            else:
                key = '/'.join(
                    filter(
                        None,
                        [self.prefix, '{{workflow.uid}}', '{{pod.name}}', name],
                    )
                )
                uri = self.uri + '/{run}/{task}/' + name
                declaration.update(
                    metadataPath='/tmp/kfp/metadata/' + name + '.json',
                    identityPath='/tmp/kfp/identities/' + name + '.json',
                    uri=uri,
                )
                outputs['artifacts'].extend(
                    [
                        {
                            'name': name,
                            'path': path,
                            'archive': {'none': {}},
                            's3': {'bucket': self.bucket, 'key': key},
                        },
                        {
                            'name': name + '-metadata',
                            'path': declaration['metadataPath'],
                            'archive': {'none': {}},
                            's3': {
                                'bucket': self.bucket,
                                'key': '/'.join(
                                    filter(
                                        None,
                                        [
                                            self.prefix,
                                            '{{workflow.uid}}',
                                            '{{pod.name}}',
                                            'metadata',
                                            name + '.json',
                                        ],
                                    )
                                ),
                            },
                        },
                    ]
                )
                outputs['parameters'].append(
                    {
                        'name': 'kfp-uri-' + name,
                        'value': 's3://' + self.bucket + '/' + key,
                    }
                )
                replacements[f"{{{{$.outputs.artifacts['{name}'].path}}}}"] = (
                    path
                )
                replacements[f"{{{{$.outputs.artifacts['{name}'].uri}}}}"] = (
                    's3://' + self.bucket + '/' + key
                )
            contract['outputs'][name] = declaration
        args = list(container.args or [])
        python = '--executor_input' in args and '{{$}}' in args
        if python:
            index = args.index('--executor_input')
            if args[index + 1] != '{{$}}':
                raise ValueError(
                    f'Task {task.name!r} has a nonstandard executor input; use a standard Python component.'
                )
            args[index : index + 2] = [
                '--argo_input',
                _json(contract),
                '--argo_parameters',
                '{{inputs.parameters}}' if inputs['parameters'] else '[]',
            ]
            outputs['artifacts'].append(
                {
                    'name': 'kfp-metadata',
                    'path': '/tmp/kfp/outputs/metadata.json',
                    'archive': {'none': {}},
                    's3': {
                        'bucket': self.bucket,
                        'key': '/'.join(
                            filter(
                                None,
                                [
                                    self.prefix,
                                    '{{workflow.uid}}',
                                    '{{pod.name}}',
                                    'kfp-metadata.json',
                                ],
                            )
                        ),
                    },
                }
            )
            for name, port in contract['outputs'].items():
                if 'metadataPath' in port:
                    outputs['parameters'].append(
                        {
                            'name': 'kfp-meta-' + name,
                            'valueFrom': {'path': port['identityPath']},
                        }
                    )
        else:
            if any(
                structures.spec_type_is_parameter(port.type)
                and _kind(port) != 'String'
                for port in (spec.outputs or {}).values()
            ):
                raise ValueError(
                    'Non-Python container scalar outputs must be String; use a Python component bridge for validated numeric, boolean, list or dict returns.'
                )
            self.native_metadata(template, contract, outputs)
            # Container images need not contain Python or a shell. Use /tmp,
            # which already exists, rather than requiring a wrapper mkdir.
            for declaration in contract['outputs'].values():
                old = declaration['path']
                new = '/tmp/kfp-' + old.rsplit('/', 1)[-1]
                replacements = {
                    key: value.replace(old, new)
                    for key, value in replacements.items()
                }
                for output in outputs['parameters']:
                    if output.get('valueFrom', {}).get('path') == old:
                        output['valueFrom']['path'] = new
                for output in outputs['artifacts']:
                    if output.get('path') == old:
                        output['path'] = new

        def convert(value):
            if not isinstance(value, str):
                value = _json(value)
            if pipeline_channel.extract_pipeline_channels_from_string(value):
                raise ValueError(
                    f'Task {task.name}: dynamic execution configuration needs explicit native input binding.'
                )
            for old, new in replacements.items():
                value = value.replace(old, new)
            if '{{$.' in value or '{{$}}' in value:
                raise ValueError(
                    f'Task {task.name!r} contains an unsupported KFP placeholder; replace it with a supported parameter/path.'
                )
            return value

        def command_elements(value):
            if isinstance(value, str) and (
                value.startswith('{"Concat":')
                or value.startswith('{"IfPresent":')
            ):
                value = json.loads(value)
            if isinstance(value, dict) and 'Concat' in value:
                return [
                    ''.join(
                        part
                        for item in value['Concat']
                        for part in command_elements(item)
                    )
                ]
            if isinstance(value, dict) and 'IfPresent' in value:
                declaration = value['IfPresent']
                name = declaration['InputName']
                present = (
                    name in task.inputs
                    or name in (spec.outputs or {})
                    or (
                        name in (spec.inputs or {})
                        and spec.inputs[name].default is not None
                    )
                )
                branch = declaration.get('Then' if present else 'Else', [])
                return [
                    part
                    for item in (
                        branch if isinstance(branch, list) else [branch]
                    )
                    for part in command_elements(item)
                ]
            return [convert(value)]

        digest = self.options.image_digests.get(container.image)
        rendered = {
            'image': local_image_reference(container.image, digest)
            if digest
            else container.image
        }
        if container.command:
            rendered['command'] = [
                part
                for value in container.command
                for part in command_elements(value)
            ]
        if args:
            rendered['args'] = [
                part for value in args for part in command_elements(value)
            ]
        rendered['env'] = [
            {'name': name, 'value': convert(value)}
            for name, value in (container.env or {}).items()
        ]
        if python:
            rendered['env'].extend(
                [
                    {'name': 'KFP_ARGO_RUN_UID', 'value': '{{workflow.uid}}'},
                    {'name': 'KFP_ARGO_NODE_ID', 'value': '{{pod.name}}'},
                    {
                        'name': 'KFP_ARGO_NAMESPACE',
                        'value': '{{workflow.namespace}}',
                    },
                ]
            )
        if container.resources:
            resources = dataclasses.asdict(container.resources)
            rendered['resources'] = {}
            for suffix, target in [
                ('request', 'requests'),
                ('limit', 'limits'),
            ]:
                values = {
                    kind: resources[kind + '_' + suffix]
                    for kind in ('cpu', 'memory')
                    if resources[kind + '_' + suffix] is not None
                }
                if values:
                    rendered['resources'][target] = values
            if resources['accelerator_count'] or resources['accelerator_type']:
                if (
                    not resources['accelerator_type']
                    or not resources['accelerator_count']
                ):
                    raise ValueError(
                        'Set both accelerator type and count for native GPU resources.'
                    )
                rendered['resources'].setdefault('limits', {})[
                    resources['accelerator_type']
                ] = resources['accelerator_count']
        template['container'] = rendered
        for key, ports in [('inputs', inputs), ('outputs', outputs)]:
            if any(ports.values()):
                template[key] = {
                    name: values for name, values in ports.items() if values
                }
        self.platform(task, template)
        if pipeline_channel.extract_pipeline_channels_from_string(
            _json(template)
        ):
            raise ValueError(
                f'Task {task.name!r} has dynamic execution configuration without native input binding; use fixed values for this prototype.'
            )
        retry = task._task_spec.retry_policy
        if retry:
            count = (
                retry.max_retry_count
                if retry.max_retry_count is not None
                else 0
            )
            if (
                isinstance(count, bool)
                or not isinstance(count, int)
                or count < 0
            ):
                raise ValueError('num_retries must be a non-negative integer.')
            factor = (
                retry.backoff_factor if retry.backoff_factor is not None else 2
            )
            if (
                isinstance(factor, bool)
                or not isinstance(factor, (int, float))
                or factor < 1
                or not float(factor).is_integer()
            ):
                raise ValueError(
                    'Argo requires a positive integer backoff_factor; use e.g. 2 or 2.0, not a fractional factor.'
                )
            template['retryStrategy'] = {
                'limit': str(count),
                'retryPolicy': retry.policy or 'OnError',
                'backoff': {
                    'duration': f'{structures.convert_duration_to_seconds(retry.backoff_duration or "0s")}s',
                    'factor': str(int(factor)),
                    'maxDuration': f'{structures.convert_duration_to_seconds(retry.backoff_max_duration or "3600s")}s',
                },
            }
        if task._task_spec.enable_caching and not self.disable_caching:
            external = set(task.platform_config.get('kubernetes', {})) & {
                'secretAsEnv',
                'configMapAsEnv',
                'secretAsVolume',
                'configMapAsVolume',
                'pvcMount',
            }
            if external:
                raise ValueError(
                    f'Task {task.name!r} reads mutable external configuration {sorted(external)}; disable caching or bake versioned inputs into the image/artifact contract.'
                )
            if any(
                'PIP_DISABLE_PIP_VERSION_CHECK=1' in command
                for command in (container.command or [])
            ):
                raise ValueError(
                    f'Task {task.name!r} installs packages at execution time; bake them into an immutable image or disable caching.'
                )
            identity = self.options.image_digests.get(
                container.image, container.image.split('@')[-1]
            )
            if not re.fullmatch(r'sha256:[0-9a-f]{64}', identity):
                raise ValueError(
                    f'Caching {task.name!r} requires an immutable image: pin {container.image!r} by digest, provide Compiler(image_digests=...), or call set_caching_options(False).'
                )
            fingerprint = hashlib.sha256(
                _json(
                    {
                        'template': template,
                        'image': identity,
                        'cache_key': task._task_spec.cache_key,
                    }
                ).encode()
            ).hexdigest()
            template['memoize'] = {
                'key': "{{=sprig.sha256sum('"
                + fingerprint
                + "' + toJson(inputs.parameters))}}",
                'maxAge': '1h',
                'cache': {'configMap': {'name': self.options.cache_config_map}},
            }
        if task._task_spec.display_name:
            template.setdefault('metadata', {}).setdefault('annotations', {})[
                'workflows.argoproj.io/display-name'
            ] = task._task_spec.display_name
        return template['name']

    def native_metadata(self, template, contract, outputs):
        """Emit small static metadata as parameters; Argo stages raw
        companions."""
        expressions = {}

        def expression(value):
            marker = f'__kfp_expression_{len(expressions)}__'
            expressions[_json(marker)] = value
            return marker

        producer = {
            'run': expression('workflow.uid'),
            'task': expression('pod.name'),
            'namespace': expression('workflow.namespace'),
        }
        references = []
        for name, port in contract['inputs'].items():
            if 'path' not in port:
                continue
            uri = f'inputs.parameters[{_json("kfp-uri-" + name)}]'
            source = f'fromJSON(inputs.parameters[{_json("kfp-meta-" + name)}])'
            references.append(
                {
                    'port': name,
                    'uri': expression(uri),
                    'producer': expression(source + '.producer'),
                }
            )
        artifacts = []
        for name, port in contract['outputs'].items():
            if 'metadataPath' not in port:
                continue
            uri = (
                _json(self.uri + '/')
                + ' + workflow.uid + "/" + pod.name + '
                + _json('/' + name)
            )
            metadata = {
                'schema': SCHEMA,
                'port': name,
                'type': port['type'],
                'producer': producer,
                'uri': expression(uri),
                'metadata': {},
                'inputs': references,
            }
            artifacts.append(metadata)
            companion = next(
                output
                for output in outputs['artifacts']
                if output['name'] == name + '-metadata'
            )
            companion['optional'] = True
            outputs['parameters'].append(
                {'name': 'kfp-meta-' + name, 'value': metadata}
            )
        outputs['parameters'].append(
            {
                'name': 'kfp-metadata',
                'value': {
                    'schema': SCHEMA,
                    'producer': producer,
                    'inputs': references,
                    'artifacts': artifacts,
                    'metrics': {},
                },
            }
        )
        for parameter in outputs['parameters']:
            if not isinstance(parameter.get('value'), dict):
                continue
            value = _json(parameter['value'])
            for marker, replacement in expressions.items():
                value = value.replace(marker, replacement)
            parameter['value'] = '{{=toJson(' + value + ')}}'

    def platform(self, task, template):
        if set(task.platform_config) - {'kubernetes'}:
            raise ValueError(
                'Only kfp-kubernetes task settings have native lowering; remove unsupported platform settings.'
            )
        config = copy.deepcopy(task.platform_config.get('kubernetes', {}))
        container = template['container']
        for source, destination in [
            ('nodeSelector', 'nodeSelector'),
            ('podMetadata', 'metadata'),
            ('tolerations', 'tolerations'),
            ('activeDeadlineSeconds', 'activeDeadlineSeconds'),
        ]:
            if source in config:
                value = config.pop(source)
                if source == 'nodeSelector':
                    if set(value) - {'labels'}:
                        raise ValueError(
                            'Dynamic node selectors are not yet supported; use literal labels.'
                        )
                    value = value.get('labels', {})
                if source == 'activeDeadlineSeconds':
                    value = int(value)
                template[destination] = value
        if 'imagePullPolicy' in config:
            container['imagePullPolicy'] = config.pop('imagePullPolicy')
        for source, kind, field in [
            ('secretAsEnv', 'secretKeyRef', 'secret'),
            ('configMapAsEnv', 'configMapKeyRef', 'configMap'),
        ]:
            for declaration in config.pop(source, []):
                dynamic = declaration.get(field + 'NameParameter', {}).get(
                    'runtimeValue', {}
                )
                if dynamic and 'constant' not in dynamic:
                    raise ValueError(
                        'Dynamic secret/config-map names need native parameter lowering; use a literal name.'
                    )
                name = dynamic.get('constant', declaration.get(field + 'Name'))
                for entry in declaration['keyToEnv']:
                    container.setdefault('env', []).append(
                        {
                            'name': entry['envVar'],
                            'valueFrom': {
                                kind: {
                                    'name': name,
                                    'key': entry[field + 'Key'],
                                    'optional': declaration.get(
                                        'optional', False
                                    ),
                                }
                            },
                        }
                    )
        for index, mount in enumerate(config.pop('pvcMount', [])):
            name = (
                mount.get('pvcNameParameter', {})
                .get('runtimeValue', {})
                .get('constant', mount.get('constant'))
            )
            if not name:
                raise ValueError(
                    'Dynamic PVC names are not yet supported; use a literal existing PVC name.'
                )
            volume = f'kfp-pvc-{index}'
            template.setdefault('volumes', []).append(
                {'name': volume, 'persistentVolumeClaim': {'claimName': name}}
            )
            container.setdefault('volumeMounts', []).append(
                {
                    'name': volume,
                    'mountPath': mount['mountPath'],
                    **(
                        {'subPath': mount['subPath']}
                        if 'subPath' in mount
                        else {}
                    ),
                }
            )
        if config:
            raise ValueError(
                f'Kubernetes settings have no native lowering yet: {sorted(config)}. Remove them or add an explicit mapping.'
            )


def compile_argo(
    component, options, pipeline_name, display_name, parameters, kind, schedule
):
    if kind not in {'WorkflowTemplate', 'Workflow', 'CronWorkflow'}:
        raise ValueError(
            'resource_kind must be WorkflowTemplate, Workflow or CronWorkflow.'
        )
    if (kind == 'CronWorkflow') != bool(schedule):
        raise ValueError(
            'Specify schedule only when resource_kind="CronWorkflow".'
        )
    builder = _Compiler(options)
    entrypoint = builder.compile_component(component.component_spec)
    arguments = []
    unknown = set(parameters) - set(component.component_spec.inputs or {})
    if unknown:
        raise ValueError(f'Unknown pipeline parameters: {sorted(unknown)}.')
    for name, port in (component.component_spec.inputs or {}).items():
        if not structures.spec_type_is_parameter(port.type):
            raise ValueError(
                'Root artifact inputs need an explicit native artifact source; pass a produced artifact through a nested DAG.'
            )
        declaration = {'name': name}
        value = parameters.get(name, port.default)
        if value is not None:
            declaration['value'] = serialize(value, _kind(port))
        arguments.append(declaration)
    spec = {
        'entrypoint': entrypoint,
        'serviceAccountName': options.service_account_name,
        'arguments': {'parameters': arguments},
        'templates': builder.templates,
    }
    graph = component.component_spec.implementation.graph
    if isinstance(graph, structures.CapturedGraph):
        if graph.pipeline_root and graph.pipeline_root.rstrip(
            '/'
        ) != options.artifact_repository.rstrip('/'):
            raise ValueError(
                'pipeline_root must match Compiler(artifact_repository=...) so recorded artifact URIs match storage.'
            )
        if graph.config:
            if graph.config.workspace:
                raise ValueError(
                    'Pipeline workspaces need native volumeClaimTemplates; use explicit PVC mounts until this mapping is implemented.'
                )
            for source, destination in [
                ('ttl_seconds_after_finished', 'secondsAfterCompletion'),
                ('ttl_seconds_after_success', 'secondsAfterSuccess'),
                ('ttl_seconds_after_failure', 'secondsAfterFailure'),
            ]:
                value = getattr(graph.config, source)
                if value is not None:
                    spec.setdefault('ttlStrategy', {})[destination] = value
            if graph.config.active_deadline_seconds is not None:
                spec['activeDeadlineSeconds'] = (
                    graph.config.active_deadline_seconds
                )
    name = _name(pipeline_name or component.name)[:48]
    version = hashlib.sha256(_json(spec).encode()).hexdigest()[:12]
    metadata = {
        'labels': {
            'pipelines.kubeflow.org/pipeline': name,
            'pipelines.kubeflow.org/version': version,
        },
        'annotations': {
            'pipelines.kubeflow.org/display-name': display_name
            or (
                graph.display_name
                if isinstance(graph, structures.CapturedGraph)
                else None
            )
            or component.name,
            'pipelines.kubeflow.org/description': component.description or '',
            'pipelines.kubeflow.org/contract': SCHEMA,
        },
    }
    if kind == 'Workflow':
        metadata['generateName'] = name + '-'
    else:
        metadata['name'] = name + '-' + version
    if kind == 'WorkflowTemplate':
        spec['workflowMetadata'] = {'labels': dict(metadata['labels'])}
    if kind == 'CronWorkflow':
        spec = {
            'schedules': [schedule],
            'suspend': True,
            'workflowSpec': spec,
            'workflowMetadata': {'labels': metadata['labels']},
        }
    return {
        'apiVersion': 'argoproj.io/v1alpha1',
        'kind': kind,
        'metadata': metadata,
        'spec': spec,
    }
