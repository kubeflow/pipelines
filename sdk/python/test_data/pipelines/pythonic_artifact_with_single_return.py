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

from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Model


@dsl.component(packages_to_install=['dill==0.3.7'])
def make_language_model(text_dataset: Dataset) -> Model:
    # dill allows pickling objects belonging to a function's local namespace
    import dill

    with open(text_dataset.path) as f:
        text = f.read()

    # insert train on text here #

    def dummy_model(x: str) -> str:
        return x

    model = Model(
        uri=dsl.get_uri(suffix='model'),
        metadata={'data': text_dataset.name},
    )

    with open(model.path, 'wb') as f:
        dill.dump(dummy_model, f)

    return model


@dsl.pipeline
def make_language_model_pipeline() -> Model:
    importer = dsl.importer(
        artifact_uri='gs://ml-pipeline-playground/shakespeare1.txt',
        artifact_class=Dataset,
        reimport=False,
        metadata={'key': 'value'})
    return make_language_model(text_dataset=importer.output).output


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=make_language_model_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
