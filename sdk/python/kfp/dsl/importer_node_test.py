# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Importer declarations remain typed authoring data, never a backend fallback."""

import pytest

from kfp import compiler, dsl


@pytest.mark.parametrize('shape', ['dict', 'list', 'fstring'])
def test_importer_captures_dynamic_metadata(shape, tmp_path):
    @dsl.pipeline
    def pipeline(value: str, number: int):
        metadata = (
            {'first': value, 'second': value}
            if shape == 'dict'
            else {'items': [value, number], value: value}
            if shape == 'list'
            else {f'prefix-{value}': f'value-{number}'}
        )
        dsl.importer('s3://bucket/object', dsl.Dataset, metadata=metadata)

    task = pipeline.pipeline_spec.pipeline.tasks['importer']
    assert 'metadata' in task.component_spec.inputs
    assert task.inputs['metadata'].name == 'value'
    text = str(task.importer_spec.metadata)
    assert "{{$.inputs.parameters['metadata']}}" in text
    if shape != 'dict':
        assert task.inputs['metadata-2'].name == 'number'
    with pytest.raises(
        ValueError,
        match='Importer components need a versioned native artifact source',
    ):
        compiler.Compiler(disable_caching=True).compile(
            pipeline, str(tmp_path / 'pipeline.yaml')
        )


def test_importer_reuses_task_output_metadata_binding():
    @dsl.component
    def string_task() -> str:
        return 'value'

    @dsl.pipeline
    def pipeline():
        value = string_task().output
        dsl.importer(
            's3://bucket/object',
            dsl.Dataset,
            metadata={'items': [value, value], value: value},
        )

    task = pipeline.pipeline_spec.pipeline.tasks['importer']
    assert 'metadata' in task.inputs and 'metadata-2' not in task.inputs
    assert task.inputs['metadata'].task_name == 'string-task'
    assert (
        task.importer_spec.metadata['items']
        == ["{{$.inputs.parameters['metadata']}}"] * 2
    )


def test_importer_captures_loop_item_references():
    @dsl.component
    def items() -> list:
        return [{'uri': 's3://bucket/key', 'key': 'value'}]

    @dsl.pipeline
    def pipeline():
        with dsl.ParallelFor(items().output) as item:
            dsl.importer(item.uri, dsl.Dataset, metadata={'key': item.key})

    task = pipeline.pipeline_spec.pipeline.tasks['importer']
    assert task.inputs['uri'].subvar_name == 'uri'
    assert task.inputs['metadata'].subvar_name == 'key'
    assert task.importer_spec.metadata == {
        'key': "{{$.inputs.parameters['metadata']}}"
    }
