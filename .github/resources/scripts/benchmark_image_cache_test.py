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
"""Test benchmark isolation and its cache correctness assertions without
Docker."""

import importlib.util
import json
import os
from pathlib import Path
import shutil
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location(
    'benchmark_image_cache',
    Path(__file__).with_name('benchmark_image_cache.py'))
BENCHMARK = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(BENCHMARK)


class BenchmarkImageCacheTest(unittest.TestCase):

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.repository = self.directory / 'repository'
        backend = self.repository / 'backend'
        (backend / 'src/apiserver').mkdir(parents=True)
        (backend / 'Dockerfile').write_text(
            'FROM golang:pinned AS builder\n'
            'RUN go build -o /bin/apiserver backend/src/apiserver/*.go\n')
        self.source = backend / 'src/apiserver/main.go'
        self.source.write_text('package main\nfunc main() {}\n')
        self.output = self.directory / 'results'
        self.calls = []
        self.fault = None
        self.addCleanup(mock.patch.stopall)
        mock.patch.object(
            BENCHMARK, 'archive_source', side_effect=self.archive).start()
        mock.patch.object(
            BENCHMARK, 'command', side_effect=self.command).start()
        mock.patch.dict(os.environ, {'GITHUB_STEP_SUMMARY': ''}).start()

    def archive(self, repository, context, log, env):
        shutil.copytree(repository, context)

    def command(self, args, log, env, cwd=None):
        self.calls.append(args)
        log.write_text('')
        if args[:3] == ['docker', 'buildx', 'build']:
            scenario = log.name.removesuffix('.build.log')
            if self.fault == 'build-failure':
                raise RuntimeError('Build failed')
            edited = BENCHMARK.MARKER in (
                Path(args[-1]) / 'backend/src/apiserver/main.go').read_text()
            content = b'edited' if edited else b'original'
            if self.fault == 'stale' and scenario == 'source-edit-warm':
                content = b'original'
            destination = Path(args[args.index('--output') +
                                    1].split('dest=')[1])
            destination.mkdir()
            (destination / 'apiserver').write_bytes(content)
            count = {
                'cold': 12,
                'warm': 0,
                'source-edit-warm': 1,
                'source-edit-cold': 12
            }[scenario]
            if self.fault == 'no-go-cache' and scenario == 'source-edit-warm':
                count = 12
            log.write_text(
                '#1 CACHED\n' * (scenario in ('warm', 'source-edit-warm')) +
                '#2 1.234 /usr/local/go/pkg/tool/linux_amd64/compile -o pkg.a\n'
                * count)
        elif args[0] == 'bash' and args[1].endswith('buildkit-go-cache.sh'):
            cache = Path(env['RUNNER_TEMP']) / 'go-build-cache'
            if args[2] == 'save':
                cache.mkdir()
                if self.fault != 'empty-cache':
                    (cache / 'go-build-cache.tar.gz').write_bytes(b'compiled')
            else:
                self.assertEqual((cache / 'go-build-cache.tar.gz').read_bytes(),
                                 b'compiled')
        return 1.5

    def run_benchmark(self):
        return BENCHMARK.benchmark(self.repository, self.output, self.directory)

    def test_four_fresh_builders_and_independent_cache_restores(self):
        summary = self.run_benchmark()
        self.assertEqual(summary['status'], 'passed')
        self.assertEqual([row['scenario'] for row in summary['results']],
                         list(BENCHMARK.SCENARIOS))
        builders = [row['builder'] for row in summary['results']]
        self.assertEqual(len(set(builders)), 4)
        removals = [
            args[-1]
            for args in self.calls
            if args[:4] == ['docker', 'buildx', 'rm', '--force']
        ]
        self.assertEqual(removals, builders)
        builds = [
            args for args in self.calls
            if args[:3] == ['docker', 'buildx', 'build']
        ]
        self.assertIn('--cache-to', builds[0])
        self.assertNotIn('--cache-from', builds[0])
        for args in builds[1:3]:
            self.assertIn('--cache-from', args)
            self.assertNotIn('--cache-to', args)
        self.assertNotIn('--cache-from', builds[3])
        self.assertNotIn('--cache-to', builds[3])
        for args in builds:
            self.assertEqual(args[args.index('--platform') + 1], 'linux/amd64')
            self.assertEqual(args[args.index('--target') + 1],
                             'cache-benchmark-output')
        transfers = [
            args[2]
            for args in self.calls
            if args[0] == 'bash' and args[1].endswith('buildkit-go-cache.sh')
        ]
        self.assertEqual(transfers, ['save', 'restore', 'restore'])
        self.assertNotIn(BENCHMARK.MARKER, self.source.read_text())
        self.assertEqual(list(self.directory.glob('kfp-cache-benchmark-*')), [])
        self.assertIn('GitHub cache upload/download latency is not measured',
                      (self.output / 'summary.md').read_text())
        self.assertEqual(
            json.loads((self.output / 'summary.json').read_text()), summary)

    def test_stale_source_edit_binary_fails(self):
        self.fault = 'stale'
        with self.assertRaisesRegex(RuntimeError,
                                    'warm and cold binaries differ'):
            self.run_benchmark()
        self.assertEqual(
            json.loads((self.output / 'summary.json').read_text())['status'],
            'failed')

    def test_no_compiler_cache_reuse_fails(self):
        self.fault = 'no-go-cache'
        with self.assertRaisesRegex(RuntimeError, 'Go cache reuse'):
            self.run_benchmark()

    def test_empty_cache_export_fails(self):
        self.fault = 'empty-cache'
        with self.assertRaisesRegex(RuntimeError, 'cache export was empty'):
            self.run_benchmark()

    def test_build_failure_removes_only_its_builder_and_preserves_diagnostics(
            self):
        self.fault = 'build-failure'
        with self.assertRaisesRegex(RuntimeError, 'Build failed'):
            self.run_benchmark()
        removals = [
            args for args in self.calls
            if args[:3] == ['docker', 'buildx', 'rm']
        ]
        self.assertEqual(len(removals), 1)
        self.assertTrue(removals[0][-1].startswith('kfp-cache-benchmark-'))
        self.assertTrue((self.output / 'cold.build.log').is_file())
        self.assertIn('Build failed', (self.output / 'summary.md').read_text())
        self.assertEqual(list(self.directory.glob('kfp-cache-benchmark-*')), [])

    def test_compiler_counts_ignore_echoed_dockerfile_instruction(self):
        self.assertEqual(
            BENCHMARK.log_counts(
                '#3 [builder 8/8] RUN go build -x -o /bin/apiserver main.go\n'
                '#3 0.2 /usr/local/go/pkg/tool/linux_amd64/compile -o main.a\n'
                '#4 CACHED\n'), {
                    'cached_steps': 1,
                    'compiler_invocations': 1
                })


if __name__ == '__main__':
    unittest.main()
