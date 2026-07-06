"""Release steps and step registry."""

import getpass
import re
import subprocess
import time
from dataclasses import dataclass
from pathlib import Path

from . import rtd
from .core import (
    REPO,
    ReleaseContext,
    confirm,
    emphasize_prompt,
    image_workflow_command,
    kfp_kubernetes_docs_build_command,
    kfp_requirements_command,
    kubernetes_requirements_command,
    parse_github_owner,
    prompt_choice,
    prompt_required,
    release_version_bump_command,
    sdk_workflow_command,
    underline_links,
    wait_for_pr_merge,
    watch_pr_ci,
    watch_latest_workflow_run,
)


@dataclass(frozen=True)
class Step:
  """Represents a release step with ID, description, and handler."""

  step_id: str
  description: str
  handler: str


def manual_checklist(step_id: str, metadata) -> str:
  """Return manual checkpoint text for status output."""
  if step_id == 'confirm-rtd':
    return underline_links(f'''ReadTheDocs manual checkpoint:
1. Open https://app.readthedocs.org/projects/kubeflow-pipelines/
2. Confirm sdk-{metadata.tag} build succeeded.
3. Set Default version to sdk-{metadata.tag}.
4. Set Default branch to {metadata.release_branch}.
5. Open https://app.readthedocs.org/projects/kfp-kubernetes/
6. Add or resync version kfp-kubernetes-{metadata.tag}.
7. Confirm kfp-kubernetes-{metadata.tag} build succeeded.
8. Set Default version to kfp-kubernetes-{metadata.tag}.''')
  if step_id == 'confirm-website-and-slack':
    return underline_links(f'''Manual final checkpoint:
1. In kubeflow/website, write the version without a trailing newline:
   echo -n {metadata.tag} > layouts/shortcodes/pipelines/latest-version.html
2. Open a PR titled: pipelines: release kfp {metadata.tag}
3. Announce the release in #kubeflow-pipelines in the CNCF Slack workspace.''')
  return ''


def step_preflight(context: ReleaseContext) -> None:
  """Verify required tools and GitHub authentication.

  Args:
    context: Release context with runner for command execution.
  """
  tools = ['git', 'gh', 'docker', 'python3', 'sed']
  if context.include_sdk:
    tools.append('pip-compile')
  for tool in tools:
    context.runner.run(['which', tool])
  context.runner.run(['gh', 'auth', 'status'])
  context.runner.run(['git', 'status', '--short'], cwd=context.root)


def step_prepare_release_branch(context: ReleaseContext) -> None:
  """Create release branch from the selected source branch.

  Args:
    context: Release context with runner and metadata.
  """
  branch = context.metadata.release_branch
  source_branch = str(context.state.answers.get('release_source_branch', 'master'))
  context.runner.run(['git', 'checkout', source_branch], cwd=context.root)
  pull_if_upstream(context)
  context.runner.run(['git', 'checkout', '-b', branch], cwd=context.root)


def pull_if_upstream(context: ReleaseContext) -> None:
  """Fast-forward pull only when the current branch tracks an upstream."""
  upstream = context.runner.run(['git', 'rev-parse', '--abbrev-ref', '--symbolic-full-name', '@{u}'], cwd=context.root, check=False)
  if upstream.returncode == 0:
    context.runner.run(['git', 'pull', '--ff-only'], cwd=context.root)


def step_prepare_patch_branch(context: ReleaseContext) -> None:
  """Create patch branch from release branch.

  Args:
    context: Release context with runner and metadata.
  """
  metadata = context.metadata
  context.runner.run(['git', 'checkout', metadata.release_branch], cwd=context.root)
  context.runner.run(['git', 'pull', '--ff-only'], cwd=context.root)
  context.runner.run(['git', 'checkout', '-b', metadata.patch_branch], cwd=context.root)


def step_cherry_pick_prs(context: ReleaseContext) -> None:
  """Cherry-pick requested PRs.

  Args:
    context: Release context with runner, state containing patch_prs.
  """
  prs = str(context.state.answers.get('patch_prs', '')).split(',')
  for pr in [item.strip() for item in prs if item.strip()]:
    sha = context.runner.capture(['gh', '--repo', REPO, 'pr', 'view', pr, '--json', 'mergeCommit', '--jq', '.mergeCommit.oid // ""'])
    if sha:
      context.runner.run(['git', 'cherry-pick', sha], cwd=context.root)
    elif not context.runner.dry_run:
      print(f'Warning: Could not find merge commit for PR {pr}')


