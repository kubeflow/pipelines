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

try:
    from .version import __version__
except ImportError:
    __version__ = 'dev'

import sys

if sys.version_info < (3, 11):
    raise RuntimeError(
        'KFP requires Python 3.11 or later. Upgrade your Python interpreter '
        'before importing kfp.'
    )

TYPE_CHECK = True

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from kfp import compiler, components, dsl
    from kfp.client import Client

_LAZY_MODULES = {
    'Client': ('kfp.client', 'Client'),
    'components': ('kfp.components', None),
    'compiler': ('kfp.compiler', None),
    'dsl': ('kfp.dsl', None),
}


def __getattr__(name: str):
    if name in _LAZY_MODULES:
        module_name, attr_name = _LAZY_MODULES[name]
        import importlib

        mod = importlib.import_module(module_name)
        val = getattr(mod, attr_name) if attr_name else mod
        globals()[name] = val
        return val
    raise AttributeError(f"module '{__name__}' has no attribute '{name}'")


def __dir__():
    return sorted(list(globals().keys()) + list(_LAZY_MODULES.keys()))
