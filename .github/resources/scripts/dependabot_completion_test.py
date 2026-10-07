#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import base64
import json
from pathlib import Path
import unittest
from unittest.mock import patch

import dependabot_completion as completion
import yaml

ROOT = Path(__file__).resolve().parents[3]
HEAD = 'a' * 40
BASE = 'b' * 40


def plan():
    return {
        'kind': 'python',
        'version': '',
        'repository': 'kubeflow/pipelines',
        'number': 1,
        'head': HEAD,
        'base': BASE,
        'branch': 'dependabot/uv/example'
    }


def event():
    return {
        'repository': {
            'full_name': 'kubeflow/pipelines'
        },
        'workflow_run': {
            'event': 'pull_request',
            'conclusion': 'success',
            'path': '.github/workflows/dependabot-completion-request.yml',
            'head_repository': {
                'full_name': 'kubeflow/pipelines'
            },
            'head_sha': HEAD,
            'head_branch': 'dependabot/uv/example'
        }
    }


def bundle():
    return {
        'plan': plan(),
        'files': {
            'requirements.txt': base64.b64encode(b'idna==3.20\n').decode()
        }
    }


def pr():
    return {
        'state': 'open',
        'number': 1,
        'user': {
            'login': 'dependabot[bot]'
        },
        'head': {
            'ref': 'dependabot/uv/example',
            'sha': HEAD,
            'repo': {
                'full_name': 'kubeflow/pipelines'
            }
        },
        'base': {
            'ref': 'master',
            'sha': BASE,
            'repo': {
                'full_name': 'kubeflow/pipelines'
            }
        }
    }


class ClassificationTest(unittest.TestCase):

    def test_transitive_lock_only_update_is_exported(self):
        old = {'uv.lock': '[[package]]\nname="idna"\nversion="3.11"\n'}
        new = {'uv.lock': '[[package]]\nname="idna"\nversion="3.20"\n'}
        self.assertEqual(
            completion.classify(old, new), {
                'kind': 'python',
                'version': ''
            })

    def test_direct_update_preserves_package_config(self):
        old = {
            'uv.lock':
                '[[package]]\nname="idna"\nversion="3.11"\n',
            'pyproject.toml':
                '[project]\nname="example"\ndependencies=["idna==3.11"]\n'
        }
        new = {key: value.replace('3.11', '3.20') for key, value in old.items()}
        self.assertEqual(completion.classify(old, new)['kind'], 'python')
        new['pyproject.toml'] += '[build-system]\nrequires=["untrusted-build"]\n'
        with self.assertRaisesRegex(ValueError, 'project configuration'):
            completion.classify(old, new)

    def test_generated_only_upgrade_is_not_silently_reverted(self):
        with self.assertRaisesRegex(ValueError, 'Generated-only'):
            completion.classify({'requirements.txt': 'idna==3.11'},
                                {'requirements.txt': 'idna==3.20'})

    def test_source_change_mixed_with_dependency_update_is_unsupported(self):
        self.assertIsNone(
            completion.classify({
                'uv.lock': '',
                'hack/build.sh': ''
            }, {
                'uv.lock': '',
                'hack/build.sh': 'bad'
            }))

    def test_go_digest_is_preserved_as_explicit_target(self):
        path = 'backend/Dockerfile.driver'
        old = f'FROM golang:1.27.1-alpine@sha256:{"1" * 64} AS builder\nRUN echo unchanged\n'
        new = old.replace('1' * 64, '2' * 64)
        result = completion.classify({path: old}, {path: new})
        self.assertEqual(
            result, {
                'kind': 'go',
                'version': '1.27.1',
                'go_images': [f'1.27.1-alpine@sha256:{"2" * 64}']
            })
        self.assertIsNone(
            completion.classify({path: old},
                                {path: new.replace('unchanged', 'changed')}))

    def test_go_conflicting_target_versions_fail_closed(self):
        old = {
            path: f'FROM golang:1.27.1-alpine@sha256:{"1" * 64} AS builder\n'
            for path in ('backend/Dockerfile.driver',
                         'backend/Dockerfile.launcher')
        }
        new = {
            path: value.replace('1.27.1', f'1.27.{i + 2}')
            for i, (path, value) in enumerate(old.items())
        }
        with self.assertRaisesRegex(ValueError, 'one compiler'):
            completion.classify(old, new)

    def test_argo_module_and_image_proposals_select_same_target(self):
        old_mod = 'module example\ngo 1.27.0\ntoolchain go1.27.1\nrequire (\n\tgithub.com/argoproj/argo-workflows/v4 v4.1.2\n)\n'
        self.assertEqual(
            completion.classify({
                'go.mod': old_mod,
                'go.sum': ''
            }, {
                'go.mod': old_mod.replace('4.1.2', '4.1.4'),
                'go.sum': 'updated'
            }), {
                'kind': 'argo',
                'version': 'v4.1.4'
            })
        path = 'manifests/kustomize/third-party/argo/base/workflow-controller-deployment-patch.yaml'
        old = '    image: quay.io/argoproj/workflow-controller:v4.1.2\n    args: [--executor-image, quay.io/argoproj/argoexec:v4.1.2]\n'
        self.assertEqual(
            completion.classify({path: old},
                                {path: old.replace('4.1.2', '4.1.4')}), {
                                    'kind': 'argo',
                                    'version': 'v4.1.4'
                                })

    def test_gh_aw_uses_canonical_pin_not_generated_runtime(self):
        old = (ROOT / completion.GH_AW_SOURCE).read_text()
        new = old.replace('d46b759176d4402dd99ea454e40841ad3c88c85e # v0.87.2',
                          f'{"c" * 40} # v0.89.17')
        self.assertEqual(
            completion.classify({completion.GH_AW_SOURCE: old},
                                {completion.GH_AW_SOURCE: new}), {
                                    'kind': 'gh-aw',
                                    'version': 'v0.89.17',
                                    'compiler_sha': 'c' * 40
                                })
        self.assertIsNone(
            completion.classify(
                {'.github/workflows/ai-analyzer.lock.yml': old},
                {'.github/workflows/ai-analyzer.lock.yml': new}))

    def test_conflicting_argo_images_on_one_line_cannot_revert_target(self):
        path = 'manifests/kustomize/third-party/argo/base/workflow-controller-deployment-patch.yaml'
        old = 'images: [quay.io/argoproj/workflow-controller:v4.1.2, quay.io/argoproj/argoexec:v4.1.2]'
        new = old.replace('workflow-controller:v4.1.2',
                          'workflow-controller:v4.1.4')
        with self.assertRaisesRegex(ValueError, 'conflicting target'):
            completion.classify({path: old}, {path: new})


