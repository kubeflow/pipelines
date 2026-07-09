"""Core release metadata and utilities."""

import argparse
import datetime
import json
import os
import re
import shutil
import subprocess
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, List, Optional

# Constants
REPO = 'kubeflow/pipelines'
STATE_FILE = 'release-state.json'
VERSION_RE = r'^(\d+)\.(\d+)\.(\d+)$'
ANSI_BOLD = '\033[1m'
ANSI_UNDERLINE = '\033[4m'
ANSI_RESET = '\033[0m'
URL_RE = re.compile(r'https?://\S+')


def underline_links(text: str) -> str:
  """Underline URLs in terminal output."""
  return URL_RE.sub(lambda match: f'{ANSI_UNDERLINE}{match.group(0)}{ANSI_RESET}', text)


def emphasize_prompt(text: str) -> str:
  """Bold a prompt that needs user input."""
  return f'{ANSI_BOLD}{text}{ANSI_RESET}'


@dataclass(frozen=True)
class ReleaseMetadata:
  """Immutable release metadata derived from version and release type."""

  release_type: str
  version: str
  major: int
  minor: int
  patch: int
  tag: str
  sdk_tag: str
  release_branch: str
  patch_branch: Optional[str]

  @classmethod
  def from_version(cls, release_type: str, version: str) -> 'ReleaseMetadata':
    """Derive release metadata from release type and version.

    Args:
      release_type: One of 'major', 'minor', or 'patch'.
      version: Semantic version string in MAJOR.MINOR.PATCH format.

    Returns:
      ReleaseMetadata instance with all derived fields.

    Raises:
      ValueError: If release_type is invalid, version format is invalid,
        or version constraints are violated (major must be X.0.0, minor must
        be X.Y.0).
    """
    if release_type not in ('major', 'minor', 'patch'):
      raise ValueError(
          f'release_type must be major, minor, or patch; got {release_type}'
      )

    match = re.match(VERSION_RE, version)
    if not match:
      raise ValueError(f'version must be in MAJOR.MINOR.PATCH format; got {version}')

    major, minor, patch = int(match.group(1)), int(match.group(2)), int(match.group(3))

    if release_type == 'major' and (minor != 0 or patch != 0):
      raise ValueError(f'major release version must be X.0.0; got {version}')

    if release_type == 'minor' and patch != 0:
      raise ValueError(f'minor release version must be X.Y.0; got {version}')

    tag = version
    sdk_tag = f'sdk-{version}'

    if release_type == 'major':
      release_branch = f'release-{major}.0'
      patch_branch = None
    elif release_type == 'minor':
      release_branch = f'release-{major}.{minor}'
      patch_branch = None
    else:  # patch
      release_branch = f'release-{major}.{minor}'
      patch_branch = f'release-{major}.{minor}.{patch}'

    return cls(
        release_type=release_type,
        version=version,
        major=major,
        minor=minor,
        patch=patch,
        tag=tag,
        sdk_tag=sdk_tag,
        release_branch=release_branch,
        patch_branch=patch_branch,
    )


