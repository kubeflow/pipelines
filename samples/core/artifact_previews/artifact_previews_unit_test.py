# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Local tests of actual producer outputs; no cluster or containers
required."""

from collections.abc import Iterator
import csv
import json
import os
from pathlib import Path
from typing import Any
from unittest import mock

from artifact_previews import artifact_preview_disabled_pipeline
from artifact_previews import artifact_preview_formats_pipeline
from artifact_previews import artifact_preview_list_pipeline
from artifact_previews import artifact_preview_pipeline
from artifact_previews import create_dataframe_artifact
from artifact_previews import create_model_artifact
from artifact_previews import create_preview_artifacts
from kfp import compiler
from kfp import dsl
from kfp.dsl import _artifact_preview
from kfp.dsl import constants
from kfp.dsl import executor
import numpy as np
import pyarrow as pa
import pyarrow.feather as feather
import pyarrow.parquet as pq
import pytest
import xgboost as xgb
import yaml

# The macOS wheels' OpenMP runtimes require XGBoost to load before PyTorch.
# isort: split
import torch
from transformers import BertConfig
from transformers import BertForSequenceClassification


@pytest.fixture(autouse=True)
def default_pipeline_policy(monkeypatch: pytest.MonkeyPatch) -> None:
    """Exercise the default without inheriting an enclosing task's opt-out."""
    monkeypatch.delenv(constants.ARTIFACT_PREVIEWS_ENV_VAR, raising=False)


def produce(
    directory: Path,
    previews_enabled: bool = True,
    include_rows: bool = True,
) -> tuple[dsl.Dataset, dsl.Model]:
    """Run normal component code through the SDK executor's default policy."""
    preview_options = {'artifact_preview_rows': include_rows}
    if not previews_enabled:
        preview_options['artifact_previews'] = False
    runtime = executor.Executor(
        {
            'inputs': {
                'parameterValues': {
                    'seed': 42
                }
            },
            'outputs': {
                'artifacts': {
                    name: {
                        'artifacts': [{
                            'uri': str(directory / name),
                            'type': {
                                'schemaTitle': title
                            },
                        }]
                    } for name, title in (('dataset', 'system.Dataset'),
                                         ('model', 'system.Model'))
                },
                'outputFile': str(directory / 'executor_output.json'),
            },
        },
        create_preview_artifacts,
        **preview_options,
    )
    output_file = runtime.execute()
    recorded = json.loads(Path(output_file).read_text())
    for name, artifact in runtime.output_artifacts.items():
        assert recorded['artifacts'][name]['artifacts'][0][
            'metadata'] == artifact.metadata
    return runtime.output_artifacts['dataset'], runtime.output_artifacts[
        'model']


@pytest.mark.parametrize('include_rows', [True, False])
def test_parquet_preview_matches_the_actual_file(
    tmp_path: Path,
    include_rows: bool,
) -> None:
    dataset, model = produce(tmp_path, include_rows=include_rows)
    for artifact in (dataset, model):
        assert artifact.metadata['_kfp_preview'] == {
            'version': 1,
            'path': 'preview.json'
        }
        assert (Path(artifact.path) /
                'preview.json').stat().st_size <= 64 * 1024
        assert not list(Path(artifact.path).glob('.kfp-preview-*.tmp'))

    preview = json.loads(
        (Path(dataset.path) / 'preview.json').read_text(encoding='utf-8'))
    parquet = pq.ParquetFile(Path(dataset.path) / 'data.parquet')
    assert preview['kind'] == 'dataset'
    assert preview['format'] == 'parquet'
    assert preview['total_rows'] == parquet.metadata.num_rows == 32
    assert preview['total_columns'] == len(parquet.schema_arrow) == 25
    assert len(preview['columns']) == 20
    assert preview['columns'] == [{
        'name': field.name,
        'dtype': str(field.type),
    } for field in list(parquet.schema_arrow)[:20]]
    assert preview['sample']['columns_truncated'] is True
    assert preview['sample']['rows_omitted'] is not include_rows
    if include_rows:
        expected = parquet.read(columns=parquet.schema_arrow.names[:20]).slice(
            0, 10)
        expected_rows = [[column[row].as_py()
                          for column in expected.columns]
                         for row in range(10)]
        assert preview['rows'] == expected_rows
        assert preview['sample']['rows_truncated'] is True
    else:
        assert preview['rows'] == []


