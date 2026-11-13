"""Validation of release answers and checkpoint state."""

import re

from .metadata import ReleaseMetadata
from .state import ReleaseState


def parse_github_owner(fork_remote: str) -> str:
    """Parse GitHub owner from fork remote URL.

    Args:
      fork_remote: Git remote URL (SSH or HTTPS format).

    Returns:
      GitHub username/owner extracted from the URL.

    Raises:
      ValueError: If owner cannot be parsed from the remote URL.

    Examples:
      >>> parse_github_owner('git@github.com:user123/pipelines.git')
      'user123'
      >>> parse_github_owner('https://github.com/user123/pipelines.git')
      'user123'
    """
    # SSH format: git@github.com:USER/REPO.git
    ssh_match = re.match(r'^git@github\.com:([^/]+)/', fork_remote)
    if ssh_match:
        return ssh_match.group(1)

    # HTTPS format: https://github.com/USER/REPO.git
    https_match = re.match(r'^https://github\.com/([^/]+)/', fork_remote)
    if https_match:
        return https_match.group(1)

    raise ValueError(
        f'Cannot parse GitHub owner from fork_remote: {fork_remote}. '
        f'Expected format: git@github.com:USER/REPO.git or https://github.com/USER/REPO.git'
    )


def normalize_fork_remote(fork_remote: str) -> str:
    """Normalize a fork owner or remote URL to a GitHub remote URL."""
    if fork_remote == 'kubeflow':
        raise ValueError('fork_remote must not be the kubeflow organization')
    if re.match(r'^git@github\.com:[^/]+/[^/]+(?:\.git)?$', fork_remote):
        normalized = fork_remote
    elif re.match(r'^https://github\.com/[^/]+/[^/]+(?:\.git)?$', fork_remote):
        normalized = fork_remote
    elif re.match(r'^[A-Za-z0-9][A-Za-z0-9-]*$', fork_remote):
        normalized = f'https://github.com/{fork_remote}/pipelines.git'
    else:
        raise ValueError(
            f'Cannot parse fork_remote: {fork_remote}. '
            'Use a GitHub username, git@github.com:USER/pipelines.git, '
            'or https://github.com/USER/pipelines.git'
        )
    if parse_github_owner(normalized).lower() == 'kubeflow':
        raise ValueError('fork_remote must not point to kubeflow/pipelines')
    return normalized


def validate_answers(
    answers: dict[str, object], require_all: bool = True
) -> tuple[ReleaseMetadata | None, list[str]]:
    """Validate release answers and return derived metadata when possible."""
    errors = []
    release_type = answers.get('release_type')
    version = answers.get('version')
    fork_remote = answers.get('fork_remote')

    if require_all and not release_type:
        errors.append('missing release_type')
    if require_all and not version:
        errors.append('missing version')
    metadata = None
    if release_type and version:
        try:
            metadata = ReleaseMetadata.from_version(
                str(release_type), str(version)
            )
        except ValueError as error:
            errors.append(str(error))

    if require_all and not fork_remote:
        errors.append('missing fork_remote')
    if fork_remote:
        try:
            normalize_fork_remote(str(fork_remote))
        except ValueError as error:
            errors.append(str(error))
    return metadata, errors


def validate_state(state: ReleaseState) -> list[str]:
    """Validate a checkpoint state file for resume safety."""
    if state.load_errors:
        return list(state.load_errors)

    metadata, errors = validate_answers(state.answers)
    if metadata is None:
        return errors

    include_backend = bool(state.answers.get('include_backend', True))
    include_sdk = bool(state.answers.get('include_sdk', True))
    from .steps import build_steps

    step_ids = {
        step.step_id
        for step in build_steps(
            metadata.release_type, include_backend, include_sdk
        )
    }
    for completed_step in state.completed_steps:
        if completed_step not in step_ids:
            errors.append(f'unknown completed step: {completed_step}')
    return errors
