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
import json
from pathlib import Path
from typing import Any
from unittest import mock

from artifact_previews import artifact_preview_list_pipeline
from artifact_previews import artifact_preview_pipeline
from artifact_previews import create_preview_artifacts
from kfp import compiler
from kfp import dsl
from kfp.dsl import _artifact_preview
from kfp.dsl import executor
import numpy as np
import pyarrow as pa
import pyarrow.parquet as pq
import pytest
import xgboost as xgb
import yaml


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


@pytest.mark.parametrize('control', ['component', 'deployment'])
def test_disabling_previews_preserves_dataset_and_model_outputs(
    tmp_path: Path,
    control: str,
) -> None:
    enabled_dataset, enabled_model = produce(tmp_path / 'enabled')
    with mock.patch.object(_artifact_preview, '_parquet_preview') as sample:
        if control == 'deployment':
            policy = tmp_path / 'artifactPreviewsEnabled'
            policy.write_text('false')
            with mock.patch.object(_artifact_preview, '_DEPLOYMENT_CONFIG_PATH',
                                   policy):
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
