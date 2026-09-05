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

"""DRA static claim check: verifies a pod with a static ResourceClaim
schedules and runs via the dra-example-driver."""

from kfp import compiler, dsl, kubernetes


@dsl.container_component
def verify_dra_static():
    return dsl.ContainerSpec(
        image='python:3.11',
        command=['sh', '-c'],
        args=['echo "DRA static claim check: PASS"'],
    )


@dsl.pipeline(
    name="dra-static-claim-check",
    description=(
        "Verifies a task with a static DRA ResourceClaim can schedule "
        "and complete using the dra-example-driver."
    ),
)
def dra_static_claim_check():
    task = verify_dra_static().set_caching_options(False)
    kubernetes.add_resource_claim(
        task, resource_claim_template_name='dra-test-claim')
    kubernetes.add_resource_claim(
        task, resource_claim_template_name='dra-test-claim-secondary')


if __name__ == "__main__":
    compiler.Compiler().compile(
        dra_static_claim_check,
        package_path=__file__.replace(".py", ".yaml"),
    )
