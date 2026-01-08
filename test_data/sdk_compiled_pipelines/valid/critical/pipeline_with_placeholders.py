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
        create_time: str,
        schedule_time: str,
):
    allPlaceholders = [job_name, job_resource_name, job_id, task_name, task_id, create_time, schedule_time]

    for placeholder in allPlaceholders:
        if "{{" in placeholder or placeholder == "":
            raise RuntimeError(
                "Expected the placeholder to be replaced with a value: " + placeholder
            )

    assert task_name == "print-all-placeholders"
    assert job_resource_name.startswith("pipeline-with-placeholders-")
    
    # Verify timestamp format (RFC3339/ISO 8601 format)
    import re
    timestamp_pattern = r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
    assert re.match(timestamp_pattern, create_time), f"create_time should be in RFC3339 format, got: {create_time}"
    assert re.match(timestamp_pattern, schedule_time), f"schedule_time should be in RFC3339 format, got: {schedule_time}"

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
        create_time=dsl.PIPELINE_JOB_CREATE_TIME_UTC_PLACEHOLDER,
        schedule_time=dsl.PIPELINE_JOB_SCHEDULE_TIME_UTC_PLACEHOLDER,
    ).set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_placeholders,
        package_path=__file__.replace(".py", ".yaml"),
    )