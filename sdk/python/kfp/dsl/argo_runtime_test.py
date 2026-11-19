# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import importlib.util
import json
from pathlib import Path

import pytest
import yaml

from kfp.dsl import argo_runtime

ROOT = Path(__file__).resolve().parents[4]


def test_large_metadata_is_not_inlined_into_workflow_parameters(
    tmp_path, monkeypatch
):
    from kfp import dsl

    def produce(dataset: dsl.Output[dsl.Dataset]):
        Path(dataset.path).write_text('payload')
        dataset.metadata['large'] = 'x' * 100_000

    for key, value in [
        ('KFP_ARGO_RUN_UID', 'run'),
        ('KFP_ARGO_NODE_ID', 'task'),
        ('KFP_ARGO_NAMESPACE', 'ns'),
    ]:
        monkeypatch.setenv(key, value)
    port = {
        'type': 'system.Dataset@0.0.1',
        'path': str(tmp_path / 'payload'),
        'metadataPath': str(tmp_path / 'dataset.json'),
        'identityPath': str(tmp_path / 'identity.json'),
        'uri': 's3://bucket/{run}/{task}/dataset',
    }
    argo_runtime.execute(
        produce,
        {
            'schema': argo_runtime.SCHEMA,
            'outputDirectory': str(tmp_path),
            'outputs': {'dataset': port},
        },
        [],
    )
    full = json.loads(Path(port['metadataPath']).read_text())
    identity = json.loads(Path(port['identityPath']).read_text())
    assert full['metadata']['large'] == 'x' * 100_000
    assert 'metadata' not in identity
    assert len(json.dumps(identity)) < 1000
    assert identity['producer'] == full['producer']

    def consume(dataset: dsl.Input[dsl.Dataset]):
        assert len(dataset.metadata['large']) == 100_000

    inputs = {
        'dataset': {key: port[key] for key in ('type', 'path', 'metadataPath')}
    }
    contract = {
        'schema': argo_runtime.SCHEMA,
        'outputDirectory': str(tmp_path / 'consumer'),
        'inputs': inputs,
    }
    parameters = [{'name': 'kfp-meta-dataset', 'value': json.dumps(identity)}]
    argo_runtime.execute(consume, contract, parameters)
    Path(port['metadataPath']).unlink()
    with pytest.raises(ValueError, match='metadata unavailable'):
        argo_runtime.execute(consume, contract, parameters)


@pytest.mark.parametrize(
    'kind,value',
    [
        ('String', '"{{workflow.uid}}"\n'),
        ('Boolean', False),
        ('Integer', 2),
        ('Float', 1.25),
        ('List', [1, 'two']),
        ('Dict', {'quote': '"'}),
    ],
)
def test_parameter_roundtrip(kind, value):
    assert (
        argo_runtime.deserialize(argo_runtime.serialize(value, kind), kind)
        == value
    )


@pytest.mark.parametrize(
    'kind,value',
    [('Boolean', '1'), ('Integer', 'true'), ('Float', 'NaN'), ('List', '{}')],
)
def test_bad_parameters(kind, value):
    with pytest.raises(ValueError):
        argo_runtime.deserialize(value, kind)


@pytest.mark.parametrize(
    'kind,value',
    [
        ('Integer', True),
        ('Float', '1.2'),
        ('Boolean', 1),
        ('List', {}),
        ('Unknown', 1),
    ],
)
def test_serialize_rejects_wrong_type(kind, value):
    with pytest.raises(ValueError):
        argo_runtime.serialize(value, kind)


@pytest.mark.parametrize('value', ['[NaN]', '{"value":Infinity}'])
def test_nested_nonfinite_parameters(value):
    with pytest.raises(ValueError):
        argo_runtime.deserialize(
            value, 'List' if value.startswith('[') else 'Dict'
        )


def test_handwritten_workflow_contract(tmp_path, monkeypatch):
    """Stage the exact handwritten ports as Argo does, without transferring
    data."""
    spec = importlib.util.spec_from_file_location(
        'demo', ROOT / 'samples/argo-native/components.py'
    )
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    workflow = yaml.safe_load(
        (ROOT / 'samples/argo-native/handwritten.yaml').read_text()
    )
    artifacts = {}
    for template in workflow['spec']['templates'][1:]:
        name = template['name']
        monkeypatch.setenv('KFP_ARGO_RUN_UID', 'run-1')
        monkeypatch.setenv('KFP_ARGO_NODE_ID', name)
        monkeypatch.setenv('KFP_ARGO_NAMESPACE', 'demo')
        args = template['container']['args']
        contract = json.loads(
            args[args.index('--argo_input') + 1].replace(
                '/tmp/kfp', str(tmp_path / name)
            )
        )
        for port, declaration in contract.get('inputs', {}).items():
            if 'path' in declaration:
                Path(declaration['path']).parent.mkdir(
                    parents=True, exist_ok=True
                )
                Path(declaration['path']).write_bytes(artifacts[port][0])
                Path(declaration['metadataPath']).write_bytes(
                    artifacts[port][1]
                )
        summary = argo_runtime.execute(
            getattr(module, name),
            contract,
            [{'name': 'factor', 'value': '1.0'}],
        )
        for port, declaration in contract['outputs'].items():
            if 'metadataPath' in declaration:
                artifacts[port] = (
                    Path(declaration['path']).read_bytes(),
                    Path(declaration['metadataPath']).read_bytes(),
                )
                metadata = json.loads(artifacts[port][1])
                assert (
                    metadata['uri']
                    == f's3://argo-artifacts/run-1/{name}/{port}'
                )
        if name == 'train':
            assert summary['inputs'][0]['producer']['task'] == 'prepare'
            assert summary['artifacts'][0]['metadata']['dataset_uri'].endswith(
                '/prepare/dataset'
            )
        if name == 'evaluate':
            assert summary['metrics']['score'] == pytest.approx(2 / 3)
            assert float(
                Path(contract['outputs']['Output']['path']).read_text()
            ) == pytest.approx(2 / 3)
