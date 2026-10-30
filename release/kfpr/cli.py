"""Typer CLI for Kubeflow Pipelines release automation."""

import json
import subprocess
from enum import Enum
from pathlib import Path
from typing import Optional

import typer

from .core import (
    STATE_FILE,
    CommandRunner,
    ReleaseContext,
    ReleaseMetadata,
    ReleaseState,
    collect_context,
    doctor_errors,
    emphasize_prompt,
    normalize_fork_remote,
    validate_answers,
    validate_state,
    watch_latest_workflow_run,
)
from .editorial import validate_notes
from .inventory import build_inventory, review_warnings, write_bundle
from .ranges import resolve_range
from .steps import (
    STEP_HANDLERS,
    Step,
    build_steps,
    manual_checklist,
    regenerate_release_notes,
    run_steps,
)

HELP_OPTION_NAMES = ['-h', '--help']


class NotesProduct(str, Enum):
    backend = 'backend'
    sdk = 'sdk'


class ImageQualification(str, Enum):
    ci = 'ci'


app = typer.Typer(
    help='Kubeflow Pipelines release automation.',
    context_settings={'help_option_names': HELP_OPTION_NAMES},
)
run_app = typer.Typer(
    help='Run the full release flow or one release step.',
    context_settings={'help_option_names': HELP_OPTION_NAMES},
    invoke_without_command=True,
)


def _load_state(path: Path) -> ReleaseState:
    return ReleaseState.load(path)


def _set_answer(state: ReleaseState, key: str, value: object | None) -> None:
    if value is not None:
        state.answers[key] = value


def _context(
    state_file: Path,
    dry_run: bool,
    release_type: Optional[str],
    version: Optional[str],
    fork_remote: Optional[str],
    patch_prs: Optional[str],
    include_backend: Optional[bool],
    include_sdk: Optional[bool],
    prompt_release_source_branch: bool = False,
    release_source_branch: Optional[str] = None,
    skip_local_review: bool = False,
    previous_release: Optional[str] = None,
    previous_sdk_release: Optional[str] = None,
    release_target: Optional[str] = None,
    save_state: bool = True,
    require_previous_release: bool = True,
    qualification_pythons: Optional[str] = None,
    security_policy: Optional[str] = None,
    security_exceptions: Optional[Path] = None,
    image_qualification: Optional[str] = None,
) -> ReleaseContext:
    state = _load_state(state_file)
    _set_answer(state, 'release_type', release_type)
    _set_answer(state, 'version', version)
    _set_answer(
        state,
        'fork_remote',
        normalize_fork_remote(fork_remote) if fork_remote is not None else None,
    )
    _set_answer(state, 'patch_prs', patch_prs)
    _set_answer(state, 'include_backend', include_backend)
    _set_answer(state, 'include_sdk', include_sdk)
    if skip_local_review:
        state.answers['skip_local_review'] = True
    _set_answer(state, 'previous_release', previous_release)
    _set_answer(state, 'previous_sdk_release', previous_sdk_release)
    _set_answer(state, 'release_target', release_target)
    if qualification_pythons:
        raise typer.BadParameter(
            'CI selects Python versions from the release source',
            param_hint='--qualification-pythons',
        )
    _set_answer(state, 'qualification_pythons', qualification_pythons)
    if security_policy not in (
        None,
        'zero-fixable',
        'approved-exceptions',
        '2.17.3-approved-exceptions-v1',
    ):
        raise typer.BadParameter(
            'Unknown security policy', param_hint='--security-policy'
        )
    if security_exceptions is not None:
        if security_policy not in (None, 'approved-exceptions'):
            raise typer.BadParameter(
                'Exception files require approved-exceptions policy',
                param_hint='--security-exceptions',
            )
        from .security_exceptions import load_exceptions

        try:
            load_exceptions(security_exceptions)
        except (ValueError, OSError) as error:
            raise typer.BadParameter(
                str(error), param_hint='--security-exceptions'
            ) from error
        state.answers['security_exceptions'] = str(
            security_exceptions.resolve()
        )
        security_policy = 'approved-exceptions'
    elif security_policy in ('zero-fixable', '2.17.3-approved-exceptions-v1'):
        state.answers.pop('security_exceptions', None)
    _set_answer(state, 'security_policy', security_policy)
    if state.answers.get(
        'security_policy'
    ) == 'approved-exceptions' and not state.answers.get('security_exceptions'):
        raise typer.BadParameter(
            'Provide an exception file', param_hint='--security-exceptions'
        )
    _set_answer(state, 'image_qualification', image_qualification)
    state.answers.setdefault('image_qualification', 'ci')
    if state.answers['image_qualification'] != 'ci':
        raise typer.BadParameter(
            'Qualification scans require ci', param_hint='--image-qualification'
        )

    if 'release_type' in state.answers and 'version' in state.answers:
        ReleaseMetadata.from_version(
            str(state.answers['release_type']), str(state.answers['version'])
        )

    args = type(
        'Args',
        (),
        {
            'dry_run': dry_run,
            'prompt_release_source_branch': prompt_release_source_branch,
            'release_source_branch': release_source_branch,
            'save_state': save_state,
            'require_previous_release': require_previous_release,
        },
    )()
    return collect_context(args, state)


