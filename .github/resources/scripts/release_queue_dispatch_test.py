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

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOWS = ROOT / '.github/workflows'


def definition(path):
    return yaml.safe_load(path.read_text())


class ReleaseQueueDispatchTest(unittest.TestCase):

    def test_every_queue_lane_accepts_dispatch_and_requires_identity_check(
            self):
        lanes = []
        for path in WORKFLOWS.glob('*.y*ml'):
            data = definition(path)
            trigger = data.get('on', data.get(True, {}))
            if not isinstance(trigger, dict) or 'merge_group' not in trigger:
                continue
            lanes.append(path)
            with self.subTest(workflow=path.name):
                self.assertEqual(trigger['merge_group']['branches'],
                                 ['release-2.18'])
                inputs = trigger['workflow_dispatch']['inputs']
                for key in [
                        'queue_head_sha', 'queue_base_ref', 'queue_base_sha'
                ]:
                    self.assertEqual(inputs[key]['type'], 'string')
                guard = data['jobs']['queue-dispatch-guard']
                self.assertEqual(
                    guard['uses'],
                    './.github/workflows/queue-dispatch-guard.yml')
                self.assertIn(
                    "startsWith(github.ref, 'refs/heads/gh-readonly-queue/')",
                    guard['if'])
                self.assertEqual(guard['permissions'], {'contents': 'read'})
        self.assertEqual(len(lanes), 37)

    def guard(self,
              branch='release-2.18',
              head='a' * 40,
              base='b' * 40,
              moved=False):
        data = definition(WORKFLOWS / 'queue-dispatch-guard.yml')
        script = data['jobs']['validate']['steps'][0]['with']['script']
        context = {
            'eventName': 'workflow_dispatch',
            'sha': 'a' * 40,
            'ref': f'refs/heads/gh-readonly-queue/{branch}/pr-7-candidate',
            'repo': {
                'owner': 'kubeflow',
                'repo': 'pipelines'
            }
        }
        source = '''
const context = JSON.parse(process.argv[1]);
const processEnv = JSON.parse(process.argv[2]); Object.assign(process.env, processEnv);
const github = {rest: {git: {getRef: async ({ref}) => ({data: {
 ref: `refs/${ref}`, object: {type: 'commit', sha: ref.includes('gh-readonly-queue/') ?
 context.sha : process.argv[3]}}})}}};
(async () => { try { await (async () => { SCRIPT })(); console.log('ok'); }
catch (error) { console.log(error.message); } })();
'''.replace('SCRIPT', script)
        result = subprocess.run([
            'node', '-e', source,
            json.dumps(context),
            json.dumps({
                'QUEUE_HEAD_SHA': head,
                'QUEUE_BASE_REF': branch,
                'QUEUE_BASE_SHA': base
            }), 'c' * 40 if moved else 'b' * 40
        ],
                                capture_output=True,
                                text=True,
                                check=True)
        return result.stdout.strip()

    def test_dispatch_rejects_wrong_branch_head_missing_base_and_moved_base(
            self):
        self.assertEqual(self.guard(), 'ok')
        for options in [{
                'branch': 'release-2.19'
        }, {
                'branch': 'main'
        }, {
                'head': 'c' * 40
        }, {
                'base': ''
        }, {
                'moved': True
        }]:
            with self.subTest(options=options):
                self.assertNotEqual(self.guard(**options), 'ok')

    def test_compiler_queue_dispatch_keeps_automatic_test_label(self):
        data = definition(WORKFLOWS / 'compiler-tests.yml')
        steps = data['jobs']['compiler-tests']['steps']
        script = next(
            step['run'] for step in steps if step.get('id') == 'configure')
        script = script.replace('${{ env.TESTS_LABEL }}',
                                data['env']['TESTS_LABEL'])
        script = script.replace('${{ github.event_name }}', 'workflow_dispatch')
        for ref, expected in [
            ('refs/heads/release-2.18', 'ApiServerTests'),
            ('refs/heads/gh-readonly-queue/release-2.18/pr-7-candidate',
             'WorkflowCompiler')
        ]:
            with self.subTest(
                    ref=ref), tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / 'output'
                subprocess.run(
                    ['bash', '-eu', '-c', script],
                    check=True,
                    env={
                        **os.environ, 'GITHUB_REF': ref,
                        'GITHUB_OUTPUT': str(output),
                        'DISPATCH_TEST_LABEL': 'ApiServerTests'
                    })
                self.assertEqual(output.read_text().strip(),
                                 f'TEST_LABEL={expected}')

    def test_dispatched_queue_uses_complete_lanes_and_exact_base(self):
        for name in ['api-server-tests.yml', 'e2e-test.yml']:
            text = (WORKFLOWS / name).read_text()
            self.assertNotIn(
                'if [ "${{ github.event_name }}" == "workflow_dispatch" ]; then',
                text)
            self.assertIn('"${GITHUB_REF:-}" != refs/heads/gh-readonly-queue/*',
                          text)
        for name in [
                'pre-commit.yml', 'frontend.yml',
                'third-party-image-upgrades.yml'
        ]:
            self.assertIn('inputs.queue_base_sha',
                          (WORKFLOWS / name).read_text())
        image = (WORKFLOWS / 'image-builds.yml').read_text()
        self.assertIn(
            "REQUIRE_SOURCE_SHA: ${{ github.event_name == 'merge_group' || startsWith(github.ref, 'refs/heads/gh-readonly-queue/') }}",
            image)
        self.assertNotIn('-f event=merge_group', image)


if __name__ == '__main__':
    unittest.main()
