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

import datetime
import json
import logging
import os
import re
import subprocess
import tempfile
import warnings
from collections import deque
from typing import Any, Callable, Dict, List, Mapping, Optional, Union, cast

from kfp import dsl
from kfp.components.utils import maybe_rename_for_k8s


class _Dag:
    """DAG stands for Direct Acyclic Graph.

    DAG here is used to decide the order to execute pipeline ops.

    For more information on DAG, please refer to `wiki <https://en.wikipedia.org/wiki/Directed_acyclic_graph>`_.
    """

    def __init__(self, nodes: List[str]) -> None:
        """

        Args::
          nodes: List of DAG nodes, each node is identified by an unique name.
        """
        self._graph = {node: [] for node in nodes}
        self._reverse_graph = {node: [] for node in nodes}

    @property
    def graph(self):
        return self._graph

    @property
    def reverse_graph(self):
        return self._reverse_graph

    def add_edge(self, edge_source: str, edge_target: str) -> None:
        """Add an edge between DAG nodes.

        Args::
          edge_source: the source node of the edge
          edge_target: the target node of the edge
        """
        self._graph[edge_source].append(edge_target)
        self._reverse_graph[edge_target].append(edge_source)

    def get_follows(self, source_node: str) -> List[str]:
        """Get all target nodes start from the specified source node.

        Args::
          source_node: the source node
        """
        return self._graph.get(source_node, [])

    def get_dependencies(self, target_node: str) -> List[str]:
        """Get all source nodes end with the specified target node.

        Args::
          target_node: the target node
        """
        return self._reverse_graph.get(target_node, [])

    def topological_sort(self) -> List[str]:
        """List DAG nodes in topological order."""

        in_degree = {node: 0 for node in self._graph.keys()}

        for i in self._graph:
            for j in self._graph[i]:
                in_degree[j] += 1

        queue = deque()
        for node, degree in in_degree.items():
            if degree == 0:
                queue.append(node)

        sorted_nodes = []

        while queue:
            u = queue.popleft()
            sorted_nodes.append(u)

            for node in self._graph[u]:
                in_degree[node] -= 1

                if in_degree[node] == 0:
                    queue.append(node)

        return sorted_nodes


class LocalClient:

    class ExecutionMode:
        """Configuration to decide whether the client executes a component in
        docker or in local process."""

        DOCKER = "docker"
        LOCAL = "local"

        def __init__(
            self,
            mode: str = DOCKER,
            images_to_exclude: List[str] = [],
            ops_to_exclude: List[str] = [],
            docker_options: List[str] = [],
        ) -> None:
            """Constructor.

            Args:
                mode: Default execution mode, default 'docker'
                images_to_exclude: If the image of op is in images_to_exclude, the op is
                    executed in the mode different from default_mode.
                ops_to_exclude: If the name of op is in ops_to_exclude, the op is
                    executed in the mode different from default_mode.
                docker_options: Docker options used in docker mode,
                    e.g. docker_options=["-e", "foo=bar"].
            """
            if mode not in [self.DOCKER, self.LOCAL]:
                raise Exception(
                    "Invalid execution mode, must be docker of local")
            self._mode = mode
            self._images_to_exclude = images_to_exclude
            self._ops_to_exclude = ops_to_exclude
            self._docker_options = docker_options

        @property
        def mode(self) -> str:
            return self._mode

        @property
        def images_to_exclude(self) -> List[str]:
            return self._images_to_exclude

        @property
        def ops_to_exclude(self) -> List[str]:
            return self._ops_to_exclude

        @property
        def docker_options(self) -> List[str]:
            return self._docker_options

    def __init__(self, pipeline_root: Optional[str] = None) -> None:
        """Construct the instance of LocalClient.

        Args：
            pipeline_root: The root directory where the output artifact of component
              will be saved.
        """
        warnings.warn(
            'LocalClient is an Alpha[1] feature. It may be deprecated in the future.\n'
            '[1] https://github.com/kubeflow/pipelines/blob/master/docs/release/feature-stages.md#alpha',
            category=FutureWarning,
        )

        pipeline_root = pipeline_root or tempfile.tempdir
        self._pipeline_root = pipeline_root


    def _make_output_file_path_unique(self, run_name: str, op_name: str,
                                      output_file: str) -> str:
        """Alter the file path of output artifact to make sure it's unique in
        local runner.

        kfp compiler will bound a tmp file for each component output,
        which is unique in kfp runtime, but not unique in local runner.
        We alter the file path of the name of current run and op, to
        make it unique in local runner.
        """
        if not output_file.startswith("/tmp/"):
            return output_file
        return f'{self._pipeline_root}/{run_name}/{op_name.lower()}/{output_file[len("/tmp/"):]}'

