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

from absl.testing import parameterized
from kfp.components import for_loop
from kfp.components import pipeline_context
from kfp.components import tasks_group


class ParallelForTest(parameterized.TestCase):

    def test_basic(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with pipeline_context.Pipeline('pipeline') as p:
            with tasks_group.ParallelFor(items=loop_items) as parallel_for:
                loop_argument = for_loop.LoopArgument.from_raw_items(
                    loop_items, '1')
                self.assertEqual(parallel_for.group_type, 'for-loop')
                self.assertEqual(parallel_for.parallelism, 0)
                self.assertEqual(parallel_for.loop_argument, loop_argument)

    def test_parallelfor_valid_parallelism(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with pipeline_context.Pipeline('pipeline') as p:
            with tasks_group.ParallelFor(
                    items=loop_items, parallelism=3) as parallel_for:
                loop_argument = for_loop.LoopArgument.from_raw_items(
                    loop_items, '1')
                self.assertEqual(parallel_for.group_type, 'for-loop')
                self.assertEqual(parallel_for.parallelism, 3)
                self.assertEqual(parallel_for.loop_argument, loop_argument)

    def test_parallelfor_zero_parallelism(self):
        loop_items = ['pizza', 'hotdog', 'pasta']
        with pipeline_context.Pipeline('pipeline') as p:
            with tasks_group.ParallelFor(
                    items=loop_items, parallelism=0) as parallel_for:
                loop_argument = for_loop.LoopArgument.from_raw_items(
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