def _all_steps() -> list[Step]:
    by_id: dict[str, Step] = {}
    for release_type in ('major', 'minor', 'patch'):
        for step in build_steps(
            release_type, include_backend=True, include_sdk=True
        ):
            by_id.setdefault(step.step_id, step)
    return list(by_id.values())


def _state_steps(
    state: ReleaseState,
) -> tuple[list[Step], ReleaseMetadata | None, list[str]]:
    metadata, errors = validate_answers(state.answers)
    if metadata is None:
        return [], None, errors
    include_backend = bool(state.answers.get('include_backend', True))
    include_sdk = bool(state.answers.get('include_sdk', True))
    return (
        build_steps(metadata.release_type, include_backend, include_sdk),
        metadata,
        validate_state(state),
    )


def _print_state_status(state: ReleaseState) -> None:
    steps, metadata, errors = _state_steps(state)
    typer.echo('answers:')
    if state.answers:
        for key in sorted(state.answers):
            typer.echo(f'  {key}: {state.answers[key]}')
    else:
        typer.echo('  (none)')

    if errors:
        typer.echo('errors:')
        for error in errors:
            typer.echo(f'  {error}')
        return

    next_step = None
    typer.echo('steps:')
    for step in steps:
        if state.is_done(step.step_id):
            status = 'done'
        elif next_step is None:
            status = 'next'
            next_step = step
        else:
            status = 'pending'
        typer.echo(f'  [{status}] {step.step_id}: {step.description}')

    typer.echo(f'resume: kfpr run --state-file {state.path}')
    if next_step and metadata:
        checklist = manual_checklist(next_step.step_id, metadata)
        if checklist:
            typer.echo(checklist)


def _steps_diagram() -> str:
    return """preflight
  |
  +-- patch --> prepare-patch-branch -> cherry-pick-prs -> merge-cherry-pick-pr
  |
  +-- major/minor --> prepare-release-branch
  |
update-version-tags -> merge-version-pr
  |
  +-- include-backend --> publish-images
  |
  +-- include-sdk --> create-sdk-tag -> publish-sdks
                      -> [split SDK: create-kfp-kubernetes-docs-branch]
                      -> confirm-rtd -> create-sdk-release
  |

create-backend-release -> sync-master -> confirm-website-and-slack"""


