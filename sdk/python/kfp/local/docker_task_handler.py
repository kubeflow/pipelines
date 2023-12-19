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

from kfp.dsl import component_factory
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
        # TODO: remove when full placeholder support is added
        self.validate_not_container_component(full_command)
        self.image = image
        self.full_command = full_command
        self.pipeline_root = pipeline_root
        self.runner = runner

    def get_volumes_to_mount(self) -> Dict[str, Any]:
        """Gets the volume configuration to mount the pipeline root to the
        container so that outputs can be obtained outside of the container."""
        if not os.path.isabs(self.pipeline_root):
            # defensive check. this is enforced by upstream code.
            # users should not hit this,
            raise ValueError(
                "'pipeline_root' should be an absolute path to correctly construct the volume mount specification."
            )
        return {self.pipeline_root: {'bind': self.pipeline_root, 'mode': 'rw'}}

    def run(self) -> status.Status:
        """Runs the Docker container and returns the status."""
        try:
            import docker
            client = docker.from_env()
            volumes = self.get_volumes_to_mount()
            return_code = run_docker_container(
                client=client,
                image=self.image,
                command=self.full_command,
                volumes=volumes,
            )
        finally:
            client.close()
        return status.Status.SUCCESS if return_code == 0 else status.Status.FAILURE

    def validate_not_container_component(
        self,
        full_command: List[str],
    ) -> None:
        if not any(component_factory.EXECUTOR_MODULE in part
                   for part in full_command):
            raise RuntimeError(
                f'The {config.DockerRunner.__name__} only supports running Lightweight Python Components. You are attempting to run a Container Component.'
            )


def pull_image(client: 'docker.DockerClient', image: str) -> None:
    if ':' in image:
        repository, tag = image.split(':')
    else:
        repository, tag = image, 'latest'
    client.images.pull(repository=repository, tag=tag)


def run_docker_container(
    client: 'docker.DockerClient',
    image: str,
    command: List[str],
    volumes: Dict[str, Any],
) -> int:
    image_exists = any(
        image in existing_image.tags for existing_image in client.images.list())
    if image_exists:
        print(f'Found image {image!r}')
    else:
        print(f'Pulling image {image!r}')
        pull_image(client, image)
        print('Image pull complete')
    container = client.containers.run(
        image=image,
        command=command,
        detach=True,
        stdout=True,
        stderr=True,
        volumes=volumes,
    )
    for line in container.logs(stream=True):
        print(line.decode())
    return container.wait()['StatusCode']
