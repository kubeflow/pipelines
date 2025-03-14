# Copyright 2025 The Kubeflow Authors
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

from kfp import compiler
from kfp import dsl
from kfp.dsl import component


@component
def print_all_placeholders(
    job_name: str,
    job_resource_name: str,
    job_id: str,
    task_name: str,
    task_id: str,
):
    allPlaceholders = [job_name, job_resource_name, job_id, task_name, task_id]

    for placeholder in allPlaceholders:
        if "\{\{" in placeholder or placeholder == "":
            raise RuntimeError(
                "Expected the placeholder to be replaced with a value: " + placeholder
            )

    assert task_name == "print-all-placeholders"
    assert job_name.startswith("pipeline-with-placeholders ")
    assert job_resource_name.startswith("pipeline-with-placeholders-")

    output = ", ".join(allPlaceholders)
    print(output)


@dsl.pipeline(name="pipeline-with-placeholders")
def pipeline_with_placeholders():
    print_all_placeholders(
        job_name=dsl.PIPELINE_JOB_NAME_PLACEHOLDER,
        job_resource_name=dsl.PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER,
        job_id=dsl.PIPELINE_JOB_ID_PLACEHOLDER,
        task_name=dsl.PIPELINE_TASK_NAME_PLACEHOLDER,
        task_id=dsl.PIPELINE_TASK_ID_PLACEHOLDER,
    ).set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_placeholders,
        package_path=__file__.replace(".py", ".yaml"),
    )