def test_model_summary_and_predictions_survive_native_io(
        tmp_path: Path) -> None:
    dataset, model = produce(tmp_path)
    preview = json.loads(
        (Path(model.path) / 'preview.json').read_text(encoding='utf-8'))
    loaded = xgb.XGBClassifier()
    loaded.load_model(Path(model.path) / 'model.ubj')
    booster = loaded.get_booster()
    assert preview['framework'] == {
        'name': 'xgboost',
        'version': xgb.__version__
    }
    assert preview['estimator_class'] == 'xgboost.XGBClassifier'
    assert preview['hyperparameters']['max_depth'] == 2
    assert preview['structure'] == {
        'feature_count': loaded.n_features_in_,
        'boosting_rounds': booster.num_boosted_rounds(),
        'tree_count': len(booster.get_dump()),
    }
    assert preview['structure']['boosting_rounds'] == 5

    data = pq.read_table(Path(dataset.path) / 'data.parquet')
    features = np.column_stack(
        [column.to_numpy() for column in data.columns[:-1]])
    trained = xgb.XGBClassifier(
        **{
            name: preview['hyperparameters'][name]
            for name in ('objective', 'n_estimators', 'max_depth',
                         'learning_rate', 'n_jobs', 'random_state')
        })
    trained.fit(features, data.column('label').to_numpy())
    np.testing.assert_array_equal(
        loaded.predict(features), trained.predict(features))
    assert 'weights' not in preview


@pytest.mark.parametrize('control', ['component', 'pipeline'])
def test_disabling_previews_preserves_dataset_and_model_outputs(
    tmp_path: Path,
    control: str,
) -> None:
    enabled_dataset, enabled_model = produce(tmp_path / 'enabled')
    with mock.patch.object(_artifact_preview, '_parquet_preview') as sample:
        if control == 'pipeline':
            package = tmp_path / 'disabled.yaml'
            compiler.Compiler().compile(artifact_preview_disabled_pipeline,
                                        str(package))
            specification = yaml.safe_load(package.read_text())
            container = next(
                iter(specification['deploymentSpec']
                     ['executors'].values()))['container']
            environment = {
                item['name']: item['value'] for item in container['env']
            }
            with mock.patch.dict(os.environ, environment):
                dataset, model = produce(tmp_path / 'disabled')
        else:
            dataset, model = produce(
                tmp_path / 'disabled', previews_enabled=False)
    sample.assert_not_called()
    for artifact in (dataset, model):
        assert '_kfp_preview' not in artifact.metadata
        assert not (Path(artifact.path) / 'preview.json').exists()
    assert pq.read_table(Path(dataset.path) / 'data.parquet').equals(
        pq.read_table(Path(enabled_dataset.path) / 'data.parquet'))
    assert (Path(model.path) /
            'model.ubj').read_bytes() == (Path(enabled_model.path) /
                                          'model.ubj').read_bytes()


def test_pipeline_needs_no_additional_preview_task(tmp_path: Path) -> None:
    package = tmp_path / 'pipeline.yaml'
    compiler.Compiler().compile(artifact_preview_pipeline, str(package))
    specification = yaml.safe_load(package.read_text(encoding='utf-8'))
    assert len(specification['root']['dag']['tasks']) == 1
    assert len(specification['deploymentSpec']['executors']) == 1
    container = next(
        iter(
            specification['deploymentSpec']['executors'].values()))['container']
    assert '--artifact_previews' not in container['args']
    assert '--disable_artifact_previews' not in container['args']
    assert '_kfp_preview' not in '\n'.join(container['command'])


def test_collected_artifacts_retain_individual_outputs(tmp_path: Path) -> None:
    package = tmp_path / 'lists.yaml'
    compiler.Compiler().compile(artifact_preview_list_pipeline, str(package))
    specification = yaml.safe_load(package.read_text(encoding='utf-8'))
    assert len(specification['deploymentSpec']['executors']) == 1
    outputs = specification['root']['outputDefinitions']['artifacts']
    assert outputs['datasets']['isArtifactList'] is True
    assert outputs['models']['isArtifactList'] is True
    assert len(specification['root']['dag']['tasks']) == 1


def preview_table(directory: Path,
                  table: pa.Table,
                  include_rows: bool = True) -> dict:
    """Write a real Parquet file and generate its bounded SDK preview."""
    directory.mkdir()
    pq.write_table(table, directory / 'data.parquet')
    return _artifact_preview._parquet_preview(directory, include_rows)


