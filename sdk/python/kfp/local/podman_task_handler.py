import os
from typing import Any, Dict, List

from kfp.local import config
from kfp.local import status
from kfp.local import task_handler_interface


class PodmanTaskHandler(task_handler_interface.ITaskHandler):
    """The task handler corresponding to PodmanRunner."""

    def __init__(
        self,
        image: str,
        full_command: List[str],
        pipeline_root: str,
        runner: config.PodmanRunner,
    ) -> None:
        self.image = image
        self.full_command = full_command
        self.pipeline_root = pipeline_root
        self.runner = runner

    def get_volumes_to_mount(self) -> Dict[str, Any]:
        """Gets the volume configuration to mount the pipeline root to the
        container so that outputs can be obtained outside of the container."""
        if not os.path.isabs(self.pipeline_root):
            # Defensive check that is enforced by upstream code.
            # users should not hit this
            raise ValueError(
                "'pipeline_root' should be an absolute path to correctly construct the volume mount specification."
            )
        return {self.pipeline_root: {'bind': self.pipeline_root, 'mode': 'rw'}}

    def run(self) -> status.Status:
        """Runs the container and returns the status."""
        # nest podman import in case not available in user env so that
        # this module is runnable, even if not using PodmanRunner
        import podman
        client = podman.PodmanClient()
        try:
            volumes = self.get_volumes_to_mount()
            return_code = run_container(
                client=client,
                image=self.image,
                command=self.full_command,
                volumes=volumes,
            )
        finally:
            client.close()
        return status.Status.SUCCESS if return_code == 0 else status.Status.FAILURE


def add_latest_tag_if_not_present(image: str) -> str:
    """Adds the 'latest' tag if no tag is present in the image name."""
    if ':' not in image:
        return f'{image}:latest'
    return image


def run_container(
    client: 'podman.PodmanClient',
    image: str,
    command: List[str],
    volumes: Dict[str, Dict[str, str]],
) -> int:
    """Runs a container using Podman.

    Args:
        client: The Podman client instance.
        image: The container image to use.
        command: The command to run in the container.
        volumes: Dictionary mapping host paths to volume configuration.

    Returns:
        The exit code of the container.
    """

    image = add_latest_tag_if_not_present(image=image)

    image_exists = any(
        image in existing_image.tags for existing_image in client.images.list())
    if image_exists:
        print(f'Found image {image!r}\n')
    else:
        print(f'Pulling image {image!r}')
        repository, tag = image.split(':')
        client.images.pull(repository=repository, tag=tag)
        print('Image pull complete\n')

    # Create and run container
    container = client.containers.create(
        image=image,
        command=command,
        volumes=volumes,
        detach=True,
    )

    # Start container and stream logs
    container.start()
    for line in container.logs(stream=True):
        print(line.decode(), end='')

    # Wait for container to finish and get exit code
    container.wait()
    return container.exit_code