@run_app.callback(invoke_without_command=True)
def run(
    ctx: typer.Context,
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
    dry_run: bool = typer.Option(
        False, help='Print commands without executing.'
    ),
    force: bool = typer.Option(
        False, help='Resume an existing checkpoint without prompting.'
    ),
    release_type: Optional[str] = typer.Option(
        None, help='major, minor, or patch.'
    ),
    version: Optional[str] = typer.Option(None, help='Release version.'),
    fork_remote: Optional[str] = typer.Option(None, help='Fork remote.'),
    release_source_branch: Optional[str] = typer.Option(
        None, help='Branch to cut the release branch from.'
    ),
    patch_prs: Optional[str] = typer.Option(
        None, help='Comma-separated patch PRs.'
    ),
    include_backend: Optional[bool] = typer.Option(
        None,
        '--include-backend/--no-include-backend',
        help='Include backend/image steps.',
    ),
    include_sdk: Optional[bool] = typer.Option(
        None,
        '--include-sdk/--no-include-sdk',
        help='Include SDK steps.',
    ),
    skip_local_review: bool = typer.Option(
        False,
        '--skip-local-review',
        help='Skip local review prompts before git commit.',
    ),
    previous_release: Optional[str] = typer.Option(
        None,
        '--previous-release',
        help='Previous release tag for changelog generation.',
    ),
    previous_sdk_release: Optional[str] = typer.Option(
        None,
        '--previous-sdk-release',
        help='Previous SDK release tag for SDK notes.',
    ),
    release_target: Optional[str] = typer.Option(
        None,
        '--release-target',
        help='Explicit Git ref for release contents; resolved to a commit SHA.',
    ),
    qualification_pythons: Optional[str] = typer.Option(
        None,
        help='Deprecated: CI selects every supported Python minor from the release source.',
    ),
    security_policy: Optional[str] = typer.Option(
        None,
        help='Security policy: zero-fixable (default), approved-exceptions, or historical 2.17.3-approved-exceptions-v1.',
    ),
    security_exceptions: Optional[Path] = typer.Option(
        None,
        help='JSON exception file with exact findings, justifications and UTC expiries or an exact release bound; selects approved-exceptions.',
    ),
    image_qualification: Optional[ImageQualification] = typer.Option(
        None,
        help='Qualification scans run in upstream CI; local scanning is unsupported.',
    ),
) -> None:
    """Run the full checkpointed release flow."""
    if ctx.invoked_subcommand is not None:
        return
    if state_file.exists() and not force:
        _print_state_status(_load_state(state_file))
        if not typer.confirm(
            emphasize_prompt('Continue from this checkpoint?')
        ):
            typer.echo('Release flow aborted.')
            raise typer.Exit(1)

    context = _context(
        state_file,
        dry_run,
        release_type,
        version,
        fork_remote,
        patch_prs,
        include_backend,
        include_sdk,
        prompt_release_source_branch=True,
        release_source_branch=release_source_branch,
        skip_local_review=skip_local_review,
        previous_release=previous_release,
        previous_sdk_release=previous_sdk_release,
        release_target=release_target,
        qualification_pythons=qualification_pythons,
        security_policy=security_policy,
        security_exceptions=security_exceptions,
        image_qualification=image_qualification,
    )
    run_steps(context)
    typer.echo(f'Release flow complete for {context.metadata.tag}')


@app.command(
    'promote-latest', context_settings={'help_option_names': HELP_OPTION_NAMES}
)
def promote_latest(
    version: str = typer.Option(
        ..., help='Published stable backend release version.'
    ),
    dry_run: bool = typer.Option(
        False, help='Preview CI promotion without dispatch.'
    ),
) -> None:
    """Promote existing published image indexes through CI without
    rebuilding."""
    from .promotion import dispatch

    dispatch(version, dry_run=dry_run)


