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
"""Exercise the dev Makefile without building images or changing a cluster."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]
TOOL_STUB = '''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys
name = Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ['CALL_LOG'], 'a') as log:
    log.write(json.dumps([name] + args) + '\\n')
if name == 'ko':
    print(os.environ['KO_DOCKER_REPO'] + '@sha256:' + 'a' * 64)
elif name == 'go':
    output = Path(args[args.index('-o') + 1])
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text('#!/bin/sh\\nprintf "workflow\\\\n"\\n')
    output.chmod(0o755)
elif name == 'kfp' and args[:2] == ['dsl', 'compile']:
    Path(args[args.index('--output') + 1]).write_text('pipelineInfo: {name: test}')
elif name == 'kubectl' and 'rollout' in args and os.environ.get('FAIL_ROLLOUT'):
    sys.exit(1)
'''


class V2DevWorkflowTest(unittest.TestCase):

    def run_pipeline(self, fail_rollout=False, context='dev-cluster'):
        with tempfile.TemporaryDirectory() as directory:
            cwd = Path(directory)
            bin_dir = cwd / 'bin'
            bin_dir.mkdir()
            for name in ('go', 'ko', 'kfp', 'kubectl', 'argo'):
                tool = bin_dir / name
                tool.write_text(TOOL_STUB)
                tool.chmod(0o755)
            log = cwd / 'calls.jsonl'
            command = [
                'make',
                '-f',
                str(ROOT / 'backend/src/v2/Makefile'),
                'pipeline/hello_world',
                'DEV_IMAGE_PREFIX=registry.test/dev-',
                'DEV_NAMESPACE=dev-ns',
                f'REPO_ROOT={ROOT}',
            ]
            if context is not None:
                command.append(f'DEV_KUBE_CONTEXT={context}')
            result = subprocess.run(
                command,
                cwd=cwd,
                env={
                    **os.environ,
                    'PATH':
                        str(bin_dir) + os.pathsep + os.environ['PATH'],
                    'CALL_LOG':
                        str(log),
                    'FAIL_ROLLOUT':
                        '1' if fail_rollout else '',
                    'TMPDIR':
                        directory,
                },
                text=True,
                capture_output=True,
                timeout=30,
            )
            calls = [json.loads(line) for line in log.read_text().splitlines()]
            return result, calls

    def test_submits_ir_after_configuring_published_runtime_images(self):
        result, calls = self.run_pipeline()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        updates = [c for c in calls if c[0] == 'kubectl' and 'env' in c]
        self.assertEqual(len(updates), 1, calls)
        update = updates[0]
        self.assertIn('dev-cluster', update)
        self.assertIn('dev-ns', update)
        self.assertIn('deployment/ml-pipeline', update)
        for name, image in (('DRIVER', 'driver'), ('LAUNCHER', 'launcher-v2')):
            self.assertIn(
                f'V2_{name}_IMAGE=registry.test/dev-{image}@sha256:' + 'a' * 64,
                update)
        rollout = next(c for c in calls if c[0] == 'kubectl' and 'rollout' in c)
        submit = next(c for c in calls if c[:3] == ['kfp', 'run', 'submit'])
        self.assertLess(calls.index(update), calls.index(rollout))
        self.assertLess(calls.index(rollout), calls.index(submit))
        self.assertTrue(submit[submit.index('-f') + 1].endswith('-spec.yaml'))

    def test_does_not_submit_after_failed_runtime_rollout(self):
        result, calls = self.run_pipeline(fail_rollout=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(c[:3] == ['kfp', 'run', 'submit'] for c in calls))

    def test_requires_explicit_development_context(self):
        for context in (None, '', ' '):
            with self.subTest(context=context):
                result, calls = self.run_pipeline(context=context)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('DEV_KUBE_CONTEXT', result.stderr)
                self.assertFalse(any(c[0] == 'kubectl' for c in calls))


if __name__ == '__main__':
    unittest.main()
