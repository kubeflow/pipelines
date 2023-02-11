
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
"""Setup configuration of  Google Cloud Pipeline Components client side libraries."""


def make_required_install_packages():
    return [
        # Pin google-api-core version for the bug fixing in 1.31.5
        # https://github.com/googleapis/python-api-core/releases/tag/v1.31.5
        "google-api-core>=1.31.5,<3.0.0dev,!=2.0.*,!=2.1.*,!=2.2.*,!=2.3.0",
        "google-cloud-storage<3,>=2.2.1",
        "kfp>=2.0.0b10",
        "google-cloud-aiplatform>=1.14.0,<2",
        "google-cloud-notebooks>=0.4.0",
    ]


def make_required_test_packages():
    return make_required_install_packages() + [
        "mock>=4.0.0",
        "flake8>=3.0.0",
        "pytest>=6.0.0",
    ]


def make_dependency_links():
    return []
