
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
        # Explicity add google-api-core as a dependancy to avoid conflict
        # between kfp & aiplatform.
        "google-api-core<2dev,>=1.26.0",
        "kfp>=1.8.9,<2.0.0",
        "google-cloud-aiplatform>=1.4.3",
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