def step_merge_cherry_pick_pr(context: ReleaseContext) -> None:
  """Push patch branch and create PR for cherry-picks.

  Args:
    context: Release context with runner, metadata, and fork_remote.
  """
  metadata = context.metadata
  owner = parse_github_owner(context.fork_remote)
  context.runner.run(['git', 'push', '--set-upstream', context.fork_remote, metadata.patch_branch], cwd=context.root)
  if context.runner.dry_run:
    print(f'[dry-run] would open PR: cherry-picks for {metadata.tag}')
    return
  pr_url = context.runner.capture(
      [
          'gh',
          'pr',
          'create',
          '--repo',
          REPO,
          '--base',
          metadata.release_branch,
          '--head',
          f'{owner}:{metadata.patch_branch}',
          '--title',
          f'chore(release): cherry-pick fixes for {metadata.tag}',
          '--body',
          f'Cherry-picks fixes for KFP {metadata.tag}.',
      ]
  )
  print(underline_links(f'Cherry-pick PR: {pr_url}'))
  wait_for_pr_merge(context.runner, pr_url)


def step_update_version_tags(context: ReleaseContext) -> None:
  """Update version tags in repository files.

  Args:
    context: Release context with runner, metadata, and root path.
  """
  metadata = context.metadata
  root = context.root
  version_branch = f'{metadata.tag}-update-version-tags'

  context.runner.run(['git', 'checkout', metadata.release_branch], cwd=root)
  pull_if_upstream(context)
  if context.runner.dry_run:
    context.runner.run(['git', 'checkout', '-b', version_branch], cwd=root)
  else:
    existing_branch = context.runner.run(['git', 'rev-parse', '--verify', f'refs/heads/{version_branch}'], cwd=root, check=False)
    if existing_branch.returncode == 0:
      choice = prompt_choice(f'Branch {version_branch} already exists. Use it or recreate it?', ['u', 'r'], default='u')
      if choice == 'u':
        context.runner.run(['git', 'checkout', version_branch], cwd=root)
      else:
        context.runner.run(['git', 'branch', '-D', version_branch], cwd=root)
        context.runner.run(['git', 'checkout', '-b', version_branch], cwd=root)
    else:
      context.runner.run(['git', 'checkout', '-b', version_branch], cwd=root)
  
  # Update manifests
  context.runner.run(['sed', '-i.bak', '-E', f's#^([[:space:]]*-[[:space:]]*kubeflow_pipelines_version=).*#\\1v{metadata.tag}#', 'manifests/kustomize/base/pipeline/kustomization.yaml'], cwd=root)
  
  # Write VERSION file
  if context.runner.dry_run:
    print(f'[dry-run] would write VERSION: {metadata.tag}')
  else:
    (root / 'VERSION').write_text(metadata.tag)
  
  release_image_tag = 'master' if metadata.release_type in ('major', 'minor') else metadata.release_branch
  context.runner.run(release_version_bump_command(root, metadata.release_branch, release_image_tag), cwd=root)
  if context.include_sdk:
    _update_sdk_version_files(context)
  
  # Commit and tag
  context.runner.run(['git', 'add', '--all'], cwd=root)
  context.runner.run(['git', 'commit', '-s', '-m', f'chore(release): bump version to {metadata.tag}'], cwd=root)
  if context.runner.dry_run:
    print(f'[dry-run] would create tag: {metadata.tag}')
    context.runner.run(['git', 'tag', '-a', metadata.tag, '-m', f'Kubeflow Pipelines {metadata.tag} release'], cwd=root)
    return
  existing_tag = context.runner.run(['git', 'rev-parse', '--verify', f'refs/tags/{metadata.tag}'], cwd=root, check=False)
  if existing_tag.returncode == 0:
    choice = prompt_choice(f'Tag {metadata.tag} already exists. Use it or recreate it?', ['u', 'r'], default='u')
    if choice == 'u':
      return
    context.runner.run(['git', 'tag', '-d', metadata.tag], cwd=root)
  context.runner.run(['git', 'tag', '-a', metadata.tag, '-m', f'Kubeflow Pipelines {metadata.tag} release'], cwd=root)


