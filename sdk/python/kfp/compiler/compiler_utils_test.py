# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
from kfp import dsl
from kfp.compiler.compiler_utils import (
    additional_input_name_for_pipeline_channel,
    get_all_groups,
)


def test_channel_alias_and_group_traversal():
    @dsl.container_component
    def echo(value: int):
        return dsl.ContainerSpec(
            image='busybox', command=['echo'], args=[value]
        )

    @dsl.pipeline
    def pipeline(value: int = 1):
        with dsl.If(value > 0):
            with dsl.ParallelFor([1, 2]) as item:
                echo(value=item)

    groups = get_all_groups(
        pipeline.component_spec.implementation.graph.pipeline.groups[0]
    )
    assert len(groups) == 3
    assert groups[1].parent_task_group is groups[0]
    assert groups[2].parent_task_group is groups[1]
    assert (
        additional_input_name_for_pipeline_channel('input')
        == 'pipelinechannel--input'
    )
