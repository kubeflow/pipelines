# Copyright 2021 The Kubeflow Authors
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
"""Definitions of utils methods."""

import re


def maybe_rename_for_k8s(name: str) -> str:
    """Cleans and converts a name to be k8s compatible.

    Args:
      name: The original name.

    Returns:
      A sanitized name.
    """
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-',
                                    name.lower())).lstrip('-').rstrip('-')
