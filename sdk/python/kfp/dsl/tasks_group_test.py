# Copyright 2022 The Kubeflow Authors
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

import os
import tempfile
import unittest

from kfp import compiler, dsl
from kfp.dsl import (
    Artifact,
    Input,
    Output,
    for_loop,
    pipeline_context,
    tasks_group,
)


class ParallelForTest(unittest.TestCase):
    def test_basic(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with pipeline_context.Pipeline('pipeline') as p:
            with tasks_group.ParallelFor(items=loop_items) as parallel_for:
                loop_argument = for_loop.LoopParameterArgument.from_raw_items(
                    loop_items, '1'
                )
                self.assertEqual(parallel_for.group_type, 'for-loop')
                self.assertEqual(parallel_for.parallelism, 0)
                self.assertEqual(parallel_for.loop_argument, loop_argument)

    def test_parallelfor_valid_parallelism(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with pipeline_context.Pipeline('pipeline') as p:
            with tasks_group.ParallelFor(
                items=loop_items, parallelism=3
            ) as parallel_for:
                loop_argument = for_loop.LoopParameterArgument.from_raw_items(
                    loop_items, '1'
                )
                self.assertEqual(parallel_for.group_type, 'for-loop')
                self.assertEqual(parallel_for.parallelism, 3)
                self.assertEqual(parallel_for.loop_argument, loop_argument)

    def test_parallelfor_zero_parallelism(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with pipeline_context.Pipeline('pipeline') as p:
            with tasks_group.ParallelFor(
                items=loop_items, parallelism=0
            ) as parallel_for:
                loop_argument = for_loop.LoopParameterArgument.from_raw_items(
                    loop_items, '1'
                )
                self.assertEqual(parallel_for.group_type, 'for-loop')
                self.assertEqual(parallel_for.parallelism, 0)
                self.assertEqual(parallel_for.loop_argument, loop_argument)

    def test_parallelfor_invalid_parallelism(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with self.assertRaisesRegex(
            ValueError, 'ParallelFor parallelism must be >= 0.'
        ):
            with pipeline_context.Pipeline('pipeline') as p:
                tasks_group.ParallelFor(items=loop_items, parallelism=-1)

    def test_parallelfor_empty_dict_items_raises(self):
        with self.assertRaisesRegex(
            ValueError, r'Dictionary items in ParallelFor cannot be empty\.'
        ):
            with pipeline_context.Pipeline('pipeline') as p:
                tasks_group.ParallelFor(items=[{}])


class TestConditionDeprecated(unittest.TestCase):
    def test(self):

        @dsl.component
        def foo() -> str:
            return 'foo'

        @dsl.pipeline
        def my_pipeline(string: str):
            with self.assertWarnsRegex(
                DeprecationWarning,
                r'dsl\.Condition is deprecated\. Please use dsl\.If instead\.',
            ):
                with dsl.Condition(string == 'text'):
                    foo()


class ExitHandlerTest(unittest.TestCase):
    def test_exit_task_with_upstream_artifact_raises(self):
        """Tests that an exit task cannot depend on an upstream artifact."""

        @dsl.component
        def producer(a: Output[Artifact]):
            pass

        @dsl.component
        def exit_comp(a: Input[Artifact]):
            pass

        with self.assertRaisesRegex(
            ValueError, r'exit_task cannot depend on any other tasks\.'
        ):

            @dsl.pipeline
            def my_pipeline():
                t1 = producer()
                exit_task = exit_comp(a=t1.outputs['a'])
                with dsl.ExitHandler(exit_task):
                    pass

    def test_exit_task_with_upstream_parameter_raises(self):
        """Tests that an exit task cannot depend on an upstream parameter."""

        @dsl.component
        def producer() -> str:
            return 'out'

        @dsl.component
        def exit_comp(param: str):
            pass

        with self.assertRaisesRegex(
            ValueError, r'exit_task cannot depend on any other tasks\.'
        ):

            @dsl.pipeline
            def my_pipeline():
                t1 = producer()
                exit_task = exit_comp(param=t1.output)
                with dsl.ExitHandler(exit_task):
                    pass

    def test_exit_task_with_pipeline_inputs_allowed(self):
        """Tests that an exit task can consume pipeline-level inputs."""

        @dsl.component
        def exit_comp(param: str, art: Input[Artifact]):
            pass

        @dsl.pipeline
        def my_pipeline(p: str, a: Input[Artifact]):
            exit_task = exit_comp(param=p, art=a)
            with dsl.ExitHandler(exit_task):
                pass

        with tempfile.TemporaryDirectory() as tmpdir:
            package_path = os.path.join(tmpdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path
            )
