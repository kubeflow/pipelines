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

from kfp.dsl import PipelineTask

def enable_debug_pause(
    task: PipelineTask,
    before: bool = False,
    after: bool = True,
    on_error: bool = False,
) -> PipelineTask:
    """
    Enable debug-pause for a pipeline task, keeping the pod alive so you 
    can ``kubectl exec`` into it interactively.

    When enabled, Argo Workflows' executor (the ``wait`` container) detects the
    corresponding ``ARGO_DEBUG_PAUSE_*`` environment variable and pauses the
    workflow node. Preventing the pod from terminating. The pod annotation
    ``workflows.argoproj.io/debug`` is also set to signal Argo Workflows to 
    honor the pause. This is a native Argo Workflows feature - no KFP launcher
    support is required.

    This function requires an Argo Workflows server that supports the 
    ``workflows.argoproj.io/debug`` annotation. This is available in Argo 
    Workflows 3.5.0 and later.
    
    Args:
        task: Pipeline Task.
        before: If ``True``, pause before the main process starts. Useful for 
            inspecting the environment, installing tools, or modifying inputs
            before execution.
        after: If ``True`` (default), pause after the main process completes.
            Modified by ``on_error``.
        on_error: If ``True``, only pause after execution when the
            component fails (sets ``ARGO_DEBUG_PAUSE_ON_ERROR`` instead of
            ``ARGO_DEBUG_PAUSE_AFTER``). Requires ``after=True``.
    
    Returns:
        Task object with debug-pause configured

    Raises:
        ValueError: If ``after=False`` and ``on_error=True`` (contradictory)
        ValueError: If both ``before`` and ``after`` are ``False``
    Example:
    ::
        @dsl.pipeline(name="my-pipeline")
        def my_pipeline():
            task = my_component()
            # Pause after execution (default)
            kubernetes.enable_debug_pause(task)

            task2 = my_component()
            # Pause before execution
            kubernetes.enable_debug_pause(task2, before=True, after=False)

            task3 = my_component()
            # Pause after execution only on error
            kubernetes.enable_debug_pause(task3, on_error=True)
    """
    if not after and on_error:
        raise ValueError(
            '"on_error" applies to post-execution pause and requires '
            'after=True. Got after=False, on_error=True - contradictory configuration.')
    
    if not before and not after:
        raise ValueError(
            'At least one of "before" or "after" must be True. '
            'Got before=False, after=False - nothing to pause on.')
    
    # Set environment variable
    if before:
        task.set_env_variable("ARGO_DEBUG_PAUSE_BEFORE", "true")
    if after:
        if on_error:
            task.set_env_variable("ARGO_DEBUG_PAUSE_ON_ERROR", "true")
        else:
            task.set_env_variable("ARGO_DEBUG_PAUSE_AFTER", "true")

    return task