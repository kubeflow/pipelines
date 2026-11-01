"""Release steps and step registry."""

import getpass
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path

from . import qualification, rtd
from .core import (
    REPO,
    ReleaseContext,
    ReleaseMetadata,
    confirm,
    emphasize_prompt,
    image_workflow_command,
    is_pending_merge_queue_check,
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
    watch_latest_workflow_run,
    watch_pr_ci,
)
from .dependency_pins import preserve_dependency_pins
from .editorial import editorial_scaffold, validate_notes
from .generated import restore_generated_sections
from .inventory import build_inventory, release_scaffolding
from .notes import render_notes
from .packages import clean_legacy_distributions, validate_legacy_wheels
from .previews import _write_atomic, has_conflicts, regenerate_preview
from .ranges import ReleaseRange, commit_date, read_commits, resolve_range


@dataclass(frozen=True)
class Step:
    """Represents a release step with ID, description, and handler."""

    step_id: str
    description: str
    handler: str


def manual_checklist(
    step_id: str, metadata: ReleaseMetadata, consolidated: bool | None = None
) -> str:
    """Return manual checkpoint text for status output."""
    if step_id == 'create-backend-release' and metadata.major >= 3:
        return underline_links(
            f"""Architecture validation checkpoint for KFP {metadata.tag}:
1. Confirm image-builds-release.yml succeeded on {metadata.release_branch} for target tag
   {metadata.tag} and the intended source commit, including both native shared-tag checks
   (Linux AMD64 and ARM64) and the native ARM64 installation/pipeline smoke.
2. Retain the successful run URL and its same-run published-index, native image-validation, and
   smoke artifacts as release evidence. A master run or an index-only check does not replace it.
3. Review the architecture support policy before publishing the backend GitHub release:
   https://github.com/kubeflow/pipelines/blob/{metadata.release_branch}/docs/operator-guides/supported-platforms.md"""
        )
    if step_id == 'confirm-rtd':
        checklist = f"""ReadTheDocs manual checkpoint:
1. Open https://app.readthedocs.org/projects/kubeflow-pipelines/
2. Confirm {metadata.release_branch} build succeeded.
3. Set Default version to {metadata.release_branch}.
4. Set Default branch to {metadata.release_branch}."""
        if consolidated is not True:
            if consolidated is None:
                checklist += f"""

For split-package SDK tags only (skip when {metadata.sdk_tag} contains
sdk/python/kfp/server_api):"""
            checklist += f"""
5. Open https://app.readthedocs.org/projects/kfp-kubernetes/
6. Confirm kfp-kubernetes-{metadata.major}.{metadata.minor} build succeeded.
7. Set Default version to kfp-kubernetes-{metadata.major}.{metadata.minor}."""
        if metadata.release_type == 'patch':
            checklist = re.sub(
                r'Set Default (?:version|branch) to [^\n]+\.',
                'Preserve the existing project default.',
                checklist,
            )
        return underline_links(checklist)
    if step_id == 'confirm-website-and-slack':
        compatibility_steps = ''
        slack_step = 4
        if metadata.tag == '3.0.0':
            compatibility_steps = """
4. In the same website PR, update the Kubeflow Pipelines operator installation and upgrade
   documentation to announce that KFP 3.0 does not support Argo Workflows 3.x and requires a
   supported Argo Workflows 4.x release.
5. Confirm the KFP 3.0 GitHub release notes list removal of Argo Workflows 3.x support as a breaking
   change and close https://github.com/kubeflow/pipelines/issues/14139 only after every migration and
   documentation item is complete."""
            slack_step = 6
        if metadata.major >= 3:
            compatibility_steps += f"""
{slack_step}. Link the architecture support policy in release communications: Linux AMD64 and ARM64
   are supported starting with KFP 3.0; use the shared image tags and architecture-compatible
   pipeline component images.
   https://github.com/kubeflow/pipelines/blob/{metadata.tag}/docs/operator-guides/supported-platforms.md"""
            slack_step += 1
        website_steps = f"""1. Open https://github.com/kubeflow/website/edit/master/layouts/shortcodes/pipelines/latest-version.html
2. Write the version without a trailing newline:
   echo -n {metadata.tag} > layouts/shortcodes/pipelines/latest-version.html
3. Open a PR titled: pipelines: release kfp {metadata.tag}"""
        if metadata.release_type == 'patch':
            website_steps = f"""1. Preserve layouts/shortcodes/pipelines/latest-version.html; do not change the latest release pointer.
2. Review any documentation updates needed for KFP {metadata.tag}.
3. If documentation changes are needed, open a PR titled: pipelines: release kfp {metadata.tag}"""
        return underline_links(f"""Manual final checkpoint:
{website_steps}{compatibility_steps}
{slack_step}. After the website PR merges, announce in #kubeflow-pipelines:
   KFP {metadata.tag} has officially been released!

   Backend - https://github.com/kubeflow/pipelines/releases/tag/{metadata.tag}
   SDK - https://github.com/kubeflow/pipelines/releases/tag/{metadata.sdk_tag}

   Changes: https://github.com/kubeflow/pipelines/blob/master/CHANGELOG.md""")
    return ''


def _run_reviewed_commit(
    context: ReleaseContext, command: list[str], cwd: Path | None = None
) -> None:
    skip_local_review = context.skip_local_review or bool(
        context.state.answers.get('skip_local_review', False)
    )
    if not context.runner.dry_run and not skip_local_review:
        if sys.stdin.isatty():
            context.runner.run(['git', 'status', '--short'], cwd=cwd)
            confirm('Ready to commit changes?')
        else:
            print(
                'Skipping local review prompt because stdin is not interactive.'
            )
    context.runner.run(command, cwd=cwd)


def step_preflight(context: ReleaseContext) -> None:
    """Verify required tools and GitHub authentication.

    Args:
      context: Release context with runner for command execution.
    """
    tools = ['git', 'gh', 'docker', 'python3', 'sed']
    if context.include_sdk:
        tools.append(
            'uv' if (context.root / 'uv.lock').exists() else 'pip-compile'
        )
    for tool in tools:
        context.runner.run(['which', tool])
    context.runner.run(['gh', 'auth', 'status'])
    if context.runner.capture(
        ['git', 'status', '--short'], cwd=context.root
    ).strip():
        raise RuntimeError('git working tree is dirty')


def step_prepare_release_branch(context: ReleaseContext) -> None:
    """Create release branch from the selected source branch.

    Args:
      context: Release context with runner and metadata.
    """
    branch = context.metadata.release_branch
    source_branch = str(
        context.state.answers.get('release_source_branch', 'master')
    )
    context.runner.run(['git', 'checkout', source_branch], cwd=context.root)
    pull_if_upstream(context)
    context.runner.run(['git', 'checkout', '-b', branch], cwd=context.root)


def pull_if_upstream(context: ReleaseContext) -> None:
    """Fast-forward pull only when the current branch tracks an upstream."""
    upstream = context.runner.run(
        ['git', 'rev-parse', '--abbrev-ref', '--symbolic-full-name', '@{u}'],
        cwd=context.root,
        check=False,
    )
    if upstream.returncode == 0:
        context.runner.run(['git', 'pull', '--ff-only'], cwd=context.root)


def step_prepare_patch_branch(context: ReleaseContext) -> None:
    """Create patch branch from release branch.

    Args:
      context: Release context with runner and metadata.
    """
    metadata = context.metadata
    context.runner.run(
        ['git', 'checkout', metadata.release_branch], cwd=context.root
    )
    context.runner.run(['git', 'pull', '--ff-only'], cwd=context.root)
    context.runner.run(
        ['git', 'checkout', '-b', metadata.patch_branch], cwd=context.root
    )