@dataclass
class ReleaseState:
  """Checkpoint for release process state.

  Stores answers to prompts and completed steps in JSON format.
  """

  path: Path
  answers: Dict[str, str] = field(default_factory=dict)
  completed_steps: List[str] = field(default_factory=list)

  @classmethod
  def load(cls, path: Path) -> 'ReleaseState':
    """Load state from JSON file.

    Args:
      path: Path to state file.

    Returns:
      ReleaseState instance with loaded data.

    Raises:
      ValueError: If the state file contains malformed JSON.
    """
    if not path.exists():
      return cls(path)
    try:
      with open(path) as f:
        data = json.load(f)
    except json.JSONDecodeError as e:
      raise ValueError(
          f'Invalid state file: {path} contains malformed JSON. '
          f'Please fix or delete the file. Details: {e}'
      ) from e
    if not isinstance(data, dict):
      return cls(path, answers={'__schema_error__': 'state file must be an object'})
    answers = data.get('answers', {})
    completed_steps = data.get('completed_steps', [])
    return cls(
        path=path,
        answers=answers if isinstance(answers, dict) else {'__schema_error__': 'answers must be an object'},
        completed_steps=completed_steps if isinstance(completed_steps, list) else ['__schema_error__:completed_steps must be a list'],
    )

  def save(self) -> None:
    """Save state to JSON file atomically.
    
    Writes to a temporary file with .tmp suffix and then replaces the
    original file to ensure atomicity and avoid corruption on partial writes.
    """
    data = {
        'answers': self.answers,
        'completed_steps': self.completed_steps,
    }
    self.path.parent.mkdir(parents=True, exist_ok=True)
    tmp_path = self.path.with_suffix(self.path.suffix + '.tmp')
    try:
      with open(tmp_path, 'w') as f:
        json.dump(data, f, indent=2)
      tmp_path.replace(self.path)
    except OSError:
      try:
        tmp_path.unlink(missing_ok=True)
      except OSError:
        pass
      raise

  def is_done(self, step_id: str) -> bool:
    """Check if a step has been marked as done.

    Args:
      step_id: Identifier for the step.

    Returns:
      True if the step is in completed_steps, False otherwise.
    """
    return step_id in self.completed_steps

  def mark_done(self, step_id: str) -> None:
    """Mark a step as done.

    Args:
      step_id: Identifier for the step.
    """
    if step_id not in self.completed_steps:
      self.completed_steps.append(step_id)

  def reset_step(self, step_id: str) -> None:
    """Remove a step from the completed checkpoint list."""
    self.completed_steps = [completed for completed in self.completed_steps if completed != step_id]


class CommandRunner:
  """Runs commands with optional dry-run mode."""

  def __init__(self, dry_run: bool = False):
    self.dry_run = dry_run
    self.commands: list[list[str]] = []

  def run(self, command: list[str], cwd: Path | None = None, check: bool = True) -> subprocess.CompletedProcess[str]:
    """Run a command, recording it and optionally executing it.

    Args:
     command: Command as a list of strings.
     cwd: Working directory for command execution.
     check: Whether to raise on non-zero exit code.

    Returns:
     CompletedProcess instance.
    """
    self.commands.append(command)
    print(f'+ {" ".join(command)}')
    if self.dry_run:
      return subprocess.CompletedProcess(command, 0, '', '')
    return subprocess.run(command, cwd=cwd, check=check, text=True)

  def capture(self, command: list[str], cwd: Path | None = None) -> str:
    """Capture command output, recording command and optionally executing it.

    Args:
     command: Command as a list of strings.
     cwd: Working directory for command execution.

    Returns:
     Command output as string (empty string in dry-run mode).
    """
    self.commands.append(command)
    print(f'+ {" ".join(command)}')
    if self.dry_run:
      return ''
    return subprocess.check_output(command, cwd=cwd, text=True).strip()


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


def image_workflow_command(metadata: ReleaseMetadata) -> list[str]:
  """Build command to trigger image-builds-release.yml workflow.

  Args:
    metadata: Release metadata for version and branch info.

  Returns:
    Command list for gh workflow run with appropriate parameters.
  """
  return [
      'gh',
      'workflow',
      'run',
      'image-builds-release.yml',
      '--ref',
      metadata.release_branch,
      '-f',
      f'src_branch={metadata.release_branch}',
      '-f',
      f'target_tag={metadata.tag}',
      '-f',
      'overwrite_imgs=false',
      '-f',
      'set_latest=true',
      '-f',
      'add_sha_tag=true',
      '-f',
      'dry_run=false',
  ]


def sdk_workflow_command(metadata: ReleaseMetadata, packages: str = 'all') -> list[str]:
  """Build command to trigger publish-packages.yml workflow.

  Args:
    metadata: Release metadata for tag and branch info.

  Returns:
    Command list for gh workflow run with appropriate parameters.
  """
  return [
      'gh',
      'workflow',
      'run',
      'publish-packages.yml',
      '--ref',
      metadata.release_branch,
      '-f',
      f'tag={metadata.sdk_tag}',
      '-f',
      f'packages={packages}',
      '-f',
      'dry_run=false',
  ]


def kfp_requirements_command() -> list[str]:
  """Build command for regenerating sdk/python requirements.

  Returns:
    Command list for pip-compile with local unpublished package dists.
  """
  return [
      'bash',
      '-c',
      '''
GIT_ROOT="$(git rev-parse --show-toplevel)"
pip-compile --no-emit-find-links --no-header --no-emit-index-url requirements.in \
  --find-links="$GIT_ROOT/api/v2alpha1/python/dist" \
  --find-links="$GIT_ROOT/backend/api/v2beta1/python_http_client/dist" > requirements.txt
'''.strip(),
  ]


