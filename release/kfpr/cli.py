"""Typer CLI for Kubeflow Pipelines release automation."""

from pathlib import Path
from typing import Optional

import typer

from .core import (
    CommandRunner,
    STATE_FILE,
    ReleaseContext,
    ReleaseMetadata,
    ReleaseState,
    collect_context,
    doctor_errors,
    normalize_fork_remote,
    validate_answers,
    validate_state,
    watch_latest_workflow_run,
)
from .steps import STEP_HANDLERS, Step, build_steps, manual_checklist, run_steps

HELP_OPTION_NAMES = ['-h', '--help']

app = typer.Typer(help='Kubeflow Pipelines release automation.', context_settings={'help_option_names': HELP_OPTION_NAMES})


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
    save_state: bool = True,
) -> ReleaseContext:
  state = _load_state(state_file)
  _set_answer(state, 'release_type', release_type)
  _set_answer(state, 'version', version)
  _set_answer(state, 'fork_remote', normalize_fork_remote(fork_remote) if fork_remote is not None else None)
  _set_answer(state, 'patch_prs', patch_prs)
  _set_answer(state, 'include_backend', include_backend)
  _set_answer(state, 'include_sdk', include_sdk)

  if 'release_type' in state.answers and 'version' in state.answers:
    ReleaseMetadata.from_version(str(state.answers['release_type']), str(state.answers['version']))

  args = type(
      'Args',
      (),
      {
          'dry_run': dry_run,
          'prompt_release_source_branch': prompt_release_source_branch,
          'release_source_branch': release_source_branch,
          'save_state': save_state,
      },
  )()
  return collect_context(args, state)


def _all_steps() -> list[Step]:
  by_id: dict[str, Step] = {}
  for release_type in ('major', 'minor', 'patch'):
    for step in build_steps(release_type, include_backend=True, include_sdk=True):
      by_id.setdefault(step.step_id, step)
  return list(by_id.values())


def _state_steps(state: ReleaseState) -> tuple[list[Step], ReleaseMetadata | None, list[str]]:
  metadata, errors = validate_answers(state.answers)
  if metadata is None:
    return [], None, errors
  include_backend = bool(state.answers.get('include_backend', True))
  include_sdk = bool(state.answers.get('include_sdk', True))
  return build_steps(metadata.release_type, include_backend, include_sdk), metadata, validate_state(state)


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
  return '''preflight
  |
  +-- patch --> prepare-patch-branch -> cherry-pick-prs -> merge-cherry-pick-pr
  |
  +-- major/minor --> prepare-release-branch
  |
update-version-tags -> merge-version-pr
  |
  +-- include-backend --> publish-images
  |
  +-- include-sdk --> create-sdk-release -> publish-sdks -> create-kfp-kubernetes-docs-branch -> confirm-rtd
  |
create-backend-release -> sync-master -> confirm-website-and-slack'''


@app.command(context_settings={'help_option_names': HELP_OPTION_NAMES})
def run(
    state_file: Path = typer.Option(Path(STATE_FILE), help='Path to checkpoint state file.'),
    dry_run: bool = typer.Option(False, help='Print commands without executing.'),
    force: bool = typer.Option(False, help='Resume an existing checkpoint without prompting.'),
    release_type: Optional[str] = typer.Option(None, help='major, minor, or patch.'),
    version: Optional[str] = typer.Option(None, help='Release version.'),
    fork_remote: Optional[str] = typer.Option(None, help='Fork remote.'),
    release_source_branch: Optional[str] = typer.Option(None, help='Branch to cut the release branch from.'),
    patch_prs: Optional[str] = typer.Option(None, help='Comma-separated patch PRs.'),
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
) -> None:
  """Run the full checkpointed release flow."""
  if state_file.exists() and not force:
    _print_state_status(_load_state(state_file))
    if not typer.confirm('Continue from this checkpoint?'):
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
  )
  run_steps(context)
  typer.echo(f'Release flow complete for {context.metadata.tag}')


