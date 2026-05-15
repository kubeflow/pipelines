# Copyright 2026 The Kubeflow Authors
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

from typing import Any, Dict, List, Optional

import pytest
from kfp import dsl
from kfp import kubernetes
from google.protobuf import json_format


class TestEnableDebugPause:
    def test_default_sets_pause_after_env_var(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.enable_debug_pause(task)
        
        env = _get_env(my_pipeline)
        assert _has_env(env, 'ARGO_DEBUG_PAUSE_AFTER', 'true')
        assert not _has_env(env, 'ARGO_DEBUG_PAUSE_BEFORE')
        assert not _has_env(env, 'ARGO_DEBUG_PAUSE_ON_ERROR')
    
    def test_on_error_sets_on_error_env_var(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.enable_debug_pause(task, on_error=True)

        env = _get_env(my_pipeline)
        assert _has_env(env, 'ARGO_DEBUG_PAUSE_ON_ERROR', 'true')
        assert not _has_env(env, 'ARGO_DEBUG_PAUSE_AFTER')
        assert not _has_env(env, 'ARGO_DEBUG_PAUSE_BEFORE')
    
    def test_before_only_sets_pause_before_env_var(self):
        
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.enable_debug_pause(task, before=True, after=False)
        
        env = _get_env(my_pipeline)
        assert _has_env(env, 'ARGO_DEBUG_PAUSE_BEFORE', 'true')
        assert not _has_env(env, 'ARGO_DEBUG_PAUSE_AFTER')
        assert not _has_env(env, 'ARGO_DEBUG_PAUSE_ON_ERROR')
    
    def test_before_and_after_sets_both_env_vars(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.enable_debug_pause(task, before=True, after=True)
        
        env = _get_env(my_pipeline)
        assert _has_env(env, 'ARGO_DEBUG_PAUSE_BEFORE', 'true')
        assert _has_env(env, 'ARGO_DEBUG_PAUSE_AFTER', 'true')
        assert not _has_env(env, 'ARGO_DEBUG_PAUSE_ON_ERROR')
    
    def test_raises_when_before_and_after_both_false(self):
        with pytest.raises(
            ValueError, 
            match=(
                'At least one of "before" or "after" must be True'
            'Got before=False, after=False - nothing to pause on.'
            )
        ):
            @dsl.pipeline
            def my_pipeline():
                task = comp()
                kubernetes.enable_debug_pause(task, before=False, after=False)
    
    def test_raises_when_on_error_without_after(self):
        with pytest.raises(
            ValueError, 
            match=(
                '"on_error" applies to post-execution pause and requires '
                'after=True. Got after=False, on_error=True - contradictory configuration.'
            )
        ):
            @dsl.pipeline
            def my_pipeline():
                task = comp()
                kubernetes.enable_debug_pause(task, on_error=True, after=False)
    
    def test_composes_with_other_kubernetes_helpers(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_pod_label(task, 'team', 'ml')
            kubernetes.enable_debug_pause(task)
        
        env = _get_env(my_pipeline)
        assert _has_env(env, 'ARGO_DEBUG_PAUSE_AFTER', 'true')
        # Other Kubernetes features should still be present
        platform_spec = json_format.MessageToDict(my_pipeline.platform_spec)
        labels = (platform_spec['platforms']['kubernetes']['deploymentSpec']['executors']['exec-comp']['podMetadata']['labels'])
        assert labels == {'team': 'ml'}


def _get_env(pipeline) -> List[Dict[str, Any]]:
    """Returns the container environment variables from a compiled pipeline.

    Args:
        pipeline: A ``@dsl.pipeline``-decorated function whose component spec
            contains exactly one executor named ``exec-comp``.

    Returns:
        A list of environment variable dicts, each with ``'name'`` and
        ``'value'`` keys, as produced by
        ``PipelineSpec.deploymentSpec.executors['exec-comp'].container.env``.

    Example:
    ::

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.enable_debug_pause(task)

        env = _get_env(my_pipeline)
        # [{'name': 'ARGO_DEBUG_PAUSE_AFTER', 'value': 'true'}]
    """
    # pipeline_spec = pipeline.component_spec.to_dict()
    pipeline_spec = json_format.MessageToDict(pipeline.component_spec.to_pipeline_spec())
    return (pipeline_spec['deploymentSpec']['executors']['exec-comp']['container']['env'])

def _has_env(env: List[Dict[str, Any]], name: str, value: Optional[str] = None) -> bool:
    """Returns whether an environment variable exists in an env list.

    Args:
        env: A list of environment variable dicts, each with at least a
            ``'name'`` key and optionally a ``'value'`` key.
        name: The name of the environment variable to look for.
        value: If provided, also assert that the matching entry's ``'value'``
            equals this string. If ``None`` (default), only the name is
            checked.

    Returns:
        ``True`` if an entry matching ``name`` (and ``value``, when given) is
        found; ``False`` otherwise.

    Example:
    ::

        env = [{'name': 'ARGO_DEBUG_PAUSE_AFTER', 'value': 'true'}]
        _has_env(env, 'ARGO_DEBUG_PAUSE_AFTER')              # True
        _has_env(env, 'ARGO_DEBUG_PAUSE_AFTER', 'true')      # True
        _has_env(env, 'ARGO_DEBUG_PAUSE_BEFORE')             # False
    """
    for e in env:
        if e.get('name') == name:
            if value is None:
                return True
            return e.get('value') == value
    return False

@dsl.component
def comp():
    pass