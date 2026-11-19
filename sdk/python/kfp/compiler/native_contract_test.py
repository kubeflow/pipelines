# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Public native-execution contract, including absence of in-process runners."""

import importlib.util

import pytest

from kfp import dsl


def test_no_local_runner_package():
    assert importlib.util.find_spec('kfp.local') is None


def test_component_calls_only_capture_tasks():
    @dsl.container_component
    def task():
        return dsl.ContainerSpec(image='busybox:1.36.1', command=['true'])

    assert not hasattr(task, 'execute')
    with pytest.raises(RuntimeError, match='inside @dsl.pipeline'):
        task()

    @dsl.pipeline
    def pipeline():
        invocation = task().set_cpu_request('100m').set_caching_options(False)
        assert invocation.container_spec.resources.cpu_request == '100m'
        assert not hasattr(invocation, 'state')

    assert not hasattr(pipeline, 'execute')