def test_schema_only_never_reads_row_data(tmp_path: Path) -> None:
    with mock.patch.object(pq.ParquetFile, 'iter_batches') as read_rows:
        preview = preview_table(tmp_path / 'schema',
                                pa.table({'value': [1, 2]}), False)
    read_rows.assert_not_called()
    assert preview['rows'] == []
    assert preview['total_rows'] == 2
    assert preview['sample']['rows_omitted'] is True


def test_sample_projects_columns_and_reads_one_batch(tmp_path: Path) -> None:
    table = pa.table({f'column_{index}': range(100) for index in range(25)})
    original = pq.ParquetFile.iter_batches
    calls = []

    def iter_batches(parquet: pq.ParquetFile,
                     **kwargs: Any) -> Iterator[pa.RecordBatch]:
        calls.append(kwargs)
        yield next(original(parquet, **kwargs))
        raise AssertionError('The preview must not read a second batch.')

    with mock.patch.object(pq.ParquetFile, 'iter_batches', iter_batches):
        preview = preview_table(tmp_path / 'projected', table)
    assert calls == [{'batch_size': 10, 'columns': table.column_names[:20]}]
    assert len(preview['rows']) == 10
    assert len(preview['columns']) == 20


def test_large_unicode_values_are_bounded_in_utf8_bytes(tmp_path: Path) -> None:
    table = pa.table(
        {f'column_{index}': ['\U0001f600' * 400] * 10 for index in range(20)})
    preview = preview_table(tmp_path / 'bounded', table)
    assert len(_artifact_preview._encode(preview)) <= 64 * 1024
    assert 0 < len(preview['rows']) < 10
    assert len(preview['rows'][0][0]) == 256
    assert preview['sample']['rows_truncated'] is True
    assert preview['sample']['values_truncated'] is True


def test_cells_are_scalars_and_preserve_numeric_precision(
        tmp_path: Path) -> None:
    table = pa.table({
        'integer': [2**63 - 1, None],
        'float': [1e30, float('nan')],
        'binary': [b'not displayed', None],
        'nested': [[1, 2], None],
        'dictionary': pa.array(['category', None]).dictionary_encode(),
    })
    preview = preview_table(tmp_path / 'scalars', table)
    assert preview['rows'][0][:2] == [str(2**63 - 1), str(1e30)]
    assert preview['rows'][1] == [None, 'NaN', None, None, None]
    assert 'omitted' in preview['rows'][0][2]
    assert 'omitted' in preview['rows'][0][3]
    assert preview['rows'][0][4] == 'category'
    assert preview['sample']['values_truncated'] is True


@pytest.mark.parametrize('error', [
    pa.ArrowNotImplementedError('unsupported encoding'),
    OverflowError('timestamp outside Python datetime range'),
])
def test_optional_arrow_failures_do_not_fail_artifact_production(
    tmp_path: Path,
    error: Exception,
    caplog: pytest.LogCaptureFixture,
) -> None:
    artifact = dsl.Dataset(uri=str(tmp_path))
    pq.write_table(pa.table({'value': [1]}), tmp_path / 'data.parquet')
    with mock.patch.object(pq.ParquetFile, 'iter_batches', side_effect=error):
        _artifact_preview.ArtifactPreviewSession({
            'dataset': artifact
        }).generate()
    assert '_kfp_preview' not in artifact.metadata
    assert 'could not read a bounded Parquet sample' in caplog.text


@pytest.mark.parametrize('booster_type', ['gbtree', 'dart'])
def test_native_booster_preview_without_model_reload_or_tree_dump(
    tmp_path: Path,
    booster_type: str,
) -> None:
    artifact = dsl.Model(uri=str(tmp_path))
    training = xgb.DMatrix(
        np.arange(40).reshape(20, 2), label=np.arange(20) % 2)
    original = xgb.Booster.save_model
    with _artifact_preview.ArtifactPreviewSession({'model': artifact
                                                  }) as session:
        booster = xgb.train(
            {
                'booster': booster_type,
                'objective': 'binary:logistic',
                'max_depth': 2,
                'eta': 0.1,
                'nthread': 1,
            },
            training,
            num_boost_round=3)
        with mock.patch.object(
                xgb.Booster, 'get_dump',
                side_effect=AssertionError('no tree dump')):
            booster.save_model(tmp_path / 'model.ubj')
            session.generate()
    assert xgb.Booster.save_model is original
    preview = json.loads((tmp_path / 'preview.json').read_text())
    assert preview['estimator_class'] == 'xgboost.Booster'
    assert preview['hyperparameters']['training'][
        'objective'] == 'binary:logistic'
    assert preview['structure'] == {
        'feature_count': 2,
        'boosting_rounds': 3,
        'tree_count': 3,
    }


