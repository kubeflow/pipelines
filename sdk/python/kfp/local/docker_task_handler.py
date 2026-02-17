# Copyright 2023 The Kubeflow Authors
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
import os
from typing import Any, Dict, List

import docker
from kfp.dsl import constants as dsl_constants
from kfp.local import config
from kfp.local import status
from kfp.local import task_handler_interface


class DockerTaskHandler(task_handler_interface.ITaskHandler):
    """The task handler corresponding to DockerRunner."""

    def __init__(
        self,
        image: str,
        full_command: List[str],
        pipeline_root: str,
        runner: config.DockerRunner,
    ) -> None:
        self.image = image
        self.full_command = full_command
        self.pipeline_root = pipeline_root
        self.runner = runner

    def get_volumes_to_mount(self,
                             client: docker.DockerClient = None
                            ) -> Dict[str, Any]:
        """Gets the volume configuration to mount the pipeline root and
        workspace to the container so that outputs and workspace can be
        accessed outside of the container."""
        default_mode = 'rw'
        if client is not None and 'name=selinux' in client.info().get(
                'SecurityOptions', []):
            default_mode = f'{default_mode},z'

        if not os.path.isabs(self.pipeline_root):
            # defensive check. this is enforced by upstream code.
            # users should not hit this,
            raise ValueError(
                "'pipeline_root' should be an absolute path to correctly construct the volume mount specification."
            )
        volumes = {
            self.pipeline_root: {
                'bind': self.pipeline_root,
                'mode': default_mode
            }
        }
        # Add workspace volume mount if workspace is configured
        if (config.LocalExecutionConfig.instance and
                config.LocalExecutionConfig.instance.workspace_root):
            workspace_root = config.LocalExecutionConfig.instance.workspace_root
            if not os.path.isabs(workspace_root):
                workspace_root = os.path.abspath(workspace_root)
            # Mount workspace to the standard KFP workspace path
            volumes[workspace_root] = {
                'bind': dsl_constants.WORKSPACE_MOUNT_PATH,
                'mode': default_mode
            }
        return volumes

    def run(self) -> status.Status:
        """Runs the Docker container and returns the status."""
        # nest docker import in case not available in user env so that
        # this module is runnable, even if not using DockerRunner
        import docker
        client = docker.from_env()
        try:
            volumes = self.get_volumes_to_mount(client)

            # Check if command contains pip install operations
            command_str = ' '.join(
                self.full_command) if self.full_command else ''
            has_pip_install = 'pip install' in command_str or 'python3 -m pip install' in command_str

            if has_pip_install:
                # Use managed install context for pip operations
                from kfp.local.pip_install_manager import pip_install_manager
                with pip_install_manager.managed_install('docker'):
                    return_code = run_docker_container(
                        client=client,
                        image=self.image,
                        command=self.full_command,
                        volumes=volumes,
                        **self.runner.container_run_args)
            else:
                return_code = run_docker_container(
                    client=client,
                    image=self.image,
                    command=self.full_command,
                    volumes=volumes,
                    **self.runner.container_run_args)
        finally:
            client.close()
        return status.Status.SUCCESS if return_code == 0 else status.Status.FAILURE


def add_latest_tag_if_not_present(image: str) -> str:
    if ':' not in image:
        image = f'{image}:latest'
    return image


def run_docker_container(client: 'docker.DockerClient', image: str,
                         command: List[str], volumes: Dict[str, Any],
                         **container_run_args) -> int:
    image = add_latest_tag_if_not_present(image=image)
    image_exists = any(
        image in (existing_image.tags + existing_image.attrs['RepoDigests'])
        for existing_image in client.images.list())
    if image_exists:
        print(f'Found image {image!r}\n')
    else:
        print(f'Pulling image {image!r}')
        client.images.pull(image)
        print('Image pull complete\n')
    container = client.containers.run(
        image=image,
        command=command,
        detach=True,
        stdout=True,
        stderr=True,
        volumes=volumes,
        auto_remove=True,
        **container_run_args)
    try:
        for line in container.logs(stream=True):
            # the inner logs should already have trailing \n
            # we do not need to add another
            print(line.decode(), end='')
    except docker.errors.NotFound:
        # Container was auto-removed before we could stream logs
        # This can happen if the container exits very quickly
        pass
    try:
        return container.wait()['StatusCode']
    except docker.errors.NotFound:
        # Container was auto-removed after logs completed
        # This means the container exited successfully (logs streaming completed)
        # We assume success since we couldn't get the actual status code
        return 0