def step_merge_version_pr(context: ReleaseContext) -> None:
  """Push release branch and tag, create version PR.

  Args:
    context: Release context with runner, metadata, and fork_remote.
  """
  metadata = context.metadata
  version_branch = f'{metadata.tag}-update-version-tags'
  upstream_remote = f'https://github.com/{REPO}.git'
  owner = parse_github_owner(context.fork_remote)
  if not context.runner.dry_run:
    upstream_branch = context.runner.run(['git', 'ls-remote', '--exit-code', '--heads', upstream_remote, metadata.release_branch], cwd=context.root, check=False)
    if upstream_branch.returncode != 0:
      context.runner.run(['git', 'push', upstream_remote, metadata.release_branch], cwd=context.root)
  if context.runner.dry_run:
    context.runner.run(['git', 'push', '--set-upstream', context.fork_remote, version_branch], cwd=context.root)
  else:
    remote_branch = context.runner.run(['git', 'ls-remote', '--exit-code', '--heads', context.fork_remote, version_branch], cwd=context.root, check=False)
    if remote_branch.returncode == 0:
      choice = prompt_choice(f'Remote branch {version_branch} already exists. Use it or recreate it?', ['u', 'r'], default='u')
      if choice == 'r':
        context.runner.run(['git', 'push', '--delete', context.fork_remote, version_branch], cwd=context.root)
        context.runner.run(['git', 'push', '--set-upstream', context.fork_remote, version_branch], cwd=context.root)
    else:
      context.runner.run(['git', 'push', '--set-upstream', context.fork_remote, version_branch], cwd=context.root)
  if context.runner.dry_run:
    context.runner.run(['git', 'push', context.fork_remote, metadata.tag], cwd=context.root)
    print(f'[dry-run] would open PR: version bump for {metadata.tag}')
    return
  remote_tag = context.runner.run(['git', 'ls-remote', '--exit-code', '--tags', context.fork_remote, metadata.tag], cwd=context.root, check=False)
  if remote_tag.returncode == 0:
    choice = prompt_choice(f'Remote tag {metadata.tag} already exists. Use it or recreate it?', ['u', 'r'], default='u')
    if choice == 'r':
      context.runner.run(['git', 'push', '--delete', context.fork_remote, metadata.tag], cwd=context.root)
      context.runner.run(['git', 'push', context.fork_remote, metadata.tag], cwd=context.root)
  else:
    context.runner.run(['git', 'push', context.fork_remote, metadata.tag], cwd=context.root)
  pr_url = context.runner.capture(
      [
          'gh',
          'pr',
          'create',
          '--repo',
          REPO,
          '--base',
          metadata.release_branch,
          '--head',
          f'{owner}:{version_branch}',
          '--title',
          f'chore(release): bump version to {metadata.tag}',
          '--body',
          f'Updates release branch version tags for KFP {metadata.tag}.',
      ]
  )
  print(underline_links(f'Version PR: {pr_url}'))
  watch_pr_ci(context.runner, pr_url)
  wait_for_pr_merge(context.runner, pr_url)


def step_publish_images(context: ReleaseContext) -> None:
  """Trigger and watch image publication workflow.

  Args:
    context: Release context with runner and metadata.
  """
  context.runner.run(image_workflow_command(context.metadata))
  watch_latest_workflow_run(context.runner, 'image-builds-release.yml', context.metadata.release_branch)


def _replace(path: Path, pattern: str, replacement: str) -> None:
  if path.exists():
    path.write_text(re.sub(pattern, replacement, path.read_text()))


def _update_sdk_docs_versions(path: Path, version: str) -> None:
  if not path.exists():
    return
  text = path.read_text()
  if f'"title": "{version}"' in text:
    return
  text = re.sub(r'"aliases": \[\s*"stable",\s*"latest"\s*\]', '"aliases": []', text, count=1)
  entry = f'''    {{
      "version": "https://kubeflow-pipelines.readthedocs.io/en/sdk-{version}/",
      "title": "{version}",
      "aliases": [
        "stable",
        "latest"
      ]
    }},
'''
  path.write_text(text.replace('[\n', '[\n' + entry, 1))