def execute_sample(directory: Path,
                   kind: str,
                   include_rows: bool = True,
                   **parameters: str) -> dsl.Artifact:
    """Run each native writer through the real executor, without preview
    calls."""
    name = 'dataset' if kind == 'Dataset' else 'model'
    component = create_dataframe_artifact if kind == 'Dataset' else create_model_artifact
    runtime = executor.Executor(
        {
            'inputs': {
                'parameterValues': parameters
            },
            'outputs': {
                'artifacts': {
                    name: {
                        'artifacts': [{
                            'uri': str(directory / name),
                            'type': {
                                'schemaTitle': f'system.{kind}'
                            },
                        }]
                    },
                },
                'outputFile': str(directory / 'executor_output.json'),
            },
        },
        component,
        artifact_preview_rows=include_rows)
    output = json.loads(Path(runtime.execute()).read_text())
    artifact = runtime.output_artifacts[name]
    assert output['artifacts'][name]['artifacts'][0][
        'metadata'] == artifact.metadata
    return artifact


def load_preview(artifact: dsl.Artifact) -> dict:
    """Require the descriptor, encoded budget, and completed publication."""
    assert artifact.metadata['_kfp_preview'] == {
        'version': 1,
        'path': 'preview.json'
    }
    path = Path(artifact.path) / 'preview.json'
    assert path.stat().st_size <= _artifact_preview.MAX_BYTES
    assert not list(path.parent.glob('.kfp-preview-*.tmp'))
    return json.loads(path.read_text())


@pytest.mark.parametrize('library', ['pandas', 'polars'])
@pytest.mark.parametrize('file_format', [
    'parquet',
    'csv',
    'json',
    'jsonl',
    'arrow',
    'arrow-stream',
    'feather',
])
@pytest.mark.parametrize('include_rows', [True, False])
def test_dataframe_native_writer_matrix(tmp_path: Path, library: str,
                                        file_format: str,
                                        include_rows: bool) -> None:
    artifact = execute_sample(
        tmp_path,
        'Dataset',
        include_rows,
        library=library,
        file_format=file_format)
    preview = load_preview(artifact)
    assert preview['format'] == ('arrow' if file_format == 'arrow-stream' else
                                 file_format)
    assert len(preview['columns']) == 20
    assert [column['name'] for column in preview['columns'][:4]
           ] == ['record_id', 'category', 'score', 'accepted']
    assert preview['total_columns'] == (None if file_format in ('json', 'jsonl')
                                        else 25)
    assert preview['sample']['columns_truncated'] is True
    assert preview['sample']['rows_omitted'] is not include_rows
    assert len(preview['rows']) == (10 if include_rows else 0)
    if include_rows:
        if file_format == 'csv':
            with (Path(artifact.path) / 'data.csv').open(newline='') as source:
                native_rows = list(csv.reader(source))
            assert preview['rows'] == [row[:20] for row in native_rows[1:11]]
        else:
            assert [row[:4] for row in preview['rows']] == [[
                row, 'group_a' if row % 2 == 0 else 'group_b', row / 8,
                row % 2 == 0
            ] for row in range(10)]
        assert preview['sample']['rows_truncated'] is True
    if file_format == 'parquet' or (file_format == 'feather' and include_rows):
        assert preview['total_rows'] == 32
    else:
        assert preview['total_rows'] is None


@pytest.mark.parametrize('file_format', ['json-columns', 'json-split'])
def test_pandas_json_orientations(tmp_path: Path, file_format: str) -> None:
    artifact = execute_sample(
        tmp_path, 'Dataset', library='pandas', file_format=file_format)
    preview = load_preview(artifact)
    assert preview['format'] == 'json'
    assert preview['total_rows'] == 32
    assert preview['total_columns'] == 25
    assert len(preview['rows']) == 10
    assert preview['rows'][0][:4] == [0, 'group_a', 0, True]


