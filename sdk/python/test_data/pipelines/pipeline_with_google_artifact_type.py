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
import sys
import tempfile

# NOTE: this is a compilation test only and is not executable, since dummy_third_party_package does not exist and cannot be installed or imported at runtime


def create_temporary_google_artifact_package(
        temp_dir: tempfile.TemporaryDirectory) -> None:
    """Creates a fake temporary module that can be used as a Vertex SDK mock
    for testing purposes."""
    import inspect
    import os
    import textwrap

    from kfp import dsl

    class VertexModel(dsl.Artifact):
        schema_title = 'google.VertexModel'
        schema_version = '0.0.0'

        def __init__(self, name: str, uri: str, metadata: dict) -> None:
            self.name = name
            self.uri = uri
            self.metadata = metadata

        @property
        def path(self) -> str:
            return self.uri.replace('gs://', '/')

    class VertexDataset(dsl.Artifact):
        schema_title = 'google.VertexDataset'
        schema_version = '0.0.0'

        def __init__(self, name: str, uri: str, metadata: dict) -> None:
            self.name = name
            self.uri = uri
            self.metadata = metadata

        @property
        def path(self) -> str:
            return self.uri.replace('gs://', '/')

    class_source = 'from kfp import dsl' + '\n\n' + textwrap.dedent(
        inspect.getsource(VertexModel)) + '\n\n' + textwrap.dedent(
            inspect.getsource(VertexDataset))

    with open(os.path.join(temp_dir.name, 'aiplatform.py'), 'w') as f:
        f.write(class_source)


# remove try finally when a third-party package adds pre-registered custom artifact types that we can use for testing
try:
    temp_dir = tempfile.TemporaryDirectory()
    sys.path.append(temp_dir.name)
    create_temporary_google_artifact_package(temp_dir)

    import aiplatform
    from aiplatform import VertexDataset
    from aiplatform import VertexModel
    from kfp import compiler
    from kfp import dsl
    from kfp.dsl import Input
    from kfp.dsl import Output

    PACKAGES_TO_INSTALL = ['aiplatform']

    @dsl.component(packages_to_install=PACKAGES_TO_INSTALL)
    def model_producer(model: Output[aiplatform.VertexModel]):

        assert isinstance(model, aiplatform.VertexModel), type(model)
        with open(model.path, 'w') as f:
            f.write('my model')

    @dsl.component(packages_to_install=PACKAGES_TO_INSTALL)
    def model_consumer(model: Input[VertexModel],
                       dataset: Input[VertexDataset]):
        print('Model')
        print('artifact.type: ', type(model))
        print('artifact.name: ', model.name)
        print('artifact.uri: ', model.uri)
        print('artifact.metadata: ', model.metadata)

        print('Dataset')
        print('artifact.type: ', type(dataset))
        print('artifact.name: ', dataset.name)
        print('artifact.uri: ', dataset.uri)
        print('artifact.metadata: ', dataset.metadata)

    @dsl.pipeline(name='pipeline-with-google-types')
    def my_pipeline():
        producer_task = model_producer()
        importer = dsl.importer(
            artifact_uri='gs://ml-pipeline-playground/shakespeare1.txt',
            artifact_class=VertexDataset,
            reimport=False,
            metadata={'key': 'value'})
        model_consumer(
            model=producer_task.outputs['model'],
            dataset=importer.output,
        )

    if __name__ == '__main__':
        ir_file = __file__.replace('.py', '.yaml')
        compiler.Compiler().compile(
            pipeline_func=my_pipeline, package_path=ir_file)
finally:
    sys.path.pop()
    temp_dir.cleanup()
