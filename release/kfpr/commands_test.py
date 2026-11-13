#!/usr/bin/env python3
"""Tests for release command construction."""

import os
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from kfpr import core


class GithubCommandTest(unittest.TestCase):
    def test_image_workflow_command(self):
        metadata = core.ReleaseMetadata.from_version('patch', '3.2.1')
        command = core.image_workflow_command(metadata)
        self.assertEqual(
            command[:4], ['gh', 'workflow', 'run', 'image-builds-release.yml']
        )
        self.assertIn('src_branch=release-3.2', command)
        self.assertIn('target_tag=3.2.1', command)

    def test_image_workflow_dispatch_preserves_release_branch_contract(self):
        for version, release_type, branch in (
            ('2.18.0', 'minor', 'release-2.18'),
            ('2.18.1', 'patch', 'release-2.18'),
            ('3.0.0', 'major', 'release-3.0'),
            ('3.0.1', 'patch', 'release-3.0'),
        ):
            with self.subTest(version=version):
                metadata = core.ReleaseMetadata.from_version(
                    release_type, version
                )
                self.assertEqual(
                    core.image_workflow_command(metadata, set_latest=True),
                    [
                        'gh',
                        'workflow',
                        'run',
                        'image-builds-release.yml',
                        '--repo',
                        core.REPO,
                        '--ref',
                        branch,
                        '-f',
                        f'src_branch={branch}',
                        '-f',
                        f'target_tag={version}',
                        '-f',
                        'overwrite_imgs=false',
                        '-f',
                        'set_latest=true',
                        '-f',
                        'add_sha_tag=true',
                        '-f',
                        'dry_run=false',
                    ],
                )

    def test_sdk_workflow_command(self):
        metadata = core.ReleaseMetadata.from_version('minor', '3.2.0')
        command = core.sdk_workflow_command(metadata)
        self.assertEqual(
            command[:4], ['gh', 'workflow', 'run', 'publish-packages.yml']
        )
        self.assertIn('tag=sdk-3.2.0', command)
        self.assertIn('packages=all', command)

    def test_release_container_does_not_run_a_second_changelog_generator(self):
        command = core.release_version_bump_command(
            Path('/repo'), 'release-3.2'
        )
        self.assertNotIn('git-cliff', command[-1])
        self.assertNotIn('CHANGELOG.md', command[-1])
        self.assertIn('manifests/kustomize/hack/release.sh', command[-1])


