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

# `kfp` is a namespace package.
# https://packaging.python.org/guides/packaging-namespace-packages/#pkgutil-style-namespace-packages
__path__ = __import__("pkgutil").extend_path(__path__, __name__)

__version__ = '2.0.0-alpha.5'

TYPE_CHECK = True

from kfp.client import Client  # pylint: disable=wrong-import-position

# TODO: clean up COMPILING_FOR_V2
# COMPILING_FOR_V2 is True when using kfp.compiler or use (v1) kfp.compiler
# with V2_COMPATIBLE or V2_ENGINE mode
COMPILING_FOR_V2 = False
