# Copyright 2018-2022 The Kubeflow Authors
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

# declare an explicit namespace using pkg_resources and use the namespace_packages argument in setup.py to avoid module collision between old versions of kfp and kfp-dsl on pip install
# https://setuptools.pypa.io/en/latest/userguide/package_discovery.html#pkg-resource-style-namespace-package
__import__('pkg_resources').declare_namespace(__name__)

__version__ = '2.1.2'

TYPE_CHECK = True

from kfp import components
from kfp import dsl
from kfp.client import Client