@app.command(
    'prepare-editorial',
    context_settings={'help_option_names': HELP_OPTION_NAMES},
)
def prepare_editorial(
    version: str = typer.Option(
        ..., help='Release version, without sdk- prefix.'
    ),
    base: str = typer.Option(
        ..., help='Explicit previous release tag or commit.'
    ),
    target: str = typer.Option(..., help='Exact release target ref.'),
    output: Path = typer.Option(
        ..., help='New directory for evidence and document templates.'
    ),
) -> None:
    """Create an offline editorial inventory and backend/SDK/upgrade templates."""
    try:
        root = Path(
            subprocess.check_output(
                ['git', 'rev-parse', '--show-toplevel'], text=True
            ).strip()
        )
        selected = resolve_range(root, base, target)
        inventory = build_inventory(root, selected, version)
        write_bundle(root, inventory, output)
    except (ValueError, OSError, subprocess.CalledProcessError) as error:
        raise typer.BadParameter(str(error)) from error
    for warning in review_warnings(inventory):
        typer.echo(f'Warning: {warning}')
    typer.echo(
        f'Editorial bundle: {output}. Review inventory.json, then use render-editorial.'
    )


@app.command(
    'render-editorial',
    context_settings={'help_option_names': HELP_OPTION_NAMES},
)
def render_editorial(
    inventory_path: Path = typer.Argument(..., help='Reviewed inventory.json.'),
    output: Path = typer.Option(
        ..., help='New output directory; existing drafts are never overwritten.'
    ),
) -> None:
    """Render reviewed benefits and shared facts without publishing or changing state."""
    try:
        root = Path(
            subprocess.check_output(
                ['git', 'rev-parse', '--show-toplevel'], text=True
            ).strip()
        )
        inventory = json.loads(inventory_path.read_text(encoding='utf-8'))
        write_bundle(root, inventory, output)
    except (ValueError, OSError, subprocess.CalledProcessError) as error:
        raise typer.BadParameter(str(error)) from error
    for warning in review_warnings(inventory):
        typer.echo(f'Warning: {warning}')
    typer.echo(
        f'Review rendered documents in {output}; this command does not publish them.'
    )


@app.command(
    'validate-notes', context_settings={'help_option_names': HELP_OPTION_NAMES}
)
def validate_notes_command(
    path: Path = typer.Argument(..., help='Markdown release notes to review.'),
    product: Optional[str] = typer.Option(
        None, help='Optional backend, sdk, or upgrade editorial checks.'
    ),
) -> None:
    """Report advisory editorial warnings without changing release notes."""
    try:
        notes = path.read_text(encoding='utf-8')
    except (OSError, UnicodeError) as error:
        raise typer.BadParameter(
            f'Cannot read release notes: {error}', param_hint='PATH'
        ) from error
    try:
        warnings = validate_notes(notes, product=product)
    except ValueError as error:
        raise typer.BadParameter(str(error), param_hint='--product') from error
    if warnings:
        for warning in warnings:
            typer.echo(f'Warning: {warning}')
    else:
        typer.echo('No editorial warnings found.')


@app.command(
    'regenerate-notes',
    context_settings={'help_option_names': HELP_OPTION_NAMES},
)
def regenerate_notes(
    product: NotesProduct = typer.Option(
        ..., help='Release notes to regenerate: backend or sdk.'
    ),
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
    version: Optional[str] = typer.Option(None, help='Release version.'),
    previous_release: Optional[str] = typer.Option(
        None, help='Previous backend release tag.'
    ),
    previous_sdk_release: Optional[str] = typer.Option(
        None, help='Previous SDK release tag.'
    ),
    release_target: Optional[str] = typer.Option(
        None, help='Git ref for release contents.'
    ),
) -> None:
    """Regenerate a local notes preview while preserving handwritten edits."""
    state = _load_state(state_file)
    answers = state.answers
    selected_version = (
        version if version is not None else answers.get('version')
    )
    if not selected_version:
        raise typer.BadParameter(
            'Provide a release version or an existing checkpoint.',
            param_hint='--version',
        )
    try:
        metadata = ReleaseMetadata.from_version(
            str(answers.get('release_type', 'patch')), str(selected_version)
        )
    except ValueError as error:
        raise typer.BadParameter(str(error), param_hint='--version') from error
    root = Path(
        subprocess.check_output(
            ['git', 'rev-parse', '--show-toplevel'], text=True
        ).strip()
    )
    context = ReleaseContext(
        root=root,
        state=state,
        runner=CommandRunner(dry_run=True),
        metadata=metadata,
        fork_remote=str(answers.get('fork_remote', '')),
        include_backend=product == NotesProduct.backend,
        include_sdk=product == NotesProduct.sdk,
        previous_release=previous_release
        if previous_release is not None
        else str(answers.get('previous_release', '')),
        previous_sdk_release=previous_sdk_release
        if previous_sdk_release is not None
        else str(answers.get('previous_sdk_release', '')),
        release_target=release_target
        if release_target is not None
        else str(answers.get('release_target', '')),
    )
    if not regenerate_release_notes(context, product.value):
        typer.echo(
            'Resolve the conflict markers in the preview, then review it with the individual release preview command before publishing.'
        )
        raise typer.Exit(1)
    typer.echo(
        'Release notes preview regenerated. Review it before publication.'
    )


