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
"""Framework-independent tests of automatic preview lifecycle and bounds."""

import csv
import json
import os
from pathlib import Path
import sys
import tempfile
import types
import unittest
from unittest import mock
import zoneinfo

from kfp import dsl
from kfp.dsl import _artifact_preview as previews
from kfp.dsl import executor
from kfp.dsl import executor_main


class FakeBooster:
    """A native save API with inspectable configuration and no framework
    dependency."""

    def save_model(self, fname: str) -> str:
        Path(fname).write_bytes(b'native-model')
        return 'saved'

    def save_config(self) -> str:
        return json.dumps({
            'learner': {
                'learner_train_param': {
                    'objective': 'binary:logistic'
                },
                'gradient_booster': {
                    'gbtree_model_param': {
                        'num_trees': '5'
                    },
                    'tree_train_param': {
                        'max_depth': '2'
                    },
                },
            },
        })

    def num_features(self) -> int:
        return 24

    def num_boosted_rounds(self) -> int:
        return 5


class FakeEstimator:
    """Exercise the nested sklearn-to-Booster save used by XGBoost."""

    def __init__(self) -> None:
        self.depth = 2
        self.booster = FakeBooster()

    def save_model(self, fname: str) -> str:
        return self.booster.save_model(fname)

    def get_booster(self) -> FakeBooster:
        return self.booster

    def get_params(self, deep: bool = True) -> dict:
        return {
            'max_depth': self.depth,
            'missing': float('nan'),
            'learning_rate': None
        }


class FakeTensor:
    """Expose tensor structure without importing PyTorch or inspecting
    values."""

    shape = (2, 3)
    dtype = 'torch.float32'

    def numel(self) -> int:
        return 6


FRAMEWORK = types.SimpleNamespace(
    Booster=FakeBooster,
    XGBModel=FakeEstimator,
    __version__='3.0.5',
    core=types.SimpleNamespace(XGBoostError=RuntimeError),
)


def fake_framework_import(name: str) -> types.SimpleNamespace:
    """Keep lifecycle tests independent of installed scientific libraries."""
    if name == 'xgboost':
        return FRAMEWORK
    raise ImportError(f'{name} is not installed in this test')


class UnexpectedLibraryError(Exception):
    """An error type that preview code does not anticipate."""


def sibling_preview(artifact: dsl.Artifact) -> Path:
    """Return the generated preview object beside an artifact directory."""
    path = Path(artifact.path)
    return path.with_name(path.name + previews.PREVIEW_SIBLING_SUFFIX)