def step_cherry_pick_prs(context: ReleaseContext) -> None:
    """Cherry-pick requested PRs.

    Args:
      context: Release context with runner, state containing patch_prs.
    """
    prs = str(context.state.answers.get('patch_prs', '')).split(',')
    for pr in [item.strip() for item in prs if item.strip()]:
        sha = context.runner.capture(
            [
                'gh',
                '--repo',
                REPO,
                'pr',
                'view',
                pr,
                '--json',
                'mergeCommit',
                '--jq',
                '.mergeCommit.oid // ""',
            ]
        )
        if sha:
            context.runner.run(['git', 'cherry-pick', sha], cwd=context.root)
        elif not context.runner.dry_run:
            raise RuntimeError(f'Could not find merge commit for PR {pr}')


def step_merge_cherry_pick_pr(context: ReleaseContext) -> None:
    """Push patch branch and create PR for cherry-picks.

    Args:
      context: Release context with runner, metadata, and fork_remote.
    """
    metadata = context.metadata
    owner = parse_github_owner(context.fork_remote)
    context.runner.run(
        [
            'git',
            'push',
            '--set-upstream',
            context.fork_remote,
            metadata.patch_branch,
        ],
        cwd=context.root,
    )
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
        existing_branch = context.runner.run(
            ['git', 'rev-parse', '--verify', f'refs/heads/{version_branch}'],
            cwd=root,
            check=False,
        )
        if existing_branch.returncode == 0:
            choice = prompt_choice(
                f'Branch {version_branch} already exists. Use it or recreate it?',
                ['u', 'r'],
                default='u',
            )
            if choice == 'u':
                context.runner.run(
                    ['git', 'checkout', version_branch], cwd=root
                )
            else:
                context.runner.run(
                    ['git', 'branch', '-D', version_branch], cwd=root
                )
                context.runner.run(
                    ['git', 'checkout', '-b', version_branch], cwd=root
                )
        else:
            context.runner.run(
                ['git', 'checkout', '-b', version_branch], cwd=root
            )

    backend_range = sdk_range = None
    if not context.runner.dry_run:
        if context.include_backend:
            backend_range = _preparation_range(context, 'backend')
        if context.include_sdk:
            sdk_range = _preparation_range(context, 'sdk')

    if context.include_backend:
        _update_changelog(context, backend_range)
        context.runner.run(
            [
                'sed',
                '-i.bak',
                '-E',
                f's#^([[:space:]]*-[[:space:]]*kubeflow_pipelines_version=).*#\\1v{metadata.tag}#',
                'manifests/kustomize/base/pipeline/kustomization.yaml',
            ],
            cwd=root,
        )
        if context.runner.dry_run:
            print(f'[dry-run] would write VERSION: {metadata.tag}')
        else:
            (root / 'VERSION').write_text(metadata.tag + '\n')
        release_image_tag = (
            'master'
            if metadata.release_type in ('major', 'minor')
            else metadata.release_branch
        )
        context.runner.run(
            release_version_bump_command(
                root, metadata.release_branch, release_image_tag
            ),
            cwd=root,
        )
        for path, expression in [
            (
                'backend/api/Makefile',
                r's#^(PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:).*#\1'
                + metadata.release_branch
                + '#',
            ),
            (
                'backend/api/Makefile',
                r's#^(RELEASE_IMAGE=ghcr.io/kubeflow/kfp-release:).*#\1'
                + metadata.release_branch
                + '#',
            ),
        ]:
            context.runner.run(
                ['sed', '-i.bak', '-E', expression, path], cwd=root
            )
    for path, expression in [
        (
            'api/Makefile',
            r's#^(PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:).*#\1'
            + metadata.release_branch
            + '#',
        ),
        (
            'sdk/Makefile',
            r's#^(PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:).*#\1'
            + metadata.release_branch
            + '#',
        ),
        (
            'kubernetes_platform/Makefile',
            r's#^(PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:).*#\1'
            + metadata.release_branch
            + '#',
        ),
        (
            'release/Dockerfile.release',
            r's#^(FROM ghcr.io/kubeflow/kfp-api-generator:).*#\1'
            + metadata.release_branch
            + '#',
        ),
    ]:
        context.runner.run(['sed', '-i.bak', '-E', expression, path], cwd=root)
    if context.include_sdk:
        _update_sdk_version_files(context, sdk_range)

    # Commit preparation; tags are created only after merged-source qualification.
    context.runner.run(['git', 'add', '--all'], cwd=root)
    _run_reviewed_commit(
        context,
        [
            'git',
            'commit',
            '-s',
            '-m',
            f'chore(release): bump version to {metadata.tag}',
        ],
        cwd=root,
    )


def step_merge_version_pr(context: ReleaseContext) -> None:
    """Push the version branch, create the PR, and pin its verified merge.

    Args:
      context: Release context with runner, metadata, and fork_remote.
    """
    recorded_pr = str(context.state.answers.get('version_pr_url', ''))
    if recorded_pr:
        _wait_for_version_pr(context, recorded_pr)
        return
    metadata = context.metadata
    version_branch = f'{metadata.tag}-update-version-tags'
    upstream_remote = f'https://github.com/{REPO}.git'
    owner = parse_github_owner(context.fork_remote)
    if not context.runner.dry_run:
        upstream_branch = context.runner.run(
            [
                'git',
                'ls-remote',
                '--exit-code',
                '--heads',
                upstream_remote,
                metadata.release_branch,
            ],
            cwd=context.root,
            check=False,
        )
        if upstream_branch.returncode != 0:
            context.runner.run(
                ['git', 'push', upstream_remote, metadata.release_branch],
                cwd=context.root,
            )
    if context.runner.dry_run:
        context.runner.run(
            [
                'git',
                'push',
                '--set-upstream',
                context.fork_remote,
                version_branch,
            ],
            cwd=context.root,
        )
    else:
        remote_branch = context.runner.run(
            [
                'git',
                'ls-remote',
                '--exit-code',
                '--heads',
                context.fork_remote,
                version_branch,
            ],
            cwd=context.root,
            check=False,
        )
        if remote_branch.returncode == 0:
            choice = prompt_choice(
                f'Remote branch {version_branch} already exists. Use it or recreate it?',
                ['u', 'r'],
                default='u',
            )
            if choice == 'u':
                # A normal push uploads local fixes but rejects divergent history.
                context.runner.run(
                    ['git', 'push', context.fork_remote, version_branch],
                    cwd=context.root,
                )
            else:
                context.runner.run(
                    [
                        'git',
                        'push',
                        '--delete',
                        context.fork_remote,
                        version_branch,
                    ],
                    cwd=context.root,
                )
                context.runner.run(
                    [
                        'git',
                        'push',
                        '--set-upstream',
                        context.fork_remote,
                        version_branch,
                    ],
                    cwd=context.root,
                )
        else:
            context.runner.run(
                [
                    'git',
                    'push',
                    '--set-upstream',
                    context.fork_remote,
                    version_branch,
                ],
                cwd=context.root,
            )
    if context.runner.dry_run:
        print(f'[dry-run] would open PR: version bump for {metadata.tag}')
        return
    pr_url = str(context.state.answers.get('version_pr_url', ''))
    if not pr_url:
        existing = json.loads(
            context.runner.capture(
                [
                    'gh',
                    'pr',
                    'list',
                    '--repo',
                    REPO,
                    '--state',
                    'all',
                    '--base',
                    metadata.release_branch,
                    '--head',
                    f'{owner}:{version_branch}',
                    '--json',
                    'url',
                    '--limit',
                    '100',
                ]
            )
        )
        if len(existing) > 1:
            raise RuntimeError(
                'Multiple version PRs match; record the intended version_pr_url before continuing.'
            )
        if existing:
            pr_url = existing[0]['url']
    if not pr_url:
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
    context.state.answers['version_pr_url'] = pr_url.strip()
    context.state.save()
    _wait_for_version_pr(context, pr_url)


def _wait_for_version_pr(context: ReleaseContext, pr_url: str) -> None:
    if not re.fullmatch(
        r'https://github[.]com/kubeflow/pipelines/pull/[0-9]+', pr_url
    ):
        raise ValueError(
            'Record an upstream kubeflow/pipelines version_pr_url before resuming.'
        )
    if context.runner.dry_run:
        print(
            f'[dry-run] would verify CI and merge for recorded version PR: {pr_url}'
        )
        return
    print(underline_links(f'Version PR: {pr_url}'))
    watch_pr_ci(context.runner, pr_url)
    wait_for_pr_merge(context.runner, pr_url)
    validate_release_pr(context)
    context.state.save()


