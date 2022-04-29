# Copyright 2021-2022 The Kubeflow Authors
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
from kfp.components import pipeline_context


class TestPipeline(unittest.TestCase):

    def test_is_pipeline_true_1(self):

        @dsl.pipeline(name='my-pipeline')
        def my_pipeline(a: str, b: int):
            pass

        self.assertTrue(pipeline_context.Pipeline.is_pipeline_func(my_pipeline))

    def test_is_pipeline_true2(self):

        @dsl.component
        def my_comp():
            pass

        @dsl.pipeline(name='my-pipeline')
        def my_pipeline(a: str, b: int):
            pass

        self.assertTrue(pipeline_context.Pipeline.is_pipeline_func(my_pipeline))

    def test_is_pipeline_false_1(self):

        @dsl.component
        def my_comp():
            pass

        self.assertFalse(pipeline_context.Pipeline.is_pipeline_func(my_comp))

    def test_is_pipeline_false_2(self):

        def my_pipeline(a: str, b: int):
            pass

        self.assertFalse(
            pipeline_context.Pipeline.is_pipeline_func(my_pipeline))


if __name__ == '__main__':
    unittest.main()