class ArtifactPreviewTest(unittest.TestCase):
    """Automatic previews must never alter the primary artifact or native
    save."""

    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.model = dsl.Model(uri=str(self.root / 'model'))
        Path(self.model.path).mkdir()
        self.import_module = mock.patch.object(
            previews.importlib,
            'import_module',
            side_effect=fake_framework_import)
        self.import_module.start()
        self.addCleanup(self.import_module.stop)

    def load_preview(self, artifact: dsl.Artifact) -> dict:
        return json.loads(sibling_preview(artifact).read_bytes())

    def runtime_input(self, **artifacts: dsl.Artifact) -> dict:
        """Build executor inputs for the declared output artifacts."""
        return {
            'inputs': {},
            'outputs': {
                'artifacts': {
                    name: {
                        'artifacts': [{
                            'uri': artifact.uri,
                            'type': {
                                'schemaTitle': artifact.schema_title
                            },
                        }]
                    } for name, artifact in artifacts.items()
                },
                'outputFile': str(self.root / 'executor_output.json'),
            },
        }

    def test_disabled_previews_leave_outputs_unchanged_without_framework_imports(
            self) -> None:
        """Component and pipeline opt-outs both reach the executor as this
        setting, which skips dataset sampling and model hooks."""
        dataset = dsl.Dataset(uri=str(self.root / 'dataset'))
        original = FakeEstimator.save_model

        def produce(dataset: dsl.Output[dsl.Dataset],
                    model: dsl.Output[dsl.Model]) -> None:
            self.assertIs(FakeEstimator.save_model, original)
            Path(dataset.path).mkdir()
            (Path(dataset.path) / 'data.parquet').write_bytes(b'native-parquet')
            FakeEstimator().save_model(Path(model.path) / 'model.ubj')

        with mock.patch.object(previews.importlib, 'import_module') as imported:
            with mock.patch.object(previews, '_dataset_preview') as sample:
                runtime = executor.Executor(
                    self.runtime_input(dataset=dataset, model=self.model),
                    produce,
                    artifact_previews=False)
                result = runtime.execute()
        self.assertIsNone(runtime._artifact_previews)
        imported.assert_not_called()
        sample.assert_not_called()
        metadata = json.loads(Path(result).read_text())
        for name, artifact in (('dataset', dataset), ('model', self.model)):
            self.assertNotIn(
                '_kfp_preview',
                metadata['artifacts'][name]['artifacts'][0]['metadata'])
            self.assertFalse(sibling_preview(artifact).exists())
        self.assertEqual((Path(dataset.path) / 'data.parquet').read_bytes(),
                         b'native-parquet')
        self.assertEqual((Path(self.model.path) / 'model.ubj').read_bytes(),
                         b'native-model')

    def test_disabled_previews_keep_producer_written_previews(self) -> None:
        """Opting out stops generation without hiding a manual preview."""
        descriptor = {'version': 1, 'path': '_kfp_preview.json'}
        content = b'{"version":1,"kind":"model"}'

        def produce(model: dsl.Output[dsl.Model]) -> None:
            (Path(model.path) / '_kfp_preview.json').write_bytes(content)
            model.metadata['_kfp_preview'] = descriptor

        result = executor.Executor(
            self.runtime_input(model=self.model),
            produce,
            artifact_previews=False).execute()
        metadata = json.loads(Path(result).read_text())
        self.assertEqual(
            metadata['artifacts']['model']['artifacts'][0]['metadata']
            ['_kfp_preview'], descriptor)
        self.assertEqual(
            (Path(self.model.path) / '_kfp_preview.json').read_bytes(), content)

    def test_previews_do_not_read_deployment_configuration(self) -> None:
        with mock.patch.object(
                Path,
                'read_text',
                side_effect=AssertionError('No deployment configuration read')):
            with previews.ArtifactPreviewSession({'model': self.model
                                                 }) as session:
                FakeEstimator().save_model(Path(self.model.path) / 'model.ubj')
                session.generate()
        self.assertIn('_kfp_preview', self.model.metadata)

    def test_native_save_is_unchanged_and_configuration_is_captured_at_save_time(
            self) -> None:
        original_model_save = FakeEstimator.save_model
        original_booster_save = FakeBooster.save_model
        estimator = FakeEstimator()
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            self.assertEqual(
                estimator.save_model(fname=Path(self.model.path) / 'model.ubj'),
                'saved')
            estimator.depth = 9
            session.generate()
        self.assertIs(FakeEstimator.save_model, original_model_save)
        self.assertIs(FakeBooster.save_model, original_booster_save)
        preview = self.load_preview(self.model)
        self.assertEqual(preview['hyperparameters']['max_depth'], 2)
        self.assertEqual(preview['hyperparameters']['missing'], 'NaN')
        self.assertNotIn('learning_rate', preview['hyperparameters'])
        self.assertEqual(preview['structure']['tree_count'], 5)
        self.assertEqual(self.model.metadata['_kfp_preview'], {
            'version': 2,
            'storage': 'sibling',
        })
        self.assertEqual((Path(self.model.path) / 'model.ubj').read_bytes(),
                         b'native-model')
        self.assertFalse(
            list(Path(self.model.path).parent.glob('.kfp-preview-*.tmp')))

    def test_native_booster_configuration_is_supported(self) -> None:
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            FakeBooster().save_model(Path(self.model.path) / 'model.ubj')
            session.generate()
        self.assertEqual(
            self.load_preview(
                self.model)['hyperparameters']['tree_parameters']['max_depth'],
            '2')

    def test_unrelated_saves_and_unknown_model_files_are_not_inspected(
            self) -> None:
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            FakeEstimator().save_model(self.root / 'unrelated.ubj')
            (Path(self.model.path) / 'untrusted.pkl').write_bytes(b'not loaded')
            with self.assertLogs(
                    previews.__name__, level='WARNING') as messages:
                session.generate()
        self.assertIn('No supported XGBoost or PyTorch save',
                      '\n'.join(messages.output))
        self.assertNotIn('_kfp_preview', self.model.metadata)

    def test_changed_model_does_not_get_stale_configuration(self) -> None:
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            path = Path(self.model.path) / 'model.ubj'
            FakeEstimator().save_model(path)
            path.write_bytes(b'different-model')
            with self.assertLogs(
                    previews.__name__, level='WARNING') as messages:
                session.generate()
        self.assertIn('changed after', '\n'.join(messages.output))
        self.assertNotIn('_kfp_preview', self.model.metadata)

    def test_changed_artifact_directory_does_not_get_another_models_preview(
            self) -> None:
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            FakeEstimator().save_model(Path(self.model.path) / 'model.ubj')
            self.model.uri = str(self.root / 'replacement')
            Path(self.model.path).mkdir()
            with self.assertLogs(
                    previews.__name__, level='WARNING') as messages:
                session.generate()
        self.assertIn('output artifact path changed',
                      '\n'.join(messages.output))
        self.assertNotIn('_kfp_preview', self.model.metadata)
        self.assertFalse(sibling_preview(self.model).exists())

    def test_resaving_the_same_model_keeps_the_final_configuration(
            self) -> None:
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            model = FakeEstimator()
            path = Path(self.model.path) / 'model.ubj'
            model.save_model(path)
            model.depth = 7
            model.save_model(path)
            session.generate()
        self.assertEqual(
            self.load_preview(self.model)['hyperparameters']['max_depth'], 7)

    def test_multiple_models_in_one_artifact_are_not_misrepresented(
            self) -> None:
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            FakeEstimator().save_model(Path(self.model.path) / 'first.ubj')
            FakeEstimator().save_model(Path(self.model.path) / 'second.ubj')
            with self.assertLogs(previews.__name__, level='WARNING'):
                session.generate()
        self.assertNotIn('_kfp_preview', self.model.metadata)

    def test_capture_failure_does_not_fail_the_native_save(self) -> None:
        for error in (ValueError('bad config'),
                      UnexpectedLibraryError('unexpected library error')):
            with self.subTest(error=type(error).__name__):
                with previews.ArtifactPreviewSession({'model': self.model
                                                     }) as session:
                    with mock.patch.object(
                            FakeEstimator, 'get_params', side_effect=error):
                        with self.assertLogs(
                                previews.__name__, level='WARNING'):
                            result = FakeEstimator().save_model(
                                Path(self.model.path) / 'model.ubj')
                    self.assertEqual(result, 'saved')
                    self.assertTrue(
                        (Path(self.model.path) / 'model.ubj').exists())
                    self.assertNotIn('model', session.models)
                self.assertEqual(previews._SESSIONS, [])

    def test_hooks_restore_after_component_and_native_save_exceptions(
            self) -> None:
        original = FakeEstimator.save_model
        with self.assertRaisesRegex(RuntimeError, 'training failed'):
            with previews.ArtifactPreviewSession({'model': self.model}):
                raise RuntimeError('training failed')
        self.assertIs(FakeEstimator.save_model, original)
        with mock.patch.object(
                FakeBooster, 'save_model',
                side_effect=OSError('disk full')) as save:
            with self.assertRaisesRegex(OSError, 'disk full'):
                with previews.ArtifactPreviewSession({'model': self.model}):
                    FakeEstimator().save_model(
                        Path(self.model.path) / 'model.ubj')
            self.assertEqual(save.call_count, 1)
        self.assertIs(FakeEstimator.save_model, original)
        self.assertNotIn('_kfp_preview', self.model.metadata)

    def test_partial_hook_installation_is_rolled_back(self) -> None:
        original = FakeBooster.save_model
        unsupported = types.SimpleNamespace(
            Booster=FakeBooster, XGBModel=object)
        with mock.patch.object(
                previews.importlib, 'import_module', return_value=unsupported):
            with self.assertLogs(previews.__name__, level='WARNING'):
                with previews.ArtifactPreviewSession({'model': self.model}):
                    self.assertIs(FakeBooster.save_model, original)
        self.assertEqual(previews._SESSIONS, [])
        self.assertEqual(previews._SAVE_METHODS, [])

    def test_broken_optional_import_does_not_block_other_adapters_or_execution(
            self) -> None:
        """A broken XGBoost install must not prevent native PyTorch outputs."""
        for error_type in (ValueError, TypeError, AttributeError, KeyError,
                           OSError, RuntimeError):
            with self.subTest(error=error_type.__name__):
                model = dsl.Model(
                    uri=str(self.root / f'model-{error_type.__name__}'))
                Path(model.path).mkdir()
                native_save = FakeBooster.save_model
                torch = types.SimpleNamespace(
                    __version__='2.14.1',
                    Tensor=FakeTensor,
                    nn=types.SimpleNamespace(Module=FakeEstimator),
                    save=native_save,
                    jit=types.SimpleNamespace(
                        save=native_save,
                        ScriptModule=types.SimpleNamespace(save=native_save)))

                def import_framework(name: str) -> types.SimpleNamespace:
                    if name == 'xgboost':
                        raise error_type('binary-incompatible optional library')
                    self.assertEqual(name, 'torch')
                    return torch

                def produce(model: dsl.Output[dsl.Model]) -> None:
                    torch.save({'weight': FakeTensor()},
                               Path(model.path) / 'model.pt')

                with mock.patch.object(
                        previews.importlib,
                        'import_module',
                        side_effect=import_framework):
                    with self.assertLogs(
                            previews.__name__, level='WARNING') as messages:
                        result = executor.Executor(
                            self.runtime_input(model=model), produce).execute()
                self.assertIn('xgboost', '\n'.join(messages.output))
                self.assertIn('binary-incompatible', '\n'.join(messages.output))
                metadata = json.loads(Path(result).read_text())
                self.assertIn(
                    '_kfp_preview',
                    metadata['artifacts']['model']['artifacts'][0]['metadata'])
                preview = self.load_preview(model)
                self.assertEqual(preview['framework']['name'], 'pytorch')
                self.assertEqual(preview['structure']['tensor_element_count'],
                                 6)
                self.assertEqual((Path(model.path) / 'model.pt').read_bytes(),
                                 b'native-model')
                self.assertIs(torch.save, native_save)
                self.assertEqual(previews._SESSIONS, [])
                self.assertEqual(previews._SAVE_METHODS, [])

    def test_import_process_control_exceptions_propagate_and_restore_hooks(
            self) -> None:
        """Interrupted entry must not leave hooks installed without a
        session."""
        original_booster_save = FakeBooster.save_model
        original_estimator_save = FakeEstimator.save_model
        for error_type in (KeyboardInterrupt, SystemExit):
            for failing_framework in ('xgboost', 'torch'):
                with self.subTest(
                        error=error_type.__name__, framework=failing_framework):

                    def import_framework(name: str) -> types.SimpleNamespace:
                        if name == failing_framework:
                            raise error_type('interrupted import')
                        return fake_framework_import(name)

                    with mock.patch.object(
                            previews.importlib,
                            'import_module',
                            side_effect=import_framework):
                        with self.assertRaisesRegex(error_type,
                                                    'interrupted import'):
                            with previews.ArtifactPreviewSession(
                                {'model': self.model}):
                                self.fail('The component must not execute.')
                    self.assertIs(FakeBooster.save_model, original_booster_save)
                    self.assertIs(FakeEstimator.save_model,
                                  original_estimator_save)
                    self.assertEqual(previews._SESSIONS, [])
                    self.assertEqual(previews._SAVE_METHODS, [])

    def test_native_booster_does_not_require_the_optional_sklearn_api(
            self) -> None:
        native_only = types.SimpleNamespace(
            Booster=FakeBooster,
            __version__=FRAMEWORK.__version__,
            core=FRAMEWORK.core)
        with mock.patch.object(
                previews.importlib, 'import_module', return_value=native_only):
            with previews.ArtifactPreviewSession({'model': self.model
                                                 }) as session:
                FakeBooster().save_model(Path(self.model.path) / 'model.ubj')
                session.generate()
        self.assertEqual(
            self.load_preview(self.model)['structure']['boosting_rounds'], 5)

    def test_missing_framework_dependencies_are_logged(self) -> None:
        dataset = dsl.Dataset(uri=str(self.root / 'dataset'))
        Path(dataset.path).mkdir()
        (Path(dataset.path) / 'data.parquet').touch()
        with mock.patch.object(
                previews.importlib, 'import_module', side_effect=ImportError):
            with self.assertLogs(
                    previews.__name__, level='WARNING') as messages:
                with previews.ArtifactPreviewSession({
                        'dataset': dataset,
                        'model': self.model
                }) as session:
                    session.generate()
        self.assertIn('XGBoost', '\n'.join(messages.output))
        self.assertIn('PyArrow', '\n'.join(messages.output))
        self.assertEqual(dataset.metadata, {})
        self.assertEqual(self.model.metadata, {})

    def test_later_framework_integrations_are_not_overwritten(self) -> None:
        original = FakeEstimator.save_model

        def integration(model: FakeEstimator, filename: str) -> str:
            return original(model, filename)

        try:
            with self.assertLogs(previews.__name__, level='WARNING'):
                with previews.ArtifactPreviewSession({'model': self.model}):
                    FakeEstimator.save_model = integration
            self.assertIs(FakeEstimator.save_model, integration)
        finally:
            FakeEstimator.save_model = original

    def test_overlapping_sessions_do_not_cross_artifact_boundaries(
            self) -> None:
        other = dsl.Model(uri=str(self.root / 'other'))
        Path(other.path).mkdir()
        original = FakeEstimator.save_model
        first = previews.ArtifactPreviewSession({'model': self.model})
        second = previews.ArtifactPreviewSession({'model': other})
        first.__enter__()
        second.__enter__()
        try:
            FakeEstimator().save_model(Path(self.model.path) / 'model.ubj')
            self.assertNotIn('model', second.models)
            first.generate()
            first.__exit__()
            self.assertIsNot(FakeEstimator.save_model, original)
            FakeEstimator().save_model(Path(other.path) / 'model.ubj')
            second.generate()
        finally:
            if first in previews._SESSIONS:
                first.__exit__()
            second.__exit__()
        self.assertIs(FakeEstimator.save_model, original)
        self.assertTrue(sibling_preview(other).exists())

    def test_manual_previews_are_not_overwritten(self) -> None:
        self.model.metadata['_kfp_preview'] = {
            'version': 1,
            'path': '_kfp_preview.json'
        }
        path = Path(self.model.path) / '_kfp_preview.json'
        path.write_text('existing preview')
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            FakeEstimator().save_model(Path(self.model.path) / 'model.ubj')
            session.generate()
        self.assertEqual(path.read_text(), 'existing preview')

    def test_unadvertised_existing_file_is_not_overwritten(self) -> None:
        path = sibling_preview(self.model)
        path.write_text('user file')
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            FakeEstimator().save_model(Path(self.model.path) / 'model.ubj')
            with self.assertLogs(previews.__name__, level='WARNING'):
                session.generate()
        self.assertEqual(path.read_text(), 'user file')
        self.assertNotIn('_kfp_preview', self.model.metadata)

    def test_single_file_artifacts_keep_their_layout(self) -> None:
        artifact = dsl.Model(uri=str(self.root / 'single-file'))
        with previews.ArtifactPreviewSession({'model': artifact}) as session:
            FakeEstimator().save_model(artifact.path)
            with self.assertLogs(previews.__name__, level='WARNING'):
                session.generate()
        self.assertTrue(Path(artifact.path).is_file())
        self.assertEqual(artifact.metadata, {})

    def test_values_are_bounded_and_json_safe(self) -> None:
        values = previews._Values()
        self.assertEqual(values.value(2**63), str(2**63))
        self.assertEqual(values.value(float('inf')), 'Infinity')
        self.assertEqual(values.value(1e30), str(1e30))
        self.assertEqual(values.value(None), None)
        self.assertEqual(len(values.value('😀' * 400)), 256)
        self.assertLessEqual(len(values.value(list(range(1000)))), 100)
        cyclic = []
        cyclic.append(cyclic)
        self.assertIn('omitted', json.dumps(values.value(cyclic)))
        self.assertTrue(values.shortened)

    def test_text_samples_bound_rows_columns_and_preserve_unknown_totals(
            self) -> None:
        for extension in ('csv', 'json', 'jsonl'):
            with self.subTest(extension=extension):
                directory = self.root / extension
                directory.mkdir()
                records = [{
                    f'column_{column}': row + column for column in range(25)
                } for row in range(32)]
                path = directory / f'data.{extension}'
                if extension == 'csv':
                    with path.open('w', newline='') as stream:
                        writer = csv.DictWriter(
                            stream, fieldnames=list(records[0]))
                        writer.writeheader()
                        writer.writerows(records)
                else:
                    path.write_text(
                        json.dumps(records) if extension ==
                        'json' else '\n'.join(
                            json.dumps(record) for record in records))
                preview = previews._dataset_preview(directory, True)
                self.assertEqual(len(preview['rows']), 10)
                self.assertEqual(len(preview['columns']), 20)
                self.assertIsNone(preview['total_rows'])
                self.assertEqual(preview['total_columns'],
                                 25 if extension == 'csv' else None)
                self.assertTrue(preview['sample']['rows_truncated'])
                self.assertTrue(preview['sample']['columns_truncated'])
                self.assertEqual(preview['rows'][0][0],
                                 '0' if extension == 'csv' else 0)
                schema = previews._dataset_preview(directory, False)
                self.assertEqual(schema['rows'], [])
                self.assertTrue(schema['sample']['rows_omitted'])
                self.assertLessEqual(
                    len(previews._encode(preview)), previews.MAX_BYTES)

    def test_csv_preserves_quoted_multiline_and_empty_cells(self) -> None:
        directory = self.root / 'csv'
        directory.mkdir()
        (directory /
         'data.csv').write_text('name,note,empty\n"a,b","first\nsecond",\n')
        preview = previews._dataset_preview(directory, True)
        self.assertEqual(preview['rows'], [['a,b', 'first\nsecond', '']])
        self.assertEqual(preview['total_rows'], 1)
        self.assertFalse(preview['sample']['rows_truncated'])

    def test_json_types_precision_and_nested_omissions(self) -> None:
        directory = self.root / 'json'
        directory.mkdir()
        (directory / 'data.json').write_text(
            json.dumps([
                {
                    'integer': 2**63,
                    'nested': {
                        'secret': 1
                    },
                    'nullable': None
                },
                {
                    'integer': 7,
                    'nested': [1, 2],
                    'nullable': False
                },
            ]))
        preview = previews._dataset_preview(directory, True)
        self.assertEqual(preview['rows'][0],
                         [str(2**63), '<nested JSON value omitted>', None])
        self.assertEqual(preview['total_rows'], 2)
        self.assertEqual(preview['total_columns'], 3)
        self.assertTrue(preview['sample']['values_truncated'])
        self.assertEqual(preview['columns'][2]['dtype'],
                         'boolean | null (sample)')

    def test_large_json_records_use_only_a_bounded_prefix(self) -> None:
        payload = json.dumps([{'value': index} for index in range(200)])
        path = self.root / 'data.json'
        path.write_text(payload)
        with mock.patch.object(previews, 'MAX_TEXT_SOURCE_BYTES', 256):
            preview = previews._text_dataset_preview(path, True)
        self.assertEqual(preview['rows'], [[index] for index in range(10)])
        self.assertIsNone(preview['total_rows'])
        self.assertTrue(preview['sample']['rows_truncated'])

    def test_bounded_text_read_does_not_decode_partial_utf8(self) -> None:
        path = self.root / 'data.json'
        path.write_text('["\U0001f600"]', encoding='utf-8')
        with mock.patch.object(previews, 'MAX_TEXT_SOURCE_BYTES', 4):
            text, complete = previews._text_prefix(path)
        self.assertEqual(text, '["')
        self.assertFalse(complete)

    def test_malformed_text_data_is_not_published(self) -> None:
        for extension, content in (('csv', 'a,b\n1\n'),
                                   ('csv', 'a\n"unterminated'), ('json',
                                                                 '[{"a": 1},]'),
                                   ('json', '[{"a": 1}]trailing'),
                                   ('json', '[{"a": 1} {"a": 2}]'),
                                   ('json', '[{"a": 1, "a": 2}]'),
                                   ('json', '[1, 2]'), ('jsonl',
                                                        '{"a": 1}{"a": 2}')):
            with self.subTest(content=content):
                path = self.root / f'data.{extension}'
                path.write_text(content)
                with self.assertRaises((ValueError, csv.Error)):
                    previews._text_dataset_preview(path, True)

    def test_transformers_configuration_never_opens_weight_files(self) -> None:
        directory = Path(self.model.path)
        (directory / 'config.json').write_text(
            json.dumps({
                'model_type': 'bert',
                'transformers_version': '5.20.0',
                'architectures': ['BertForSequenceClassification'],
                'hidden_size': 16,
                '_name_or_path': 'private-training-path',
            }))
        (directory / 'model.safetensors').write_bytes(b'not inspected')
        session = previews.ArtifactPreviewSession({'model': self.model})
        with mock.patch.object(previews.importlib, 'import_module') as imported:
            session.generate()
        imported.assert_not_called()
        preview = self.load_preview(self.model)
        self.assertEqual(preview['framework']['name'], 'transformers')
        self.assertEqual(preview['estimator_class'],
                         'transformers.BertForSequenceClassification')
        self.assertEqual(preview['hyperparameters'], {'hidden_size': 16})
        self.assertEqual(preview['structure'], {'model_type': 'bert'})

    def test_transformers_config_size_and_symlinks_are_rejected(self) -> None:
        path = Path(self.model.path) / 'config.json'
        path.write_bytes(b' ' * (previews.MAX_BYTES + 1))
        with self.assertRaisesRegex(ValueError, '64 KiB'):
            previews._transformers_preview(Path(self.model.path))
        path.unlink()
        external_config = self.root / 'external-config.json'
        external_config.write_text('{}')
        path.symlink_to(external_config)
        with self.assertRaisesRegex(ValueError, 'regular file'):
            previews._transformers_preview(Path(self.model.path))

    def test_recursive_model_configuration_does_not_fail_executor_finalization(
            self) -> None:
        """Skip one recursive configuration while finalizing all native
        outputs."""
        # Python 3.13's JSON C-stack limit can exceed sys.getrecursionlimit().
        depth = 32_000
        config = ('{"model_type":"bert","transformers_version":"5.20.0",'
                  '"nested":' + '[' * depth + '0' + ']' * depth + '}')
        self.assertLess(len(config.encode()), previews.MAX_BYTES)
        with self.assertRaises(RecursionError):
            json.loads(config, object_pairs_hook=previews._json_object)
        dataset = dsl.Dataset(uri=str(self.root / 'dataset'))

        def produce(model: dsl.Output[dsl.Model],
                    dataset: dsl.Output[dsl.Dataset]) -> None:
            (Path(model.path) / 'config.json').write_text(config)
            (Path(model.path) /
             'model.safetensors').write_bytes(b'native-weights')
            Path(dataset.path).mkdir()
            (Path(dataset.path) / 'data.csv').write_text('value\n1\n')

        with self.assertLogs(previews.__name__, level='WARNING') as messages:
            result = executor.Executor(
                self.runtime_input(model=self.model, dataset=dataset),
                produce).execute()
        self.assertIn("output 'model'", '\n'.join(messages.output))
        metadata = json.loads(Path(result).read_text())['artifacts']
        self.assertNotIn('_kfp_preview',
                         metadata['model']['artifacts'][0]['metadata'])
        self.assertIn('_kfp_preview',
                      metadata['dataset']['artifacts'][0]['metadata'])
        self.assertFalse(sibling_preview(self.model).exists())
        self.assertEqual((Path(self.model.path) / 'config.json').read_text(),
                         config)
        self.assertEqual(
            (Path(self.model.path) / 'model.safetensors').read_bytes(),
            b'native-weights')
        self.assertEqual(self.load_preview(dataset)['rows'], [['1']])

    def test_unexpected_conversion_errors_do_not_fail_executor_finalization(
            self) -> None:
        """Skip only a preview whose library conversion fails unexpectedly."""
        failing = dsl.Dataset(uri=str(self.root / 'failing'))
        dataset = dsl.Dataset(uri=str(self.root / 'dataset'))
        sample = previews._dataset_preview

        def produce(failing: dsl.Output[dsl.Dataset],
                    dataset: dsl.Output[dsl.Dataset],
                    model: dsl.Output[dsl.Model]) -> None:
            for output in (failing, dataset):
                Path(output.path).mkdir()
                (Path(output.path) / 'data.csv').write_text('value\n1\n')
            FakeEstimator().save_model(Path(model.path) / 'model.ubj')

        def unresolvable_time_zone(directory: Path, include_rows: bool) -> dict:
            # PyArrow raises this KeyError subclass for time zone-aware
            # cells when no time zone database can resolve the zone.
            if directory == Path(failing.path):
                raise zoneinfo.ZoneInfoNotFoundError(
                    'No time zone found with key America/New_York')
            return sample(directory, include_rows)

        with mock.patch.object(
                previews, '_dataset_preview',
                side_effect=unresolvable_time_zone):
            with self.assertLogs(
                    previews.__name__, level='WARNING') as messages:
                result = executor.Executor(
                    self.runtime_input(
                        failing=failing, dataset=dataset, model=self.model),
                    produce).execute()
        self.assertIn("output 'failing'", '\n'.join(messages.output))
        metadata = json.loads(Path(result).read_text())['artifacts']
        self.assertNotIn('_kfp_preview',
                         metadata['failing']['artifacts'][0]['metadata'])
        self.assertFalse(sibling_preview(failing).exists())
        self.assertEqual((Path(failing.path) / 'data.csv').read_text(),
                         'value\n1\n')
        for name in ('dataset', 'model'):
            self.assertIn('_kfp_preview',
                          metadata[name]['artifacts'][0]['metadata'])
        self.assertEqual(self.load_preview(dataset)['rows'], [['1']])

    def test_large_model_structure_is_trimmed_to_the_byte_limit(self) -> None:
        preview = previews._model_preview('pytorch', '2', 'module', {}, {
            'tensors': {
                f'{index}' + '\U0001f600' * 250: {
                    'shape': [8, 4],
                    'dtype': 'float32'
                } for index in range(100)
            }
        }, [])
        self.assertLessEqual(len(previews._encode(preview)), previews.MAX_BYTES)
        self.assertTrue(preview['structure']['tensors'])
        self.assertTrue(any('64 KiB' in note for note in preview['notes']))

    def test_oversized_preview_does_not_publish_a_descriptor_or_partial_file(
            self) -> None:
        with self.assertRaisesRegex(ValueError, 'exceeds 64 KiB'):
            previews.ArtifactPreviewSession._publish(
                Path(self.model.path), self.model,
                {'value': 'x' * previews.MAX_BYTES})
        self.assertFalse(sibling_preview(self.model).exists())
        self.assertEqual(self.model.metadata, {})

    def test_executor_automatically_generates_preview_without_a_user_adapter_call(
            self) -> None:

        def produce(model: dsl.Output[dsl.Model]) -> None:
            FakeEstimator().save_model(Path(model.path) / 'model.ubj')

        result = executor.Executor(
            self.runtime_input(model=self.model), produce).execute()
        metadata = json.loads(Path(result).read_text())
        self.assertEqual(
            metadata['artifacts']['model']['artifacts'][0]['metadata']
            ['_kfp_preview'], {
                'version': 2,
                'storage': 'sibling',
            })

    def test_executor_cli_defaults_to_previews_with_schema_only_support(
            self) -> None:
        dataset = dsl.Dataset(uri=str(self.root / 'dataset'))

        def produce(dataset: dsl.Output[dsl.Dataset],
                    model: dsl.Output[dsl.Model]) -> None:
            Path(dataset.path).mkdir()
            FakeEstimator().save_model(Path(model.path) / 'model.ubj')

        runtime_input = self.runtime_input(model=self.model, dataset=dataset)
        arguments = [
            'executor_main',
            '--component_module_path',
            str(self.root / 'producer.py'),
            '--function_to_execute',
            'produce',
            '--executor_input',
            json.dumps(runtime_input),
            '--artifact_preview_schema_only',
        ]
        with mock.patch.object(sys, 'argv', arguments), mock.patch.object(
                executor_main.utils,
                'load_module',
                return_value=types.SimpleNamespace(
                    produce=produce)), mock.patch.object(
                        previews,
                        '_dataset_preview',
                        return_value={'kind': 'dataset'}) as sample:
            executor_main.executor_main()
        sample.assert_called_once_with(Path(dataset.path), False)
        metadata = json.loads((self.root / 'executor_output.json').read_text())
        for name in ('dataset', 'model'):
            self.assertIn(
                '_kfp_preview',
                metadata['artifacts'][name]['artifacts'][0]['metadata'])

    def test_executor_cli_preview_flags(self) -> None:

        def produce() -> None:
            pass

        runtime_input = self.runtime_input()
        arguments = [
            'executor_main', '--component_module_path',
            str(self.root / 'producer.py'), '--function_to_execute', 'produce',
            '--executor_input',
            json.dumps(runtime_input)
        ]
        for flags, enabled, rows in (
            ([], True, True),
            (['--disable_artifact_previews'], False, True),
            (['--artifact_preview_schema_only'], True, False),
                # A pipeline opt-out overrides a schema-only component.
            (['--artifact_preview_schema_only',
              '--disable_artifact_previews'], False, False),
            (['--disable_artifact_previews',
              '--disable_artifact_previews'], False, True),
                # Executors ignore flags introduced by newer SDKs.
            (['--future_preview_option'], True, True),
        ):
            with self.subTest(flags=flags):
                with mock.patch.object(sys, 'argv', arguments + flags):
                    with mock.patch.object(
                            executor_main.utils,
                            'load_module',
                            return_value=types.SimpleNamespace(
                                produce=produce)):
                        with mock.patch.object(executor, 'Executor') as runtime:
                            executor_main.executor_main()
                runtime.assert_called_once_with(
                    executor_input=runtime_input,
                    function_to_execute=produce,
                    artifact_previews=enabled,
                    artifact_preview_rows=rows)

    def test_non_chief_workers_do_not_publish_previews(self) -> None:

        def produce(model: dsl.Output[dsl.Model]) -> None:
            FakeEstimator().save_model(Path(model.path) / 'model.ubj')

        runtime_input = {
            'inputs': {},
            'outputs': {
                'artifacts': {
                    'model': {
                        'artifacts': [{
                            'uri': self.model.uri,
                            'type': {
                                'schemaTitle': 'system.Model'
                            },
                        }]
                    }
                },
                'outputFile': str(self.root / 'executor_output.json'),
            },
        }
        with mock.patch.dict(os.environ,
                             {'CLUSTER_SPEC': '{"task":{"type":"worker"}}'}):
            self.assertIsNone(
                executor.Executor(
                    runtime_input, produce, artifact_previews=True).execute())
        self.assertFalse(sibling_preview(self.model).exists())