def validate_release_pr(context: ReleaseContext) -> str:
    """Revalidate the merged version PR and bind state to its tested head."""
    url = str(context.state.answers.get('version_pr_url', ''))
    if not re.fullmatch(
        r'https://github[.]com/kubeflow/pipelines/pull/[0-9]+', url
    ):
        raise RuntimeError(
            'The version PR must be recorded and merged before qualification.'
        )
    pr = json.loads(
        context.runner.capture(
            [
                'gh',
                'pr',
                'view',
                url,
                '--repo',
                REPO,
                '--json',
                'state,baseRefName,headRefName,headRefOid,mergeCommit,statusCheckRollup',
            ],
            cwd=context.root,
        )
    )
    # The explicitly recorded PR may be a follow-up release fix on another
    # branch; repository, destination, CI and immutable commits define trust.
    if (
        pr.get('state') != 'MERGED'
        or pr.get('baseRefName') != context.metadata.release_branch
    ):
        raise RuntimeError(
            'Version PR must be merged into the intended release branch.'
        )
    sha = (pr.get('mergeCommit') or {}).get('oid', '')
    head = pr.get('headRefOid', '')
    if not re.fullmatch(r'[0-9a-f]{40}', sha) or not re.fullmatch(
        r'[0-9a-f]{40}', head
    ):
        raise RuntimeError(
            'Version PR has no valid merged commit or tested head.'
        )
    checks = pr.get('statusCheckRollup') or []
    passed = False
    for check in checks:
        # Tide reports merge-queue eligibility, not a CI result. Its pending
        # status can remain on the tested head after the PR has been merged.
        if is_pending_merge_queue_check(check):
            continue
        status = check.get('status') or check.get('state')
        result = check.get('conclusion') if status == 'COMPLETED' else status
        if result not in ('SUCCESS', 'NEUTRAL', 'SKIPPED'):
            raise RuntimeError(
                f'Latest version PR CI is not green: {check.get("name") or check.get("context")}'
            )
        passed = passed or result == 'SUCCESS'
    if not passed:
        raise RuntimeError(
            'Version PR requires successful CI on its latest head.'
        )
    for key, value in (('version_pr_head_sha', head), ('release_sha', sha)):
        previous = context.state.answers.get(key)
        if previous and previous != value:
            raise RuntimeError(
                f'{key} changed; qualify the new release commit explicitly.'
            )
        context.state.answers[key] = value
    return sha


def step_qualify_release(context: ReleaseContext) -> None:
    """Qualify the exact merged release before any publication."""
    qualification.qualify_release(context)


def _ensure_release_tag(context: ReleaseContext, tag: str) -> None:
    """Create an immutable upstream tag on the qualified merge commit."""
    sha = str(context.state.answers.get('release_sha', ''))
    if context.runner.dry_run:
        print(f'[dry-run] would tag qualified merged commit as {tag}')
        return
    if not re.fullmatch(r'[0-9a-f]{40}', sha):
        raise RuntimeError('No qualified merged release commit is recorded.')
    upstream = f'https://github.com/{REPO}.git'
    published = False
    upstream_exists = False
    for remote in dict.fromkeys((upstream, context.fork_remote)):
        output = context.runner.capture(
            [
                'git',
                'ls-remote',
                '--tags',
                remote,
                f'refs/tags/{tag}',
                f'refs/tags/{tag}^{{}}',
            ],
            cwd=context.root,
        )
        refs = dict(
            line.split()[1::-1] for line in output.splitlines() if line.strip()
        )
        target = refs.get(f'refs/tags/{tag}^{{}}', refs.get(f'refs/tags/{tag}'))
        if target:
            if target != sha:
                raise RuntimeError(
                    f'Published tag {tag} on {remote} differs from qualified commit; refusing replacement.'
                )
            published = True
            upstream_exists = upstream_exists or remote == upstream
    context.runner.run(
        ['git', 'fetch', '--no-tags', upstream, sha], cwd=context.root
    )
    local = context.runner.run(
        ['git', 'rev-parse', '--verify', f'refs/tags/{tag}^{{commit}}'],
        cwd=context.root,
        check=False,
    )
    local_sha = (
        context.runner.capture(
            ['git', 'rev-parse', '--verify', f'refs/tags/{tag}^{{commit}}'],
            cwd=context.root,
        ).strip()
        if local.returncode == 0
        else ''
    )
    if local.returncode == 0 and local_sha != sha:
        if published:
            raise RuntimeError(
                f'Local tag {tag} differs from a published tag; refusing replacement.'
            )
        context.runner.run(['git', 'tag', '-d', tag], cwd=context.root)
        local = None
    if local is None or local.returncode != 0:
        context.runner.run(
            [
                'git',
                'tag',
                '-a',
                tag,
                sha,
                '-m',
                f'Kubeflow Pipelines {tag} release',
            ],
            cwd=context.root,
        )
    if not upstream_exists:
        context.runner.run(
            ['git', 'push', upstream, f'refs/tags/{tag}:refs/tags/{tag}'],
            cwd=context.root,
        )


def _should_set_backend_latest(context: ReleaseContext) -> bool:
    """Promote only a stable version at least as new as every backend
    release."""
    if context.runner.dry_run:
        print(
            '[dry-run] would compare published stable backend releases before selecting latest'
        )
        return False
    version_pattern = re.compile(
        r'^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)'
        r'(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?'
        r'(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$'
    )
    target = version_pattern.fullmatch(context.metadata.tag)
    if not target:
        raise ValueError('Cannot select latest for a malformed release version')
    if target.group(4):
        return False
    target_version = tuple(int(target.group(index)) for index in (1, 2, 3))
    pages = json.loads(
        context.runner.capture(
            [
                'gh',
                'api',
                '--paginate',
                '--slurp',
                f'repos/{REPO}/releases?per_page=100',
            ],
            cwd=context.root,
        )
    )
    if not isinstance(pages, list) or any(
        not isinstance(page, list) for page in pages
    ):
        raise ValueError(
            'Malformed published release discovery; cannot select latest'
        )
    latest = (0, 0, 0)
    for page in pages:
        for release in page:
            if (
                not isinstance(release, dict)
                or type(release.get('draft')) is not bool
                or type(release.get('prerelease')) is not bool
                or not isinstance(release.get('tag_name'), str)
            ):
                raise ValueError(
                    'Malformed published release metadata; cannot select latest'
                )
            tag = release['tag_name']
            if (
                release['draft']
                or release['prerelease']
                or tag.startswith('sdk-')
            ):
                continue
            version = version_pattern.fullmatch(tag)
            if not version:
                raise ValueError(f'Malformed stable backend release tag: {tag}')
            if version.group(4):
                continue
            latest = max(
                latest, tuple(int(version.group(index)) for index in (1, 2, 3))
            )
    return target_version >= latest


def step_publish_images(context: ReleaseContext) -> None:
    """Trigger and watch image publication workflow.

    Args:
      context: Release context with runner and metadata.
    """
    qualification.require_qualification(context)
    set_latest = _should_set_backend_latest(context)
    _ensure_release_tag(context, context.metadata.tag)
    context.runner.run(
        image_workflow_command(
            context.metadata,
            source_sha=context.state.answers.get('release_sha'),
            workflow_ref=context.metadata.tag,
            set_latest=set_latest,
        )
    )
    watch_latest_workflow_run(
        context.runner,
        'image-builds-release.yml',
        context.metadata.tag,
        expected_sha=context.state.answers.get('release_sha'),
    )


def _replace(path: Path, pattern: str, replacement: str) -> None:
    if path.exists():
        path.write_text(re.sub(pattern, replacement, path.read_text()))


