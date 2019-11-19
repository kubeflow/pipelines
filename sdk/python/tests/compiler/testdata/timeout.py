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


import kfp.dsl as dsl


class RandomFailure1Op(dsl.ContainerOp):
  """A component that fails randomly."""

  def __init__(self, exit_codes):
    super(RandomFailure1Op, self).__init__(
      name='random_failure',
      image='python:alpine3.6',
      command=['python', '-c'],
      arguments=["import random; import sys; exit_code = random.choice([%s]); print(exit_code); sys.exit(exit_code)" % exit_codes])


@dsl.pipeline(
  name='pipeline includes two steps which fail randomly.',
  description='shows how to use ContainerOp set_retry().'
)
def retry_sample_pipeline():
  op1 = RandomFailure1Op('0,1,2,3').set_timeout(10)
  op2 = RandomFailure1Op('0,1')
  dsl.get_pipeline_conf().set_timeout(50)


if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(retry_sample_pipeline, __file__ + '.tar.gz')
