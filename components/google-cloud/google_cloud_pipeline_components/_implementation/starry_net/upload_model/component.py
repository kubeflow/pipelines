# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Starry Net Upload Model Component."""

from kfp import dsl
from google_cloud_pipeline_components.types import artifact_types as google_artifact_types


@dsl.container_component
def model_upload(
        project: str,
        display_name: str,
        model: dsl.Output[google_artifact_types.VertexModel],
        gcp_resources: dsl.OutputPath(str),
        location: str = 'us-central1',
        description: str = '',
        unmanaged_container_model: dsl.Input[
            google_artifact_types.UnmanagedContainerModel] = None,
        encryption_spec_key_name: str = '',
        labels: dict = {},
        parent_model: dsl.Input[google_artifact_types.VertexModel] = None):
    return dsl.ContainerSpec(
        image='gcr.io/ml-pipeline/automl-tables-private:1.0.17',
        command=['python3', '-u', '-m', 'launcher'],
        args=[
            '--type', 'UploadModel', '--payload',
            dsl.ConcatPlaceholder([
                '{', '"display_name": "', display_name, '"',
                ', "description": "', description, '"',
                ', "encryption_spec": {"kms_key_name":"',
                encryption_spec_key_name, '"}', ', "labels": ', labels, '}'
            ]), '--project', project, '--location', location, '--gcp_resources',
            gcp_resources, '--executor_input', '{{$}}',
            dsl.IfPresentPlaceholder(
                input_name='parent_model',
                then=[
                    '--parent_model_name',
                    "{{$.inputs.artifacts['parent_model'].metadata['resourceName']}}"
                ])
        ],
    )


upload_model = model_upload
