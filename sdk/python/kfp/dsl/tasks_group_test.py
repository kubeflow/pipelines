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

import unittest

from kfp import dsl
from kfp.dsl import for_loop
from kfp.dsl import pipeline_context
from kfp.dsl import tasks_group


class ParallelForTest(unittest.TestCase):

    def test_basic(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with pipeline_context.Pipeline('pipeline') as p:
            with tasks_group.ParallelFor(items=loop_items) as parallel_for:
                loop_argument = for_loop.LoopParameterArgument.from_raw_items(
                    loop_items, '1')
                self.assertEqual(parallel_for.group_type, 'for-loop')
                self.assertEqual(parallel_for.parallelism, 0)
                self.assertEqual(parallel_for.loop_argument, loop_argument)

    def test_parallelfor_valid_parallelism(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with pipeline_context.Pipeline('pipeline') as p:
            with tasks_group.ParallelFor(
                    items=loop_items, parallelism=3) as parallel_for:
                loop_argument = for_loop.LoopParameterArgument.from_raw_items(
                    loop_items, '1')
                self.assertEqual(parallel_for.group_type, 'for-loop')
                self.assertEqual(parallel_for.parallelism, 3)
                self.assertEqual(parallel_for.loop_argument, loop_argument)

    def test_parallelfor_zero_parallelism(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with pipeline_context.Pipeline('pipeline') as p:
            with tasks_group.ParallelFor(
                    items=loop_items, parallelism=0) as parallel_for:
                loop_argument = for_loop.LoopParameterArgument.from_raw_items(
                    loop_items, '1')
                self.assertEqual(parallel_for.group_type, 'for-loop')
                self.assertEqual(parallel_for.parallelism, 0)
                self.assertEqual(parallel_for.loop_argument, loop_argument)

    def test_parallelfor_invalid_parallelism(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with self.assertRaisesRegex(ValueError,
                                    'ParallelFor parallelism must be >= 0.'):
            with pipeline_context.Pipeline('pipeline') as p:
                tasks_group.ParallelFor(items=loop_items, parallelism=-1)


class TestConditionDeprecated(unittest.TestCase):

    def test(self):

        @dsl.component
        def foo() -> str:
            return 'foo'

        @dsl.pipeline
        def my_pipeline(string: str):
            with self.assertWarnsRegex(
                    DeprecationWarning,
                    r'dsl\.Condition is deprecated\. Please use dsl\.If instead\.'
            ):
                with dsl.Condition(string == 'text'):
                    foo()