class InlineCommandTest(unittest.TestCase):
    def test_kfp_requirements_command(self):
        command = core.kfp_requirements_command()
        self.assertEqual(command[:2], ['bash', '-c'])
        self.assertIn('pip-compile', command[2])
        self.assertIn('$GIT_ROOT/api/v2alpha1/python/dist', command[2])
        self.assertIn(
            '$GIT_ROOT/backend/api/v2beta1/python_http_client/dist', command[2]
        )
        self.assertIn('--output-file=requirements.txt', command[2])

    def test_kubernetes_requirements_command(self):
        command = core.kubernetes_requirements_command()
        self.assertEqual(command[:2], ['bash', '-c'])
        self.assertIn('pip-compile', command[2])
        self.assertIn('$GIT_ROOT/sdk/python/dist', command[2])
        self.assertIn('$GIT_ROOT/api/v2alpha1/python/dist', command[2])
        self.assertIn(
            '$GIT_ROOT/backend/api/v2beta1/python_http_client/dist', command[2]
        )
        self.assertIn('--output-file=requirements.txt', command[2])

    def test_requirements_commands_preserve_existing_pins(self):
        for command in (
            core.kfp_requirements_command(),
            core.kubernetes_requirements_command(),
        ):
            with self.subTest(command=command), TemporaryDirectory() as tmpdir:
                root = Path(tmpdir)
                subprocess.run(['git', 'init', '-q', str(root)], check=True)
                requirements = root / 'requirements.txt'
                requirements.write_text('existing-package==1.2.3\n')
                compiler = root / 'pip-compile'
                compiler.write_text(
                    '#!/bin/bash\nset -eu\n'
                    'test "$(cat requirements.txt)" = "existing-package==1.2.3"\n'
                    'case " $* " in *" --output-file=requirements.txt "*) ;; *) exit 2;; esac\n'
                    'printf "existing-package==1.2.3\\nnew-package==4.5.6\\n" > requirements.txt\n'
                )
                compiler.chmod(0o755)
                subprocess.run(
                    command,
                    cwd=root,
                    check=True,
                    env={
                        **os.environ,
                        'PATH': f'{root}{os.pathsep}{os.environ["PATH"]}',
                    },
                )
                self.assertEqual(
                    requirements.read_text(),
                    'existing-package==1.2.3\nnew-package==4.5.6\n',
                )

    def test_release_container_mounts_linked_worktree_git_paths_readonly(self):
        with TemporaryDirectory() as tmpdir:
            base = Path(tmpdir).resolve()
            main = base / 'main'
            linked = base / 'linked'
            subprocess.run(['git', 'init', '-q', str(main)], check=True)
            subprocess.run(
                [
                    'git',
                    '-C',
                    str(main),
                    '-c',
                    'user.name=Test',
                    '-c',
                    'user.email=test@example.invalid',
                    'commit',
                    '-q',
                    '--allow-empty',
                    '-m',
                    'initial',
                ],
                check=True,
            )
            subprocess.run(
                [
                    'git',
                    '-C',
                    str(main),
                    'worktree',
                    'add',
                    '--detach',
                    str(linked),
                ],
                check=True,
                capture_output=True,
            )
            command = core.release_version_bump_command(linked, 'release-2.17')
            mounts = [
                command[i + 1]
                for i, arg in enumerate(command)
                if arg == '--mount'
            ]
            self.assertEqual(
                mounts,
                [
                    f'type=bind,source={linked},target=/go/src/github.com/kubeflow/pipelines',
                    f'type=bind,source={main / ".git"},target={main / ".git"},readonly',
                    f'type=bind,source={linked},target={linked},readonly',
                ],
            )
            # Every absolute Git metadata reference resolves inside the added mounts.
            git_dir = Path(
                (linked / '.git').read_text().strip().removeprefix('gitdir: ')
            )
            common = (
                git_dir / (git_dir / 'commondir').read_text().strip()
            ).resolve()
            backlink = Path((git_dir / 'gitdir').read_text().strip())
            self.assertEqual(common, main / '.git')
            self.assertEqual(backlink, linked / '.git')
            self.assertEqual(
                subprocess.check_output(
                    ['git', '-C', str(linked), 'rev-parse', 'HEAD']
                ),
                subprocess.check_output(
                    ['git', '-C', str(main), 'rev-parse', 'HEAD']
                ),
            )
            self.assertNotIn('git config', command[-1])
            self.assertEqual(core._release_git_mounts(main), [])

    def test_release_version_bump_command_runs_container_directly(self):
        command = core.release_version_bump_command(
            Path('/repo'), 'release-3.2'
        )
        self.assertEqual(command[:3], ['docker', 'run', '--rm'])
        self.assertIn('ghcr.io/kubeflow/kfp-release:release-3.2', command)
        self.assertIn('/bin/bash', command)
        self.assertIn('-c', command)
        self.assertNotIn('-lc', command)
        script = command[-1]
        self.assertNotIn('git-cliff', script)
        self.assertNotIn('manifests/gcp_marketplace/hack/release.sh', script)
        self.assertIn(
            '"$REPO_ROOT/backend/api/build_kfp_server_api_python_package.sh"',
            script,
        )
        self.assertIn('REQUIRED_NODE_VERSION=', script)
        self.assertIn('which node >/dev/null', script)
        self.assertIn('version v4[.]', script)
        self.assertIn('python3 -c "import setuptools"', script)
        self.assertIn('go env GOPATH', script)
        self.assertNotIn('check-release-needed-tools.sh', script)
        self.assertNotIn('make release-in-place', command)

    def test_release_version_bump_selects_api_from_release_source(self):
        for declaration, expected in (
            ('package kubeflow.pipelines.backend.api.v2;', 'v2'),
            ('  package  kubeflow.pipelines.backend.api.v2 ;', 'v2'),
            ('package kubeflow.pipelines.backend.api.v2beta1;', 'v2beta1'),
            (
                '// package kubeflow.pipelines.backend.api.v2;\npackage old;',
                'v2beta1',
            ),
            ('package kubeflow.pipelines.backend.api.v20;', 'v2beta1'),
            ('', 'v2beta1'),
            (None, 'v2beta1'),
        ):
            with (
                self.subTest(declaration=declaration),
                TemporaryDirectory() as tmpdir,
            ):
                root = Path(tmpdir)
                api = root / 'backend/api'
                (api / 'v2beta1').mkdir(parents=True)
                if declaration is not None:
                    (api / 'v2').mkdir()
                    (api / 'v2/run.proto').write_text(declaration)
                if declaration is not None:
                    (api / 'v1beta1').mkdir()
                    (api / 'v1beta1/run.proto').write_text('package old;')
                for relative in (
                    'hack/generator.sh',
                    'build_kfp_server_api_python_package.sh',
                ):
                    generator = api / relative
                    generator.parent.mkdir(parents=True, exist_ok=True)
                    generator.write_text(
                        '#!/bin/bash\nprintf "%s\\n" "$API_VERSION"\n'
                    )
                    generator.chmod(0o755)

                for directory, target in (
                    ('api', 'generate'),
                    ('kubernetes_platform', 'generate-go-in-container'),
                ):
                    build_dir = root / directory
                    build_dir.mkdir(exist_ok=True)
                    (build_dir / 'Makefile').write_text(
                        f'{target}:\n\t@echo {directory}-{target}\n'
                    )
                command = core.release_version_bump_command(root, 'release-3.2')
                script = command[-1]
                selection = script[
                    script.index(
                        'if compgen -G "$REPO_ROOT/backend/api/v1beta1/*.proto"'
                    ) :
                ]
                # Execute the real version selection and both generator calls
                # without running unrelated release prerequisites or Docker.
                result = subprocess.run(
                    ['bash', '-ec', selection],
                    timeout=10,
                    env={
                        **os.environ,
                        'REPO_ROOT': str(root),
                        'API_VERSION': 'stale',
                        'MAKEFLAGS': '--no-print-directory',
                    },
                    capture_output=True,
                    text=True,
                    check=True,
                )
                self.assertEqual(
                    result.stdout.splitlines(),
                    (['v1beta1'] if declaration is not None else [])
                    + [
                        expected,
                        expected,
                        'api-generate',
                        'kubernetes_platform-generate-go-in-container',
                    ],
                )

    def test_release_yq_matches_target_checkout(self):
        script = core.release_version_bump_command(
            Path('/repo'), 'release-2.17'
        )[-1]
        prerequisites = script[
            script.index('which yq') : script.index('echo "java')
        ]
        for legacy in (True, False):
            for major in (3, 4):
                with (
                    self.subTest(legacy=legacy, major=major),
                    TemporaryDirectory() as tmp,
                ):
                    root = Path(tmp)
                    release = root / 'manifests/kustomize/hack/release.sh'
                    release.parent.mkdir(parents=True)
                    release.write_text(
                        'yq w -i file key value'
                        if legacy
                        else 'yq eval -i expression file'
                    )
                    binary = root / 'yq'
                    version = (
                        'yq version 3.3.0'
                        if major == 3
                        else 'yq (https://github.com/mikefarah/yq/) version v4.54.1'
                    )
                    binary.write_text('#!/bin/sh\necho "' + version + '"\n')
                    binary.chmod(0o755)
                    result = subprocess.run(
                        ['bash', '-ec', prerequisites],
                        env={
                            **os.environ,
                            'REPO_ROOT': tmp,
                            'PATH': tmp + ':' + os.environ['PATH'],
                        },
                        capture_output=True,
                        text=True,
                        timeout=10,
                    )
                    self.assertEqual(
                        result.returncode == 0,
                        major == (3 if legacy else 4),
                        result.stdout + result.stderr,
                    )

    def test_docs_build_without_checkout_uv_helper(self):
        script = core.kfp_kubernetes_docs_build_command(
            Path('/repo'), 'release-2.17'
        )[-1]
        build = script[script.index('if [ -f pyproject.toml ]') :]
        for bundled in (True, False):
            for workspace in (True, False):
                with (
                    self.subTest(bundled=bundled, workspace=workspace),
                    TemporaryDirectory() as tmp,
                ):
                    root = Path(tmp)
                    bindir = root / 'bin'
                    bindir.mkdir()
                    if workspace:
                        (root / 'pyproject.toml').touch()
                    uv_body = '#!/bin/sh\nprintf "%s\\n" "$@"\n'
                    if bundled:
                        (bindir / 'uv').write_text(uv_body)
                        (bindir / 'uv').chmod(0o755)
                    else:
                        local = root / '.local/bin'
                        local.mkdir(parents=True)
                        (local / 'uv').write_text(uv_body)
                        (local / 'uv').chmod(0o755)
                    python = bindir / 'python3'
                    python.write_text(
                        '#!/bin/sh\nprintf "%s\\n" "$@" > "$HOME/bootstrap"\n'
                    )
                    python.chmod(0o755)
                    # Isolate PATH so the host uv cannot hide the bootstrap path.
                    setup = (
                        script[
                            script.index('UV=') : script.index(
                                'rm -rf kubernetes_platform'
                            )
                        ]
                        if 'UV=' in script
                        else ''
                    )
                    result = subprocess.run(
                        ['/bin/bash', '-ec', setup + build],
                        cwd=root,
                        env={**os.environ, 'HOME': tmp, 'PATH': str(bindir)},
                        capture_output=True,
                        text=True,
                        timeout=10,
                    )
                    self.assertEqual(result.returncode, 0, result.stderr)
                    expected = (
                        ['build']
                        + (['--package', 'kfp-kubernetes'] if workspace else [])
                        + ['--out-dir', 'dist']
                        + ([] if workspace else ['.'])
                    )
                    self.assertEqual(result.stdout.splitlines(), expected)
                    self.assertEqual((root / 'bootstrap').exists(), not bundled)
                    if not bundled:
                        self.assertEqual(
                            (root / 'bootstrap').read_text().splitlines(),
                            [
                                '-m',
                                'pip',
                                'install',
                                '--user',
                                '--break-system-packages',
                                'uv==0.13.5',
                            ],
                        )

    def test_release_version_bump_command_can_use_master_image(self):
        command = core.release_version_bump_command(
            Path('/repo'), 'release-3.2', 'master'
        )

        self.assertIn('ghcr.io/kubeflow/kfp-release:master', command)
        self.assertNotIn('ghcr.io/kubeflow/kfp-release:release-3.2', command)

    def test_kfp_kubernetes_docs_build_command_runs_container_directly(self):
        command = core.kfp_kubernetes_docs_build_command(
            Path('/repo'), 'release-3.2'
        )
        self.assertEqual(command[:3], ['docker', 'run', '--rm'])
        self.assertIn('ghcr.io/kubeflow/kfp-api-generator:release-3.2', command)
        script = command[-1]
        self.assertIn('wget -qO api/v2alpha1/google/rpc/status.proto', script)
        self.assertIn('python3 generate_proto.py', script)
        self.assertIn('if [ -f pyproject.toml ]; then', script)
        self.assertIn(
            '"$UV" build --package kfp-kubernetes --out-dir dist', script
        )
        self.assertIn('"$UV" build --out-dir dist .', script)
        self.assertNotIn('make', command)


if __name__ == '__main__':
    unittest.main()