def kubernetes_requirements_command() -> list[str]:
  """Build command for regenerating kubernetes_platform/python requirements.

  Returns:
    Command list for pip-compile with local unpublished package dists.
  """
  return [
      'bash',
      '-c',
      '''
GIT_ROOT="$(git rev-parse --show-toplevel)"
pip-compile --no-emit-find-links --no-header --no-emit-index-url requirements.in \
  --find-links="$GIT_ROOT/sdk/python/dist" \
  --find-links="$GIT_ROOT/backend/api/v2beta1/python_http_client/dist" \
  --find-links="$GIT_ROOT/api/v2alpha1/python/dist" > requirements.txt
'''.strip(),
  ]


def release_version_bump_command(root: Path, release_branch: str, previous_release: str, image_tag: str | None = None) -> list[str]:
  """Build command that runs release version updates in the release container."""
  image_tag = image_tag or release_branch
  repo_path = '/go/src/github.com/kubeflow/pipelines'
  script = f'''
set -e
REPO_ROOT={repo_path}
PREVIOUS_RELEASE={previous_release}
TAG_NAME="$(cat "$REPO_ROOT/VERSION")"
if [ -z "$TAG_NAME" ]; then
  echo "ERROR: $REPO_ROOT/VERSION is empty" >&2
  exit 1
fi
normalize_node_version() {{
  printf '%s\\n' "${{1#v}}"
}}
REQUIRED_NODE_VERSION="$(normalize_node_version "$(tr -d '\\r\\n' < "$REPO_ROOT/frontend/.nvmrc")")"
echo "The following tools are needed when releasing KFP:"
echo "node==${{REQUIRED_NODE_VERSION}}"
which node >/dev/null || (echo "node not found in PATH, recommend install via https://github.com/nvm-sh/nvm#installing-and-updating" && exit 1)
ACTUAL_NODE_VERSION="$(normalize_node_version "$(node -v)")"
test "${{ACTUAL_NODE_VERSION}}" = "${{REQUIRED_NODE_VERSION}}" || (echo "node version should match ${{REQUIRED_NODE_VERSION}}" && exit 1)
echo "jq>=1.6"
which jq >/dev/null || (echo "jq not found in PATH" && exit 1)
echo "yq>=3.3 <4.0"
which yq >/dev/null || (echo "yq not found in PATH" && exit 1)
yq -V | grep 3. || (echo "yq version 3.x should be used" && exit 1)
echo "java>=11"
which java >/dev/null || (echo "java not found in PATH" && exit 1)
echo "python>3"
which python3 >/dev/null || (echo "python not found in PATH" && exit 1)
python3 -c "import setuptools" || (echo "setuptools should be installed in python" && exit 1)
echo "go"
which go >/dev/null || (echo "go not found in PATH" && exit 1)
go_path=$(go env GOPATH)
echo "$PATH" | grep "${{go_path}}/bin" >/dev/null || (echo "\\$GOPATH/bin: ${{go_path}}/bin should be in PATH" && exit 1)
echo "All tools installed"
cd "$REPO_ROOT"
git-cliff -c cliff.toml --tag "$TAG_NAME" --prepend CHANGELOG.md "$PREVIOUS_RELEASE..$TAG_NAME"
"$REPO_ROOT/manifests/gcp_marketplace/hack/release.sh" "$TAG_NAME"
"$REPO_ROOT/manifests/kustomize/hack/release.sh" "$TAG_NAME"
export API_VERSION=v1beta1
"$REPO_ROOT/backend/api/hack/generator.sh"
"$REPO_ROOT/backend/api/build_kfp_server_api_python_package.sh"
export API_VERSION=v2beta1
"$REPO_ROOT/backend/api/hack/generator.sh"
"$REPO_ROOT/backend/api/build_kfp_server_api_python_package.sh"
'''.strip()
  return [
      'docker',
      'run',
      '--rm',
      '--user',
      f'{os.getuid()}:{os.getgid()}',
      '--mount',
      f'type=bind,source={root},target={repo_path}',
      f'ghcr.io/kubeflow/kfp-release:{image_tag}',
      '/bin/bash',
      '-c',
      script,
  ]