class PublicationTest(unittest.TestCase):

    @patch.object(completion, 'git')
    @patch.object(completion, 'api')
    def test_proposal_uses_authoritative_master_not_stale_pr_base(
            self, api, git):
        commit = {
            'sha': HEAD,
            'author': {
                'login': 'dependabot[bot]'
            },
            'committer': {
                'login': 'web-flow'
            },
            'commit': {
                'message': 'chore(deps): bump idna',
                'verification': {
                    'verified': True
                }
            },
        }
        master = 'd' * 40
        api.side_effect = [pr(), [pr()], [commit], {'object': {'sha': master}}]

        def git_result(*args):
            if args[0] == 'fetch':
                return b''
            if args[0] == 'rev-parse':
                return HEAD.encode()
            if args[0] == 'merge-base':
                self.assertEqual(args[1], master)
                return BASE.encode()
            if args[0] == 'diff':
                return b'M\tuv.lock\n'
            if args[0] == 'ls-tree':
                return b'100644 blob aaaa\tuv.lock\n'
            if args[0] == 'show':
                version = '3.20' if args[1].startswith(HEAD) else '3.11'
                return f'[[package]]\nname="idna"\nversion="{version}"\n'.encode(
                )
            raise AssertionError(args)

        git.side_effect = git_result
        _, actual = completion.proposal('kubeflow/pipelines', 1, HEAD)
        self.assertEqual(actual['base'], master)

    @patch.object(completion, 'git')
    @patch.object(completion, 'api')
    def test_human_and_unverified_commits_are_never_processed(self, api, git):
        for author, verified in [('human', True), ('dependabot[bot]', False)]:
            commit = {
                'sha': HEAD,
                'author': {
                    'login': author
                },
                'committer': {
                    'login': 'web-flow'
                },
                'commit': {
                    'message': 'update',
                    'verification': {
                        'verified': verified
                    }
                }
            }
            api.side_effect = [pr(), [pr()], [commit]]
            with self.assertRaisesRegex(ValueError, 'verified Dependabot'):
                completion.proposal('kubeflow/pipelines', 1, HEAD)
            git.assert_not_called()

    @patch.object(completion, 'git')
    @patch.object(completion, 'api')
    def test_disposable_completion_commit_does_not_recurse(self, api, git):
        api.side_effect = [
            pr(), [pr()],
            [{
                'sha': HEAD,
                'commit': {
                    'message': 'complete files [dependabot skip]'
                }
            }]
        ]
        self.assertIsNone(completion.proposal('kubeflow/pipelines', 1, HEAD)[1])
        git.assert_not_called()

    def test_forks_nonbots_closed_and_wrong_heads_are_rejected(self):
        mutations = [('state', 'closed'), ('user', {'login': 'human'})]
        for key, value in mutations:
            item = pr()
            item[key] = value
            with self.assertRaises(ValueError):
                completion.validate_pr(item, 'kubeflow/pipelines', HEAD)
        for section, key, value in (('head', 'repo', {
                'full_name': 'fork/pipelines'
        }), ('head', 'ref', 'master'), ('head', 'sha', BASE), ('base', 'ref',
                                                               'release')):
            item = pr()
            item[section][key] = value
            with self.assertRaises(ValueError):
                completion.validate_pr(item, 'kubeflow/pipelines', HEAD)

    def test_bundle_cannot_write_source_or_escape_destination(self):
        for path in ('../requirements.txt', '.github/workflows/pr-gate.yml',
                     'pyproject.toml', 'uv.lock'):
            item = bundle()
            item['files'] = {path: base64.b64encode(b'malicious').decode()}
            with self.assertRaisesRegex(ValueError, 'output set'):
                completion.validate_bundle(item)

    def test_wrong_event_repository_branch_head_or_workflow_is_rejected(self):
        for key, value in (('path', '.github/workflows/other.yml'),
                           ('head_sha', BASE), ('event', 'push'), ('conclusion',
                                                                   'failure'),
                           ('head_branch',
                            'dependabot/uv/other'), ('head_repository', {
                                'full_name': 'other/repo'
                            })):
            item = event()
            item['workflow_run'][key] = value
            with self.assertRaises(ValueError):
                completion.validate_event_binding(plan(), item)

    @patch.object(completion, 'api')
    @patch.object(completion, 'proposal')
    @patch.object(completion, 'git')
    def test_append_is_atomic_and_signed_and_cannot_overwrite_new_head(
            self, git, proposal, api):
        proposal.return_value = (pr(), plan())
        git.side_effect = [
            b'example[bot]\n', b'1+example[bot]@users.noreply.github.com\n'
        ]
        commit_result = {
            'data': {
                'createCommitOnBranch': {
                    'commit': {
                        'oid': 'c' * 40,
                        'url': 'https://github.com/commit'
                    }
                }
            }
        }
        api.side_effect = [pr(), {'object': {'sha': BASE}}, commit_result]
        completion.publish(bundle(), event())
        mutation = api.call_args.kwargs['body']['variables']['input']
        self.assertEqual(mutation['expectedHeadOid'], HEAD)
        self.assertEqual(mutation['branch']['branchName'], plan()['branch'])
        self.assertIn('[dependabot skip]', mutation['message']['headline'])
        self.assertIn('Signed-off-by: example[bot]',
                      mutation['message']['body'])
        self.assertNotIn('deletions', mutation['fileChanges'])
        api.reset_mock()
        changed = plan()
        changed['base'] = 'd' * 40
        proposal.return_value = (pr(), changed)
        with self.assertRaisesRegex(ValueError, 'base changed'):
            completion.publish(bundle(), event())
        api.assert_not_called()

    @patch.object(completion, 'api')
    @patch.object(completion, 'proposal')
    def test_moved_head_fails_before_any_write(self, proposal, api):
        proposal.side_effect = ValueError('head changed')
        with self.assertRaisesRegex(ValueError, 'head changed'):
            completion.publish(bundle(), event())
        api.assert_not_called()

    def test_workflow_separates_generation_from_scoped_writer(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/dependabot-completion.yml').read_text())
        generation = workflow['jobs']['generate']
        self.assertEqual(generation['permissions'], {
            'contents': 'read',
            'pull-requests': 'read'
        })
        self.assertFalse(
            any('secrets.' in json.dumps(step) for step in generation['steps']))
        publisher = workflow['jobs']['publish']
        self.assertEqual(publisher['needs'], 'generate')
        mint = next(
            step for step in publisher['steps'] if step.get('id') == 'app')
        self.assertEqual(mint['with']['permission-contents'], 'write')
        self.assertEqual(mint['with']['permission-pull-requests'], 'read')
        for job in (generation, publisher):
            checkout = next(
                step for step in job['steps']
                if step.get('uses', '').startswith('actions/checkout@'))
            self.assertEqual(checkout['with']['ref'],
                             '${{ github.workflow_sha }}')
            self.assertFalse(checkout['with']['persist-credentials'])


if __name__ == '__main__':
    unittest.main()
