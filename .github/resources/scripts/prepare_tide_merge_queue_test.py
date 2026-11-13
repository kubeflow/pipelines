#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Verify offline Tide migration preserves every nondevelopment merge route."""

import copy
import itertools
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import yaml
from prepare_tide_merge_queue import prepare_config, prepare_queries

BLOCKERS = ['do-not-merge', 'do-not-merge/hold', 'needs-rebase']
QUERIES = [
    {
        'repos': ['kubeflow/pipelines'],
        'labels': ['lgtm', 'approved'],
        'missingLabels': BLOCKERS,
    },
    {
        'repos': ['kubeflow/pipelines'],
        'author': 'dependabot[bot]',
        'includedBranches': ['master', 'main'],
        'missingLabels': BLOCKERS + ['needs-ok-to-test'],
    },
    {
        'orgs': ['kubeflow'],
        'labels': ['lgtm', 'approved'],
        'missingLabels': BLOCKERS,
    },
    {'repos': ['another/project'], 'labels': ['lgtm']},
]


def eligible(queries, repository, branch, author, labels):
    for query in queries:
        scope = repository in query.get('repos', []) or (
            repository.split('/')[0] in query.get('orgs', [])
            and repository not in query.get('excludedRepos', [])
        )
        if (
            scope
            and branch not in query.get('excludedBranches', [])
            and (
                'includedBranches' not in query
                or branch in query['includedBranches']
            )
            and ('author' not in query or author == query['author'])
            and set(query.get('labels', [])).issubset(labels)
            and not set(query.get('missingLabels', [])).intersection(labels)
        ):
            return True
    return False


class TideQueuePreparationTest(unittest.TestCase):
    def test_cli_rejects_patch_paths_aliasing_input(self):
        source = 'tide:\n  queries:\n  - repos: [kubeflow/pipelines]\n'
        for alias in ['same', 'symlink', 'hardlink']:
            with (
                self.subTest(alias=alias),
                tempfile.TemporaryDirectory() as directory,
            ):
                config = Path(directory) / 'config.yaml'
                config.write_text(source)
                output = (
                    config
                    if alias == 'same'
                    else Path(directory) / 'queue.patch'
                )
                if alias == 'symlink':
                    output.symlink_to(config)
                elif alias == 'hardlink':
                    output.hardlink_to(config)
                result = subprocess.run(
                    [
                        sys.executable,
                        str(
                            Path(__file__).with_name(
                                'prepare_tide_merge_queue.py'
                            )
                        ),
                        str(config),
                        '--patch',
                        str(output),
                    ],
                    capture_output=True,
                    text=True,
                )
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('must not alias', result.stderr)
                self.assertEqual(config.read_text(), source)

    def test_cli_writes_distinct_patch_without_changing_input(self):
        source = 'tide:\n  queries:\n  - repos: [kubeflow/pipelines]\n'
        with tempfile.TemporaryDirectory() as directory:
            config = Path(directory) / 'config.yaml'
            config.write_text(source)
            output = Path(directory) / 'queue.patch'
            result = subprocess.run(
                [
                    sys.executable,
                    str(
                        Path(__file__).with_name('prepare_tide_merge_queue.py')
                    ),
                    str(config),
                    '--patch',
                    str(output),
                ],
                capture_output=True,
                text=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(config.read_text(), source)
            self.assertTrue(
                output.read_text().startswith('--- a/prow/oss/config.yaml\n')
            )
            self.assertIn('+    excludedBranches:', output.read_text())

    def test_all_overlapping_development_routes_removed_and_other_policy_preserved(
        self,
    ):
        before = copy.deepcopy(QUERIES)
        after = prepare_queries(before)
        self.assertEqual(before, QUERIES)
        labels = ['lgtm', 'approved', *BLOCKERS, 'needs-ok-to-test']
        for bits in itertools.product([False, True], repeat=len(labels)):
            selected = {
                label for label, present in zip(labels, bits) if present
            }
            for repo, branch, author in itertools.product(
                ['kubeflow/pipelines', 'kubeflow/trainer', 'another/project'],
                ['master', 'main', 'release-2.18', 'sdk/release-1.8'],
                ['human', 'dependabot[bot]'],
            ):
                with self.subTest(
                    repo=repo, branch=branch, author=author, labels=selected
                ):
                    actual = eligible(after, repo, branch, author, selected)
                    if repo == 'kubeflow/pipelines' and branch in [
                        'master',
                        'main',
                    ]:
                        self.assertFalse(actual)
                    else:
                        self.assertEqual(
                            actual,
                            eligible(before, repo, branch, author, selected),
                        )

    def test_mixed_repository_queries_preserve_unrelated_repositories(self):
        queries = [
            {
                'repos': ['kubeflow/pipelines', 'kubeflow/trainer'],
                'author': 'dependabot[bot]',
                'includedBranches': ['main', 'release-2.18'],
            }
        ]
        after = prepare_queries(queries)
        self.assertFalse(
            eligible(
                after, 'kubeflow/pipelines', 'main', 'dependabot[bot]', set()
            )
        )
        self.assertTrue(
            eligible(
                after, 'kubeflow/trainer', 'main', 'dependabot[bot]', set()
            )
        )
        self.assertTrue(
            eligible(
                after,
                'kubeflow/pipelines',
                'release-2.18',
                'dependabot[bot]',
                set(),
            )
        )

    def test_existing_org_exclusion_is_preserved(self):
        queries = [
            {'orgs': ['kubeflow'], 'excludedRepos': ['kubeflow/pipelines']}
        ]
        self.assertEqual(prepare_queries(queries), queries)

    def test_patch_changes_only_queries_and_is_idempotent(self):
        source = (
            '# retain header\ntide:\n  queries:\n'
            + ''.join(
                '  ' + line + '\n'
                for line in yaml.safe_dump(QUERIES).splitlines()
            )
            + '  # retain context comment\n  context_options: {required-contexts: [ci-passed]}\nplugins: {approve: true}\n'
        )
        output = prepare_config(source)
        original = yaml.safe_load(source)
        original['tide']['queries'] = prepare_queries(QUERIES)
        self.assertEqual(yaml.safe_load(output), original)
        self.assertIn('# retain header', output)
        self.assertIn('# retain context comment', output)
        self.assertEqual(prepare_config(output), output)

    def test_malformed_or_ambiguous_config_is_rejected(self):
        for source in [
            'tide: {}',
            'tide: {queries: []}',
            'tide: {queries: [{}]}',
            'tide: {queries: [{repos: kubeflow/pipelines}]}',
            'tide: {queries: [{repos: [kubeflow/pipelines], repos: [other/repo]}]}',
        ]:
            with (
                self.subTest(source=source),
                self.assertRaises((ValueError, yaml.YAMLError)),
            ):
                prepare_config(source)


if __name__ == '__main__':
    unittest.main()