@pytest.mark.parametrize('container', ['file', 'stream', 'feather'])
def test_arrow_schema_only_does_not_decode_record_batches(
        tmp_path: Path, container: str) -> None:
    table = pa.table({'value': [1, 2]})
    path = tmp_path / ('data.feather'
                       if container == 'feather' else 'data.arrow')
    if container == 'feather':
        feather.write_feather(table, path)
    else:
        writer = pa.ipc.new_file if container == 'file' else pa.ipc.new_stream
        with pa.OSFile(str(path), 'wb') as output:
            with writer(output, table.schema) as stream:
                stream.write_table(table)
    reader_type = pa.ipc.RecordBatchStreamReader if container == 'stream' else pa.ipc.RecordBatchFileReader
    method = 'read_next_batch' if container == 'stream' else 'get_batch'
    with mock.patch.object(
            reader_type, method, side_effect=AssertionError('schema only')):
        preview = _artifact_preview._dataset_preview(tmp_path, False)
    assert preview['columns'] == [{'name': 'value', 'dtype': 'int64'}]
    assert preview['rows'] == []
    assert preview['sample']['rows_omitted'] is True


def test_arrow_projection_does_not_read_the_complete_table(
        tmp_path: Path) -> None:
    table = pa.table({f'column_{index}': range(100) for index in range(25)})
    feather.write_feather(table, tmp_path / 'data.feather', chunksize=8)
    original = pa.ipc.RecordBatchFileReader.get_batch
    selected_widths = []

    def get_batch(reader: pa.ipc.RecordBatchFileReader,
                  index: int) -> pa.RecordBatch:
        assert index < 2, 'Only enough projected batches for ten rows should be decoded.'
        batch = original(reader, index)
        selected_widths.append(batch.num_columns)
        return batch

    with mock.patch.object(pa.ipc.RecordBatchFileReader, 'get_batch',
                           get_batch):
        preview = _artifact_preview._dataset_preview(tmp_path, True)
    assert selected_widths == [20, 20]
    assert len(preview['rows']) == 10
    assert preview['total_rows'] is None


@pytest.mark.parametrize('serialization',
                         ['state_dict', 'checkpoint', 'module', 'torchscript'])
def test_pytorch_native_serialization_and_structure(tmp_path: Path,
                                                    serialization: str) -> None:
    original_save = torch.save
    original_jit_save = torch.jit.save
    with mock.patch.object(
            torch,
            'load',
            side_effect=AssertionError('no checkpoint deserialization')):
        with mock.patch.object(
                torch.jit,
                'load',
                side_effect=AssertionError('no TorchScript loading')):
            artifact = execute_sample(
                tmp_path,
                'Model',
                framework='pytorch',
                serialization=serialization)
    assert torch.save is original_save
    assert torch.jit.save is original_jit_save
    preview = load_preview(artifact)
    assert preview['framework'] == {
        'name': 'pytorch',
        'version': torch.__version__
    }
    assert preview['hyperparameters'] == {}
    tensors = preview['structure']['tensors']
    assert tensors['0.weight'] == {'shape': [8, 4], 'dtype': 'torch.float32'}
    assert tensors['2.bias'] == {'shape': [2], 'dtype': 'torch.float32'}
    assert len(tensors) == 4
    if serialization in ('state_dict', 'checkpoint'):
        assert preview['structure']['tensor_element_count'] == 58
        assert preview['structure']['tensor_count'] == 4
        assert 'parameter_count' not in preview['structure']
    else:
        assert preview['structure']['parameter_count'] == 58
        assert preview['structure']['trainable_parameter_count'] == 58
    path = Path(artifact.path) / 'model.pt'
    torch.manual_seed(42)
    expected = torch.nn.Sequential(
        torch.nn.Linear(4, 8), torch.nn.ReLU(), torch.nn.Linear(8, 2)).eval()
    if serialization == 'torchscript':
        loaded = torch.jit.load(str(path))
    elif serialization == 'module':
        loaded = torch.load(path, weights_only=False)
    else:
        state = torch.load(path, weights_only=True)
        loaded = torch.nn.Sequential(
            torch.nn.Linear(4, 8), torch.nn.ReLU(), torch.nn.Linear(8,
                                                                    2)).eval()
        loaded.load_state_dict(state['model_state_dict'] if serialization ==
                               'checkpoint' else state)
    inputs = torch.arange(12, dtype=torch.float32).reshape(3, 4)
    torch.testing.assert_close(loaded(inputs), expected(inputs))


