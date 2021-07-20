# Copyright 2021 The Kubeflow Authors
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

from kfp.v2 import dsl
from kfp.v2 import compiler
from kfp.v2 import components


@components.create_component_from_func
def hello_world(text: str):
    print(text)
    return text


@dsl.pipeline(name='hello-world', description='A simple intro pipeline')
def pipeline_parameter_to_consumer(text: str = 'hi there'):
    '''Pipeline that passes small pipeline parameter string to consumer op'''

    consume_task = hello_world(
        text
    )  # Passing pipeline parameter as argument to consumer op


if __name__ == "__main__":
    # execute only if run as a script
    compiler.Compiler().compile(
        pipeline_func=pipeline_parameter_to_consumer,
        package_path='hello_world_pipeline.json'
    )