def _update_sdk_docs_versions(path: Path, version: str) -> None:
    if not path.exists():
        return
    text = path.read_text()
    if f'"title": "{version}"' in text:
        return
    text = re.sub(
        r'"aliases": \[\s*"stable",\s*"latest"\s*\]',
        '"aliases": []',
        text,
        count=1,
    )
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
    major_minor_version = '.'.join(version.split('.')[:2])
    text = path.read_text()
    if f"'kfp-kubernetes-{major_minor_version}/'" in text:
        return
    text = re.sub(r"'aliases': \['stable'\]", "'aliases': []", text, count=1)
    entry = f"""        {{
            'version':
                'https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-{major_minor_version}/',
            'title':
                '{major_minor_version}',
            'aliases': ['stable'],
        }},
"""
    path.write_text(
        text.replace(
            "    'version_info': [\n", "    'version_info': [\n" + entry, 1
        )
    )


def _notes_range(
    context: ReleaseContext, product: str, publishing: bool = False
) -> ReleaseRange:
    key = 'previous_sdk_release' if product == 'sdk' else 'previous_release'
    base = getattr(context, key)
    if not base:
        base = prompt_required(f'Previous {product} release tag')
        setattr(context, key, base)
        context.state.answers[key] = base
    target = context.release_target
    tag = context.metadata.sdk_tag if product == 'sdk' else context.metadata.tag
    if not target:
        tag_ref = f'refs/tags/{tag}'
        tag_exists = (
            publishing
            and subprocess.run(
                ['git', 'show-ref', '--verify', '--quiet', tag_ref],
                cwd=context.root,
                check=False,
            ).returncode
            == 0
        )
        if publishing:
            target = (
                tag_ref
                if tag_exists
                else (
                    f'refs/remotes/upstream/{context.metadata.release_branch}'
                )
            )
        else:
            target = f'refs/heads/{context.metadata.tag}-update-version-tags'
    selected = resolve_range(context.root, base, target)
    qualified_sha = context.state.answers.get('release_sha')
    if publishing and qualified_sha and selected.target_sha != qualified_sha:
        raise ValueError(
            'Release notes target differs from the qualified release commit.'
        )
    print(
        f'{product} release range: {base} ({selected.base_sha}) '
        f'.. {target} ({selected.target_sha})'
    )
    return selected


def _preparation_range(context: ReleaseContext, product: str) -> ReleaseRange:
    selected = _notes_range(context, product)
    checkout = resolve_range(context.root, selected.target_sha, 'HEAD')
    if checkout.target_sha != selected.target_sha:
        raise ValueError(
            'Release preparation target differs from the checked-out version branch. '
            'Select its current commit with --release-target before preparing notes.'
        )
    return selected


def _remote_release_sha(root: Path, tag: str) -> str:
    ref = f'refs/tags/{tag}'
    try:
        output = subprocess.check_output(
            [
                'git',
                'ls-remote',
                '--tags',
                f'https://github.com/{REPO}.git',
                ref,
                ref + '^{}',
            ],
            cwd=root,
            text=True,
        )
    except (OSError, subprocess.CalledProcessError) as error:
        raise ValueError(
            'Cannot verify the published release tag. Check GitHub '
            'access and retry before publishing.'
        ) from error
    refs = dict(line.split()[::-1] for line in output.splitlines())
    sha = refs.get(ref + '^{}', refs.get(ref, ''))
    if sha and not re.fullmatch(r'[0-9a-f]{40}|[0-9a-f]{64}', sha):
        raise ValueError('GitHub returned an invalid release tag commit.')
    return sha


def _verify_release_target(
    context: ReleaseContext, tag: str, selected: ReleaseRange
) -> None:
    remote_sha = _remote_release_sha(context.root, tag)
    if remote_sha and remote_sha != selected.target_sha:
        raise ValueError(
            f'Published tag {tag} differs from the reviewed release target. '
            'Fetch the tag and regenerate notes for its commit before publishing.'
        )
    if not remote_sha and tag.startswith('sdk-'):
        raise ValueError(
            f'SDK tag {tag} is not published. Complete create-sdk-tag first.'
        )


def _generate_sdk_release_notes(root: Path, selected: ReleaseRange) -> str:
    return render_notes(read_commits(root, selected), product='sdk')


def _generate_backend_release_notes(root: Path, selected: ReleaseRange) -> str:
    return render_notes(read_commits(root, selected), product='backend')


def _update_changelog(
    context: ReleaseContext, selected: ReleaseRange | None = None
) -> None:
    if context.runner.dry_run:
        print(
            '[dry-run] would update CHANGELOG.md from the explicit backend range'
        )
        return
    selected = selected or _preparation_range(context, 'backend')
    body = render_notes(read_commits(context.root, selected))
    if not body:
        raise ValueError(
            'No user-facing changelog entries in the release range. '
            'Check the base and target before preparing the release.'
        )
    path = context.root / 'CHANGELOG.md'
    text = path.read_text() if path.exists() else 'Changelog\n'
    version = context.metadata.tag
    heading = f'## {version} ({commit_date(context.root, selected.target_sha)})'
    entry = (
        heading
        + '\n\n'
        + re.sub(r'(?m)^## ', '### ', body)
        + f'\n\n**Full changelog:** {selected.compare_url}\n'
    )
    identity = {
        'version': version,
        'base_sha': selected.base_sha,
        'target_sha': selected.target_sha,
    }
    prepared = context.state.answers.get('prepared_changelog')
    if prepared and any(
        prepared.get(key) != value for key, value in identity.items()
    ):
        raise ValueError(
            'Changelog preparation range changed. Review the existing notes and '
            'use a fresh release state before preparing a different candidate.'
        )
    existing = re.search(rf'(?m)^## {re.escape(version)}(?:[ \n]|$)', text)
    if existing:
        end = re.search(r'(?m)^## ', text[existing.end() :])
        stop = existing.end() + end.start() if end else len(text)
        expected = prepared['entry'] if prepared else entry
        if text[existing.start() : stop].strip() != expected.strip():
            raise ValueError(
                f'CHANGELOG.md already contains reviewed or edited notes for {version}. '
                'Review that section manually; release preparation will not overwrite it.'
            )
        return
    unreleased = re.search(r'(?m)^## Unreleased[ \t]*$', text)
    if unreleased:
        end = re.search(r'(?m)^## ', text[unreleased.end() :])
        stop = unreleased.end() + end.start() if end else len(text)
        authored = text[unreleased.end() : stop].strip()
        if authored:
            entry = (
                heading
                + '\n\n'
                + authored
                + '\n\n'
                + entry[len(heading) :].lstrip('\n')
            )
        text = text[: unreleased.end()] + '\n\n' + entry + '\n' + text[stop:]
    else:
        header, separator, history = text.partition('\n')
        text = header + '\n\n' + entry + '\n' + history.lstrip('\n')
    # Persist before writing: a failure after writing the changelog must be
    # resumable without duplicating authored guidance or accepting a new range.
    context.state.answers['prepared_changelog'] = {**identity, 'entry': entry}
    context.state.save()
    _write_atomic(path, text.encode('utf-8'))


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
    match = re.match(
        r'(# Current Version \(in development\)\n\n)(.*?)(\n# \d)',
        text,
        re.DOTALL,
    )
    if not match:
        return
    authored_body = match.group(2).strip()
    template_headings = set(default_body.splitlines())
    has_authored_notes = any(
        line.strip() and line.strip() not in template_headings
        for line in authored_body.splitlines()
    )
    if release_body and has_authored_notes:
        generated_body = re.sub(r'(?m)^## ', '### ', release_body.strip())
        release_body = (
            f'{authored_body}\n\n## Generated changes\n\n{generated_body}'
        )
    else:
        release_body = release_body or authored_body or default_body
    path.write_text(
        f'{match.group(1)}{default_body}\n\n# {version}\n\n{release_body}\n{match.group(3)}{text[match.end() :]}'
    )