def _update_kfp_kubernetes_docs_versions(path: Path, version: str) -> None:
  if not path.exists():
    return
  text = path.read_text()
  if f"'kfp-kubernetes-{version}/'" in text:
    return
  text = re.sub(r"'aliases': \['stable'\]", "'aliases': []", text, count=1)
  entry = f'''        {{
            'version':
                'https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-{version}/',
            'title':
                '{version}',
            'aliases': ['stable'],
        }},
'''
  path.write_text(text.replace("    'version_info': [\n", "    'version_info': [\n" + entry, 1))


def _sdk_release_note_section(title: str, items: list[str]) -> str:
  body = '\n'.join(f'* {item}' for item in items)
  return f'## {title}\n\n{body}'.rstrip()


def _generate_sdk_release_notes(root: Path) -> str:
  try:
    tags = subprocess.check_output(
        ['git', 'tag', '--list', 'sdk-*', '--sort=-version:refname'],
        cwd=root,
        text=True,
    ).splitlines()
  except subprocess.CalledProcessError:
    return ''
  if not tags:
    return ''

  try:
    subjects = subprocess.check_output(
        ['git', 'log', f'{tags[0]}..HEAD', '--pretty=%s'],
        cwd=root,
        text=True,
    ).splitlines()
  except subprocess.CalledProcessError:
    return ''
  groups: dict[str, list[str]] = {
      'Features': [],
      'Breaking changes': [],
      'Deprecations': [],
      'Bug fixes and other changes': [],
  }
  pattern = re.compile(
      r'^(?P<type>[A-Za-z]+)(?:\((?P<scope>[^)]*)\))?(?P<breaking>!)?:\s*(?P<description>.+)$'
  )
  for subject in subjects:
    match = pattern.match(subject)
    if not match or 'sdk' not in (match.group('scope') or '').lower():
      continue
    commit_type = match.group('type').lower()
    description = match.group('description')
    if match.group('breaking') or commit_type in {'break', 'breaking'}:
      groups['Breaking changes'].append(description)
    elif commit_type == 'feat':
      groups['Features'].append(description)
    elif commit_type.startswith('deprecat'):
      groups['Deprecations'].append(description)
    else:
      groups['Bug fixes and other changes'].append(description)

  if not any(groups.values()):
    return ''
  return '\n\n'.join(_sdk_release_note_section(title, groups[title]) for title in groups)


def _update_sdk_release_notes(
    path: Path,
    version: str,
    release_body: str = '',
) -> None:
  if not path.exists():
    return
  text = path.read_text()
  if f'# {version}\n' in text:
    return
  default_body = '## Features\n\n## Breaking changes\n\n## Deprecations\n\n## Bug fixes and other changes'
  match = re.match(r'(# Current Version \(in development\)\n\n)(.*?)(\n# \d)', text, re.DOTALL)
  if not match:
    return
  release_body = release_body or match.group(2).strip() or default_body
  path.write_text(f'{match.group(1)}{default_body}\n\n# {version}\n\n{release_body}{match.group(3)}{text[match.end():]}')


