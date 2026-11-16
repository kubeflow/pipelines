"""Release context assembly from arguments, state and prompts."""

import argparse
import subprocess
from dataclasses import dataclass
from pathlib import Path

from .metadata import ReleaseMetadata
from .prompts import (
    prompt_choice,
    prompt_numbered_choice,
    prompt_required,
    prompt_validated,
)
from .runner import CommandRunner
from .state import ReleaseState
from .validation import normalize_fork_remote


@dataclass
class ReleaseContext:
    """Context for release step execution."""

    root: Path
    state: ReleaseState
    runner: CommandRunner
    metadata: ReleaseMetadata
    fork_remote: str
    include_backend: bool
    include_sdk: bool
    skip_local_review: bool = False
    previous_release: str = ''
    previous_sdk_release: str = ''
    release_target: str = ''


def collect_context(
    args: argparse.Namespace, state: ReleaseState
) -> ReleaseContext:
    """Collect release context from state and prompts.

    Args:
      args: Parsed command-line arguments.
      state: Release state with answers and completed steps.

    Returns:
      ReleaseContext instance with all required fields.
    """
    root = Path(
        subprocess.check_output(
            ['git', 'rev-parse', '--show-toplevel'], text=True
        ).strip()
    )
    answers = state.answers
    if 'release_type' not in answers:
        answers['release_type'] = prompt_numbered_choice(
            'Release type', ['major', 'minor', 'patch']
        )
    if 'version' not in answers:
        answers['version'] = prompt_validated(
            'Release version',
            lambda version: ReleaseMetadata.from_version(
                str(answers['release_type']), version
            ),
        )
    if 'fork_remote' not in answers:
        answers['fork_remote'] = prompt_validated(
            'Fork remote, for example https://github.com/USER/pipelines.git',
            normalize_fork_remote,
        )
    answers['fork_remote'] = normalize_fork_remote(str(answers['fork_remote']))
    if answers['release_type'] == 'patch':
        if 'patch_prs' not in answers:
            answers['patch_prs'] = prompt_required(
                'Comma-separated PR numbers or URLs to cherry-pick'
            )
        if 'include_backend' not in answers:
            answers['include_backend'] = (
                prompt_choice(
                    'Does this patch include backend/images?',
                    ['y', 'n'],
                    default='y',
                )
                == 'y'
            )
        if 'include_sdk' not in answers:
            answers['include_sdk'] = (
                prompt_choice(
                    'Does this patch include SDK packages?',
                    ['y', 'n'],
                    default='y',
                )
                == 'y'
            )
    else:
        if (
            getattr(args, 'release_source_branch', None)
            and 'release_source_branch' not in answers
        ):
            answers['release_source_branch'] = str(args.release_source_branch)
        if (
            getattr(args, 'prompt_release_source_branch', False)
            and 'release_source_branch' not in answers
        ):
            source_branch = input(
                'Release source branch [repository default]: '
            ).strip()
            if source_branch:
                answers['release_source_branch'] = source_branch
        if 'include_backend' not in answers:
            answers['include_backend'] = True
        if 'include_sdk' not in answers:
            answers['include_sdk'] = True
    metadata = ReleaseMetadata.from_version(
        str(answers['release_type']), str(answers['version'])
    )
    if (
        getattr(args, 'require_previous_release', True)
        and answers.get('include_backend', True)
        and 'previous_release' not in answers
    ):
        answers['previous_release'] = prompt_required('Previous release tag')
    if getattr(args, 'save_state', True):
        state.save()
    return ReleaseContext(
        root=root,
        state=state,
        runner=CommandRunner(dry_run=args.dry_run),
        metadata=metadata,
        fork_remote=str(answers['fork_remote']),
        include_backend=bool(answers['include_backend']),
        include_sdk=bool(answers['include_sdk']),
        skip_local_review=bool(answers.get('skip_local_review', False)),
        previous_release=str(answers.get('previous_release', '')),
        previous_sdk_release=str(answers.get('previous_sdk_release', '')),
        release_target=str(answers.get('release_target', '')),
    )