def _update_uv_package_versions(root: Path, sdk_version: str) -> None:
    """Update handwritten package metadata for the SDK release."""
    next_major = int(sdk_version.split('.')[0]) + 1
    _replace(
        root / 'api/v2alpha1/python/pyproject.toml',
        r'(?m)^version\s*=\s*"[^"]+"',
        f'version = "{sdk_version}"',
    )
    for package in ('kfp-pipeline-spec', 'kfp-server-api'):
        _replace(
            root / 'sdk/python/pyproject.toml',
            rf'{package}>=[^,"\']+,<\d+',
            f'{package}>={sdk_version},<{next_major}',
        )
    _replace(
        root / 'sdk/python/pyproject.toml',
        r'kfp-kubernetes==[^"\']+',
        f'kfp-kubernetes=={sdk_version}',
    )
    _replace(
        root / 'kubernetes_platform/python/pyproject.toml',
        r'kfp>=[^,"\']+,<\d+',
        f'kfp>={sdk_version},<{next_major}',
    )


def _refresh_uv_release_packages(
    context: ReleaseContext, verify_pins: Callable[[], None] | None = None
) -> None:
    """Regenerate the workspace lock and release dists, preserving legacy
    exports."""
    if (context.root / 'sdk/python/kfp/server_api/__init__.py').is_file():
        context.runner.run(
            ['make', '-C', 'sdk', 'generate-python'], cwd=context.root
        )
        context.runner.run(['uv', 'lock'], cwd=context.root)
        if verify_pins:
            verify_pins()
        if (context.root / 'requirements.txt').is_file():
            # Older consolidated checkouts still commit both requirements exports.
            # Current checkouts build from metadata and the lock without exports.
            export_command = [
                'uv',
                'export',
                '--frozen',
                '--no-dev',
                '--no-hashes',
            ]
            context.runner.run(
                export_command
                + ['--format', 'requirements-txt', '-o', 'requirements.txt'],
                cwd=context.root,
            )
            context.runner.run(
                export_command
                + [
                    '--package',
                    'kfp',
                    '--format',
                    'requirements-txt',
                    '-o',
                    'sdk/python/requirements.txt',
                ],
                cwd=context.root,
            )
        context.runner.run(
            ['uv', 'build', '--package', 'kfp', '--out-dir', 'sdk/python/dist'],
            cwd=context.root,
        )
        return
    context.runner.run(
        ['make', 'API_VERSION=v2beta1', 'generate-kfp-server-api-package'],
        cwd=context.root / 'backend/api',
    )
    context.runner.run(['uv', 'lock'], cwd=context.root)
    if verify_pins:
        verify_pins()
    export_command = ['uv', 'export', '--frozen', '--no-dev', '--no-hashes']
    context.runner.run(
        export_command
        + ['--format', 'requirements-txt', '-o', 'requirements.txt'],
        cwd=context.root,
    )
    for package, path in (
        ('kfp-pipeline-spec', 'api/v2alpha1/python'),
        ('kfp-server-api', 'backend/api/v2beta1/python_http_client'),
        ('kfp', 'sdk/python'),
        ('kfp-kubernetes', 'kubernetes_platform/python'),
    ):
        if package != 'kfp-server-api':
            context.runner.run(
                export_command
                + [
                    '--package',
                    package,
                    '--format',
                    'requirements-txt',
                    '-o',
                    f'{path}/requirements.txt',
                ],
                cwd=context.root,
            )
        context.runner.run(
            ['uv', 'build', '--package', package, '--out-dir', f'{path}/dist'],
            cwd=context.root,
        )


def _update_sdk_version_files(
    context: ReleaseContext, selected: ReleaseRange | None = None
) -> None:
    """Update SDK release metadata and regenerate the matching package
    artifacts."""
    metadata = context.metadata
    root = context.root
    uses_uv = (root / 'uv.lock').exists()
    consolidated = (root / 'sdk/python/kfp/server_api/__init__.py').is_file()
    generated_notes = ''
    if not context.runner.dry_run and (root / 'sdk/RELEASE.md').exists():
        generated_notes = _generate_sdk_release_notes(
            root, selected or _preparation_range(context, 'sdk')
        )

    # Update SDK version files
    if context.runner.dry_run:
        print(f'[dry-run] would update SDK version files to {metadata.tag}')
    else:
        _replace(
            root / 'sdk/python/kfp/version.py',
            r"__version__\s*=\s*['\"]([^'\"]+)['\"]",
            f"__version__ = '{metadata.tag}'",
        )
        if not consolidated:
            _replace(
                root / 'kubernetes_platform/python/kfp/kubernetes/__init__.py',
                r"__version__\s*=\s*['\"]([^'\"]+)['\"]",
                f"__version__ = '{metadata.tag}'",
            )
        if uses_uv and not consolidated:
            _update_uv_package_versions(root, metadata.tag)
        elif not uses_uv and not consolidated:
            _replace(
                root / 'api/v2alpha1/python/setup.py',
                r"VERSION\s*=\s*['\"]([^'\"]+)['\"]",
                f"VERSION = '{metadata.tag}'",
            )
            _replace(
                root / 'backend/api/v2beta1/python_http_client/setup.py',
                r"VERSION\s*=\s*['\"]([^'\"]+)['\"]",
                f'VERSION = "{metadata.tag}"',
            )
            _replace(
                root
                / 'backend/api/v2beta1/python_http_client/kfp_server_api/__init__.py',
                r"__version__\s*=\s*['\"]([^'\"]+)['\"]",
                f'__version__ = "{metadata.tag}"',
            )
            next_major = metadata.major + 1
            _replace(
                root / 'sdk/python/requirements.in',
                r'kfp-pipeline-spec>=[^,\n]+,<\d+',
                f'kfp-pipeline-spec>={metadata.tag},<{next_major}',
            )
            _replace(
                root / 'sdk/python/requirements.in',
                r'kfp-server-api>=[^,\n]+,<\d+',
                f'kfp-server-api>={metadata.tag},<{next_major}',
            )
            _replace(
                root / 'kubernetes_platform/python/requirements.in',
                r'kfp>=[^,\n]+,<\d+',
                f'kfp>={metadata.tag},<{next_major}',
            )
        _update_sdk_docs_versions(root / 'docs/sdk/versions.json', metadata.tag)
        if not consolidated:
            _update_kfp_kubernetes_docs_versions(
                root / 'kubernetes_platform/python/docs/conf.py', metadata.tag
            )
        _update_sdk_release_notes(
            root / 'sdk/RELEASE.md',
            metadata.tag,
            generated_notes,
        )

    with preserve_dependency_pins(
        root, dry_run=context.runner.dry_run
    ) as verify_pins:
        if uses_uv:
            _refresh_uv_release_packages(context, verify_pins)
            return

        # Retain imported helpers: release checkout may remove their source files.
        if context.runner.dry_run:
            print('[dry-run] would clean legacy release distributions')
        else:
            clean_legacy_distributions(root)
        # Legacy wheels need generated sources before resolving unpublished dists.
        context.runner.run(['make', 'python'], cwd=root / 'api')
        context.runner.run(
            [sys.executable, '-m', 'build', '.'],
            cwd=root / 'api/v2alpha1/python',
        )
        context.runner.run(
            [sys.executable, '-m', 'build', '.'],
            cwd=root / 'backend/api/v2beta1/python_http_client',
        )
        context.runner.run(kfp_requirements_command(), cwd=root / 'sdk/python')
        verify_pins()
        context.runner.run(
            [sys.executable, '-m', 'build', '.'], cwd=root / 'sdk/python'
        )
        context.runner.run(
            kubernetes_requirements_command(),
            cwd=root / 'kubernetes_platform/python',
        )
        verify_pins()
        context.runner.run(
            ['make', 'python', 'USE_FIND_LINKS=true'],
            cwd=root / 'kubernetes_platform',
        )
        context.runner.run(
            [sys.executable, '-m', 'build', '.'],
            cwd=root / 'kubernetes_platform/python',
        )
        if context.runner.dry_run:
            print('[dry-run] would validate legacy release wheels')
        else:
            validate_legacy_wheels(root, metadata.tag)


def step_create_sdk_tag(context: ReleaseContext) -> None:
    """Publish the SDK tag only on the qualified merged commit."""
    qualification.require_qualification(context)
    _ensure_release_tag(context, context.metadata.sdk_tag)


