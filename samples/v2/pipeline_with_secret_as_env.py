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
"""A pipeline that passes a secret as an env variable to a container."""
import os
import sys
from kfp import dsl
from kfp import kubernetes


@dsl.component
def comp():
    if os.environ['SECRET_VAR'] == "service_account":
        print("Success")
        return 0
    else:
        print(os.environ['SECRET_VAR']  + " is not service_account")
        sys.exit("Failure: cannot access secret as env variable")



@dsl.pipeline
def pipeline_secret_env():
    task = comp()
    kubernetes.use_secret_as_env(
        task,
        secret_name='user-gcp-sa',
        secret_key_to_env={'type': 'SECRET_VAR'})


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(my_pipeline, 'pipeline_secret_env.yaml')