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
"""Pipeline preview policy survives composition without changing native
commands."""

from pathlib import Path
import tempfile
from typing import Any, List
import unittest

from kfp import compiler
from kfp import components
from kfp import dsl
from kfp.dsl import constants
from kfp.dsl import graph_component
import yaml


@dsl.component(base_image='python:3.11', install_kfp_package=False)
def producer(dataset: dsl.Output[dsl.Dataset]) -> None:
    """Produce a native directory artifact for preview policy tests."""
    from pathlib import Path

    directory = Path(dataset.path)
    directory.mkdir(parents=True, exist_ok=True)
    (directory / 'data.csv').write_text('value\n1\n')


class ArtifactPreviewPipelineTest(unittest.TestCase):
    """A disabled enclosing pipeline must disable every producer beneath it."""

    def setUp(self) -> None:
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.package = Path(directory.name) / 'pipeline.yaml'

    def compile(self,
                pipeline: graph_component.GraphComponent) -> dict[str, Any]:
        """Inspect the actual serialized IR consumed by remote and local
        runners."""
        compiler.Compiler().compile(pipeline, str(self.package))
        return yaml.safe_load(self.package.read_text())

    def containers(self, specification: dict[str, Any]) -> list[dict[str, Any]]:
        return [
            executor['container']
            for executor in specification['deploymentSpec']
            ['executors'].values()
            if 'container' in executor
        ]

    def disabled(self, container: dict[str, Any]) -> bool:
        return any(item['name'] == constants.ARTIFACT_PREVIEWS_ENV_VAR and
                   item['value'] == 'false'
                   for item in container.get('env', []))

    def test_default_and_explicit_enablement_preserve_the_original_ir(
            self) -> None:

        def definition() -> None:
            producer()

        implicit = self.compile(dsl.pipeline(definition))
        explicit = self.compile(
            dsl.pipeline(
                definition,
                pipeline_config=dsl.PipelineConfig(artifact_previews=True)))
        self.assertEqual(implicit, explicit)
        self.assertTrue(dsl.PipelineConfig().artifact_previews)
        self.assertNotIn('env', self.containers(implicit)[0])

    def test_pipeline_and_component_opt_outs_combine(self) -> None:
        for pipeline_enabled in (False, True):
            for component_enabled in (False, True):
                for containerized in (False, True):
                    with self.subTest(
                            pipeline=pipeline_enabled,
                            component=component_enabled,
                            containerized=containerized):
                        component = dsl.component(
                            producer.python_func,
                            base_image='python:3.11',
                            target_image='producer:test'
                            if containerized else None,
                            install_kfp_package=False,
                            artifact_previews=component_enabled)

                        @dsl.pipeline(
                            pipeline_config=dsl.PipelineConfig(
                                artifact_previews=pipeline_enabled))
                        def pipeline() -> None:
                            component()

                        container = self.containers(self.compile(pipeline))[0]
                        self.assertEqual(
                            self.disabled(container), not pipeline_enabled)
                        self.assertEqual(
                            '--disable_artifact_previews' in container['args'],
                            not component_enabled)

    def test_nested_and_loaded_pipelines_keep_the_most_restrictive_setting(
            self) -> None:

        @dsl.pipeline(
            pipeline_config=dsl.PipelineConfig(artifact_previews=False))
        def private_pipeline() -> None:
            producer()

        @dsl.pipeline
        def default_pipeline() -> None:
            producer()

        original = private_pipeline.pipeline_spec.SerializeToString()
        self.compile(private_pipeline)
        loaded = components.load_component_from_file(str(self.package))
        for enabled in (False, True):
            with self.subTest(enabled=enabled):

                @dsl.pipeline(
                    pipeline_config=dsl.PipelineConfig(
                        artifact_previews=enabled))
                def enclosing() -> None:
                    private_pipeline()
                    loaded()
                    default_pipeline()
                    producer()

                containers = self.containers(self.compile(enclosing))
                self.assertEqual(len(containers), 4)
                self.assertEqual(
                    sum(self.disabled(item) for item in containers),
                    2 if enabled else 4)
                self.assertEqual(
                    private_pipeline.pipeline_spec.SerializeToString(),
                    original)
                self.assertFalse(
                    self.disabled(
                        self.containers(self.compile(default_pipeline))[0]))

    def test_control_flow_exit_handlers_and_importers(self) -> None:

        @dsl.pipeline(
            pipeline_config=dsl.PipelineConfig(artifact_previews=False))
        def pipeline(run: bool = True) -> None:
            with dsl.ExitHandler(producer()):
                with dsl.ParallelFor([1, 2]):
                    producer()
                with dsl.If(run == True):
                    producer()
            dsl.importer(
                artifact_uri='gs://bucket/dataset', artifact_class=dsl.Dataset)

        specification = self.compile(pipeline)
        containers = self.containers(specification)
        self.assertEqual(len(containers), 3)
        self.assertTrue(all(self.disabled(item) for item in containers))
        importer = next(
            item
            for item in specification['deploymentSpec']['executors'].values()
            if 'importer' in item)
        self.assertNotIn('container', importer)

    def test_collected_artifacts_keep_the_producer_opt_out(self) -> None:

        @dsl.pipeline(
            pipeline_config=dsl.PipelineConfig(artifact_previews=False))
        def pipeline() -> List[dsl.Dataset]:
            with dsl.ParallelFor([1, 2]):
                task = producer()
            return dsl.Collected(task.output)

        specification = self.compile(pipeline)
        self.assertTrue(specification['root']['outputDefinitions']['artifacts']
                        ['Output']['isArtifactList'])
        self.assertTrue(
            all(self.disabled(item) for item in self.containers(specification)))

    def test_native_commands_and_unrelated_environment_are_preserved(
            self) -> None:

        @dsl.container_component
        def native() -> dsl.ContainerSpec:
            return dsl.ContainerSpec(
                image='busybox', command=['sh', '-c'], args=['echo native'])

        @dsl.pipeline(
            pipeline_config=dsl.PipelineConfig(artifact_previews=False))
        def pipeline() -> None:
            task = native()
            task.set_env_variable('UNRELATED', 'retained')
            task.set_env_variable(constants.ARTIFACT_PREVIEWS_ENV_VAR, 'true')

        container = self.containers(self.compile(pipeline))[0]
        self.assertEqual(container['command'], ['sh', '-c'])
        self.assertEqual(container['args'], ['echo native'])
        self.assertEqual(container['env'], [
            {
                'name': 'UNRELATED',
                'value': 'retained'
            },
            {
                'name': constants.ARTIFACT_PREVIEWS_ENV_VAR,
                'value': 'false'
            },
        ])
        self.assertIsNone(native.component_spec.implementation.container.env)

    def test_pipeline_config_rejects_non_boolean_settings(self) -> None:
        for value in (None, 'false', 0, 1, [], {}):
            with self.subTest(value=value):
                with self.assertRaisesRegex(TypeError,
                                            'artifact_previews must be a bool'):
                    dsl.PipelineConfig(artifact_previews=value)
