# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Graph naming/traversal shared with the DSL. No execution IR builder."""

from kfp.dsl import pipeline_channel


def additional_input_name_for_pipeline_channel(channel_or_name):
    name = (
        channel_or_name.full_name
        if isinstance(channel_or_name, pipeline_channel.PipelineChannel)
        else channel_or_name
    )
    return 'pipelinechannel--' + name


def get_all_groups(root_group):
    groups = []
    pending = [root_group]
    while pending:
        group = pending.pop()
        groups.append(group)
        pending.extend(reversed(group.groups))
    return groups
