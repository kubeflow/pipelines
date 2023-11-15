# Copyright 2023 The Kubeflow Authors
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
"""Tests for task_dispatcher.py."""
import contextlib
import os
import unittest

from absl.testing import parameterized
from kfp import dsl
from kfp import local
from kfp.dsl import Artifact
from kfp.local import task_dispatcher
from kfp.local import testing_utilities


class TestRunSingleComponent(testing_utilities.LocalRunnerEnvironmentTestCase):

    def test_initialized(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(string: str) -> str:
            return string

        # capture + discard stdout
        with open(os.devnull, 'w') as dev_null:
            with contextlib.redirect_stdout(dev_null):

                task_dispatcher.run_single_component(
                    identity.pipeline_spec,
                    arguments={'string': 'foo'},
                )

    def test_local_pipeline_unsupported_two_tasks(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.pipeline
        def my_pipeline():
            identity(string='foo')
            identity(string='bar')

        # compile and load into a YamlComponent to ensure the NotImplementedError isn't simply being thrown because this is a GraphComponent
        my_pipeline = testing_utilities.compile_and_load_component(my_pipeline)
        with self.assertRaisesRegex(
                NotImplementedError,
                'Local pipeline execution is not currently supported\.',
        ):
            my_pipeline()

    def test_local_pipeline_unsupported_one_task_different_interface(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.pipeline
        def my_pipeline():
            identity(string='foo')

        # compile and load into a YamlComponent to ensure the NotImplementedError isn't simply being thrown because this is a GraphComponent
        my_pipeline = testing_utilities.compile_and_load_component(my_pipeline)
        with self.assertRaisesRegex(
                NotImplementedError,
                'Local pipeline execution is not currently supported\.',
        ):
            my_pipeline()

    def test_local_pipeline_unsupported_if_is_graph_component(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(string: str) -> str:
            return string

        # even if there is one task with the same interface as the pipeline, the code should catch that the pipeline is a GraphComponent and throw the NotImplementedError
        @dsl.pipeline
        def my_pipeline(string: str) -> str:
            return identity(string=string).output

        with self.assertRaisesRegex(
                NotImplementedError,
                'Local pipeline execution is not currently supported\.',
        ):
            my_pipeline(string='foo')

    def test_not_initialized(self):

        @dsl.component
        def identity(string: str) -> str:
            return string

        with self.assertRaisesRegex(
                RuntimeError,
                r'You must initiatize the local execution session using kfp\.local\.init\(\)\.'
        ):
            task_dispatcher.run_single_component(
                identity.pipeline_spec,
                arguments={'string': 'foo'},
            )


# test on two runner types to ensure this validation logic is runner-independent
@parameterized.parameters([
    (local.SubprocessRunner(use_venv=False),),
    (local.SubprocessRunner(use_venv=True),),
])
class TestInvalidArguments(testing_utilities.LocalRunnerEnvironmentTestCase):

    def test_no_argument_no_default(self, runner):
        local.init(runner=runner, cleanup=False)

        @dsl.component
        def identity(x: str) -> str:
            return x

        with self.assertRaisesRegex(
                TypeError, r'identity\(\) missing 1 required argument: x\.'):
            task = identity()

    def test_default_wrong_type(self, runner):
        local.init(runner=runner, cleanup=False, raise_on_error=True)

        @dsl.component
        def identity(x: str) -> str:
            return x

        with self.assertRaisesRegex(
                dsl.types.type_utils.InconsistentTypeException,
                r"Incompatible argument passed to the input 'x' of component 'comp-identity': Argument type 'NUMBER_INTEGER' is incompatible with the input type 'STRING'"
        ):
            identity(x=1)

    def test_extra_argument(self, runner):
        local.init(runner=runner, cleanup=False)

        @dsl.component
        def identity(x: str) -> str:
            return x

        with self.assertRaisesRegex(
                TypeError,
                r'identity\(\) got an unexpected keyword argument "y"\.'):
            identity(x='foo', y='bar')

    def test_input_artifact_provided(self, runner):
        local.init(runner=runner, cleanup=False)

        @dsl.component
        def identity(a: Artifact) -> Artifact:
            return a

        with self.assertRaisesRegex(
                ValueError,
                r'Input artifacts are not yet supported for local execution.'):
            identity(a=Artifact(name='a', uri='gs://bucket/foo'))


if __name__ == '__main__':
    unittest.main()