def _sdk_release_is_consolidated(context: ReleaseContext) -> bool:
    """Inspect the published SDK tag, not the CLI's checkout or version."""
    contents = context.runner.capture(
        [
            'gh',
            'api',
            f'repos/{REPO}/contents/sdk/python/kfp?ref={context.metadata.sdk_tag}',
            '--jq',
            '.[].name',
        ],
        cwd=context.root,
    ).splitlines()
    if 'version.py' not in contents:
        raise RuntimeError(
            f'Cannot determine packaging layout for {context.metadata.sdk_tag}. '
            'Verify that create-sdk-tag completed and the tag contains the SDK.'
        )
    return 'server_api' in contents


def _release_preview_path(context: ReleaseContext, tag: str) -> Path:
    state_path = context.state.path.resolve()
    return state_path.parent / f'{state_path.stem}-notes' / f'{tag}.md'


def _editorial_evidence(
    context: ReleaseContext, selected: ReleaseRange, product: str
) -> str:
    """Keep complete evidence beside each range-bound preview, outside
    prose."""
    inventory = build_inventory(context.root, selected, context.metadata.tag)
    tag = context.metadata.sdk_tag if product == 'sdk' else context.metadata.tag
    directory = _release_preview_path(context, tag).parent
    directory.mkdir(parents=True, exist_ok=True)
    path = (
        directory
        / f'{tag}.{selected.base_sha[:12]}.{selected.target_sha[:12]}.inventory.json'
    )
    if not path.exists():
        path.write_text(json.dumps(inventory, indent=2) + '\n')
    print(f'Complete release inventory: {path}')
    return release_scaffolding(inventory, product)


def regenerate_release_notes(context: ReleaseContext, product: str) -> bool:
    """Refresh an existing preview without publishing or changing
    checkpoints."""
    if product not in ('backend', 'sdk'):
        raise ValueError('Choose backend or sdk for preview regeneration.')
    selected = _notes_range(context, product, publishing=True)
    tag = context.metadata.sdk_tag if product == 'sdk' else context.metadata.tag
    render = _sdk_release_body if product == 'sdk' else _backend_release_body
    path = _release_preview_path(context, tag)
    clean = regenerate_preview(path, selected, render(context, selected))
    print(f'Release notes preview: {path}')
    if clean:
        _print_notes_warnings(path.read_text())
    return clean


def _print_notes_warnings(notes: str) -> None:
    for warning in validate_notes(notes):
        print(f'Release notes warning: {warning}')


def _release_notes_preview(
    context: ReleaseContext,
    tag: str,
    render: Callable[[], str],
    selected: ReleaseRange,
) -> Path:
    """Create or reuse editable notes, then review them before publication."""
    path = _release_preview_path(context, tag)
    provenance_path = path.with_suffix('.json')
    if path.exists():
        try:
            provenance = json.loads(provenance_path.read_text())
        except (OSError, ValueError) as error:
            raise ValueError(
                f'Preview provenance is missing or invalid: {provenance_path}. '
                'Move the preview and sidecar aside, regenerate, and reapply reviewed edits.'
            ) from error
        if not isinstance(provenance, dict) or any(
            provenance.get(key) != selected.as_dict()[key]
            for key in ('base_sha', 'target_sha')
        ):
            raise ValueError(
                f'Release range changed for {path}. Run kfpr regenerate-notes '
                'with the selected range, then review the merged preview before publishing.'
            )
    else:
        notes = render()
        if not notes.strip():
            raise ValueError(f'Release notes for {tag} are empty.')
        path.parent.mkdir(parents=True, exist_ok=True)
        notes = notes.rstrip() + '\n'
        path.write_text(notes)
        provenance_path.write_text(
            json.dumps(
                {**selected.as_dict(), 'generated_notes': notes}, indent=2
            )
            + '\n'
        )
    print(f'Release notes preview: {path}')
    reviewed_text = path.read_text()
    print(reviewed_text)
    _print_notes_warnings(reviewed_text)
    if has_conflicts(reviewed_text):
        raise ValueError(
            f'Resolve merge conflict markers in {path} before publishing.'
        )
    if not context.runner.dry_run:
        confirm(f'Review and edit {path}, then publish these release notes?')
        if path.read_text() != reviewed_text:
            _print_notes_warnings(path.read_text())
    if has_conflicts(path.read_text()):
        raise ValueError(
            f'Resolve merge conflict markers in {path} before publishing.'
        )
    if not path.read_text().strip():
        raise ValueError(
            f'Release notes are empty: {path}. Edit them before retrying.'
        )
    baseline = json.loads(provenance_path.read_text()).get(
        'generated_notes', ''
    )
    restored = restore_generated_sections(path.read_text(), baseline)
    if restored != path.read_text():
        if context.runner.dry_run:
            print(
                '[dry-run] generated contributor and asset sections would be restored before publication.'
            )
        else:
            _write_atomic(path, restored.encode('utf-8'))
            print(
                'Restored generated contributor and asset sections; editorial prose is unchanged.'
            )
    return path


def _sdk_release_body(
    context: ReleaseContext,
    selected: ReleaseRange,
    consolidated: bool | None = None,
) -> str:
    metadata = context.metadata
    if consolidated is None and context.runner.dry_run:
        contents = subprocess.check_output(
            [
                'git',
                'ls-tree',
                '--name-only',
                selected.target_sha + ':sdk/python/kfp',
            ],
            cwd=context.root,
            text=True,
        ).splitlines()
        consolidated = 'server_api' in contents
    elif consolidated is None:
        consolidated = _sdk_release_is_consolidated(context)
    if consolidated:
        notes = f"""Release of the unified KFP SDK, including Kubernetes configuration,
the server API client, and pipeline-spec bindings.

To install the KFP SDK:

```
pip install kfp=={metadata.tag}
```

When migrating from the split SDK, remove the legacy distributions before
installing the unified wheel:

```
python -m pip uninstall -y kfp-pipeline-spec kfp-server-api kfp-kubernetes
python -m pip install --upgrade --force-reinstall kfp=={metadata.tag}
```
"""
    else:
        notes = f"""Release of:

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
"""
    notes += f"""
For changelog, see https://github.com/kubeflow/pipelines/blob/{metadata.sdk_tag}/sdk/RELEASE.md.
"""
    changes = _generate_sdk_release_notes(context.root, selected)
    if changes:
        notes += f'\n<details>\n<summary>Commit reference for editors</summary>\n\n{changes}\n</details>\n'
    return (
        re.sub(
            r'(?ms)^## Installation\n.*?(?=^## |\Z)',
            '',
            editorial_scaffold('sdk'),
        )
        + '## Installation\n\n'
        + notes
        + f'\n**Full changelog:** {selected.compare_url}\n\n'
        + _editorial_evidence(context, selected, 'sdk')
    )


def step_create_sdk_release(context: ReleaseContext) -> None:
    """Create the SDK GitHub release after packages are published."""
    qualification.require_qualification(context)
    _ensure_release_tag(context, context.metadata.sdk_tag)
    metadata = context.metadata
    if context.runner.dry_run:
        selected = _notes_range(context, 'sdk', publishing=True)
        _release_notes_preview(
            context,
            metadata.sdk_tag,
            lambda: _sdk_release_body(context, selected),
            selected,
        )
        print(f'[dry-run] would create SDK GitHub release: {metadata.sdk_tag}')
        return
    consolidated = _sdk_release_is_consolidated(context)
    existing_release = context.runner.run(
        ['gh', 'release', 'view', metadata.sdk_tag, '--repo', REPO], check=False
    )
    if existing_release.returncode == 0:
        choice = prompt_choice(
            f'SDK GitHub release {metadata.sdk_tag} already exists. Use it or recreate it?',
            ['u', 'r'],
            default='u',
        )
        if choice == 'u':
            return
    selected = _notes_range(context, 'sdk', publishing=True)
    _verify_release_target(context, metadata.sdk_tag, selected)
    notes_path = _release_notes_preview(
        context,
        metadata.sdk_tag,
        lambda: _sdk_release_body(context, selected, consolidated),
        selected,
    )
    _verify_release_target(context, metadata.sdk_tag, selected)
    if existing_release.returncode == 0:
        context.runner.run(
            [
                'gh',
                'release',
                'delete',
                metadata.sdk_tag,
                '--repo',
                REPO,
                '--yes',
            ]
        )
    create_command = [
        'gh',
        'release',
        'create',
        metadata.sdk_tag,
        '--repo',
        REPO,
        '--target',
        selected.target_sha,
        '--title',
        f'KFP SDK v{metadata.tag}',
        '--notes-file',
        str(notes_path),
    ]
    create_command.append('--latest=false')
    create_command.append('--verify-tag')
    context.runner.run(create_command)