@app.command('steps', context_settings={'help_option_names': HELP_OPTION_NAMES})
def list_steps(
    release_type: str = typer.Option('patch', help='major, minor, or patch.'),
    include_backend: bool = typer.Option(
        True, help='Include backend/image steps.'
    ),
    include_sdk: bool = typer.Option(True, help='Include SDK steps.'),
    diagram: bool = typer.Option(
        False, help='Print an ASCII release flow diagram.'
    ),
) -> None:
    """List steps for a selected release flow."""
    if diagram:
        typer.echo(_steps_diagram())
        return
    metadata = ReleaseMetadata.from_version(release_type, '0.0.0')
    for step in build_steps(release_type, include_backend, include_sdk):
        marker = ' [manual]' if manual_checklist(step.step_id, metadata) else ''
        typer.echo(f'{step.step_id}{marker}: {step.description}')


@app.command(context_settings={'help_option_names': HELP_OPTION_NAMES})
def status(
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
) -> None:
    """Print checkpoint answers, step status, and resume command."""
    _print_state_status(_load_state(state_file))


@app.command('next', context_settings={'help_option_names': HELP_OPTION_NAMES})
def next_command(
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
    dry_run: bool = typer.Option(
        False, help='Print commands without executing.'
    ),
    skip_local_review: bool = typer.Option(
        False,
        '--skip-local-review',
        help='Skip local review prompts before git commit.',
    ),
    previous_release: Optional[str] = typer.Option(
        None,
        '--previous-release',
        help='Previous release tag for changelog generation.',
    ),
    previous_sdk_release: Optional[str] = typer.Option(
        None,
        '--previous-sdk-release',
        help='Previous SDK release tag for SDK notes.',
    ),
    release_target: Optional[str] = typer.Option(
        None,
        '--release-target',
        help='Explicit Git ref for release contents; resolved to a commit SHA.',
    ),
) -> None:
    """Run the next incomplete checkpoint step."""
    state = _load_state(state_file)
    steps, _, errors = _state_steps(state)
    if errors:
        for error in errors:
            typer.echo(error)
        raise typer.Exit(1)
    next_step = next(
        (step for step in steps if not state.is_done(step.step_id)), None
    )
    if next_step is None:
        typer.echo('No pending release steps.')
        return
    context = _context(
        state_file,
        dry_run,
        None,
        None,
        None,
        None,
        None,
        None,
        skip_local_review=skip_local_review,
        previous_release=previous_release,
        previous_sdk_release=previous_sdk_release,
        release_target=release_target,
        save_state=True,
    )
    typer.echo(f'Running step: {next_step.description}')
    STEP_HANDLERS[next_step.step_id](context)
    if not dry_run:
        context.state.mark_done(next_step.step_id)
        context.state.save()


