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

# pylint: disable=import-outside-toplevel,unused-import,import-error
import json
import os
import sys
import tempfile
import unittest


class V2NamespaceAliasTest(unittest.TestCase):
    """Test that imports of both modules and objects are aliased
    (e.g. all import path variants work).
    """

    @classmethod
    def setUp(cls):
        # need to unload the kfp.v2 module before each test to ensure that the Deprecation warning is raised each time
        keys = {k for k, _ in sys.modules.items() if k.startswith("kfp")}
        for key in keys:
            del sys.modules[key]

    def test_import_namespace(self):  # pylint: disable=no-self-use
        with self.assertWarns(DeprecationWarning):
            from kfp import v2

        @v2.dsl.component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        @v2.dsl.pipeline(
            name='hello-world', description='A simple intro pipeline')
        def pipeline_hello_world(text: str = 'hi there'):
            """Hello world pipeline."""

            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            # you can e.g. create a file here:
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.json')
            v2.compiler.Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, "r") as f:
                json.load(f)

    def test_import_modules(self):  # pylint: disable=no-self-use
        with self.assertWarns(DeprecationWarning):
            from kfp.v2 import compiler
            from kfp.v2 import dsl

        @dsl.component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        @dsl.pipeline(name='hello-world', description='A simple intro pipeline')
        def pipeline_hello_world(text: str = 'hi there'):
            """Hello world pipeline."""

            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            # you can e.g. create a file here:
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.json')
            compiler.Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, "r") as f:
                json.load(f)

    def test_import_object(self):  # pylint: disable=no-self-use
        with self.assertWarns(DeprecationWarning):
            from kfp.v2.compiler import Compiler
            from kfp.v2.dsl import component
            from kfp.v2.dsl import pipeline

        @component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        @pipeline(name='hello-world', description='A simple intro pipeline')
        def pipeline_hello_world(text: str = 'hi there'):
            """Hello world pipeline."""

            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            # you can e.g. create a file here:
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.json')
            Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, "r") as f:
                json.load(f)


if __name__ == '__main__':
    unittest.main()
