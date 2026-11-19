# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""In-container adapter for Argo-staged files.

No service or storage client.
"""

import json
import math
import os
from pathlib import Path
from typing import Callable

from kfp.dsl.executor import Executor

SCHEMA = 'kfp-argo/v1'


def serialize(value, parameter_type: str) -> str:
    """Serialize an Argo parameter without losing its declared Python type."""
    if parameter_type == 'String':
        if not isinstance(value, str):
            raise ValueError('Expected a String parameter; supply a string.')
        return value
    encoded = json.dumps(value, allow_nan=False, separators=(',', ':'))
    deserialize(encoded, parameter_type)
    return encoded


def deserialize(value: str, parameter_type: str):
    """Read an Argo string parameter and validate the declared scalar type."""
    if parameter_type == 'String':
        return value
    result = json.loads(value)
    # Reject non-finite numbers inside collections as well as scalar values.
    json.dumps(result, allow_nan=False)
    expected = {
        'Integer': int,
        'Float': (int, float),
        'Boolean': bool,
        'List': list,
        'Dict': dict,
    }.get(parameter_type)
    if expected is None:
        raise ValueError(
            f'Unsupported parameter type {parameter_type!r}; use a JSON scalar, list or dict.'
        )
    if (
        not isinstance(result, expected)
        or (parameter_type in ('Integer', 'Float') and isinstance(result, bool))
        or (isinstance(result, float) and not math.isfinite(result))
    ):
        raise ValueError(f'Expected {parameter_type}, received {value!r}.')
    return result


def deserialize_collection(value: str, element_type: str, depth: int):
    """Restore Python values from Argo's nested arrays of parameter strings."""
    if depth < 1:
        raise ValueError(
            'Collected inputs must cross at least one ParallelFor scope.'
        )
    items = deserialize(value, 'List')
    return [
        deserialize_collection(
            item if isinstance(item, str) else json.dumps(item),
            element_type,
            depth - 1,
        )
        if depth > 1
        else deserialize(
            item if isinstance(item, str) else json.dumps(item), element_type
        )
        for item in items
    ]


def _write(path: str, value: str) -> None:
    target = Path(path)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(value)


class _StagedExecutor(Executor):
    def make_artifact(self, runtime_artifact, name, func, annotation=None):
        artifact = super().make_artifact(
            runtime_artifact, name, func, annotation
        )
        artifact.custom_path = runtime_artifact['path']
        return artifact


