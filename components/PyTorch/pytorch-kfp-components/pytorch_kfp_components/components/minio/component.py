#!/usr/bin/env/python3
# Copyright (c) Facebook, Inc. and its affiliates.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Minio Component Module."""
from pytorch_kfp_components.components.base.base_component import BaseComponent
from pytorch_kfp_components.components.minio.executor import Executor
from pytorch_kfp_components.types import standard_component_specs


class MinIO(BaseComponent):  #pylint: disable=too-few-public-methods
    """Minio Component Class."""
    def __init__(
        self,
        source: str,
        bucket_name: str,
        destination: str,
        endpoint: str,
    ):
        """Initializes the component class.

        Args:
            source : the source path of artifacts.
            bucket_name : minio bucket name.
            destination : the destination path in minio
            endpoint : minio endpoint url.
        """
        super(BaseComponent, self).__init__()  #pylint: disable=bad-super-call

        input_dict = {
            standard_component_specs.MINIO_SOURCE: source,
            standard_component_specs.MINIO_BUCKET_NAME: bucket_name,
            standard_component_specs.MINIO_DESTINATION: destination,
        }

        output_dict = {}

        exec_properties = {
            standard_component_specs.MINIO_ENDPOINT: endpoint,
        }

        spec = standard_component_specs.MinIoSpec()
        self._validate_spec(
            spec=spec,
            input_dict=input_dict,
            output_dict=output_dict,
            exec_properties=exec_properties,
        )

        Executor().Do(
            input_dict=input_dict,
            output_dict=output_dict,
            exec_properties=exec_properties,
        )

        self.output_dict = output_dict
