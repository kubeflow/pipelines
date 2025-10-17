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
__path__ = __import__('pkgutil').extend_path(__path__, __name__)

try:
    from .version import __version__
except ImportError:
    __version__ = 'dev'

import sys
import warnings

if sys.version_info < (3, 9):
    warnings.warn(
        ('KFP will drop support for Python 3.9 on Oct 1, 2025. To use new versions of the KFP SDK after that date, you will need to upgrade to Python >= 3.10. See https://devguide.python.org/versions/ for more details.'
        ),
        FutureWarning,
        stacklevel=2,
    )

TYPE_CHECK = True

import os

# compile-time only dependencies
if os.environ.get('_KFP_RUNTIME', 'false') != 'true':
    # make `from kfp import components` and `from kfp import dsl` valid;
    # related to namespace packaging issue
    from kfp import components  # noqa: keep unused import
    from kfp import dsl  # noqa: keep unused import
    from kfp.client import Client  # noqa: keep unused import
