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
from kfp import dsl
from kfp import kubernetes
from kfp.dsl import OutputPath
import os

# Note: this sample will only work if this secret is pre-created before running this pipeline.
# Is is pre-created by default only in the Google Cloud distribution listed here:
# https://www.kubeflow.org/docs/started/installing-kubeflow/#packaged-distributions-of-kubeflow
# If you are using a different distribution, you'll need to pre-create the secret yourself, or
# use a different secret that you know will exist.
SECRET_NAME = "user-gcp-sa"

@dsl.component
def comp():
    import os
    username = os.getenv("USER_NAME", "")
    psw1 = os.getenv("PASSWORD_VAR1", "")
    psw2 = os.getenv("PASSWORD_VAR2", "")
    assert username == "user1"
    assert psw1 == "psw1"
    assert psw2 == "psw2"


@dsl.component
def generate_secret_name(some_output: OutputPath(str)):
    secret_name = "test-secret-3"
    with open(some_output, 'w') as f:
        f.write(secret_name)

# Secrets are referenced from samples/v2/pre-requisites/test-secrets.yaml
@dsl.pipeline
def pipeline_secret_env(secret_parm: str = "test-secret-1"):
    task = comp()
    kubernetes.use_secret_as_env(
        task,
        secret_name=secret_parm,
        secret_key_to_env={'username': 'USER_NAME'})

    kubernetes.use_secret_as_env(
        task,
        secret_name="test-secret-2",
        secret_key_to_env={'password': 'PASSWORD_VAR1'})

    task2 = generate_secret_name()
    kubernetes.use_secret_as_env(
        task,
        secret_name=task2.output,
        secret_key_to_env={'password': 'PASSWORD_VAR2'})

if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(pipeline_secret_env, 'pipeline_secret_env.yaml')
