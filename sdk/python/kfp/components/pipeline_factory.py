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
import dataclasses
import inspect
import itertools
import pathlib
import re
from sys import implementation
import textwrap
from typing import Callable, List, Optional, Tuple
import warnings

import docstring_parser

from kfp.components import placeholders
from kfp.components import graph_component
from kfp.components import structures
from kfp.components.types import artifact_types, type_annotations
from kfp.components.types import type_utils


def create_graph_component_from_pipeline(func: Callable):
    """Implementation for the @pipeline decorator. This function converts the incoming pipeline into graph component.
    """
    component_spec = structures.ComponentSpec(
        name=func.__name__,
        description=func.__name__,
        inputs=None,
        outputs=None,
        implementation=structures.Implementation(),
    )
    component_spec.implementation = structures.Implementation(
        graph=structures.DagSpec(
            tasks={},
            outputs={},
        ))

    return graph_component.GraphComponent(
        component_spec=component_spec, python_func=func)
