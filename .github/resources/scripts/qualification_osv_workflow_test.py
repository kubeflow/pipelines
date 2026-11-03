"""Keep every requested qualification component in trusted read-only CI."""

import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


class QualificationOsvWorkflowTest(unittest.TestCase):
    def setUp(self):
        self.workflow = yaml.safe_load(
            (ROOT / '.github/workflows/release-qualification.yml').read_text()
        )
        self.jobs = self.workflow['jobs']

    def test_scope_defaults_and_read_only_trusted_jobs(self):
        trigger = self.workflow.get('on', self.workflow.get(True))
        self.assertEqual(set(trigger), {'workflow_dispatch'})
        for name in ('include_sdk', 'include_backend'):
            self.assertEqual(
                trigger['workflow_dispatch']['inputs'][name]['type'], 'boolean'
            )
            self.assertIs(
                trigger['workflow_dispatch']['inputs'][name]['default'], True
            )
        self.assertEqual(self.workflow['permissions'], {})
        for job in self.jobs.values():
            self.assertEqual(job['permissions'], {'contents': 'read'})
            checkouts = [
                step['with']
                for step in job['steps']
                if step.get('uses', '').startswith('actions/checkout@')
            ]
            self.assertEqual(
                [(step['path'], step['ref']) for step in checkouts],
                [
                    ('tooling', '${{ github.workflow_sha }}'),
                    ('source', '${{ inputs.source_sha }}'),
                ],
            )
            self.assertTrue(
                all(step['persist-credentials'] is False for step in checkouts)
            )

    def test_selected_source_cannot_access_repository_caches(self):
        self.assertEqual(self.workflow['cache-mode'], 'none')
        for job in self.jobs.values():
            self.assertNotIn('cache-mode', job)

    def test_dispatch_title_and_local_actions_match_existing_contracts(self):
        self.assertEqual(
            self.workflow['run-name'],
            'KFP image qualification ${{ inputs.version }} ${{ inputs.source_sha }} ${{ inputs.request_id }}',
        )
        for job in self.jobs.values():
            for step in job['steps']:
                action = step.get('uses', '')
                if action.startswith('./tooling/'):
                    path = (
                        ROOT / action.removeprefix('./tooling/') / 'action.yml'
                    )
                    self.assertTrue(
                        path.is_file(), f'Missing local action: {action}'
                    )

    def test_source_sdk_and_native_images_use_complete_osv_evidence(self):
        images = self.jobs['images']
        self.assertEqual(images['if'], '${{ inputs.include_backend }}')
        self.assertEqual(images['strategy']['max-parallel'], 6)
        image_commands = '\n'.join(
            step.get('run', '') for step in images['steps']
        )
        self.assertIn('--scanner osv-json', image_commands)
        security = self.jobs['security']
        commands = '\n'.join(step.get('run', '') for step in security['steps'])
        self.assertIn('pip install ./tooling/release', commands)
        self.assertIn('kfpr.ci_security_runner', commands)
        self.assertIn('--include-sdk', commands)
        setups = [
            step
            for step in security['steps']
            if step.get('uses', '').startswith('actions/setup-python@')
        ]
        self.assertIn(
            '${{ needs.inventory.outputs.python-versions }}',
            setups[-1]['with']['python-version'],
        )
        self.assertTrue(
            setups[-1]['with']['python-version'].strip().endswith('3.13')
        )
        for job in (security, images):
            self.assertTrue(
                any(
                    step.get('uses')
                    == './tooling/.github/actions/setup-osv-scanner'
                    for step in job['steps']
                )
            )
            upload = next(
                step
                for step in job['steps']
                if step.get('uses', '').startswith('actions/upload-artifact@')
            )
            self.assertEqual(upload['if'], '${{ always() }}')

    def test_aggregate_requires_security_and_all_requested_images(self):
        job = self.jobs['aggregate']
        self.assertEqual(
            set(job['needs']),
            {'inventory', 'images', 'security', 'third-party'},
        )
        self.assertIn('always()', job['if'])
        steps = job['steps']
        downloads = [
            step
            for step in steps
            if step.get('uses')
            == './tooling/.github/actions/download-artifact-with-retry'
        ]
        self.assertEqual(len(downloads), 3)
        self.assertEqual(downloads[0]['if'], '${{ inputs.include_backend }}')
        self.assertNotIn('if', downloads[1])
        self.assertEqual(downloads[2]['if'], '${{ inputs.include_backend }}')
        self.assertEqual(
            downloads[2]['with']['required-files'], 'third-party.json'
        )
        self.assertEqual(
            downloads[1]['with']['required-files'], 'security.json'
        )
        self.assertIn('${{ github.run_attempt }}', downloads[1]['with']['name'])
        verify = next(
            step
            for step in steps
            if 'kfpr.ci_qualification_runner aggregate' in step.get('run', '')
        )
        upload = next(
            step
            for step in steps
            if step.get('uses', '').startswith('actions/upload-artifact@')
        )
        self.assertLess(steps.index(verify), steps.index(upload))
        self.assertNotIn('if', upload)

    def test_third_party_evidence_is_backend_scoped_and_retained_on_failure(
        self,
    ):
        job = self.jobs['third-party']
        self.assertEqual(job['if'], '${{ inputs.include_backend }}')
        command = '\n'.join(step.get('run', '') for step in job['steps'])
        self.assertIn('kfpr.third_party collect --source source', command)
        self.assertNotIn('upload-sarif', str(job))
        upload = job['steps'][-1]
        self.assertEqual(upload['if'], 'always()')
        self.assertIn('${{ github.run_attempt }}', upload['with']['name'])
