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

"""DRA JSON input check: verifies a runtime-resolved ResourceClaim
(passed as a pipeline parameter) schedules via the dra-example-driver."""

from typing import Any, Dict

from kfp import compiler, dsl, kubernetes


@dsl.container_component
def verify_dra_json():
    return dsl.ContainerSpec(
        image='python:3.11',
        command=['sh', '-c'],
        args=['echo "DRA JSON input check: PASS"'],
    )


@dsl.pipeline(
    name="dra-json-input-check",
    description=(
        "Verifies a task with a runtime-resolved DRA ResourceClaim "
        "(via pipeline parameter) can schedule and complete."
    ),
)
def dra_json_input_check(
    resource_claim: Dict[str, Any] = {
        "resourceClaimTemplateName": "dra-test-claim"
    },
):
    task = verify_dra_json().set_caching_options(False)
    kubernetes.add_resource_claim_json(
        task, resource_claim_json=resource_claim)


if __name__ == "__main__":
    compiler.Compiler().compile(
        dra_json_input_check,
        package_path=__file__.replace(".py", ".yaml"),
    )