def test_pytorch_summary_does_not_read_tensor_values() -> None:
    model = torch.nn.Sequential(torch.nn.Linear(4, 8), torch.nn.BatchNorm1d(8))
    with mock.patch.object(
            torch.Tensor, 'numpy', side_effect=AssertionError('no values')):
        with mock.patch.object(
                torch.Tensor, 'tolist',
                side_effect=AssertionError('no values')):
            with mock.patch.object(
                    torch.Tensor, 'item',
                    side_effect=AssertionError('no values')):
                preview = _artifact_preview._torch_preview(model, torch)
                state_preview = _artifact_preview._torch_preview(
                    model.state_dict(), torch)
    assert preview['structure']['parameter_count'] == 56
    assert preview['structure']['buffer_count'] == 3
    assert state_preview['structure']['tensor_element_count'] == 73
    assert 'parameter_count' not in state_preview['structure']


def test_transformers_save_pretrained_and_native_reload(tmp_path: Path) -> None:
    with mock.patch.object(
            BertForSequenceClassification,
            'from_pretrained',
            side_effect=AssertionError(
                'no model deserialization during preview generation')):
        artifact = execute_sample(tmp_path, 'Model', framework='transformers')
    preview = load_preview(artifact)
    directory = Path(artifact.path)
    config = json.loads((directory / 'config.json').read_text())
    assert preview['framework'] == {
        'name': 'transformers',
        'version': config['transformers_version']
    }
    assert preview[
        'estimator_class'] == 'transformers.BertForSequenceClassification'
    assert preview['hyperparameters']['hidden_size'] == 16
    assert preview['hyperparameters']['num_hidden_layers'] == 1
    assert preview['hyperparameters']['num_attention_heads'] == 2
    assert preview['structure'] == {'model_type': 'bert'}
    assert 'weights' not in json.dumps(preview['hyperparameters']).lower()
    loaded = BertForSequenceClassification.from_pretrained(
        directory, local_files_only=True).eval()
    torch.manual_seed(42)
    expected = BertForSequenceClassification(
        BertConfig(
            vocab_size=32,
            hidden_size=16,
            num_hidden_layers=1,
            num_attention_heads=2,
            intermediate_size=32,
            max_position_embeddings=32,
            num_labels=2)).eval()
    inputs = torch.tensor([[1, 2, 3, 4]])
    with torch.no_grad():
        torch.testing.assert_close(
            loaded(inputs).logits,
            expected(inputs).logits)


def test_transformers_sharded_safetensors_need_no_weight_reads(
        tmp_path: Path) -> None:
    model = BertForSequenceClassification(
        BertConfig(
            vocab_size=32,
            hidden_size=16,
            num_hidden_layers=1,
            num_attention_heads=2,
            intermediate_size=32))
    artifact = dsl.Model(uri=str(tmp_path))
    with _artifact_preview.ArtifactPreviewSession({'model': artifact
                                                  }) as session:
        model.save_pretrained(tmp_path, max_shard_size='1KB')
        assert len(list(tmp_path.glob('model-*.safetensors'))) > 1
        session.generate()
    assert load_preview(artifact)['framework']['name'] == 'transformers'


@pytest.mark.parametrize('enabled', [True, False])
@pytest.mark.parametrize('framework', ['pytorch', 'transformers'])
def test_pipeline_policy_controls_model_previews(tmp_path: Path, enabled: bool,
                                                 framework: str) -> None:
    original = torch.save
    policy = {} if enabled else {constants.ARTIFACT_PREVIEWS_ENV_VAR: 'false'}
    with mock.patch.dict(os.environ, policy):
        artifact = execute_sample(tmp_path, 'Model', framework=framework)
    assert torch.save is original
    assert ('_kfp_preview' in artifact.metadata) is enabled
    assert (Path(artifact.path) / 'preview.json').exists() is enabled
    assert any(Path(artifact.path).iterdir())


def test_format_collection_pipeline_has_no_preview_tasks(
        tmp_path: Path) -> None:
    package = tmp_path / 'formats.yaml'
    compiler.Compiler().compile(artifact_preview_formats_pipeline, str(package))
    specification = yaml.safe_load(package.read_text())
    assert len(specification['deploymentSpec']['executors']) == 3
    for container in specification['deploymentSpec']['executors'].values():
        assert '--artifact_previews' not in container['container']['args']
        assert '_kfp_preview' not in '\n'.join(
            container['container']['command'])
    outputs = specification['root']['outputDefinitions']['artifacts']
    assert outputs['dataframes']['isArtifactList'] is True
    assert outputs['pytorch_models']['isArtifactList'] is True
