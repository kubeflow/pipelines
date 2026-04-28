# Copyright 2026 The Kubeflow Authors
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
"""Pipeline IR source for v2 integration: missing Secret → CreateContainerConfigError.

Regenerate the checked-in YAML (KFP pipeline spec IR, not an Argo Workflow):

  cd backend/test/v2/resources
  kfp dsl compile --py missing_secret_env_e2e_pipeline.py --output missing-secret-env.yaml

Requires editable installs: kfp, kfp-kubernetes (see repo AGENTS.md).
The compiled YAML embeds the SDK version at compilation time (sdkVersion field).

Cluster E2E: RunAPITestSuite.TestRunAPIx_MissingSecretEnvReportsFailedTaskDetail in
backend/test/v2/integration (requires -runIntegrationTests=true).
"""
from kfp import dsl
from kfp import kubernetes

# Intentionally does not exist in the cluster; kubelet surfaces CreateContainerConfigError.
_MISSING_SECRET_NAME = 'kfp-e2e-missing-secret-not-found'


@dsl.component(base_image='public.ecr.aws/docker/library/python:3.12')
def missing_secret_task() -> str:
    import os
    return os.getenv('KFP_E2E_MISSING_SECRET_ENV_MARKER', '')


@dsl.pipeline(name='e2e-missing-secret-env')
def e2e_missing_secret_env_pipeline():
    task = missing_secret_task()
    kubernetes.use_secret_as_env(
        task,
        secret_name=_MISSING_SECRET_NAME,
        secret_key_to_env={'placeholder-key': 'KFP_E2E_MISSING_SECRET_ENV_MARKER'},
    )


if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        e2e_missing_secret_env_pipeline,
        package_path='missing-secret-env.yaml',
    )