def kfp_kubernetes_docs_build_command(root: Path, release_branch: str) -> list[str]:
  """Build command that regenerates the kfp-kubernetes Python package without Make."""
  repo_path = '/go/src/github.com/kubeflow/pipelines'
  script = f'''
set -e
cd "{repo_path}"
rm -rf kubernetes_platform/python/build
rm -rf kubernetes_platform/python/dist
rm -rf kubernetes_platform/python/kfp_kubernetes.egg-info
rm -f kubernetes_platform/python/kfp/kubernetes/kubernetes_executor_config_pb2.py
rm -rf /tmp/protobuf-src
mkdir -p api/v2alpha1/google/rpc
wget -qO api/v2alpha1/google/rpc/status.proto https://raw.githubusercontent.com/googleapis/googleapis/fecd7d35f46753b45bf4519f6342495a181740c9/google/rpc/status.proto
git clone --depth 1 --branch v26.0 https://github.com/protocolbuffers/protobuf.git /tmp/protobuf-src
mkdir -p api/v2alpha1/google/protobuf
cp /tmp/protobuf-src/src/google/protobuf/*.proto api/v2alpha1/google/protobuf/
cd "{repo_path}/kubernetes_platform/python"
python3 -m pip install --user --break-system-packages -r requirements.txt
python3 generate_proto.py
python3 setup.py sdist
pip wheel --no-deps dist/*.tar.gz -w dist
'''.strip()
  return [
      'docker',
      'run',
      '--rm',
      '--user',
      f'{os.getuid()}:{os.getgid()}',
      '-e',
      'HOME=/tmp',
      '--mount',
      f'type=bind,source={root},target={repo_path}',
      f'ghcr.io/kubeflow/kfp-api-generator:{release_branch}',
      '/bin/bash',
      '-c',
      script,
  ]


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
  if re.match(r'^git@github\.com:[^/]+/[^/]+(?:\.git)?$', fork_remote):
    return fork_remote
  if re.match(r'^https://github\.com/[^/]+/[^/]+(?:\.git)?$', fork_remote):
    return fork_remote
  if re.match(r'^[A-Za-z0-9][A-Za-z0-9-]*$', fork_remote):
    return f'https://github.com/{fork_remote}/pipelines.git'
  raise ValueError(
      f'Cannot parse fork_remote: {fork_remote}. '
      'Use a GitHub username, git@github.com:USER/pipelines.git, '
      'or https://github.com/USER/pipelines.git'
  )


def validate_answers(answers: dict[str, object], require_all: bool = True) -> tuple[ReleaseMetadata | None, list[str]]:
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
      metadata = ReleaseMetadata.from_version(str(release_type), str(version))
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
  errors = []
  schema_error = state.answers.get('__schema_error__')
  if schema_error:
    errors.append(str(schema_error))
  if not all(isinstance(step, str) for step in state.completed_steps):
    errors.append('completed_steps must be a list of strings')
  for completed_step in state.completed_steps:
    if isinstance(completed_step, str) and completed_step.startswith('__schema_error__:'):
      errors.append(completed_step.split(':', 1)[1])
  if errors:
    return errors

  metadata, errors = validate_answers(state.answers)
  if metadata is None:
    return errors

  include_backend = bool(state.answers.get('include_backend', True))
  include_sdk = bool(state.answers.get('include_sdk', True))
  from .steps import build_steps
  step_ids = {step.step_id for step in build_steps(metadata.release_type, include_backend, include_sdk)}
  for completed_step in state.completed_steps:
    if completed_step not in step_ids:
      errors.append(f'unknown completed step: {completed_step}')
  return errors


def doctor_errors(answers: dict[str, object], root: Path) -> list[str]:
  """Return release preflight diagnostics without mutating state."""
  _, errors = validate_answers(answers)
  for tool in ['git', 'gh', 'docker', 'python3']:
    if shutil.which(tool) is None:
      errors.append(f'missing required tool: {tool}')

  if shutil.which('gh') is not None:
    auth = subprocess.run(['gh', 'auth', 'status'], check=False, capture_output=True, text=True)
    if auth.returncode != 0:
      errors.append('gh auth status failed')

  if shutil.which('git') is not None:
    status = subprocess.run(['git', 'status', '--short'], cwd=root, check=False, capture_output=True, text=True)
    if status.returncode != 0:
      errors.append('git status failed')
    elif status.stdout.strip():
      errors.append('git working tree is dirty')
  return errors


