# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercises the publisher with real Git patches and a stub GitHub boundary."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
from urllib.parse import unquote

import release_cve_publish as publisher


class PublisherTest(unittest.TestCase):

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.source = self.root / 'source'
        self.source.mkdir()
        self.bundle = self.root / 'bundle'
        self.bundle.mkdir()
        self.verification = self.root / 'verification'
        self.verification.mkdir()
        self.git('init', '-b', 'release-3.0')
        self.git('config', 'user.name', 'Fixture Author')
        self.git('config', 'user.email', 'fixture@example.test')
        self.git('config', 'commit.gpgsign', 'false')
        self.write('go.mod', 'module example.test/project\n\ngo 1.27.1\n')
        self.write(
            'backend/Dockerfile.driver',
            'FROM golang:1.27.1-alpine@sha256:' + 'a' * 64 + ' AS builder\n')
        self.write('frontend/server/package-lock.json',
                   '{"lockfileVersion":3}\n')
        self.write('.github/workflows/example.yml', 'name: original\n')
        self.git('add', '.')
        self.git('commit', '-m', 'Fixture baseline')
        self.sha = self.git('rev-parse', 'HEAD')
        self.plan = dict(
            schema_version=1,
            source_sha=self.sha,
            source_branch='release-3.0',
            go_version='1.27.2',
            npm_vulns=[],
            verify_images=[
                dict(image=image, dockerfile=dockerfile, context='.')
                for image, (_, dockerfile) in publisher.GO_IMAGES.items()
            ])
        (self.bundle / 'remediation.md').write_text('Fix CVE-2026-1000.\n')
        self.make_patch('go.mod', 'module example.test/project\n\ngo 1.27.2\n')
        self.ref_sha = self.sha
        self.branch_sha = None
        self.prs = []
        self.calls = []
        self.network = []
        self.body = ''
        self.env_patch = patch.dict(
            os.environ, {'GITHUB_STEP_SUMMARY': str(self.root / 'summary')})
        self.env_patch.start()
        self.addCleanup(self.env_patch.stop)

    def git(self, *arguments):
        return subprocess.check_output([
            'git', '-c', 'core.hooksPath=/dev/null', '-c',
            'core.fsmonitor=false', *arguments
        ],
                                       cwd=self.source,
                                       text=True,
                                       stderr=subprocess.DEVNULL).strip()

    def write(self, path, contents):
        target = self.source / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(contents)

    def make_patch(self, path, contents):
        self.write(path, contents)
        # New files are intentionally included for the rejection tests.
        self.git('add', '--', path)
        result = subprocess.check_output(
            ['git', 'diff', '--cached', '--binary'], cwd=self.source)
        (self.bundle / 'remediation.patch').write_bytes(result)
        self.git('reset', '--hard', 'HEAD')
        self.certify()

    def certify(self):
        (self.bundle / 'plan.json').write_text(json.dumps(self.plan))
        digest = hashlib.sha256(
            (self.bundle / 'remediation.patch').read_bytes()).hexdigest()
        for architecture in publisher.ARCHITECTURES:
            certificate = dict(
                source_sha=self.sha,
                patch_sha256=digest,
                architecture=architecture,
                images=[item['image'] for item in self.plan['verify_images']],
                verified=True)
            (self.verification /
             f'verification-{architecture}.json').write_text(
                 json.dumps(certificate))

    def gh(self, *arguments):
        self.calls.append(arguments)
        if arguments[0] == 'api':
            branch = unquote(arguments[1].split('/heads/', 1)[1])
            sha = self.ref_sha if branch == 'release-3.0' else self.branch_sha
            return json.dumps(
                [] if sha is None else
                [dict(ref=f'refs/heads/{branch}', object=dict(sha=sha))])
        if arguments[:2] == ('pr', 'list'):
            return json.dumps(self.prs)
        if arguments[:2] == ('pr', 'create'):
            self.body = Path(arguments[arguments.index('--body-file') +
                                       1]).read_text()
            return 'https://github.com/example/project/pull/10'
        self.fail(f'Unexpected GitHub request: {arguments}')

    def github_git(self, source, repository, operation, *arguments):
        self.network.append((operation, *arguments))
        if operation == 'fetch':
            # Git's normal FETCH_HEAD format, without any network access.
            git_dir = Path(self.git('rev-parse', '--absolute-git-dir'))
            (git_dir /
             'FETCH_HEAD').write_text(f'{self.branch_sha}\t\tfixture\n')
        return ''

    def publish(self):
        with patch.object(
                publisher, '_gh', side_effect=self.gh), patch.object(
                    publisher, '_github_git', side_effect=self.github_git):
            return publisher.publish(self.source, self.bundle,
                                     self.verification, 'example/project',
                                     'release-3.0')

    def test_publishes_signed_bot_commit_and_reviewable_pr(self):
        url = self.publish()
        self.assertEqual(url, 'https://github.com/example/project/pull/10')
        message = self.git('show', '-s', '--format=%an%n%ae%n%cn%n%ce%n%B')
        self.assertEqual(message.splitlines()[:4],
                         [publisher.BOT_NAME, publisher.BOT_EMAIL] * 2)
        self.assertIn(
            f'Signed-off-by: {publisher.BOT_NAME} <{publisher.BOT_EMAIL}>',
            message)
        self.assertEqual(len(self.network), 1)
        self.assertEqual(self.network[0][0], 'push')
        self.assertNotIn('--force', self.network[0])
        self.assertIn('Linux AMD64 and ARM64', self.body)
        self.assertIn('Approve workflows to run', self.body)
        self.assertIn(url, (self.root / 'summary').read_text())
        self.assertEqual(self.git('diff', 'HEAD^', '--name-only'), 'go.mod')

    def existing_bot_pr(self):
        self.publish()
        self.branch_sha = self.git('rev-parse', 'HEAD')
        self.git('reset', '--hard', self.sha)
        self.network.clear()
        self.calls.clear()
        branch = 'codex/cve-fix-' + hashlib.sha256(
            f'release-3.0:{self.sha}'.encode()).hexdigest()[:16]
        self.prs = [
            dict(
                url='https://github.com/example/project/pull/8',
                state='OPEN',
                author=dict(login='app/github-actions'),
                baseRefName='release-3.0',
                headRefName=branch,
                isCrossRepository=False)
        ]

    def test_reuses_existing_bot_pr_only_after_verifying_its_head(self):
        self.existing_bot_pr()
        self.assertTrue(self.publish().endswith('/8'))
        self.assertEqual(self.git('rev-parse', 'HEAD'), self.sha)
        self.assertEqual(len(self.network), 1)
        self.assertEqual(self.network[0][0], 'fetch')
        self.assertFalse(
            any(call[:2] == ('pr', 'create') for call in self.calls))

    def test_refuses_existing_bot_pr_with_newer_patch_for_same_source(self):
        self.existing_bot_pr()
        self.plan['go_version'] = '1.27.3'
        self.make_patch('go.mod', 'module example.test/project\n\ngo 1.27.3\n')
        with self.assertRaisesRegex(publisher.PublishError,
                                    'differs from the verified bot patch'):
            self.publish()
        self.assertEqual(len(self.network), 1)
        self.assertEqual(self.network[0][0], 'fetch')

    def test_refuses_existing_bot_pr_with_human_edits(self):
        self.existing_bot_pr()
        self.git('reset', '--hard', self.branch_sha)
        self.write('go.mod', 'module example.test/project\n\ngo 1.27.4\n')
        self.git('add', 'go.mod')
        self.git('commit', '-m', 'Subsequent human edit')
        self.branch_sha = self.git('rev-parse', 'HEAD')
        self.git('reset', '--hard', self.sha)
        with self.assertRaisesRegex(publisher.PublishError,
                                    'differs from the verified bot patch'):
            self.publish()
        self.assertEqual(len(self.network), 1)
        self.assertEqual(self.network[0][0], 'fetch')

    def test_refuses_existing_bot_pr_without_remote_branch(self):
        self.existing_bot_pr()
        self.branch_sha = None
        with self.assertRaisesRegex(publisher.PublishError,
                                    'no matching branch'):
            self.publish()
        self.assertFalse(self.network)

    def test_refuses_same_named_fork_pr(self):
        self.existing_bot_pr()
        self.prs[0]['isCrossRepository'] = True
        with self.assertRaisesRegex(publisher.PublishError, 'unexpected PR'):
            self.publish()
        self.assertFalse(self.network)

    def test_binds_target_branch_to_workflow_input_before_network(self):
        self.plan['source_branch'] = 'another-branch-at-the-same-sha'
        self.certify()
        with self.assertRaisesRegex(publisher.PublishError,
                                    'requested release branch'):
            self.publish()
        self.assertFalse(self.calls)
        self.assertFalse(self.network)
        self.assertEqual(self.git('status', '--porcelain'), '')

    def test_rejects_closed_or_human_pr(self):
        for state, login in [('CLOSED', 'app/github-actions'),
                             ('OPEN', 'human')]:
            with self.subTest(state=state, login=login):
                self.prs = [dict(state=state, author=dict(login=login))]
                with self.assertRaisesRegex(publisher.PublishError,
                                            'closed or unexpected'):
                    self.publish()
        self.assertFalse(self.network)

    def test_refuses_moved_source_before_patch(self):
        self.ref_sha = 'f' * 40
        with self.assertRaisesRegex(publisher.PublishError,
                                    'source branch moved'):
            self.publish()
        self.assertEqual(self.git('status', '--porcelain'), '')
        self.assertFalse(self.network)

    def test_rechecks_source_before_push(self):
        with patch.object(
                publisher,
                '_require_current_source',
                side_effect=[None, publisher.PublishError('branch moved')]):
            with self.assertRaisesRegex(publisher.PublishError, 'branch moved'):
                self.publish()
        self.assertFalse(self.network)

    def test_refuses_patch_not_covered_by_certificates(self):
        with (self.bundle / 'remediation.patch').open('a') as output:
            output.write('\n')
        with self.assertRaisesRegex(publisher.PublishError,
                                    'verification certificate'):
            self.publish()
        self.assertFalse(self.calls)

    def test_requires_both_architectures_and_exact_image_inventory(self):
        for changes in [
                dict(images=[]),
                dict(verified=False),
                dict(source_sha='f' * 40),
                dict(architecture='amd64'),
                dict(images=['kfp-driver', 'kfp-driver'])
        ]:
            with self.subTest(changes=changes):
                self.certify()
                file = self.verification / 'verification-arm64.json'
                data = json.loads(file.read_text())
                data.update(changes)
                file.write_text(json.dumps(data))
                with self.assertRaisesRegex(publisher.PublishError,
                                            'verification certificate'):
                    self.publish()
        (self.verification / 'verification-arm64.json').unlink()
        with self.assertRaisesRegex(publisher.PublishError,
                                    'Required regular file'):
            self.publish()

    def test_rejects_workflow_edits_and_new_files(self):
        for path, contents in [('.github/workflows/example.yml',
                                'name: modified\n'),
                               ('new/go.mod', 'module new\n')]:
            with self.subTest(path=path):
                self.make_patch(path, contents)
                with self.assertRaises(publisher.PublishError):
                    self.publish()
        self.assertFalse(self.calls)

    def test_rejects_mode_changes(self):
        (self.source / 'go.mod').chmod(0o755)
        self.git('add', 'go.mod')
        data = subprocess.check_output(['git', 'diff', '--cached'],
                                       cwd=self.source)
        (self.bundle / 'remediation.patch').write_bytes(data)
        self.git('reset', '--hard', 'HEAD')
        self.certify()
        with self.assertRaisesRegex(publisher.PublishError, 'file modes'):
            self.publish()

    def test_rejects_symlinked_source_path(self):
        original = self.source / 'go.mod'
        original.unlink()
        target = self.root / 'external'
        target.write_text('module example.test/project\n\ngo 1.27.1\n')
        original.symlink_to(target)
        with self.assertRaises(publisher.PublishError):
            self.publish()
        self.assertEqual(target.read_text(),
                         'module example.test/project\n\ngo 1.27.1\n')

    def test_rejects_traversal_patch(self):
        patch_file = self.bundle / 'remediation.patch'
        patch_file.write_text(patch_file.read_text().replace(
            'a/go.mod', 'a/../go.mod').replace('b/go.mod', 'b/../go.mod'))
        self.certify()
        with self.assertRaises(publisher.PublishError):
            self.publish()
        self.assertFalse(self.calls)

    def test_npm_lockfile_requires_npm_plan(self):
        self.make_patch('frontend/server/package-lock.json',
                        '{"lockfileVersion":3,"packages":{}}\n')
        with self.assertRaisesRegex(publisher.PublishError, 'unsupported file'):
            self.publish()
        self.plan['npm_vulns'] = ['CVE-2026-1000']
        self.plan['verify_images'] = [
            dict(
                image='kfp-frontend',
                dockerfile='frontend/Dockerfile',
                context='.')
        ]
        self.plan['go_version'] = ''
        self.certify()
        self.publish()
        self.assertEqual(
            self.git('diff', 'HEAD^', '--name-only'),
            'frontend/server/package-lock.json')

    def test_recovers_pushed_identical_bot_branch_without_force_push(self):
        self.publish()
        self.branch_sha = self.git('rev-parse', 'HEAD')
        self.git('reset', '--hard', self.sha)
        self.network.clear()
        self.publish()
        self.assertEqual(self.network[0][0], 'fetch')
        self.assertEqual(len(self.network), 1)

    def test_does_not_reuse_human_branch_even_with_identical_patch(self):
        self.git('apply', '--index', str(self.bundle / 'remediation.patch'))
        self.git('commit', '-m', 'Human edit')
        self.branch_sha = self.git('rev-parse', 'HEAD')
        self.git('reset', '--hard', self.sha)
        with self.assertRaisesRegex(publisher.PublishError,
                                    'differs from the verified bot patch'):
            self.publish()
        self.assertEqual(len(self.network), 1)
        self.assertEqual(self.network[0][0], 'fetch')

    def test_rejects_partial_or_untrusted_verification_inventory(self):
        for images in [[
                dict(
                    image='kfp-driver',
                    dockerfile='backend/Dockerfile.driver',
                    context='.')
        ], [dict(image='evil', dockerfile='malicious/Dockerfile',
                 context='.')]]:
            with self.subTest(images=images):
                self.plan['verify_images'] = images
                self.certify()
                with self.assertRaisesRegex(publisher.PublishError,
                                            'complete trusted inventory'):
                    self.publish()

    def test_accepts_go_builder_version_and_digest_update(self):
        self.make_patch(
            'backend/Dockerfile.driver',
            'FROM golang:1.27.2-alpine@sha256:' + 'b' * 64 + ' AS builder\n')
        self.publish()
        self.assertEqual(
            self.git('diff', 'HEAD^', '--name-only'),
            'backend/Dockerfile.driver')

    def test_accepts_toolchain_update_preserving_module_go_floor(self):
        self.write(
            'go.mod',
            'module example.test/project\n\ngo 1.27.0\n\ntoolchain go1.27.1\n')
        self.git('add', 'go.mod')
        self.git('commit', '-m', 'Fixture toolchain')
        self.sha = self.git('rev-parse', 'HEAD')
        self.ref_sha = self.sha
        self.plan['source_sha'] = self.sha
        self.make_patch(
            'go.mod',
            'module example.test/project\n\ngo 1.27.0\n\ntoolchain go1.27.2\n')
        self.publish()
        self.assertIn('toolchain go1.27.2',
                      (self.source / 'go.mod').read_text())

    def test_rejects_go_dependency_edits_disguised_as_compiler_update(self):
        self.make_patch(
            'go.mod',
            'module example.test/project\n\ngo 1.27.2\n\nrequire evil.test/module v1.0.0\n'
        )
        with self.assertRaisesRegex(publisher.PublishError,
                                    'compiler directives'):
            self.publish()
        self.assertFalse(self.network)

    def test_rejects_docker_instructions_disguised_as_builder_update(self):
        self.make_patch('backend/Dockerfile.driver',
                        'FROM golang:1.27.2\nRUN echo unexpected-command\n')
        with self.assertRaisesRegex(publisher.PublishError,
                                    'existing builder pin'):
            self.publish()
        self.assertFalse(self.network)

    def test_disables_source_git_hooks(self):
        git_dir = Path(self.git('rev-parse', '--absolute-git-dir'))
        hook = git_dir / 'hooks' / 'pre-commit'
        hook.write_text('#!/bin/sh\nexit 42\n')
        hook.chmod(0o755)
        self.publish()


if __name__ == '__main__':
    unittest.main()