@app.command(context_settings={'help_option_names': HELP_OPTION_NAMES})
def clear(
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
) -> None:
    """Delete the checkpoint state file."""
    if state_file.exists():
        state_file.unlink()
        typer.echo(f'deleted checkpoint: {state_file}')
    else:
        typer.echo(f'no checkpoint to delete: {state_file}')


@app.command(
    'validate-state', context_settings={'help_option_names': HELP_OPTION_NAMES}
)
def validate_state_command(
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
) -> None:
    """Validate a checkpoint file before resuming."""
    errors = validate_state(_load_state(state_file))
    if errors:
        for error in errors:
            typer.echo(error)
        raise typer.Exit(1)
    typer.echo(f'{state_file} is valid')


@app.command('done', context_settings={'help_option_names': HELP_OPTION_NAMES})
def done_command(
    step_id: str,
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
) -> None:
    """Mark one release step complete in the checkpoint."""
    if step_id not in STEP_HANDLERS:
        raise typer.BadParameter(f'unknown step: {step_id}')
    state = _load_state(state_file)
    state.mark_done(step_id)
    state.save()
    typer.echo(f'marked done: {step_id}')


@app.command(
    'reset-step', context_settings={'help_option_names': HELP_OPTION_NAMES}
)
def reset_step_command(
    step_id: str,
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
) -> None:
    """Remove one release step from the completed checkpoint list."""
    if step_id not in STEP_HANDLERS:
        raise typer.BadParameter(f'unknown step: {step_id}')
    state = _load_state(state_file)
    state.reset_step(step_id)
    state.save()
    typer.echo(f'reset step: {step_id}')


@app.command('goto', context_settings={'help_option_names': HELP_OPTION_NAMES})
def goto_command(
    step_id: str,
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
) -> None:
    """Mark all steps before STEP_ID complete so the flow resumes there."""
    state = _load_state(state_file)
    steps, _, errors = _state_steps(state)
    if errors:
        for error in errors:
            typer.echo(error)
        raise typer.Exit(1)
    step_ids = [step.step_id for step in steps]
    if step_id not in step_ids:
        typer.echo(f'step is not in this release flow: {step_id}')
        raise typer.Exit(1)
    state.completed_steps = []
    for prior_step_id in step_ids[: step_ids.index(step_id)]:
        state.mark_done(prior_step_id)
    state.save()
    typer.echo(f'next step: {step_id}')


@app.command(context_settings={'help_option_names': HELP_OPTION_NAMES})
def doctor(
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
    release_type: Optional[str] = typer.Option(
        None, help='major, minor, or patch.'
    ),
    version: Optional[str] = typer.Option(None, help='Release version.'),
    fork_remote: Optional[str] = typer.Option(None, help='Fork remote.'),
) -> None:
    """Run non-mutating preflight diagnostics."""
    state = _load_state(state_file)
    answers = dict(state.answers)
    if release_type is not None:
        answers['release_type'] = release_type
    if version is not None:
        answers['version'] = version
    if fork_remote is not None:
        answers['fork_remote'] = fork_remote
    root = Path.cwd()
    errors = doctor_errors(answers, root)
    if errors:
        for error in errors:
            typer.echo(error)
        raise typer.Exit(1)
    typer.echo('doctor passed')


@run_app.command(
    'watch-publish-images',
    context_settings={'help_option_names': HELP_OPTION_NAMES},
)
def watch_publish_images(
    state_file: Path = typer.Option(
        Path(STATE_FILE), help='Path to checkpoint state file.'
    ),
    dry_run: bool = typer.Option(
        False, help='Print commands without executing.'
    ),
) -> None:
    """Watch the latest publish-images workflow run without dispatching it."""
    state = _load_state(state_file)
    metadata, errors = validate_answers(state.answers)
    if metadata is None:
        for error in errors:
            typer.echo(error)
        raise typer.Exit(1)
    watch_latest_workflow_run(
        CommandRunner(dry_run=dry_run),
        'image-builds-release.yml',
        metadata.release_branch,
        fresh_only=False,
    )


