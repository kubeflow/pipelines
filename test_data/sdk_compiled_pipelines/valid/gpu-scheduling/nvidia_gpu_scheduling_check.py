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

"""Schedule-only NVIDIA GPU check for Kind + Fake GPU Operator (fake backend).

Requests nvidia.com/gpu and a NoSchedule toleration. Succeeds when the task
schedules and runs; does not assert torch.cuda / CUDA compute.
"""

from kfp import compiler, dsl, kubernetes
from kfp.dsl import PipelineTask


def add_gpu_toleration(task: PipelineTask, accelerator_type: str,
                       accelerator_limit: int):
    task.set_accelerator_type(accelerator=accelerator_type)
    task.set_accelerator_limit(accelerator_limit)
    kubernetes.add_toleration(
        task,
        key=accelerator_type,
        operator="Exists",
        effect="NoSchedule",
    )


@dsl.container_component
def verify_gpu_scheduled():
    """Lightweight success marker once the GPU-requesting pod runs."""
    return dsl.ContainerSpec(
        image='python:3.11',
        command=['sh', '-c'],
        args=['echo "GPU scheduling check: PASS"'],
    )


@dsl.pipeline(
    name="nvidia-gpu-scheduling-check",
    description=(
        "Verifies a task requesting nvidia.com/gpu with a matching toleration "
        "can schedule and complete (no CUDA/torch assertions)."
    ),
)
def nvidia_gpu_scheduling_check():
    task = verify_gpu_scheduled().set_caching_options(False)
    add_gpu_toleration(task, "nvidia.com/gpu", 1)


if __name__ == "__main__":
    compiler.Compiler().compile(
        nvidia_gpu_scheduling_check,
        package_path=__file__.replace(".py", ".yaml"),
    )
