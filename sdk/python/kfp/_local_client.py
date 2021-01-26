# Copyright 2021 Google LLC
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
import re
import subprocess
from typing import Any, Callable, Dict, List, Mapping, Tuple, Union, cast

import kfp


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

    def add_edge(self, edge_source: str, edge_target: str) -> None:
        """Add an edge between DAG nodes.

        Args::
          edge_source: the source node of the edge
          edge_target: the target node of the edge
        """
        self._graph[edge_source].append(edge_target)
        self._reverse_graph[edge_target].append(edge_source)

    def get_follows(self, source_node) -> List[str]:
        """Get all target nodes start from the specified source node

        Args::
          node: the source node
        """
        return self._graph.get(source_node, [])

    def get_dependencies(self, target_node) -> List[str]:
        """Get all source nodes end with the specified target node

        Args::
          node: the target node
        """
        return self._reverse_graph.get(target_node, [])

    def topological_sort(self) -> List[str]:
        """ List DAG nodes in topological order. """

        in_degree = {node: 0 for node in self._graph.keys()}

        for i in self._graph:
            for j in self._graph[i]:
                in_degree[j] += 1

        queue = []
        for node, degree in in_degree.items():
            if degree == 0:
                queue.append(node)

        top_order = []

        while queue:
            u = queue.pop(0)
            top_order.append(u)

            for node in self._graph[u]:
                in_degree[node] -= 1

                if in_degree[node] == 0:
                    queue.append(node)

        return top_order


