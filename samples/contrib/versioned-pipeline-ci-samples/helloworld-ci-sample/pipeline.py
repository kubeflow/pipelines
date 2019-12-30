#!/usr/bin/env python3
# Copyright 2019 Google LLC
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
from kfp.gcp import use_gcp_secret
import argparse

parser = argparse.ArgumentParser()
parser.add_argument('--commit_id', help='Commit Id', type=str)
args = parser.parse_args()


@dsl.pipeline(
    name='Mnist Sample',
    description='Normal sample to demonstrate how to use CI with KFP'
)
def helloworld_ci_pipeline(
    gcr_address: str
):
    import os
    train = dsl.ContainerOp(
        name='mnist train',
        image = os.path.join(gcr_address, 'mnist_train:', args.commit_id)
    ).apply(use_gcp_secret('user-gcp-sa'))


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(helloworld_ci_pipeline, __file__ + '.zip')