def _make_step_command(step_id: str):

    def command(
        state_file: Path = typer.Option(
            Path(STATE_FILE), help='Path to checkpoint state file.'
        ),
        dry_run: bool = typer.Option(
            False, help='Print commands without executing.'
        ),
        release_type: Optional[str] = typer.Option(
            None, help='major, minor, or patch.'
        ),
        version: Optional[str] = typer.Option(None, help='Release version.'),
        fork_remote: Optional[str] = typer.Option(None, help='Fork remote.'),
        patch_prs: Optional[str] = typer.Option(
            None, help='Comma-separated patch PRs.'
        ),
        include_backend: Optional[bool] = typer.Option(
            None,
            '--include-backend/--no-include-backend',
            help='Include backend/image steps.',
        ),
        include_sdk: Optional[bool] = typer.Option(
            None,
            '--include-sdk/--no-include-sdk',
            help='Include SDK steps.',
        ),
        done: bool = typer.Option(
            False, '--done', help='Mark this step complete after success.'
        ),
        skip_local_review: bool = typer.Option(
            False,
            '--skip-local-review',
            help='Skip local review prompts before git commit.',
        ),
        previous_release: Optional[str] = typer.Option(
            None,
            '--previous-release',
            help='Previous release tag for changelog generation.',
        ),
        previous_sdk_release: Optional[str] = typer.Option(
            None,
            '--previous-sdk-release',
            help='Previous SDK release tag for SDK notes.',
        ),
        release_target: Optional[str] = typer.Option(
            None,
            '--release-target',
            help='Explicit Git ref for release contents; resolved to a commit SHA.',
        ),
        qualification_pythons: Optional[str] = typer.Option(
            None,
            help='Deprecated: CI selects every supported Python minor from the release source.',
        ),
        security_policy: Optional[str] = typer.Option(
            None,
            help='Security policy: zero-fixable (default), approved-exceptions, or historical 2.17.3-approved-exceptions-v1.',
        ),
        security_exceptions: Optional[Path] = typer.Option(
            None,
            help='JSON exception file with exact findings, justifications and UTC expiries or an exact release bound; selects approved-exceptions.',
        ),
        image_qualification: Optional[ImageQualification] = typer.Option(
            None,
            help='Qualification scans run in upstream CI; local scanning is unsupported.',
        ),
    ) -> None:
        if step_id != 'cherry-pick-prs' and patch_prs is None:
            patch_prs = 'not-needed'
        if include_backend is None:
            include_backend = True
        if include_sdk is None:
            include_sdk = True
        context = _context(
            state_file,
            dry_run,
            release_type,
            version,
            fork_remote,
            patch_prs,
            include_backend,
            include_sdk,
            skip_local_review=skip_local_review,
            previous_release=previous_release,
            previous_sdk_release=previous_sdk_release,
            release_target=release_target,
            qualification_pythons=qualification_pythons,
            security_policy=security_policy,
            security_exceptions=security_exceptions,
            image_qualification=image_qualification,
            save_state=done,
            require_previous_release=step_id
            in ('update-version-tags', 'sync-master'),
        )
        if (
            not dry_run
            and step_id
            in {
                'prepare-release-branch',
                'prepare-patch-branch',
                'cherry-pick-prs',
                'update-version-tags',
                'sync-master',
            }
            and context.runner.capture(
                ['git', 'status', '--short'], cwd=context.root
            ).strip()
        ):
            raise typer.BadParameter('git working tree is dirty')
        STEP_HANDLERS[step_id](context)
        if done:
            context.state.mark_done(step_id)
            context.state.save()

    command.__name__ = step_id.replace('-', '_')
    command.__doc__ = f'Run only the {step_id} release step.'
    return command


for _step in _all_steps():
    run_app.command(
        _step.step_id, context_settings={'help_option_names': HELP_OPTION_NAMES}
    )(_make_step_command(_step.step_id))

app.add_typer(run_app, name='run')

if __name__ == '__main__':
    app()