def execute(function: Callable, contract: dict, parameters: list[dict]) -> dict:
    """Invoke an unchanged KFP function against Argo input/output declarations.

    The contract is static component configuration. Argo supplies
    parameter values separately, so quotes in user values cannot alter
    its structure. Each input artifact has a staged payload and a staged
    metadata companion. Output storage URIs must match the artifact
    repository's configured key.
    """
    if contract.get('schema') != SCHEMA:
        raise ValueError(
            f'Unsupported Argo runtime contract; expected {SCHEMA}.'
        )
    values = {entry['name']: entry['value'] for entry in parameters}
    inputs = {'parameterValues': {}, 'artifacts': {}}
    references = []
    for name, port in contract.get('inputs', {}).items():
        if 'path' not in port:
            if name in values:
                if 'collectionElementType' in port:
                    inputs['parameterValues'][name] = deserialize_collection(
                        values[name],
                        port['collectionElementType'],
                        port['collectionDepth'],
                    )
                else:
                    inputs['parameterValues'][name] = deserialize(
                        values[name], port['type']
                    )
            elif not port.get('optional'):
                raise ValueError(
                    f'Missing required parameter {name!r}; pass it to Argo.'
                )
            continue
        metadata_path = Path(port['metadataPath'])
        if metadata_path.exists():
            metadata = json.loads(metadata_path.read_text())
        else:
            metadata = json.loads(values.get('kfp-meta-' + name, '{}'))
            if 'metadata' not in metadata:
                raise ValueError(
                    f'Artifact metadata unavailable for {name!r}; restore the producer metadata artifact before retrying.'
                )
        if metadata.get('schema') != SCHEMA:
            raise ValueError(
                f'Unsupported metadata for input {name!r}; regenerate its producer.'
            )
        if (
            metadata['type'] != port['type']
            and port['type'] != 'system.Artifact@0.0.1'
        ):
            raise ValueError(f'Artifact type mismatch for {name!r}.')
        references.append(
            {
                'port': name,
                'producer': metadata['producer'],
                'uri': metadata['uri'],
            }
        )
        inputs['artifacts'][name] = {
            'artifacts': [
                {
                    'name': metadata['port'],
                    'uri': metadata['uri'],
                    'metadata': metadata['metadata'],
                    'path': port['path'],
                    'type': {'schemaTitle': metadata['type'].split('@')[0]},
                }
            ]
        }
    root = contract['outputDirectory']
    producer = {
        'run': os.environ['KFP_ARGO_RUN_UID'],
        'task': os.environ['KFP_ARGO_NODE_ID'],
        'namespace': os.environ['KFP_ARGO_NAMESPACE'],
    }
    outputs = {
        'outputFile': f'{root}/executor.json',
        'parameters': {},
        'artifacts': {},
    }
    for name, port in contract.get('outputs', {}).items():
        if 'metadataPath' not in port:
            outputs['parameters'][name] = {'outputFile': port['path']}
        else:
            uri = (
                port['uri']
                .replace('{run}', producer['run'])
                .replace('{task}', producer['task'])
            )
            outputs['artifacts'][name] = {
                'artifacts': [
                    {
                        'name': name,
                        'uri': uri,
                        'path': port['path'],
                        'type': {'schemaTitle': port['type'].split('@')[0]},
                    }
                ]
            }
    executor = _StagedExecutor(
        {'inputs': inputs, 'outputs': outputs},
        function,
        artifact_previews=False,
    )
    if executor.execute() is None:
        raise RuntimeError(
            'Argo components require one output-writing process per task.'
        )
    summary = {
        'schema': SCHEMA,
        'producer': producer,
        'inputs': references,
        'artifacts': [],
        'metrics': {},
    }
    for name, port in contract.get('outputs', {}).items():
        if 'metadataPath' not in port:
            if name in executor.excutor_output.get('parameterValues', {}):
                _write(
                    port['path'],
                    serialize(
                        executor.excutor_output['parameterValues'][name],
                        port['type'],
                    ),
                )
            else:
                # OutputPath values are already written by the component.
                deserialize(Path(port['path']).read_text(), port['type'])
            continue
        artifact = executor.output_artifacts[name]
        if (
            artifact.path != port['path']
            or artifact.uri != outputs['artifacts'][name]['artifacts'][0]['uri']
        ):
            raise ValueError(
                f'Output {name!r} changed its assigned path or URI; write to the supplied artifact.path.'
            )
        metadata = {
            'schema': SCHEMA,
            'port': name,
            'type': port['type'],
            'producer': producer,
            'uri': artifact.uri,
            'metadata': artifact.metadata,
            'inputs': references,
        }
        if port['type'].startswith('system.Metrics@'):
            summary['metrics'].update(artifact.metadata)
            # Metrics have metadata but need not write a payload themselves.
            if not Path(artifact.path).exists():
                _write(
                    artifact.path,
                    json.dumps(artifact.metadata, allow_nan=False),
                )
        if not Path(artifact.path).exists():
            raise ValueError(
                f'Output artifact {name!r} was not written to {artifact.path}.'
            )
        _write(port['metadataPath'], json.dumps(metadata, allow_nan=False))
        if 'identityPath' in port:
            identity = {
                key: metadata[key]
                for key in ('schema', 'port', 'type', 'producer', 'uri')
            }
            _write(port['identityPath'], json.dumps(identity, allow_nan=False))
        summary['artifacts'].append(metadata)
    _write(f'{root}/metadata.json', json.dumps(summary, allow_nan=False))
    return summary
