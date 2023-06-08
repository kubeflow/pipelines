# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Experimental Google Cloud Pipeline Components."""
import warnings

_gcpc_package_name = 'google_cloud_pipeline_components'

warnings.warn(
    f'You are using a component from the {_gcpc_package_name!r} experimental'
    ' namespace, which contains components that may not be stable across'
    ' versions of the library. To ensure the stability of your pipelines, it'
    f' is recommended that you pin {_gcpc_package_name!r} to a specific patch'
    ' version and migrate the equivalent stable component from the v1'
    ' namespace when it becomes available.',
    stacklevel=2,
)
