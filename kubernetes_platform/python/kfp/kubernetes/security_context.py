# Copyright 2024 The Kubeflow Authors
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
"""Security context configuration for Kubernetes tasks."""

from typing import List, Optional

from google.protobuf import json_format
from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


def set_security_context(
    task: PipelineTask,
    privileged: Optional[bool] = None,
    allow_privilege_escalation: Optional[bool] = None,
    run_as_user: Optional[int] = None,
    run_as_group: Optional[int] = None,
    run_as_non_root: Optional[bool] = None,
    read_only_root_filesystem: Optional[bool] = None,
    capabilities_add: Optional[List[str]] = None,
    capabilities_drop: Optional[List[str]] = None,
    se_linux_options_user: Optional[str] = None,
    se_linux_options_role: Optional[str] = None,
    se_linux_options_type: Optional[str] = None,
    se_linux_options_level: Optional[str] = None,
    seccomp_profile_type: Optional[str] = None,
    seccomp_profile_localhost_profile: Optional[str] = None,
) -> PipelineTask:
    """Set security context for a container.

    This function configures the security context for the container running
    in a Kubernetes pod. Security context settings define privilege and access
    control settings for a Pod or Container.

    For more information, see:
    https://kubernetes.io/docs/tasks/configure-pod-container/security-context/

    Args:
        task:
            Pipeline task.
        privileged:
            Run container in privileged mode. Processes in privileged containers
            are essentially equivalent to root on the host. Defaults to false.
        allow_privilege_escalation:
            Controls whether a process can gain more privileges than its parent
            process. This bool directly controls if the no_new_privs flag will
            be set on the container process. Defaults to true when the container
            is run as privileged, or has CAP_SYS_ADMIN capability added.
        run_as_user:
            The UID to run the entrypoint of the container process.
            Uses runtime default if unset.
        run_as_group:
            The GID to run the entrypoint of the container process.
            Uses runtime default if unset.
        run_as_non_root:
            Indicates that the container must run as a non-root user.
            If true, the Kubelet will validate the image at runtime to ensure
            that it does not run as UID 0 (root) and fail to start the container
            if it does.
        read_only_root_filesystem:
            Whether this container has a read-only root filesystem.
            Default is false.
        capabilities_add:
            List of capabilities to add when running containers.
            Example: ['NET_ADMIN', 'SYS_TIME']
        capabilities_drop:
            List of capabilities to drop when running containers.
            Example: ['ALL']
        se_linux_options_user:
            SELinux user label that applies to the container.
        se_linux_options_role:
            SELinux role label that applies to the container.
        se_linux_options_type:
            SELinux type label that applies to the container.
        se_linux_options_level:
            SELinux level label that applies to the container.
        seccomp_profile_type:
            Indicates which kind of seccomp profile will be applied.
            Valid options are: 'RuntimeDefault', 'Unconfined', 'Localhost'.
        seccomp_profile_localhost_profile:
            Path to a profile defined in a file on the node.
            Must only be set if seccomp_profile_type is 'Localhost'.
            Must be a descending path, relative to the kubelet's configured
            seccomp profile location.

    Returns:
        Task object with an updated security context specification.

    Example:
        ::

            from kfp import dsl
            from kfp import kubernetes

            @dsl.component
            def my_component():
                pass

            @dsl.pipeline
            def my_pipeline():
                task = my_component()
                # Run as privileged container
                kubernetes.set_security_context(
                    task,
                    privileged=True,
                )

            @dsl.pipeline
            def secure_pipeline():
                task = my_component()
                # Run with security hardening
                kubernetes.set_security_context(
                    task,
                    run_as_non_root=True,
                    run_as_user=1000,
                    run_as_group=1000,
                    read_only_root_filesystem=True,
                    allow_privilege_escalation=False,
                    capabilities_drop=['ALL'],
                )
    """
    # Validate seccomp_profile_type if provided
    if seccomp_profile_type is not None:
        valid_seccomp_types = ['RuntimeDefault', 'Unconfined', 'Localhost']
        if seccomp_profile_type not in valid_seccomp_types:
            raise ValueError(
                f"Invalid seccomp_profile_type: '{seccomp_profile_type}'. "
                f"Must be one of {valid_seccomp_types}."
            )

    # Validate that localhost_profile is only set when type is Localhost
    if (seccomp_profile_localhost_profile is not None and
            seccomp_profile_type != 'Localhost'):
        raise ValueError(
            "seccomp_profile_localhost_profile can only be set when "
            "seccomp_profile_type is 'Localhost'."
        )

    msg = common.get_existing_kubernetes_config_as_message(task)

    # Build SecurityContext message
    security_context = pb.SecurityContext()

    if privileged is not None:
        security_context.privileged = privileged

    if allow_privilege_escalation is not None:
        security_context.allow_privilege_escalation = allow_privilege_escalation

    if run_as_user is not None:
        security_context.run_as_user = run_as_user

    if run_as_group is not None:
        security_context.run_as_group = run_as_group

    if run_as_non_root is not None:
        security_context.run_as_non_root = run_as_non_root

    if read_only_root_filesystem is not None:
        security_context.read_only_root_filesystem = read_only_root_filesystem

    # Build Capabilities if any are provided
    if capabilities_add is not None or capabilities_drop is not None:
        capabilities = pb.Capabilities()
        if capabilities_add is not None:
            capabilities.add.extend(capabilities_add)
        if capabilities_drop is not None:
            capabilities.drop.extend(capabilities_drop)
        security_context.capabilities.CopyFrom(capabilities)

    # Build SELinuxOptions if any are provided
    if any([
        se_linux_options_user,
        se_linux_options_role,
        se_linux_options_type,
        se_linux_options_level,
    ]):
        se_linux_options = pb.SELinuxOptions()
        if se_linux_options_user is not None:
            se_linux_options.user = se_linux_options_user
        if se_linux_options_role is not None:
            se_linux_options.role = se_linux_options_role
        if se_linux_options_type is not None:
            se_linux_options.type = se_linux_options_type
        if se_linux_options_level is not None:
            se_linux_options.level = se_linux_options_level
        security_context.se_linux_options.CopyFrom(se_linux_options)

    # Build SeccompProfile if type is provided
    if seccomp_profile_type is not None:
        seccomp_profile = pb.SeccompProfile()
        seccomp_profile.type = seccomp_profile_type
        if seccomp_profile_localhost_profile is not None:
            seccomp_profile.localhost_profile = seccomp_profile_localhost_profile
        security_context.seccomp_profile.CopyFrom(seccomp_profile)

    msg.security_context.CopyFrom(security_context)
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
