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


import kfp.dsl as dsl


class FlipCoinOp(dsl.ContainerOp):

    def __init__(self, name):
        super(FlipCoinOp, self).__init__(
            name=name,
            image='python:alpine3.6',
            command=['sh', '-c'],
            arguments=[
                'python -c "import random; result = \'heads\' if random.randint(0,1) == 0 '
                'else \'tails\'; print(result)" | tee /tmp/output'],
            file_outputs={'output': '/tmp/output'})


class PrintOp(dsl.ContainerOp):

    def __init__(self, name, msg):
        super(PrintOp, self).__init__(
            name=name,
            image='alpine:3.6',
            command=['echo', msg])


@dsl.pipeline(
    name='pipeline flip coin',
    description='shows how to use dsl.Condition.'
)
def flipcoin():
    flip0 = FlipCoinOp('flip0')
    flip1 = FlipCoinOp('flip1')
    flip2 = FlipCoinOp('flip2')

    with dsl.Condition((flip0.output == 'tails') & (flip1.output == 'tails')):
        PrintOp('print-bad', 'Bad luck')

    with dsl.Condition((flip0.output == 'heads') & (flip1.output == 'heads') & (flip2.output == 'heads')):
        PrintOp('print-good', 'Good luck!')