def wait_for_pr_merge(runner: CommandRunner, pr_url: str) -> None:
  """Poll PR status until merged.

  Args:
    runner: CommandRunner instance for executing gh commands.
    pr_url: URL of the pull request to monitor.
  """
  print(underline_links(f'Waiting for PR merge: {pr_url}'))
  if runner.dry_run:
    print('[dry-run] PR merge check skipped')
    return
  while True:
    state = runner.capture(['gh', 'pr', 'view', pr_url, '--json', 'state,mergeStateStatus,url', '--jq', '.state + " " + .mergeStateStatus'])
    if state.startswith('MERGED '):
      return
    if 'DIRTY' in state or 'UNKNOWN' in state:
      print(f'PR is not ready: {state}')
    time.sleep(60)


def watch_pr_ci(runner: CommandRunner, pr_url: str) -> None:
  """Poll PR checks until one fails or all reported checks complete.

  Args:
    runner: CommandRunner instance for executing gh commands.
    pr_url: URL of the pull request to monitor.
  """
  print(underline_links(f'Watching PR CI: {pr_url}'))
  if runner.dry_run:
    print('[dry-run] PR CI watch skipped')
    return
  reported_check_urls = set()
  while True:
    checks = json.loads(
        runner.capture([
            'gh',
            'pr',
            'view',
            pr_url,
            '--json',
            'statusCheckRollup',
            '--jq',
            '.statusCheckRollup',
        ])
    )
    if not checks:
      print('No CI gates reported yet')
      time.sleep(60)
      continue
    latest_checks = {}
    for check in checks:
      name = check.get('name') or check.get('workflowName') or check.get('context') or 'unknown'
      timestamp = check.get('startedAt') or check.get('completedAt') or ''
      previous = latest_checks.get(name)
      if previous is None or timestamp >= previous[0]:
        latest_checks[name] = (timestamp, check)
    completed = 0
    for check in [item[1] for item in latest_checks.values()]:
      name = check.get('name') or check.get('workflowName') or check.get('context') or 'unknown'
      status = check.get('status') or check.get('state')
      conclusion = check.get('conclusion')
      details_url = check.get('detailsUrl')
      if conclusion and conclusion not in ('SUCCESS', 'NEUTRAL', 'SKIPPED') or status in ('FAILURE', 'ERROR'):
        suffix = f': {details_url}' if details_url else ''
        raise RuntimeError(f'{name} failed{suffix}')
      if status == 'COMPLETED' or status in ('SUCCESS', 'NEUTRAL', 'SKIPPED'):
        completed += 1
      elif details_url and details_url not in reported_check_urls:
        reported_check_urls.add(details_url)
        print(underline_links(f'Watching check: {name}: {details_url}'))
    if completed == len(latest_checks):
      print(f'All PR CI gates completed successfully ({completed}/{len(latest_checks)})')
      return
    print(f'PR CI still running ({completed}/{len(latest_checks)} complete)')
    time.sleep(60)


def watch_latest_workflow_run(runner: CommandRunner, workflow: str, branch: str) -> None:
  """Watch the latest workflow run for a given workflow and branch.

  Args:
    runner: CommandRunner instance for executing gh commands.
    workflow: Workflow filename to watch.
    branch: Branch name to filter runs by.
  """
  if runner.dry_run:
    print(f'[dry-run] Watching latest workflow run skipped: {workflow} on {branch}')
    return
  started_at = datetime.datetime.fromtimestamp(time.time() - 10, datetime.UTC)
  command = [
      'gh',
      'run',
      'list',
      '--workflow',
      workflow,
      '--branch',
      branch,
      '--limit',
      '1',
      '--json',
      'databaseId,url,createdAt',
      '--jq',
      '.[0] | [.databaseId, .url, .createdAt] | @tsv',
  ]
  run_info = ''
  for _ in range(12):
    candidate = runner.capture(command)
    if candidate:
      _, _, created_at = candidate.split('\t', 2)
      created = datetime.datetime.fromisoformat(created_at.replace('Z', '+00:00'))
      if created >= started_at:
        run_info = candidate
        break
    time.sleep(5)
  if not run_info:
    raise RuntimeError(f'Could not find workflow run for {workflow} on {branch}.')
  run_id, run_url, _ = run_info.split('\t', 2)
  print(underline_links(f'Workflow run: {run_url}'))
  runner.run(['gh', 'run', 'watch', run_id])


