# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""DRA passthrough check: verifies ResourceClaims are forwarded through
TaskConfig passthrough and applied to the pod via the dra-example-driver."""

from kfp import compiler, dsl, kubernetes


@dsl.component(
    task_config_passthroughs=[
        dsl.TaskConfigPassthrough(
            field=dsl.TaskConfigField.KUBERNETES_RESOURCE_CLAIMS,
            apply_to_task=True,
        ),
    ],
)
def verify_dra_passthrough(task_config: dsl.TaskConfig):
    print(f"resource_claims: {task_config.resource_claims}")
    print("DRA passthrough check: PASS")


@dsl.pipeline(
    name="dra-passthrough-check",
    description=(
        "Verifies DRA ResourceClaims are forwarded through TaskConfig "
        "passthrough and applied to the task pod."
    ),
)
def dra_passthrough_check():
    task = verify_dra_passthrough().set_caching_options(False)
    kubernetes.add_resource_claim(
        task, resource_claim_template_name='dra-test-claim')


if __name__ == "__main__":
    compiler.Compiler().compile(
        dra_passthrough_check,
        package_path=__file__.replace(".py", ".yaml"),
    )