def _is_pypi_version_published(package: str, version: str) -> bool:
    request = urllib.request.Request(
        f'https://pypi.org/pypi/{package}/{version}/json'
    )
    try:
        response = urllib.request.urlopen(request, timeout=30)
    except urllib.error.HTTPError as error:
        if error.code == 404:
            return False
        raise
    return response.status == 200


def _wait_for_pypi_packages(
    version: str,
    packages: list[str],
    sleep: Callable[[float], None] = time.sleep,
) -> list[str]:
    missing = packages
    for _ in range(30):
        missing = [
            package
            for package in packages
            if not _is_pypi_version_published(package, version)
        ]
        if not missing:
            print(f'All SDK packages are on PyPI for {version}.')
            return []
        print(f'Waiting for PyPI packages for {version}: {", ".join(missing)}')
        sleep(10)
    return missing


def step_publish_sdks(context: ReleaseContext) -> None:
    """Trigger and watch SDK publication workflow.

    Args:
      context: Release context with runner and metadata.
    """
    qualification.require_qualification(context)
    _ensure_release_tag(context, context.metadata.sdk_tag)
    if context.runner.dry_run:
        print(
            '[dry-run] would inspect the SDK tag layout and PyPI, publish only '
            'missing packages in dependency order, and wait for each to be public'
        )
        return
    packages = (
        ['kfp']
        if _sdk_release_is_consolidated(context)
        else ['kfp-pipeline-spec', 'kfp-server-api', 'kfp', 'kfp-kubernetes']
    )
    missing = [
        package
        for package in packages
        if not _is_pypi_version_published(package, context.metadata.tag)
    ]
    for package in missing:
        context.runner.run(
            sdk_workflow_command(
                context.metadata,
                packages=package,
                workflow_ref=context.metadata.sdk_tag,
            )
        )
        watch_latest_workflow_run(
            context.runner,
            'publish-packages.yml',
            context.metadata.sdk_tag,
            expected_sha=context.state.answers.get('release_sha'),
        )
        if _wait_for_pypi_packages(context.metadata.tag, [package]):
            raise RuntimeError(
                f'PyPI packages are still missing for {context.metadata.tag}: {package}. '
                'Resume publish-sdks once publication or PyPI availability is resolved.'
            )


def step_create_kfp_kubernetes_docs_branch(context: ReleaseContext) -> None:
    """Create a separate docs branch only for split-package SDK tags.

    Args:
      context: Release context with runner, metadata, and root path.
    """
    if context.runner.dry_run:
        print(
            '[dry-run] would create the Kubernetes docs branch only if '
            f'{context.metadata.sdk_tag} uses split packages'
        )
        return
    if _sdk_release_is_consolidated(context):
        print(
            'Skipping separate Kubernetes docs: included in unified SDK docs.'
        )
        return
    if context.runner.capture(
        ['git', 'status', '--short'], cwd=context.root
    ).strip():
        raise RuntimeError(
            'git working tree is dirty; commit or stash changes before '
            'creating the Kubernetes docs branch'
        )
    metadata = context.metadata
    root = context.root
    branch = f'kfp-kubernetes-{metadata.major}.{metadata.minor}'
    pkg_root = root / 'kubernetes_platform/python'
    branch_action = ''
    remote_branch = context.runner.run(
        ['git', 'ls-remote', '--exit-code', '--heads', 'upstream', branch],
        cwd=root,
        check=False,
    )
    branch_exists = remote_branch.returncode == 0
    if branch_exists:
        branch_action = prompt_choice(
            f'{branch} already exists on upstream. Reuse, replace, or force-push',
            ['reuse', 'replace', 'force-push'],
            default='reuse',
        )
        if branch_action == 'reuse':
            print(f'Reusing upstream/{branch}; skipping docs branch creation.')
            return
        if branch_action == 'replace':
            context.runner.run(
                ['git', 'push', 'upstream', f':{branch}'], cwd=root
            )

    context.runner.run(
        ['git', 'fetch', 'upstream', metadata.release_branch], cwd=root
    )
    context.runner.run(
        [
            'git',
            'checkout',
            '-B',
            metadata.release_branch,
            f'upstream/{metadata.release_branch}',
        ],
        cwd=root,
    )
    context.runner.run(
        kfp_kubernetes_docs_build_command(
            root, context.metadata.release_branch
        ),
        cwd=root,
    )
    context.runner.run(
        ['git', 'checkout', '-B' if branch_exists else '-b', branch], cwd=root
    )

    rtd_config_src = pkg_root / 'docs/.readthedocs.yml'
    rtd_config_dst = root / '.readthedocs.yml'
    if rtd_config_src.exists():
        rtd_config_dst.write_text(rtd_config_src.read_text())

    gitignore = root / 'kubernetes_platform/.gitignore'
    if gitignore.exists():
        gitignore.unlink()

    context.runner.run(
        [
            'git',
            'add',
            str(pkg_root / 'docs/.readthedocs.yml'),
            '.readthedocs.yml',
            'kubernetes_platform/.gitignore',
        ],
        cwd=root,
    )
    _run_reviewed_commit(
        context,
        [
            'git',
            'commit',
            '-s',
            '-m',
            f'chore: kfp-kubernetes docs branch for release {metadata.tag}',
        ],
        cwd=root,
    )
    push_command = ['git', 'push', '--set-upstream', 'upstream', branch]
    if branch_action == 'force-push':
        push_command = [
            'git',
            'push',
            '--force-with-lease',
            '--set-upstream',
            'upstream',
            branch,
        ]
    context.runner.run(push_command, cwd=root)


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

    consolidated = _sdk_release_is_consolidated(context)
    token = os.environ.get('RTD_TOKEN', '').strip()
    if token:
        print('Using Read the Docs API token from RTD_TOKEN.')
    while not token:
        token = getpass.getpass(
            emphasize_prompt('Read the Docs API token: ')
        ).strip()
        if not token:
            print('A Read the Docs API token is required.')
    try:
        client = rtd.ReadTheDocsClient(token)
        print('Updating ReadTheDocs projects...')
        rtd.update_release_docs(
            client,
            metadata.tag,
            metadata.release_branch,
            consolidated=consolidated,
            set_defaults=metadata.release_type != 'patch',
        )
        print('ReadTheDocs projects updated and verified.')
    except rtd.ReadTheDocsError as error:
        print(f'ReadTheDocs automation failed: {error}')
        print(manual_checklist('confirm-rtd', metadata, consolidated))
        confirm('Fall back to manual ReadTheDocs mode?')
        confirm(
            'Have the required ReadTheDocs projects been updated and verified?'
        )


def _backend_release_body(
    context: ReleaseContext, selected: ReleaseRange
) -> str:
    metadata = context.metadata
    changed = _generate_backend_release_notes(context.root, selected)
    if not changed.strip():
        raise ValueError(
            f'No backend release notes found after {selected.base_ref}. '
            'Check the previous release tag and commit history before retrying.'
        )
    notes = f"""<details>
<summary>Commit reference for editors</summary>

{changed}
</details>

**Full Changelog**: {selected.compare_url}
"""
    if metadata.major >= 3:
        notes += f"""
## Supported architectures

Linux AMD64 and ARM64 are supported starting with KFP 3.0. Standard manifests use shared image tags;
the container runtime selects the matching architecture. Pipeline component images must support
the architecture of their execution nodes. See the [supported-platforms policy](https://github.com/kubeflow/pipelines/blob/{metadata.tag}/docs/operator-guides/supported-platforms.md).
"""
    return (
        editorial_scaffold('backend')
        + notes
        + '\n'
        + _editorial_evidence(context, selected, 'backend')
    )


