# Copyright 2020-2022 The Kubeflow Authors
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

import warnings

warnings.warn(
    (f'The module `{__name__}` is deprecated and will be removed in a future'
     'version. Please import directly from the `kfp` namespace, '
     'instead of `kfp.v2`.'),
    category=DeprecationWarning,
    stacklevel=2)

from kfp import compiler
from kfp import components
from kfp import dsl
