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
"""Exercise dependency queue probes against real Git merge histories."""

import json
import os
import shlex
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

MODULE = Path(__file__).with_name('queue_dependency_merge.js').resolve()


class DependencyMergeTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.repo = self.root / 'upstream'
        self.repo.mkdir()
        self.git('init', '--quiet')
        self.git('config', 'user.name', 'Test')
        self.git('config', 'user.email', 'test@example.org')
        self.git('config', 'uploadpack.allowFilter', 'true')
        self.base = self.commit('one\ntwo\nthree\nfour\nfive\nsix\nseven\n')

        # Use a local transport for the production fetch (including its partial
        # clone reads), keeping the same bare repository and merge operations.
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        wrapper = self.bin / 'git'
        git = shlex.quote(shutil.which('git'))
        upstream = shlex.quote(self.repo.as_uri())
        wrapper.write_text(
            '#!/bin/sh\n'
            'if [ "$5 $6 $7" = "remote add origin" ]; then\n'
            f'  exec {git} remote add origin {upstream}\n'
            'fi\n'
            f'exec {git} -c protocol.file.allow=always "$@"\n'
        )
        wrapper.chmod(0o755)

    def git(self, *args):
        return subprocess.check_output(
            ['git', *args], cwd=self.repo, text=True, stderr=subprocess.PIPE
        ).strip()

    def commit(self, content):
        (self.repo / 'package-lock.json').write_text(content)
        self.git('add', 'package-lock.json')
        self.git('commit', '--quiet', '-m', 'fixture')
        return self.git('rev-parse', 'HEAD')

    def branch(self, start, content):
        self.git('checkout', '--quiet', '--detach', start)
        return self.commit(content)

    def probe(self, entries, head):
        params = json.dumps(
            {
                'context': {'repo': {'owner': 'owner', 'repo': 'repo'}},
                'base': self.base,
                'entries': entries,
                'head': head,
            }
        )
        result = subprocess.run(
            [
                'node',
                '-e',
                'require(process.argv[1]).probeDependencyConflict(JSON.parse(process.argv[2]))'
                '.then(value => console.log(JSON.stringify(value)))'
                '.catch(e => {console.error(e.message); process.exitCode = 1;});',
                str(MODULE),
                params,
            ],
            cwd=self.repo,
            text=True,
            capture_output=True,
            env={
                **os.environ,
                'PATH': f'{self.bin}{os.pathsep}{os.environ["PATH"]}',
                'TMPDIR': str(self.root),
            },
        )
        self.assertEqual([], list(self.root.glob('kfp-queue-merge-*')))
        return result

    def entry(self, queued, candidate=None):
        return {
            'baseCommit': {'oid': self.base},
            'pullRequest': {'headRefOid': queued},
            'headCommit': {'oid': candidate} if candidate else None,
        }

    def test_same_file_clean_and_conflicting_merges(self):
        original = (self.repo / 'package-lock.json').read_text()
        first = self.branch(self.base, original.replace('two', 'queued'))
        conflicting = self.branch(
            self.base, original.replace('two', 'incoming')
        )
        clean = self.branch(self.base, original.replace('seven', 'independent'))
        for candidate in [None, first]:
            with self.subTest(candidate=candidate):
                entry = self.entry(first, candidate)
                result = self.probe([entry], conflicting)
                self.assertEqual(0, result.returncode, result.stderr)
                self.assertEqual(
                    {'paths': ['package-lock.json']}, json.loads(result.stdout)
                )
                result = self.probe([entry], clean)
                self.assertEqual(0, result.returncode, result.stderr)
                self.assertIsNone(json.loads(result.stdout))

    def test_materialized_candidate_bypasses_an_unapplied_entry(self):
        original = (self.repo / 'package-lock.json').read_text()
        skipped = self.branch(self.base, original.replace('seven', 'skipped'))
        built = self.branch(self.base, original.replace('two', 'built'))
        incoming = self.branch(self.base, original.replace('two', 'incoming'))
        entries = [self.entry(skipped), self.entry(built, built)]
        result = self.probe(entries, incoming)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertEqual(
            {'paths': ['package-lock.json']}, json.loads(result.stdout)
        )
        # A split prefix cannot prove a queue conflict.
        entries[-1]['baseCommit']['oid'] = skipped
        result = self.probe(entries, incoming)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIsNone(json.loads(result.stdout))

    def test_unknown_prefix_is_not_a_conflict(self):
        original = (self.repo / 'package-lock.json').read_text()
        first = self.branch(self.base, original.replace('two', 'queued'))
        other = self.branch(self.base, original.replace('two', 'later'))
        for entries in [
            [self.entry(first), self.entry(other)],
            [self.entry(first), self.entry(other, other)],
            [dict(self.entry(first, first), baseCommit={'oid': other})],
            [dict(self.entry(first), baseCommit={'oid': other})],
            [self.entry(first)] * 6,
        ]:
            with self.subTest(entries=entries):
                result = self.probe(entries, other)
                self.assertEqual(0, result.returncode, result.stderr)
                self.assertIsNone(json.loads(result.stdout))

    def test_fetch_error_cleans_up_and_cannot_be_classified_as_conflict(self):
        result = self.probe([], 'f' * 40)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('failed during git fetch', result.stderr)