@app.command('steps', context_settings={'help_option_names': HELP_OPTION_NAMES})
def list_steps(
    release_type: str = typer.Option('patch', help='major, minor, or patch.'),
    include_backend: bool = typer.Option(True, help='Include backend/image steps.'),
    include_sdk: bool = typer.Option(True, help='Include SDK steps.'),
    diagram: bool = typer.Option(False, help='Print an ASCII release flow diagram.'),
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
def status(state_file: Path = typer.Option(Path(STATE_FILE), help='Path to checkpoint state file.')) -> None:
  """Print checkpoint answers, step status, and resume command."""
  _print_state_status(_load_state(state_file))


@app.command(context_settings={'help_option_names': HELP_OPTION_NAMES})
def clear(state_file: Path = typer.Option(Path(STATE_FILE), help='Path to checkpoint state file.')) -> None:
  """Delete the checkpoint state file."""
  if state_file.exists():
    state_file.unlink()
    typer.echo(f'deleted checkpoint: {state_file}')
  else:
    typer.echo(f'no checkpoint to delete: {state_file}')


@app.command('validate-state', context_settings={'help_option_names': HELP_OPTION_NAMES})
def validate_state_command(state_file: Path = typer.Option(Path(STATE_FILE), help='Path to checkpoint state file.')) -> None:
  """Validate a checkpoint file before resuming."""
  errors = validate_state(_load_state(state_file))
  if errors:
    for error in errors:
      typer.echo(error)
    raise typer.Exit(1)
  typer.echo(f'{state_file} is valid')


@app.command('mark-done', context_settings={'help_option_names': HELP_OPTION_NAMES})
def mark_done_command(
    step_id: str,
    state_file: Path = typer.Option(Path(STATE_FILE), help='Path to checkpoint state file.'),
) -> None:
  """Mark one release step complete in the checkpoint."""
  if step_id not in STEP_HANDLERS:
    raise typer.BadParameter(f'unknown step: {step_id}')
  state = _load_state(state_file)
  state.mark_done(step_id)
  state.save()
  typer.echo(f'marked done: {step_id}')


@app.command('reset-step', context_settings={'help_option_names': HELP_OPTION_NAMES})
def reset_step_command(
    step_id: str,
    state_file: Path = typer.Option(Path(STATE_FILE), help='Path to checkpoint state file.'),
) -> None:
  """Remove one release step from the completed checkpoint list."""
  if step_id not in STEP_HANDLERS:
    raise typer.BadParameter(f'unknown step: {step_id}')
  state = _load_state(state_file)
  state.reset_step(step_id)
  state.save()
  typer.echo(f'reset step: {step_id}')


@app.command(context_settings={'help_option_names': HELP_OPTION_NAMES})
def doctor(
    state_file: Path = typer.Option(Path(STATE_FILE), help='Path to checkpoint state file.'),
    release_type: Optional[str] = typer.Option(None, help='major, minor, or patch.'),
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


@app.command(context_settings={'help_option_names': HELP_OPTION_NAMES})
def watch_publish_images(
    state_file: Path = typer.Option(Path(STATE_FILE), help='Path to checkpoint state file.'),
    dry_run: bool = typer.Option(False, help='Print commands without executing.'),
) -> None:
  """Watch the latest publish-images workflow run without dispatching it."""
  state = _load_state(state_file)
  metadata, errors = validate_answers(state.answers)
  if metadata is None:
    for error in errors:
      typer.echo(error)
    raise typer.Exit(1)
  watch_latest_workflow_run(CommandRunner(dry_run=dry_run), 'image-builds-release.yml', metadata.release_branch)


def _make_step_command(step_id: str):

  def command(
      state_file: Path = typer.Option(Path(STATE_FILE), help='Path to checkpoint state file.'),
      dry_run: bool = typer.Option(False, help='Print commands without executing.'),
      release_type: Optional[str] = typer.Option(None, help='major, minor, or patch.'),
      version: Optional[str] = typer.Option(None, help='Release version.'),
      fork_remote: Optional[str] = typer.Option(None, help='Fork remote.'),
      patch_prs: Optional[str] = typer.Option(None, help='Comma-separated patch PRs.'),
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
      mark_done: bool = typer.Option(False, help='Mark this step complete after success.'),
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
        save_state=mark_done,
    )
    STEP_HANDLERS[step_id](context)
    if mark_done:
      context.state.mark_done(step_id)
      context.state.save()

  command.__name__ = step_id.replace('-', '_')
  command.__doc__ = f'Run only the {step_id} release step.'
  return command


for _step in _all_steps():
  app.command(_step.step_id, context_settings={'help_option_names': HELP_OPTION_NAMES})(_make_step_command(_step.step_id))


if __name__ == '__main__':
  app()
