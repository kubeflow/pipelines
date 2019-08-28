#!/usr/bin/env python3
# Copyright 2018 Google LLC
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
from pathlib import Path

sys.path.insert(0, __file__ + '/../../../../')

import kfp
from kfp import dsl


def component_with_inline_input_artifact(text: str):
    return dsl.ContainerOp(
        name='component_with_inline_input_artifact',
        image='alpine',
        command=['cat', dsl.InputArgumentPath(text, path='/tmp/inputs/text/data', input='text')], # path and input are optional
    )


def component_with_input_artifact(text):
    '''A component that passes text as input artifact'''

    return dsl.ContainerOp(
        name='component_with_input_artifact',
        artifact_argument_paths=[
            dsl.InputArgumentPath(argument=text, path='/tmp/inputs/text/data', input='text'), # path and input are optional
        ],
        image='alpine',
        command=['cat', '/tmp/inputs/text/data'],
    )

def component_with_hardcoded_input_artifact_value():
    '''A component that passes hard-coded text as input artifact'''
    return component_with_input_artifact('hard-coded artifact value')


def component_with_input_artifact_value_from_file(file_path):
    '''A component that passes contents of a file as input artifact'''
    return component_with_input_artifact(Path(file_path).read_text())


@dsl.pipeline(
    name='Pipeline with artifact input raw argument value.',
    description='Pipeline shows how to define artifact inputs and pass raw artifacts to them.'
)
def input_artifact_pipeline():
    component_with_inline_input_artifact('Constant artifact value')
    component_with_input_artifact('Constant artifact value')
    component_with_hardcoded_input_artifact_value()

    file_path = str(Path(__file__).parent.joinpath('input_artifact_raw_value.txt'))
    component_with_input_artifact_value_from_file(file_path)

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(input_artifact_pipeline, __file__ + '.yaml')
