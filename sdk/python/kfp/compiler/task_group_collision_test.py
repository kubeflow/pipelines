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
"""Regression tests for task and control-flow group name collisions
(#15755)."""

import tempfile
import unittest
from typing import Any, Callable, Dict

import yaml

from kfp import compiler, dsl


@dsl.component
def condition() -> int:
    return 1


@dsl.component
def condition_branches_1() -> None:
    print('plain task')


@dsl.component
def for_loop_2() -> None:
    print('plain task')


@dsl.component
def exit_handler_1() -> None:
    print('plain task')


@dsl.component
def work(msg: str) -> None:
    print(msg)


class TestTaskAndControlFlowGroupCollision(unittest.TestCase):
    def _compile_to_dict(
        self, pipeline_func: Callable[..., Any]
    ) -> Dict[str, Any]:
        with tempfile.TemporaryDirectory() as tmpdir:
            out_path = f'{tmpdir}/pipeline.yaml'
            compiler.Compiler().compile(pipeline_func, out_path)
            with open(out_path, 'r', encoding='utf-8') as fh:
                return yaml.safe_load(fh)

    def test_task_named_condition_does_not_overwrite_if_group(self) -> None:
        @dsl.pipeline
        def pipe() -> None:
            c = condition()
            with dsl.If(c.output == 1):
                work(msg='branch-one')
            with dsl.If(c.output == 2):
                work(msg='branch-two')
            condition()

        spec = self._compile_to_dict(pipe)
        root_tasks = spec['root']['dag']['tasks']
        self.assertEqual(len(root_tasks), 4)
        self.assertIn('condition-1', root_tasks)
        self.assertIn('condition-2', root_tasks)
        self.assertIn('triggerPolicy', root_tasks['condition-1'])
        self.assertIn('triggerPolicy', root_tasks['condition-2'])

    def test_colliding_task_feeding_if_condition_does_not_raise_index_error(
        self,
    ) -> None:
        @dsl.pipeline
        def pipe() -> None:
            c = condition()
            with dsl.If(c.output == 1):
                work(msg='x')
            c2 = condition()
            with dsl.If(c2.output == 1):
                work(msg='y')

        spec = self._compile_to_dict(pipe)
        root_tasks = spec['root']['dag']['tasks']
        self.assertEqual(len(root_tasks), 4)

    def test_task_named_like_parallel_for_does_not_overwrite_loop_group(
        self,
    ) -> None:
        @dsl.pipeline
        def pipe() -> None:
            with dsl.ParallelFor(['a', 'b']) as item:
                work(msg=item)
            for_loop_2()

        spec = self._compile_to_dict(pipe)
        root_tasks = spec['root']['dag']['tasks']
        self.assertEqual(len(root_tasks), 2)
        loop_entries = [
            t for t in root_tasks.values() if 'parameterIterator' in t
        ]
        self.assertEqual(len(loop_entries), 1)

    def test_task_named_like_exit_handler_does_not_overwrite_group(
        self,
    ) -> None:
        @dsl.pipeline
        def pipe() -> None:
            cleanup = work(msg='cleanup')
            with dsl.ExitHandler(cleanup):
                work(msg='body')
            exit_handler_1()

        spec = self._compile_to_dict(pipe)
        root_tasks = spec['root']['dag']['tasks']
        self.assertEqual(len(root_tasks), 3)

    def test_task_named_like_condition_branches_before_if_else(self) -> None:
        @dsl.pipeline
        def pipe() -> None:
            condition_branches_1()
            c = condition()
            with dsl.If(c.output == 1):
                work(msg='then')
            with dsl.Else():
                work(msg='else')

        spec = self._compile_to_dict(pipe)
        root_tasks = spec['root']['dag']['tasks']
        self.assertEqual(len(root_tasks), 3)
        self.assertIn('condition-branches-1', root_tasks)

    def test_task_named_like_condition_branches_after_if_else(self) -> None:
        @dsl.pipeline
        def pipe() -> None:
            c = condition()
            with dsl.If(c.output == 1):
                work(msg='then')
            with dsl.Else():
                work(msg='else')
            condition_branches_1()

        spec = self._compile_to_dict(pipe)
        root_tasks = spec['root']['dag']['tasks']
        self.assertEqual(len(root_tasks), 3)
        self.assertIn('condition-branches-1', root_tasks)
        self.assertIn('condition-branches-1-2', root_tasks)


if __name__ == '__main__':
    unittest.main()