def _update_sdk_version_files(context: ReleaseContext) -> None:
  metadata = context.metadata
  root = context.root

  # Update SDK version files
  if context.runner.dry_run:
    print(f'[dry-run] would update SDK version files to {metadata.tag}')
  else:
    _replace(root / 'sdk/python/kfp/version.py', r"__version__\s*=\s*['\"]([^'\"]+)['\"]", f"__version__ = '{metadata.tag}'")
    _replace(root / 'kubernetes_platform/python/kfp/kubernetes/__init__.py', r"__version__\s*=\s*['\"]([^'\"]+)['\"]", f"__version__ = '{metadata.tag}'")
    _replace(root / 'api/v2alpha1/python/setup.py', r"VERSION\s*=\s*['\"]([^'\"]+)['\"]", f"VERSION = '{metadata.tag}'")
    _replace(root / 'backend/api/v2beta1/python_http_client/setup.py', r"VERSION\s*=\s*['\"]([^'\"]+)['\"]", f'VERSION = "{metadata.tag}"')
    _replace(root / 'backend/api/v2beta1/python_http_client/kfp_server_api/__init__.py', r"__version__\s*=\s*['\"]([^'\"]+)['\"]", f'__version__ = "{metadata.tag}"')
    _replace(root / 'sdk/python/requirements.in', r'kfp-pipeline-spec>=[^,\n]+,<3', f'kfp-pipeline-spec>={metadata.tag},<3')
    _replace(root / 'sdk/python/requirements.in', r'kfp-server-api>=[^,\n]+,<3', f'kfp-server-api>={metadata.tag},<3')
    _replace(root / 'kubernetes_platform/python/requirements.in', r'kfp>=[^,\n]+,<3', f'kfp>={metadata.tag},<3')
    _update_sdk_docs_versions(root / 'docs/sdk/versions.json', metadata.tag)
    _update_kfp_kubernetes_docs_versions(root / 'kubernetes_platform/python/docs/conf.py', metadata.tag)
    _update_sdk_release_notes(
        root / 'sdk/RELEASE.md',
        metadata.tag,
        _generate_sdk_release_notes(root),
    )

  # Build local dists for unpublished package versions before resolving requirements.
  context.runner.run(['python', '-m', 'build', '.'], cwd=root / 'api/v2alpha1/python')
  context.runner.run(['python', '-m', 'build', '.'], cwd=root / 'backend/api/v2beta1/python_http_client')
  context.runner.run(kfp_requirements_command(), cwd=root / 'sdk/python')
  context.runner.run(['python', '-m', 'build', '.'], cwd=root / 'sdk/python')
  context.runner.run(kubernetes_requirements_command(), cwd=root / 'kubernetes_platform/python')


def step_create_sdk_release(context: ReleaseContext) -> None:
  """Create SDK GitHub release.

  Args:
    context: Release context with runner and metadata.
  """
  metadata = context.metadata
  notes = f'''Release of:

- KFP SDK
- KFP Kubernetes
- KFP Server API
- KFP Pipeline Spec

To install the KFP SDK:

```
pip install kfp-pipeline-spec=={metadata.tag}
pip install kfp-server-api=={metadata.tag}
pip install kfp=={metadata.tag}
pip install kfp-kubernetes=={metadata.tag}
```

For changelog, see https://github.com/kubeflow/pipelines/blob/{metadata.sdk_tag}/sdk/RELEASE.md.
'''
  if context.runner.dry_run:
    print(f'[dry-run] would create SDK GitHub release: {metadata.sdk_tag}')
  else:
    verify_existing_tag = False
    upstream_remote = f'https://github.com/{REPO}.git'
    existing_release = context.runner.run(['gh', 'release', 'view', metadata.sdk_tag, '--repo', REPO], check=False)
    if existing_release.returncode == 0:
      choice = prompt_choice(f'SDK GitHub release {metadata.sdk_tag} already exists. Use it or recreate it?', ['u', 'r'], default='u')
      if choice == 'u':
        return
      context.runner.run(['gh', 'release', 'delete', metadata.sdk_tag, '--repo', REPO, '--yes', '--cleanup-tag'])
    else:
      existing_tag = context.runner.run(['gh', 'api', '--silent', f'repos/{REPO}/git/ref/tags/{metadata.sdk_tag}'], check=False)
      update_tag = existing_tag.returncode != 0
      if existing_tag.returncode == 0:
        verify_existing_tag = True
        choice = prompt_choice(f'SDK tag {metadata.sdk_tag} already exists. Use it or recreate it?', ['u', 'r'], default='u')
        update_tag = choice == 'r'
      if update_tag:
        verify_existing_tag = True
        context.runner.run(['git', 'tag', '-f', metadata.sdk_tag, metadata.release_branch], cwd=context.root)
        context.runner.run(['git', 'push', '--force', upstream_remote, metadata.sdk_tag], cwd=context.root)
  create_command = [
      'gh',
      'release',
      'create',
      metadata.sdk_tag,
      '--repo',
      REPO,
      '--target',
      metadata.release_branch,
      '--title',
      f'KFP SDK v{metadata.tag}',
      '--notes',
      notes,
  ]
  if not context.runner.dry_run and verify_existing_tag:
    create_command.append('--verify-tag')
  context.runner.run(create_command)


def step_publish_sdks(context: ReleaseContext) -> None:
  """Trigger and watch SDK publication workflow.

  Args:
    context: Release context with runner and metadata.
  """
  context.runner.run(sdk_workflow_command(context.metadata))
  watch_latest_workflow_run(context.runner, 'publish-packages.yml', context.metadata.release_branch)


