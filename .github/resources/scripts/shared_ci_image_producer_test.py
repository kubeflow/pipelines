"""Contracts for one image producer and overlapping shared-image consumers."""

import json
import subprocess
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOWS = ROOT / '.github/workflows'
CONSUMERS = (
    'api-server-tests.yml',
    'e2e-test-frontend.yml',
    'e2e-test.yml',
    'kfp-kubernetes-native-migration-tests.yaml',
    'kfp-sdk-client-tests.yml',
    'kfp-webhooks.yml',
    'arm64-presubmit.yml',
    'upgrade-test.yml',
)
IMAGES = {
    'apiserver',
    'persistenceagent',
    'scheduledworkflow',
    'launcher',
    'driver',
    'frontend',
    'viewer-crd-controller',
}


def workflow(name):
    return yaml.safe_load((WORKFLOWS / name).read_text())


class SharedImageProducerTest(unittest.TestCase):
    def test_only_shared_producer_calls_image_builder(self):
        callers = []
        for path in WORKFLOWS.glob('*.y*ml'):
            for name, job in yaml.safe_load(path.read_text())['jobs'].items():
                if job.get('uses') == './.github/workflows/image-builds.yml':
                    callers.append((path.name, name, job.get('with', {})))
        self.assertCountEqual(
            callers,
            [
                (
                    'ci-images.yml',
                    'amd64',
                    {
                        'architecture': 'amd64',
                        'shared': True,
                        'cache-write': "${{ github.event_name == 'push' && github.ref == "
                        "format('refs/heads/{0}', github.event.repository.default_branch) }}",
                    },
                ),
                (
                    'ci-images.yml',
                    'arm64',
                    {'architecture': 'arm64', 'shared': True},
                ),
            ],
        )
        for name in CONSUMERS:
            self.assertNotIn('build', workflow(name)['jobs'])
        for job in workflow('upgrade-test.yml')['jobs'].values():
            self.assertEqual(job['if'], '${{ false }}')

    def test_source_identity_precedes_plan_and_revision_cancellation_is_disabled(
        self,
    ):
        definition = workflow('ci-images.yml')
        concurrency = definition['concurrency']
        self.assertIn('github.sha', concurrency['group'])
        self.assertFalse(concurrency['cancel-in-progress'])
        steps = definition['jobs']['source']['steps']
        source = next(
            i
            for i, step in enumerate(steps)
            if step.get('name') == 'Publish source identity before image builds'
        )
        plan = next(
            i for i, step in enumerate(steps) if step.get('id') == 'plan'
        )
        self.assertLess(source, plan)
        self.assertEqual(
            steps[source]['with']['name'],
            'ci-image-source-attempt-${{ github.run_attempt }}',
        )

    def test_producer_filters_cover_every_active_consumer(self):
        producer = workflow('ci-images.yml')[True]['pull_request']['paths']
        for name in CONSUMERS:
            if name == 'upgrade-test.yml':
                continue
            paths = workflow(name)[True]['pull_request']['paths']
            for pattern in paths:
                self.assertIn(pattern, producer, (name, pattern))
        self.assertIn('!**/*.md', producer)
        self.assertIn('!**/OWNERS', producer)

    def test_bundled_provenance_precedes_upload_without_aggregate_barrier(self):
        definition = workflow('image-builds.yml')
        self.assertFalse(
            definition[True]['workflow_call']['inputs']['shared']['default']
        )
        self.assertEqual(
            set(definition['jobs']), {'image-build', 'runtime-base-images'}
        )
        for name in ('image-build', 'runtime-base-images'):
            steps = definition['jobs'][name]['steps']
            uploads = [
                step
                for step in steps
                if step.get('uses', '').startswith('actions/upload-artifact@')
            ]
            self.assertEqual(len(uploads), 1)
            upload = uploads[0]
            self.assertIn('inputs.shared', upload['with']['name'])
            self.assertIn('github.run_attempt', upload['with']['name'])
            record = next(
                step
                for step in steps
                if 'shared_ci_images.py record' in step.get('run', '')
            )
            self.assertLess(steps.index(record), steps.index(upload))
            self.assertIn('inputs.shared', record['if'])
            self.assertIn('.provenance.json', record['run'])
            self.assertNotIn('--artifact-id', record['run'])
            self.assertNotIn('GH_TOKEN', record.get('env', {}))
            self.assertNotIn('continue-on-error', record)
            if name == 'runtime-base-images':
                self.assertEqual(
                    record['env']['ARCHIVE_DIRECTORY'], upload['with']['path']
                )
                self.assertIn('--directory "$ARCHIVE_DIRECTORY"', record['run'])
                self.assertIn(
                    '--output "$ARCHIVE_DIRECTORY/runtime-base-images.provenance.json"',
                    record['run'],
                )
            else:
                self.assertIn(
                    '--output "$ARTIFACTS_PATH/$ARTIFACT_NAME.provenance.json"',
                    record['run'],
                )
                self.assertIn(
                    '${{ env.ARTIFACTS_PATH }}/${{ env.ARTIFACT_NAME }}.provenance.json',
                    upload['with']['path'],
                )
        image_steps = definition['jobs']['image-build']['steps']
        record = next(
            step
            for step in image_steps
            if step.get('name') == 'Record shared image identity'
        )
        arm_record = next(
            step
            for step in image_steps
            if step.get('name') == 'Record ARM image identity'
        )
        self.assertLess(
            image_steps.index(arm_record), image_steps.index(record)
        )

    def test_live_pr_revision_changes_cannot_skip_an_architecture(self):
        script = """
const assert = require('node:assert/strict');
const {pullRequestArchitectures} = require('./.github/resources/scripts/shared_ci_image_plan');
const pr = {number: 23, changed_files: 1,
  head: {sha: 'a'.repeat(40), ref: 'feature', repo: {id: 2, full_name: 'fork/repo'}},
  base: {sha: 'b'.repeat(40), ref: 'master', repo: {id: 1, full_name: 'upstream/repo'}}};
const inventory = {workflows: [
  {path: '.github/workflows/e2e-test.yml', pull_request: {paths: ['test/**']}},
  {path: '.github/workflows/arm64-presubmit.yml', pull_request: {paths: ['frontend/**']}}
]};
for (const phase of ['before', 'after']) {
  for (const side of ['head', 'base']) {
    for (const field of ['sha', 'ref', 'repo.id', 'repo.full_name']) {
      const changed = structuredClone(pr);
      const [key, child] = field.split('.');
      if (child) changed[side][key][child] = child === 'id' ? 3 : 'other/repo';
      else changed[side][key] = field === 'sha' ? 'c'.repeat(40) : 'other';
      let snapshots = 0;
      const request = (endpoint) => endpoint.includes('/files')
        ? [[{filename: 'test/new-revision-only.txt'}]]
        : (++snapshots === (phase === 'before' ? 1 : 2) ? changed : pr);
      const warnings = [];
      assert.deepEqual(pullRequestArchitectures(inventory, {pull_request: pr}, request,
        message => warnings.push(message)), {amd64: true, arm64: true}, `${phase}:${side}:${field}`);
      assert.equal(warnings.length, 1);
    }
  }
}
const calls = [];
assert.deepEqual(pullRequestArchitectures(inventory, {pull_request: pr}, endpoint => {
  calls.push(endpoint);
  return endpoint.includes('/files') ? [[{filename: 'test/stable.txt'}]] : pr;
}), {amd64: true, arm64: false});
assert.deepEqual(calls, ['pulls/23', 'pulls/23/files?per_page=100', 'pulls/23']);
"""
        subprocess.run(['node', '-e', script], cwd=ROOT, check=True)

    def test_architecture_plan_reuses_consumer_filters_and_preserves_pause(
        self,
    ):
        definitions = []
        for name in CONSUMERS:
            definitions.append(
                {
                    'path': f'.github/workflows/{name}',
                    'pull_request': workflow(name)[True]['pull_request'],
                }
            )
        fixtures = [
            (
                'pull_request',
                'frontend/src/example.tsx',
                {'amd64': True, 'arm64': True},
            ),
            (
                'pull_request',
                'test/frontend-integration-test/example.ts',
                {'amd64': True, 'arm64': False},
            ),
            (
                'pull_request',
                '.github/workflows/arm64-presubmit.yml',
                {'amd64': False, 'arm64': True},
            ),
            (
                'pull_request',
                'docs/example.md',
                {'amd64': False, 'arm64': False},
            ),
            (
                'pull_request',
                '.github/workflows/upgrade-test.yml',
                {'amd64': False, 'arm64': False},
            ),
            ('push', 'docs/example.md', {'amd64': True, 'arm64': False}),
            ('workflow_dispatch', '', {'amd64': True, 'arm64': True}),
        ]
        script = """
const {architectures} = require('./.github/resources/scripts/shared_ci_image_plan');
const data = JSON.parse(process.argv[1]);
for (const [event, file, expected] of data.fixtures) {
  const actual = architectures({workflows: data.definitions}, event,
    {pull_request: {base: {ref: 'master'}}}, [file]);
  require('node:assert/strict').deepEqual(actual, expected, event + ':' + file);
}
"""
        subprocess.run(
            [
                'node',
                '-e',
                script,
                json.dumps({'definitions': definitions, 'fixtures': fixtures}),
            ],
            cwd=ROOT,
            check=True,
        )


if __name__ == '__main__':
    unittest.main()
