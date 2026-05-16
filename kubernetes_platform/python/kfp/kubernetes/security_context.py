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

from typing import Optional

from google.protobuf import json_format
from kfp.dsl import PipelineTask
from kfp.kubernetes import common


def set_security_context(
    task: PipelineTask,
    run_as_user: Optional[int] = None,
    run_as_group: Optional[int] = None,
    run_as_non_root: Optional[bool] = None,
) -> PipelineTask:
    """Set the security context for the task's container.

    Sets identity fields (``runAsUser``, ``runAsGroup``, ``runAsNonRoot``) on
    the container's `securityContext
    <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#securitycontext-v1-core>`_.

    All capabilities are automatically dropped to comply with Pod Security
    Standards (PSS) baseline.

    Note:
        Platform security defaults (``allowPrivilegeEscalation=false``,
        ``drop ALL capabilities``, ``seccompProfile=RuntimeDefault``) are
        enforced separately by the compiler and are not affected by this
        function. If an administrator or the compiler has already set
        ``runAsUser``, ``runAsGroup``, or ``runAsNonRoot``, the values
        provided here will be ignored and a warning will be logged by the
        backend. Admin-set security context values cannot be overridden by
        the SDK.

    Args:
        task: Pipeline task.
        run_as_user: The UID to run the container process as.
        run_as_group: The GID to run the container process as.
        run_as_non_root: Whether the container must run as a non-root user.

    Returns:
        Task object with an updated security context.
    """
    if run_as_user is None and run_as_group is None and run_as_non_root is None:
        raise ValueError(
            'At least one security context field must be provided.'
        )
    if run_as_user is not None and isinstance(run_as_user, bool):
        raise TypeError(
            f'Argument for "run_as_user" must be an int, not bool. Got: {run_as_user}.'
        )
    if run_as_group is not None and isinstance(run_as_group, bool):
        raise TypeError(
            f'Argument for "run_as_group" must be an int, not bool. Got: {run_as_group}.'
        )
    if run_as_user is not None and run_as_user < 0:
        raise ValueError(
            f'Argument for "run_as_user" must be greater than or equal to 0. Got invalid input: {run_as_user}.'
        )
    if run_as_group is not None and run_as_group < 0:
        raise ValueError(
            f'Argument for "run_as_group" must be greater than or equal to 0. Got invalid input: {run_as_group}.'
        )
    if run_as_non_root is not None and not isinstance(run_as_non_root, bool):
        raise TypeError(
            f'Argument for "run_as_non_root" must be a bool. Got: {type(run_as_non_root).__name__}.'
        )

    msg = common.get_existing_kubernetes_config_as_message(task)

    if run_as_user is not None:
        msg.security_context.run_as_user = run_as_user

    if run_as_group is not None:
        msg.security_context.run_as_group = run_as_group

    if run_as_non_root is not None:
        msg.security_context.run_as_non_root = run_as_non_root

    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