def step_create_kfp_kubernetes_docs_branch(context: ReleaseContext) -> None:
  """Create kfp-kubernetes docs branch for ReadTheDocs.

  Args:
    context: Release context with runner, metadata, and root path.
  """
  metadata = context.metadata
  root = context.root
  branch = f'kfp-kubernetes-{metadata.tag}'
  pkg_root = root / 'kubernetes_platform/python'
  
  context.runner.run(kfp_kubernetes_docs_build_command(root, context.metadata.release_branch), cwd=root)
  context.runner.run(['git', 'checkout', '-b', branch], cwd=root)
  
  # Move .readthedocs.yml to root and remove .gitignore - guarded in dry-run
  if context.runner.dry_run:
    print('[dry-run] would copy ReadTheDocs config to repository root')
    print('[dry-run] would remove kubernetes_platform/.gitignore')
  else:
    rtd_config_src = pkg_root / 'docs/.readthedocs.yml'
    rtd_config_dst = root / '.readthedocs.yml'
    if rtd_config_src.exists():
      rtd_config_dst.write_text(rtd_config_src.read_text())
    
    gitignore = root / 'kubernetes_platform/.gitignore'
    if gitignore.exists():
      gitignore.unlink()
  
  context.runner.run(['git', 'add', str(pkg_root / 'docs/.readthedocs.yml'), '.readthedocs.yml', 'kubernetes_platform/.gitignore'], cwd=root)
  context.runner.run(['git', 'commit', '-s', '-m', f'chore: kfp-kubernetes docs branch for release {metadata.tag}'], cwd=root)
  context.runner.run(['git', 'push', '--set-upstream', 'upstream', branch], cwd=root)


def step_confirm_rtd(context: ReleaseContext) -> None:
  """Confirm ReadTheDocs version updates.

  Args:
    context: Release context with runner and metadata.
  """
  metadata = context.metadata
  
  if context.runner.dry_run:
    print(manual_checklist('confirm-rtd', metadata))
    print('[dry-run] ReadTheDocs confirmation skipped')
    return

  token = ''
  while not token:
    token = getpass.getpass(emphasize_prompt('Read the Docs API token: ')).strip()
    if not token:
      print('A Read the Docs API token is required.')
  try:
    client = rtd.ReadTheDocsClient(token)
    rtd.update_release_docs(client, metadata.tag, metadata.release_branch)
    print('ReadTheDocs projects updated and verified.')
  except rtd.ReadTheDocsError as error:
    print(f'ReadTheDocs automation failed: {error}')
    print(manual_checklist('confirm-rtd', metadata))
    confirm('Fall back to manual ReadTheDocs mode?')
    confirm('Have both ReadTheDocs projects been updated and verified?')


def step_create_backend_release(context: ReleaseContext) -> None:
  """Create backend GitHub release.

  Args:
    context: Release context with runner, metadata, and state.
  """
  metadata = context.metadata
  
  if context.runner.dry_run:
    print(f'[dry-run] would create backend GitHub release: {metadata.tag}')
    return
  
  last_release = prompt_required('Last backend release tag for changelog comparison')
  changed = prompt_required('Backend release notes body after "What Changed"')
  
  notes = f'''## What's Changed
{changed}

**Full Changelog**: https://github.com/kubeflow/pipelines/compare/{last_release}...{metadata.tag}
'''
  
  context.runner.run([
      'gh',
      'release',
      'create',
      metadata.tag,
      '--repo',
      REPO,
      '--target',
      metadata.release_branch,
      '--title',
      f'Version {metadata.tag}',
      '--notes',
      notes,
  ])


def step_sync_master(context: ReleaseContext) -> None:
  """Sync release version to master branch.

  Args:
    context: Release context with runner, metadata, and root path.
  """
  metadata = context.metadata
  root = context.root
  branch = f'chore-release-{metadata.tag}-master'
  
  context.runner.run(['git', 'checkout', 'master'], cwd=root)
  context.runner.run(['git', 'pull', '--ff-only'], cwd=root)
  context.runner.run(['git', 'checkout', '-b', branch], cwd=root)
  
  # Write VERSION file without trailing newline - guarded in dry-run
  if context.runner.dry_run:
    print(f'[dry-run] would write VERSION: {metadata.tag}')
  else:
    (root / 'VERSION').write_text(metadata.tag)
  
  release_image_tag = 'master' if metadata.release_type in ('major', 'minor') else metadata.release_branch
  context.runner.run(release_version_bump_command(root, metadata.release_branch, release_image_tag), cwd=root)
  context.runner.run(['git', 'checkout', metadata.tag, '--', 'CHANGELOG.md'], cwd=root)
  context.runner.run(['git', 'add', '-A'], cwd=root)
  context.runner.run(['git', 'commit', '-s', '-m', f'chore(release): bump version to {metadata.tag} on master branch'], cwd=root)
  context.runner.run(['git', 'push', '--set-upstream', context.fork_remote, branch], cwd=root)


