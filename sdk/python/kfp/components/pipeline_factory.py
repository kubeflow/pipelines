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

from typing import Callable

from kfp.components import component_factory, graph_component
from kfp.components import structures


def create_graph_component_from_pipeline(func: Callable):
    """Implementation for the @pipeline decorator. This function converts the incoming pipeline into graph component.
    """
    pipeline_meta = component_factory.extract_component_interface(
      func
    )
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