def step_create_backend_release(context: ReleaseContext) -> None:
    """Review Markdown notes and create the backend GitHub release."""
    qualification.require_qualification(context)
    set_latest = _should_set_backend_latest(context)
    _ensure_release_tag(context, context.metadata.tag)
    metadata = context.metadata
    checklist = manual_checklist('create-backend-release', metadata)
    if checklist:
        print(checklist)
        if not context.runner.dry_run:
            confirm(
                'Have the release-image architecture checks succeeded and their evidence been reviewed?'
            )

    selected = _notes_range(context, 'backend', publishing=True)
    if not context.runner.dry_run:
        _verify_release_target(context, metadata.tag, selected)
    notes_path = _release_notes_preview(
        context,
        metadata.tag,
        lambda: _backend_release_body(context, selected),
        selected,
    )
    if context.runner.dry_run:
        print(f'[dry-run] would create backend GitHub release: {metadata.tag}')
        return

    _verify_release_target(context, metadata.tag, selected)
    context.runner.run(
        [
            'gh',
            'release',
            'create',
            metadata.tag,
            '--repo',
            REPO,
            '--target',
            selected.target_sha,
            '--title',
            f'Version {metadata.tag}',
            '--notes-file',
            str(notes_path),
            f'--latest={str(set_latest).lower()}',
            '--verify-tag',
        ]
    )


def step_sync_master(context: ReleaseContext) -> None:
    """Sync release version to master branch.

    Args:
      context: Release context with runner, metadata, and root path.
    """
    qualification.require_qualification(context)
    if context.metadata.release_type == 'patch':
        print(
            'Patch release: preserving master version and changelog; no sync required.'
        )
        return
    _ensure_release_tag(context, context.metadata.tag)
    metadata = context.metadata
    root = context.root
    branch = f'chore-release-{metadata.tag}-master'

    context.runner.run(['git', 'checkout', 'master'], cwd=root)
    context.runner.run(['git', 'pull', '--ff-only'], cwd=root)
    context.runner.run(['git', 'checkout', '-b', branch], cwd=root)

    # Keep VERSION compatible with the repository end-of-file formatting hook.
    if context.runner.dry_run:
        print(f'[dry-run] would write VERSION: {metadata.tag}')
    else:
        (root / 'VERSION').write_text(metadata.tag + '\n')

    release_image_tag = (
        'master'
        if metadata.release_type in ('major', 'minor')
        else metadata.release_branch
    )
    context.runner.run(
        release_version_bump_command(
            root, metadata.release_branch, release_image_tag
        ),
        cwd=root,
    )
    context.runner.run(
        ['git', 'checkout', metadata.tag, '--', 'CHANGELOG.md'], cwd=root
    )
    context.runner.run(['git', 'add', '-A'], cwd=root)
    _run_reviewed_commit(
        context,
        [
            'git',
            'commit',
            '-s',
            '-m',
            f'chore(release): bump version to {metadata.tag} on master branch',
        ],
        cwd=root,
    )
    context.runner.run(
        ['git', 'push', '--set-upstream', context.fork_remote, branch], cwd=root
    )
    owner = parse_github_owner(context.fork_remote)
    pr_url = context.runner.capture(
        [
            'gh',
            'pr',
            'create',
            '--repo',
            REPO,
            '--base',
            'master',
            '--head',
            f'{owner}:{branch}',
            '--title',
            f'chore(release): sync {metadata.tag} to master',
            '--body',
            f'Syncs KFP {metadata.tag} release version files back to master.',
        ]
    )
    print(underline_links(f'Master sync PR: {pr_url}'))
    watch_pr_ci(context.runner, pr_url)
    wait_for_pr_merge(context.runner, pr_url)


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
    print("Congratulations! You've completed the release!")


def build_steps(
    release_type: str, include_backend: bool, include_sdk: bool
) -> list[Step]:
    """Build the list of release steps based on configuration.

    Args:
      release_type: One of 'major', 'minor', or 'patch'.
      include_backend: Whether to include backend steps.
      include_sdk: Whether to include SDK steps.

    Returns:
      Ordered list of Step instances for the release.
    """
    steps = [
        Step('preflight', 'Verify tools and GitHub auth', 'step_preflight')
    ]
    if release_type == 'patch':
        steps.extend(
            [
                Step(
                    'prepare-patch-branch',
                    'Create patch branch from release branch',
                    'step_prepare_patch_branch',
                ),
                Step(
                    'cherry-pick-prs',
                    'Cherry-pick requested PR merge commits',
                    'step_cherry_pick_prs',
                ),
                Step(
                    'merge-cherry-pick-pr',
                    'Open and wait for cherry-pick PR',
                    'step_merge_cherry_pick_pr',
                ),
            ]
        )
    else:
        steps.append(
            Step(
                'prepare-release-branch',
                'Create release branch from master',
                'step_prepare_release_branch',
            )
        )
    steps.extend(
        [
            Step(
                'update-version-tags',
                'Update repository version tags',
                'step_update_version_tags',
            ),
            Step(
                'merge-version-pr',
                'Open and wait for version bump PR',
                'step_merge_version_pr',
            ),
        ]
    )
    steps.append(
        Step(
            'qualify-release',
            'Qualify the merged release commit',
            'step_qualify_release',
        )
    )
    if include_backend:
        steps.append(
            Step(
                'publish-images',
                'Run image publication workflow',
                'step_publish_images',
            )
        )
    if include_sdk:
        steps.extend(
            [
                Step('create-sdk-tag', 'Create SDK tag', 'step_create_sdk_tag'),
                Step(
                    'publish-sdks',
                    'Run SDK publication workflow',
                    'step_publish_sdks',
                ),
                Step(
                    'create-kfp-kubernetes-docs-branch',
                    'Create Kubernetes docs branch for split SDK releases',
                    'step_create_kfp_kubernetes_docs_branch',
                ),
                Step(
                    'confirm-rtd',
                    'Confirm ReadTheDocs updates',
                    'step_confirm_rtd',
                ),
                Step(
                    'create-sdk-release',
                    'Create SDK GitHub release',
                    'step_create_sdk_release',
                ),
            ]
        )
    if include_backend:
        steps.extend(
            [
                Step(
                    'create-backend-release',
                    'Create backend GitHub release',
                    'step_create_backend_release',
                ),
                Step(
                    'sync-master',
                    'Sync release version to master',
                    'step_sync_master',
                ),
                Step(
                    'confirm-website-and-slack',
                    'Confirm website PR and Slack announcement',
                    'step_confirm_website_and_slack',
                ),
            ]
        )
    return steps


def run_steps(context: ReleaseContext) -> None:
    """Execute release steps in order, skipping completed ones.

    Args:
      context: Release context with state, runner, and metadata.
    """
    steps = build_steps(
        context.metadata.release_type,
        context.include_backend,
        context.include_sdk,
    )
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
        if context.runner.dry_run and step.step_id in (
            'create-sdk-release',
            'create-backend-release',
        ):
            # Earlier simulated steps do not create the release refs or contents.
            print(
                '[dry-run] release notes preview deferred until release refs '
                'and contents exist. Then run '
                f'kfpr run {step.step_id} --dry-run to review the notes.'
            )
            continue
        handlers[step.handler](context)
        if not context.runner.dry_run:
            context.state.mark_done(step.step_id)
            context.state.save()


# Build the step registry mapping step_id to handler for all built steps
STEP_HANDLERS = {}
for release_type in ('major', 'minor', 'patch'):
    for step in build_steps(
        release_type, include_backend=True, include_sdk=True
    ):
        STEP_HANDLERS[step.step_id] = globals()[step.handler]