def step_confirm_website_and_slack(context: ReleaseContext) -> None:
  """Confirm website PR and Slack announcement.

  Args:
    context: Release context with runner and metadata.
  """
  metadata = context.metadata
  print(manual_checklist('confirm-website-and-slack', metadata))
  
  if context.runner.dry_run:
    print('[dry-run] Website and Slack confirmation skipped')
    return
  
  confirm('Have the website PR and Slack announcement been completed?')


def build_steps(release_type: str, include_backend: bool, include_sdk: bool) -> list[Step]:
  """Build the list of release steps based on configuration.

  Args:
    release_type: One of 'major', 'minor', or 'patch'.
    include_backend: Whether to include backend steps.
    include_sdk: Whether to include SDK steps.

  Returns:
    Ordered list of Step instances for the release.
  """
  steps = [Step('preflight', 'Verify tools and GitHub auth', 'step_preflight')]
  if release_type == 'patch':
    steps.extend(
       [
           Step('prepare-patch-branch', 'Create patch branch from release branch', 'step_prepare_patch_branch'),
           Step('cherry-pick-prs', 'Cherry-pick requested PR merge commits', 'step_cherry_pick_prs'),
           Step('merge-cherry-pick-pr', 'Open and wait for cherry-pick PR', 'step_merge_cherry_pick_pr'),
       ]
    )
  else:
    steps.append(Step('prepare-release-branch', 'Create release branch from master', 'step_prepare_release_branch'))
  steps.extend(
     [
         Step('update-version-tags', 'Update repository version tags', 'step_update_version_tags'),
         Step('merge-version-pr', 'Open and wait for version bump PR', 'step_merge_version_pr'),
     ]
  )
  if include_backend:
    steps.append(Step('publish-images', 'Run image publication workflow', 'step_publish_images'))
  if include_sdk:
    steps.extend(
       [
           Step('create-sdk-release', 'Create SDK GitHub release', 'step_create_sdk_release'),
           Step('publish-sdks', 'Run SDK publication workflow', 'step_publish_sdks'),
           Step('create-kfp-kubernetes-docs-branch', 'Create kfp-kubernetes docs branch', 'step_create_kfp_kubernetes_docs_branch'),
           Step('confirm-rtd', 'Confirm ReadTheDocs updates', 'step_confirm_rtd'),
       ]
    )
  steps.extend(
     [
         Step('create-backend-release', 'Create backend GitHub release', 'step_create_backend_release'),
         Step('sync-master', 'Sync release version to master', 'step_sync_master'),
         Step('confirm-website-and-slack', 'Confirm website PR and Slack announcement', 'step_confirm_website_and_slack'),
     ]
  )
  return steps


def run_steps(context: ReleaseContext) -> None:
  """Execute release steps in order, skipping completed ones.

  Args:
    context: Release context with state, runner, and metadata.
  """
  steps = build_steps(context.metadata.release_type, context.include_backend, context.include_sdk)
  handlers = globals()
  printed_step = False
  for step in steps:
    if context.runner.dry_run and printed_step:
      print()
    printed_step = True
    if context.state.is_done(step.step_id):
      print(f'Skipping completed step: {step.step_id}')
      continue
    print(f'Running step: {step.description}')
    handlers[step.handler](context)
    context.state.mark_done(step.step_id)
    context.state.save()


# Build the step registry mapping step_id to handler for all built steps
STEP_HANDLERS = {}
for release_type in ('major', 'minor', 'patch'):
  for step in build_steps(release_type, include_backend=True, include_sdk=True):
    STEP_HANDLERS[step.step_id] = globals()[step.handler]
