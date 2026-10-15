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

import json
import os
from pathlib import Path
import sys
import tempfile
import types
import unittest
from unittest import mock

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


FRAMEWORK = types.SimpleNamespace(
    Booster=FakeBooster,
    XGBModel=FakeEstimator,
    __version__='3.0.5',
    core=types.SimpleNamespace(XGBoostError=RuntimeError),
)


class ArtifactPreviewTest(unittest.TestCase):
    """Automatic previews must never alter the primary artifact or native
    save."""

    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.policy = self.root / 'artifactPreviewsEnabled'
        policy_path = mock.patch.object(previews, '_DEPLOYMENT_CONFIG_PATH',
                                        self.policy)
        policy_path.start()
        self.addCleanup(policy_path.stop)
        self.model = dsl.Model(uri=str(self.root / 'model'))
        Path(self.model.path).mkdir()
        self.import_module = mock.patch.object(
            previews.importlib, 'import_module', return_value=FRAMEWORK)
        self.import_module.start()
        self.addCleanup(self.import_module.stop)

    def load_preview(self, artifact: dsl.Artifact) -> dict:
        return json.loads((Path(artifact.path) / 'preview.json').read_bytes())

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

    def test_component_can_opt_out(self) -> None:

        def produce(model: dsl.Output[dsl.Model]) -> None:
            FakeEstimator().save_model(Path(model.path) / 'model.ubj')

        with mock.patch.object(previews.importlib, 'import_module') as imported:
            runtime = executor.Executor(
                self.runtime_input(model=self.model),
                produce,
                artifact_previews=False)
            runtime.execute()
        self.assertIsNone(runtime._artifact_previews)
        imported.assert_not_called()
        self.assertEqual((Path(self.model.path) / 'model.ubj').read_bytes(),
                         b'native-model')
        self.assertFalse((Path(self.model.path) / 'preview.json').exists())

    def test_deployment_policy_defaults_to_enabled(self) -> None:
        self.assertTrue(previews._deployment_allows_previews())
        for value, enabled in (('true', True), ('false', False),
                               (' TRUE\n', True), (' False\n', False)):
            with self.subTest(value=value):
                self.policy.write_text(value)
                self.assertEqual(previews._deployment_allows_previews(),
                                 enabled)

    def test_invalid_policy_disables_previews_without_framework_imports(
            self) -> None:
        original = FakeEstimator.save_model
        for value in (b'', b'yes', b'0', b'1', b'invalid', b'\xff'):
            with self.subTest(value=value):
                self.policy.write_bytes(value)
                with mock.patch.object(previews.importlib,
                                       'import_module') as imported:
                    with self.assertLogs(previews.__name__, level='WARNING'):
                        with previews.ArtifactPreviewSession(
                            {'model': self.model}) as session:
                            self.assertIs(FakeEstimator.save_model, original)
                            session.generate()
                imported.assert_not_called()
                self.assertFalse(
                    (Path(self.model.path) / 'preview.json').exists())
                self.assertEqual(self.model.metadata, {})

    def test_unreadable_policy_disables_previews_with_a_warning(self) -> None:
        with mock.patch.object(
                Path, 'read_text', side_effect=PermissionError('denied')):
            with self.assertLogs(
                    previews.__name__, level='WARNING') as messages:
                self.assertFalse(previews._deployment_allows_previews())
        self.assertIn('cannot read', '\n'.join(messages.output))

    def test_deployment_opt_out_overrides_component_enablement(self) -> None:
        self.policy.write_text('false')
        dataset = dsl.Dataset(uri=str(self.root / 'dataset'))
        original = FakeEstimator.save_model

        def produce(dataset: dsl.Output[dsl.Dataset],
                    model: dsl.Output[dsl.Model]) -> None:
            self.assertIs(FakeEstimator.save_model, original)
            Path(dataset.path).mkdir()
            (Path(dataset.path) / 'data.parquet').write_bytes(b'native-parquet')
            FakeEstimator().save_model(Path(model.path) / 'model.ubj')

        with mock.patch.object(previews.importlib, 'import_module') as imported:
            with mock.patch.object(previews, '_parquet_preview') as sample:
                result = executor.Executor(
                    self.runtime_input(dataset=dataset, model=self.model),
                    produce,
                    artifact_previews=True).execute()
        imported.assert_not_called()
        sample.assert_not_called()
        metadata = json.loads(Path(result).read_text())
        for name, artifact in (('dataset', dataset), ('model', self.model)):
            self.assertNotIn(
                '_kfp_preview',
                metadata['artifacts'][name]['artifacts'][0]['metadata'])
            self.assertFalse((Path(artifact.path) / 'preview.json').exists())
        self.assertEqual((Path(dataset.path) / 'data.parquet').read_bytes(),
                         b'native-parquet')
        self.assertEqual((Path(self.model.path) / 'model.ubj').read_bytes(),
                         b'native-model')

    def test_deployment_opt_out_during_execution_stops_publication(
            self) -> None:
        original = FakeEstimator.save_model
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            FakeEstimator().save_model(Path(self.model.path) / 'model.ubj')
            self.assertIn('model', session.models)
            self.policy.write_text('false')
            session.generate()
        self.assertIs(FakeEstimator.save_model, original)
        self.assertEqual(self.model.metadata, {})
        self.assertFalse((Path(self.model.path) / 'preview.json').exists())

    def test_deployment_opt_out_does_not_remove_published_previews(
            self) -> None:
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            FakeEstimator().save_model(Path(self.model.path) / 'model.ubj')
            session.generate()
        content = (Path(self.model.path) / 'preview.json').read_bytes()
        descriptor = dict(self.model.metadata['_kfp_preview'])
        self.policy.write_text('false')
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            session.generate()
        self.assertEqual((Path(self.model.path) / 'preview.json').read_bytes(),
                         content)
        self.assertEqual(self.model.metadata['_kfp_preview'], descriptor)

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
            'version': 1,
            'path': 'preview.json',
        })
        self.assertEqual((Path(self.model.path) / 'model.ubj').read_bytes(),
                         b'native-model')
        self.assertFalse(list(Path(self.model.path).glob('*.tmp')))

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
        self.assertIn('No supported XGBoost save_model call',
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
        self.assertFalse((Path(self.model.path) / 'preview.json').exists())

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
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            with mock.patch.object(
                    FakeEstimator,
                    'get_params',
                    side_effect=ValueError('bad config')):
                with self.assertLogs(previews.__name__, level='WARNING'):
                    result = FakeEstimator().save_model(
                        Path(self.model.path) / 'model.ubj')
            self.assertEqual(result, 'saved')
            self.assertTrue((Path(self.model.path) / 'model.ubj').exists())
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
        self.assertTrue((Path(other.path) / 'preview.json').exists())

    def test_manual_previews_are_not_overwritten(self) -> None:
        self.model.metadata['_kfp_preview'] = {
            'version': 1,
            'path': 'preview.json'
        }
        path = Path(self.model.path) / 'preview.json'
        path.write_text('existing preview')
        with previews.ArtifactPreviewSession({'model': self.model}) as session:
            FakeEstimator().save_model(Path(self.model.path) / 'model.ubj')
            session.generate()
        self.assertEqual(path.read_text(), 'existing preview')

    def test_unadvertised_existing_file_is_not_overwritten(self) -> None:
        path = Path(self.model.path) / 'preview.json'
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

    def test_oversized_preview_does_not_publish_a_descriptor_or_partial_file(
            self) -> None:
        with self.assertRaisesRegex(ValueError, 'exceeds 64 KiB'):
            previews.ArtifactPreviewSession._publish(
                Path(self.model.path), self.model,
                {'value': 'x' * previews.MAX_BYTES})
        self.assertFalse((Path(self.model.path) / 'preview.json').exists())
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
                'version': 1,
                'path': 'preview.json',
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
                        '_parquet_preview',
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
            (['--artifact_previews'], True, True),
            (['--disable_artifact_previews'], False, True),
            (['--artifact_preview_schema_only'], True, False),
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
        self.assertFalse((Path(self.model.path) / 'preview.json').exists())