class LocalClient:
    def _extract_pipeline_param(self, param: str) -> kfp.dsl.PipelineParam:
        """ Extract PipelineParam from string """
        matches = re.findall(
            r"{{pipelineparam:op=([\w\s_-]*);name=([\w\s_-]+)}}", param
        )
        op_dependency_name = matches[0][0]
        output_file_name = matches[0][1]
        return kfp.dsl.PipelineParam(output_file_name, op_dependency_name)

    def _get_keyword_arguments(self, cmd: List[str]) -> List[Tuple[str, str]]:
        """ Convert command arguments into list of (keyworkd, argument) tuples. """
        arg_pairs = []
        for i in range(len(cmd)):
            if cmd[i].startswith("--"):
                arg_pairs.append((cmd[i].strip("--"), cmd[i + 1]))

        return arg_pairs

    def _replace_file_args(self, cmd: List[str], file_args: Dict[str, str]) -> str:
        """ Replace the placeholder of file in cmd with actual file path. """
        for i in range(len(cmd)):
            if cmd[i].startswith("--"):
                arg = cmd[i].strip("--")
                if arg in file_args:
                    cmd[i + 1] = file_args[arg]

        return cmd

    def _find_base_group(
        self, groups: List[kfp.dsl.OpsGroup], op_name: str
    ) -> Union[kfp.dsl.OpsGroup, None]:
        """ Find the base group of op in candidate group list. """
        if groups is None or len(groups) == 0:
            return None
        for group in groups:
            if self._get_op(group.ops, op_name):
                return group
            else:
                _parent_group = self._find_base_group(group.groups, op_name)
                if _parent_group:
                    return group

        return None

    def _get_op(
        self, ops: List[kfp.dsl.ContainerOp], op_name: str
    ) -> Union[kfp.dsl.ContainerOp, None]:
        """ Get the first op with specified op name """
        return next(filter(lambda op: op.name == op_name, ops), None)

    def _get_subgroup(
        self, groups: List[kfp.dsl.OpsGroup], group_name: str
    ) -> Union[kfp.dsl.OpsGroup, None]:
        """ Get the frist OpsGroup with specified group name """
        return next(filter(lambda g: g.name == group_name, groups), None)

    def _create_group_dag(self, pipeline_dag: _Dag, group: kfp.dsl.OpsGroup) -> _Dag:
        """Create DAG within current group, it's a DAG of direct ops and direct subgroups.

        Each node of the DAG is either an op or a subgroup.
        For each node in current group, if one of its DAG follows is also an op in
        current group, add an edge to this follow op, otherwise, if this follow belongs
        to subgroups, add an edge to its subgroup. If this node has dependency from
        subgroups, then add an edge from this subgroup to current node.
        """
        group_dag = _Dag([op.name for op in group.ops] + [g.name for g in group.groups])

        for op in group.ops:
            for follow in pipeline_dag.get_follows(op.name):
                if self._get_op(group.ops, follow) is not None:
                    # add edge between direct ops
                    group_dag.add_edge(op.name, follow)
                else:
                    _base_group = self._find_base_group(group.groups, follow)
                    if _base_group:
                        # add edge to direct subgroup
                        group_dag.add_edge(op.name, _base_group.name)

            for dependency in pipeline_dag.get_dependencies(op.name):
                if self._get_op(group.ops, dependency) is None:
                    _base_group = self._find_base_group(group.groups, dependency)
                    if _base_group:
                        # add edge from direct subgroup
                        group_dag.add_edge(_base_group.name, op.name)

        return group_dag

    def _create_op_dag(self, p: kfp.dsl.Pipeline) -> _Dag:
        """ Create the DAG of the pipeline ops. """
        dag = _Dag(p.ops.keys())

        for op in p.ops.values():
            # dependencies defined by inputs
            for input_value in op.inputs:
                if isinstance(input_value, kfp.dsl.PipelineParam):
                    input_param = self._extract_pipeline_param(input_value.pattern)
                    if input_param.op_name:
                        dag.add_edge(input_param.op_name, op.name)
                    else:
                        logging.debug("%s depend on pipeline param", op.name)

            # explicit dependencies of current op
            for dependent in op.dependent_names:
                dag.add_edge(dependent, op.name)
        return dag

    def _alter_output_file_path(
        self, run_name: str, op_name: str, output_file: str
    ) -> str:
        """Alter the file path of output artifact to make sure it's unique in local runner.

        kfp compiler will bound a tmp file for each component output, which is unique
        in kfp runtime, but not unique in local runner. We alter the file path of the
        name of current run and op, to make it unique in local runner.
        """
        return re.sub(
            "/tmp",
            "/tmp/{run_name}/{op_name}".format(
                run_name=run_name, op_name=op_name.lower()
            ),
            output_file,
        )

    def get_output_file_path(
        self,
        run_name: str,
        pipeline: kfp.dsl.Pipeline,
        op_name: str,
        output_name: str = None,
    ) -> str:
        """ Get the file path of component output. """

        op_dependency = pipeline.ops[op_name]
        if output_name is None and len(op_dependency.file_outputs) == 1:
            output_name = next(iter(op_dependency.file_outputs.keys()))
        output_file = op_dependency.file_outputs[output_name]
        altered_output_file = self._alter_output_file_path(
            run_name, op_name, output_file
        )
        return altered_output_file

    def _op_cmd_locally(
        self,
        run_name: str,
        pipeline: kfp.dsl.Pipeline,
        op: kfp.dsl.ContainerOp,
        stack: Dict[str, Any],
    ) -> List[str]:
        """ Generate shell command to run the op locally. """
        cmd = op.command + op.arguments

        # In debug mode, for `python -c cmd` format command, pydev will insert code before
        # `cmd`, but there is no newline at the end of the inserted code, which will cause
        # syntax error, so we add newline before `cmd`.
        for i in range(len(cmd)):
            if cmd[i] == "-c":
                cmd[i + 1] = "\n" + cmd[i + 1]

        _arg_pairs = self._get_keyword_arguments(cmd)
        _fixed_arg_value = {}
        for arg_name, arg_value in _arg_pairs:
            if arg_value in stack:  # Argument is LoopArguments item
                arg_value = str(stack[arg_value])
            elif arg_value in op.file_outputs.values():  # Argument is output file
                output_name = next(
                    filter(lambda item: item[1] == arg_value, op.file_outputs.items())
                )[0]
                output_param = op.outputs[output_name]
                output_file = arg_value
                output_file = self._alter_output_file_path(
                    run_name, output_param.op_name, output_file
                )
                arg_value = output_file
            elif (
                arg_value in op.input_artifact_paths.values()
            ):  # Argument is input artifact file
                input_name = next(
                    filter(
                        lambda item: item[1] == arg_value,
                        op.input_artifact_paths.items(),
                    )
                )[0]
                input_param_pattern = op.artifact_arguments[input_name]
                pipeline_param = self._extract_pipeline_param(input_param_pattern)
                input_file = self.get_output_file_path(
                    run_name, pipeline, pipeline_param.op_name, pipeline_param.name
                )

                arg_value = input_file
            _fixed_arg_value[arg_name] = arg_value

        cmd = self._replace_file_args(cmd, _fixed_arg_value)
        return cmd

    def _op_cmd_on_docker(
        self,
        run_name: str,
        pipeline: kfp.dsl.Pipeline,
        op: kfp.dsl.ContainerOp,
        stack: Dict[str, Any],
    ) -> List[str]:
        """ Generate the command to run the op in docker locally. """
        cmd = self._op_cmd_locally(run_name, pipeline, op, stack)

        docker_cmd = ["docker", "run", "-v", "/tmp:/tmp", op.image] + cmd
        return docker_cmd

    def _run_group_dag(
        self,
        run_name: str,
        pipeline: kfp.dsl.Pipeline,
        pipeline_dag: _Dag,
        current_group: kfp.dsl.OpsGroup,
        stack: Dict[str, Any],
        local_env_image: str = None,
    ):
        """Run ops in current group in topological order

        Args:
            pipeline: kfp.dsl.Pipeline
            pipeline_dag: DAG of pipeline ops
            current_group: current ops group
            stack: stack to trace `LoopArguments`
            local_env_image: image name
                If the image of component equals to `local_env_image`, local runner will run
                this component locally in forked process, otherwise, local runner will run
                this component on docker.
        """
        group_dag = self._create_group_dag(pipeline_dag, current_group)

        for node in group_dag.topological_sort():
            subgroup = self._get_subgroup(current_group.groups, node)
            if subgroup is not None:  # Node of DAG is subgroup
                self._run_group(
                    run_name, pipeline, pipeline_dag, subgroup, stack, local_env_image
                )
            else:  # Node of DAG is op
                op = self._get_op(current_group.ops, node)

                can_run_locally = op.image == local_env_image
                if can_run_locally:
                    cmd = self._op_cmd_locally(run_name, pipeline, op, stack)
                else:
                    cmd = self._op_cmd_on_docker(run_name, pipeline, op, stack)

                process = subprocess.Popen(
                    cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    universal_newlines=True,
                )
                # TODO support async process
                logging.info("start task：%s", op.name)
                stdout, stderr = process.communicate()
                if stdout:
                    logging.info(stdout)
                if stderr:
                    logging.error(stderr)
                if process.returncode != 0:
                    logging.error(cmd)
                    break

    def _run_group(
        self,
        run_name: str,
        pipeline: kfp.dsl.Pipeline,
        pipeline_dag: _Dag,
        current_group: kfp.dsl.OpsGroup,
        stack: Dict[str, Any],
        local_env_image: str = None,
    ):
        """Run all ops in current group

        Args:
            run_name: str, the name of this run, can be used to query the run result
            pipeline: kfp.dsl.Pipeline
            pipeline_dag: DAG of pipeline ops
            current_group: current ops group
            stack: stack to trace `LoopArguments`
            local_env_image: image name
                If the image of component equals to `local_env_image`, local runner will run
                this component locally in forked process, otherwise, local runner will run
                this component on docker.
        """
        if current_group.type == kfp.dsl.ParallelFor.TYPE_NAME:
            current_group = cast(kfp.dsl.ParallelFor, current_group)

            if current_group.items_is_pipeline_param:
                _loop_args = current_group.loop_args
                _param_name = _loop_args.name[
                    : -len(_loop_args.LOOP_ITEM_NAME_BASE) - 1
                ]

                _op_dependency = pipeline.ops[_loop_args.op_name]
                _list_file = _op_dependency.file_outputs[_param_name]
                _altered_list_file = self._alter_output_file_path(
                    run_name, _loop_args.op_name, _list_file
                )
                with open(_altered_list_file, "r") as f:
                    _param_values = json.load(f)
                for _param_value in _param_values:
                    if isinstance(_param_values, object):
                        _param_value = json.dumps(_param_value)
                    stack[_loop_args.pattern] = _param_value
                    self._run_group_dag(
                        run_name,
                        pipeline,
                        pipeline_dag,
                        current_group,
                        stack,
                        local_env_image,
                    )
                    del stack[_loop_args.pattern]
            else:
                raise Exception("Not implemented")
        else:
            self._run_group_dag(
                run_name, pipeline, pipeline_dag, current_group, stack, local_env_image
            )

    def create_run_from_pipeline_func(
        self,
        pipeline_func: Callable,
        arguments: Mapping[str, str],
        local_env_image: str = None,
    ):
        """Run kubeflow pipeline locally

        Parameters:
          pipeline_func: pipeline function
          arguments: Arguments to the pipeline function provided as a dict,
          reference to `kfp.client.create_run_from_pipeline_func`
          local_env_image: image name
              If the image of component equals to `local_env_image`, local runner will
              run this component locally in forked process, otherwise, local runner will
              run this component on docker.
        """

        class RunPipelineResult:
            def __init__(
                self, client: LocalClient, pipeline: kfp.dsl.Pipeline, run_id: str
            ):
                self._client = client
                self._pipeline = pipeline
                self.run_id = run_id

            def get_output_file(self, op_name: str, output: str = None):
                return self._client.get_output_file_path(
                    self.run_id, self._pipeline, op_name, output
                )

            def __repr__(self):
                return "RunPipelineResult(run_id={})".format(self.run_id)

        pipeline_name = (
            pipeline_func._component_human_name
            if hasattr(pipeline_func, "_component_human_name")
            else "local_pipeline"
        )
        with kfp.dsl.Pipeline(pipeline_name) as pipeline:
            pipeline_func(**arguments)

        run_version = datetime.datetime.now().strftime("%Y%m%d%H%M%S")
        run_name = pipeline.name.replace(" ", "_").lower() + "_" + run_version

        pipeline_dag = self._create_op_dag(pipeline)
        self._run_group(
            run_name, pipeline, pipeline_dag, pipeline.groups[0], {}, local_env_image
        )

        return RunPipelineResult(self, pipeline, run_name)
