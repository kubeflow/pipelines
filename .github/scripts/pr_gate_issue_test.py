#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest

from pr_gate_issue import admission_issue


class AdmissionIssueTest(unittest.TestCase):

    def setUp(self):
        self.pr = {
            'baseRefName': 'release-2.18',
            'body': '',
            'closingIssuesReferences': [],
        }

    def test_release_branch_accepts_explicit_same_repository_reference(self):
        for body in ('Fixes #14557 for release-2.18',
                     '- Resolves kubeflow/pipelines#14557',
                     'CLOSES Kubeflow/Pipelines#14557'):
            with self.subTest(body=body):
                self.pr['body'] = body
                self.assertEqual(
                    admission_issue(self.pr, 'kubeflow/pipelines', 'master'),
                    14557)

    def test_release_branch_rejects_mentions_and_other_repositories(self):
        for body in ('See #14557', 'The docs say Fixes #14557',
                     'Fixes other/project#14557', 'Fixes #0'):
            with self.subTest(body=body):
                self.pr['body'] = body
                self.assertIsNone(
                    admission_issue(self.pr, 'kubeflow/pipelines', 'master'))

    def test_default_branch_requires_github_link(self):
        self.pr.update(baseRefName='master', body='Fixes #14557')
        self.assertIsNone(
            admission_issue(self.pr, 'kubeflow/pipelines', 'master'))

    def test_same_repository_link_takes_precedence(self):
        self.pr.update(
            body='Fixes #14557',
            closingIssuesReferences=[{
                'number': 42,
                'repository': {
                    'owner': {
                        'login': 'kubeflow'
                    },
                    'name': 'pipelines',
                },
            }])
        self.assertEqual(
            admission_issue(self.pr, 'kubeflow/pipelines', 'master'), 42)

    def test_other_repository_link_is_ignored(self):
        self.pr['closingIssuesReferences'] = [{
            'number': 42,
            'repository': {
                'owner': {
                    'login': 'other'
                },
                'name': 'project',
            },
        }]
        self.assertIsNone(
            admission_issue(self.pr, 'kubeflow/pipelines', 'master'))


if __name__ == '__main__':
    unittest.main()