def prompt_choice(question: str, choices: list[str], default: str | None = None) -> str:
  """Prompt user to choose from a list of options.

  Args:
    question: The question to ask.
    choices: List of valid choices.
    default: Default choice if user just presses enter.

  Returns:
    The selected choice.
  """
  suffix = f' [{default}]' if default else ''
  while True:
    answer = input(emphasize_prompt(f'{question} ({"/".join(choices)}){suffix}: ')).strip().lower()
    if not answer and default:
      return default
    if answer in choices:
      return answer
    print(f'Choose one of: {", ".join(choices)}')


def prompt_numbered_choice(question: str, choices: list[str]) -> str:
  """Prompt user to choose from a numbered list."""
  while True:
    print(emphasize_prompt(f'{question}:'))
    for index, choice in enumerate(choices, start=1):
      print(f'{index}) {choice}')
    answer = input(emphasize_prompt('Select number: ')).strip()
    if answer.isdigit() and 1 <= int(answer) <= len(choices):
      return choices[int(answer) - 1]
    print(f'Choose a number from 1 to {len(choices)}.')


def prompt_required(question: str) -> str:
  """Prompt user for required input.

  Args:
    question: The question to ask.

  Returns:
    The user input (non-empty).
  """
  while True:
    answer = input(emphasize_prompt(f'{question}: ')).strip()
    if answer:
      return answer
    print('A value is required.')


def prompt_validated(question: str, validator) -> str:
  """Prompt until validator accepts the answer."""
  while True:
    answer = prompt_required(question)
    try:
      validator(answer)
      return answer
    except ValueError as error:
      print(error)


def confirm(question: str) -> None:
  """Prompt user for confirmation.

  Args:
    question: The question to ask.

  Raises:
    SystemExit: If user does not confirm.
  """
  answer = prompt_choice(question, ['y', 'n'], default='n')
  if answer != 'y':
    raise SystemExit('Stopped by user.')


def collect_context(args: argparse.Namespace, state: ReleaseState) -> ReleaseContext:
  """Collect release context from state and prompts.

  Args:
    args: Parsed command-line arguments.
    state: Release state with answers and completed steps.

  Returns:
    ReleaseContext instance with all required fields.
  """
  root = Path(subprocess.check_output(['git', 'rev-parse', '--show-toplevel'], text=True).strip())
  answers = state.answers
  if 'release_type' not in answers:
    answers['release_type'] = prompt_numbered_choice('Release type', ['major', 'minor', 'patch'])
  if 'version' not in answers:
    answers['version'] = prompt_validated(
        'Release version',
        lambda version: ReleaseMetadata.from_version(str(answers['release_type']), version),
    )
  if 'fork_remote' not in answers:
    answers['fork_remote'] = prompt_validated(
        'Fork remote, for example https://github.com/USER/pipelines.git',
        normalize_fork_remote,
    )
  answers['fork_remote'] = normalize_fork_remote(str(answers['fork_remote']))
  if answers['release_type'] == 'patch':
    if 'patch_prs' not in answers:
      answers['patch_prs'] = prompt_required('Comma-separated PR numbers or URLs to cherry-pick')
    if 'include_backend' not in answers:
      answers['include_backend'] = prompt_choice('Does this patch include backend/images?', ['y', 'n'], default='y') == 'y'
    if 'include_sdk' not in answers:
      answers['include_sdk'] = prompt_choice('Does this patch include SDK packages?', ['y', 'n'], default='y') == 'y'
  else:
    if getattr(args, 'release_source_branch', None) and 'release_source_branch' not in answers:
      answers['release_source_branch'] = str(args.release_source_branch)
    if getattr(args, 'prompt_release_source_branch', False) and 'release_source_branch' not in answers:
      answers['release_source_branch'] = input('Release source branch [master]: ').strip() or 'master'
    if 'include_backend' not in answers:
      answers['include_backend'] = True
    if 'include_sdk' not in answers:
      answers['include_sdk'] = True
  metadata = ReleaseMetadata.from_version(str(answers['release_type']), str(answers['version']))
  if getattr(args, 'require_previous_release', True) and 'previous_release' not in answers:
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
  )
