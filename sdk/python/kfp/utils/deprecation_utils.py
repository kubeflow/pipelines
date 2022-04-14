# Copyright 2022 The Kubeflow Authors
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

import functools
import warnings
from typing import Callable, Dict, TypeVar

WrappedSignature = TypeVar('WrappedSignature', bound=Callable)


def _alias_deprecated_params(
        aliases: Dict[str, str]) -> Callable[[WrappedSignature], Callable]:
    """A second-order decorator to alias deprecated parameters.

    Args:
        aliases: A dictionary of new parameter name to deprecated parameter name pairs.

    Returns:
        Decorator that aliases deprecated parameters.
    """

    def _deprecation_callback(new: str, deprecated: str) -> None:
        warnings.warn(
            f"{deprecated} parameter is deprecated, use {new} instead.",
            DeprecationWarning)

    def decorator(func: WrappedSignature) -> Callable:

        @functools.wraps(func)
        def wrapper(*args, **kwargs) -> None:
            for name, alias in aliases.items():
                if name not in kwargs and alias in kwargs:
                    _deprecation_callback(name, alias)
                    kwargs[name] = kwargs[alias]
            return func(*args, **kwargs)

        return wrapper

    return decorator