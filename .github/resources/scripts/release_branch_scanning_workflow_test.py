"""Keep release image scans isolated, complete, and stably attributed."""

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


class ReleaseBranchScanningWorkflowTest(unittest.TestCase):
    def test_release_push_and_weekly_scan_workflow_exists(self):
        path = ROOT / '.github/workflows/release-branch-image-scanning.yml'
        self.assertTrue(
            path.is_file(), 'release-2.18 has no build-and-scan workflow'
        )

    def workflow(self):
        return yaml.safe_load(
            (
                ROOT / '.github/workflows/release-branch-image-scanning.yml'
            ).read_text()
        )

    def test_triggers_include_all_completed_push_wakeups_weekly_and_manual(
        self,
    ):
        workflow = self.workflow()
        triggers = workflow.get('on', workflow.get(True))
        self.assertEqual(
            triggers['workflow_run'],
            {
                'workflows': ['pre-commit'],
                'branches': ['release-2.18'],
                'types': ['completed'],
            },
        )
        self.assertIn('schedule', triggers)
        self.assertIn('workflow_dispatch', triggers)
        self.assertNotIn('conclusion', str(workflow))
        self.assertIn(
            "github.ref == 'refs/heads/master'",
            workflow['jobs']['inventory']['if'],
        )

    def test_concurrency_keeps_valid_scans_separate_from_untrusted_wakeups(
        self,
    ):
        concurrency = self.workflow()['concurrency']
        self.assertFalse(concurrency['cancel-in-progress'])
        group = concurrency['group']
        for guard in (
            "github.ref == 'refs/heads/master'",
            "event == 'push'",
            "path == '.github/workflows/pre-commit.yml'",
            "head_branch == 'release-2.18'",
            "head_repository.full_name == 'kubeflow/pipelines'",
            "'images' || 'ignored'",
        ):
            self.assertIn(guard, group)

    def test_only_post_aggregate_upload_job_can_write_security_events(self):
        workflow = self.workflow()
        self.assertEqual(workflow['permissions'], {})
        for name, job in workflow['jobs'].items():
            permissions = {'contents': 'read'}
            if name == 'upload':
                permissions['security-events'] = 'write'
            self.assertEqual(job['permissions'], permissions)
        upload = workflow['jobs']['upload']
        self.assertEqual(upload['needs'], ['inventory', 'aggregate'])
        self.assertNotIn('if', upload)
        scripts = [step for step in upload['steps'] if 'run' in step]
        self.assertEqual(len(scripts), 1)
        self.assertEqual(scripts[0]['name'], 'Verify exact release source')
        self.assertNotIn('packages: write', str(workflow))

    def test_source_and_tool_checkouts_are_separate_and_immutable(self):
        workflow = self.workflow()
        for name, job in workflow['jobs'].items():
            checkouts = [
                step['with']
                for step in job['steps']
                if step.get('uses', '').startswith('actions/checkout@')
            ]
            self.assertEqual(len(checkouts), 2)
            self.assertEqual(checkouts[0]['ref'], '${{ github.workflow_sha }}')
            self.assertEqual(checkouts[0]['path'], 'tooling')
            self.assertEqual(checkouts[1]['path'], 'source')
            expected = (
                '${{ steps.source.outputs.source-sha }}'
                if name == 'inventory'
                else '${{ needs.inventory.outputs.source-sha }}'
            )
            self.assertEqual(checkouts[1]['ref'], 'refs/heads/release-2.18')
            source_index = next(
                index
                for index, step in enumerate(job['steps'])
                if step.get('with', {}).get('path') == 'source'
            )
            guard = job['steps'][source_index + 1]
            self.assertEqual(guard['name'], 'Verify exact release source')
            self.assertEqual(guard['env']['EXPECTED_SOURCE_SHA'], expected)
            self.assertIn('^[0-9a-f]{40}$', guard['run'])
            self.assertIn('git -C source rev-parse HEAD', guard['run'])
            self.assertIn('!= "$EXPECTED_SOURCE_SHA"', guard['run'])
            self.assertIn('exit 1', guard['run'])
            self.assertTrue(
                all(item['persist-credentials'] is False for item in checkouts)
            )

    def test_source_guard_rejects_changed_and_invalid_commits(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / 'source'
            subprocess.run(['git', 'init', '--quiet', str(source)], check=True)
            subprocess.run(
                [
                    'git',
                    '-C',
                    str(source),
                    '-c',
                    'user.name=Test',
                    '-c',
                    'user.email=test@example.invalid',
                    'commit',
                    '--quiet',
                    '--allow-empty',
                    '-m',
                    'Fixture',
                ],
                check=True,
            )
            sha = subprocess.check_output(
                ['git', '-C', str(source), 'rev-parse', 'HEAD'],
                text=True,
            ).strip()
            for name, job in self.workflow()['jobs'].items():
                guard = next(
                    step
                    for step in job['steps']
                    if step.get('name') == 'Verify exact release source'
                )
                for expected, code in ((sha, 0), ('a' * 40, 1), ('invalid', 1)):
                    with self.subTest(job=name, expected=expected):
                        result = subprocess.run(
                            ['bash', '-e', '-c', guard['run']],
                            cwd=directory,
                            env={**os.environ, 'EXPECTED_SOURCE_SHA': expected},
                            capture_output=True,
                            text=True,
                        )
                        self.assertEqual(result.returncode, code, result.stderr)

    def test_native_builds_reuse_qualification_and_always_preserve_evidence(
        self,
    ):
        images = self.workflow()['jobs']['images']
        self.assertIn("matrix.platform == 'linux/arm64'", images['runs-on'])
        self.assertFalse(images['strategy']['fail-fast'])
        build = next(
            step
            for step in images['steps']
            if 'ci_images_runner lane' in step.get('run', '')
        )
        self.assertIn('--source source', build['run'])
        evidence = images['steps'][-1]
        self.assertEqual(evidence['if'], '${{ always() }}')
        self.assertIn('${{ github.run_attempt }}', evidence['with']['name'])
        self.assertEqual(evidence['with']['retention-days'], 30)

    def test_complete_aggregate_and_conversion_gate_verified_artifact(self):
        steps = self.workflow()['jobs']['aggregate']['steps']
        aggregate = next(
            index
            for index, step in enumerate(steps)
            if 'ci_images_runner aggregate' in step.get('run', '')
        )
        convert = next(
            index
            for index, step in enumerate(steps)
            if 'branch_scan convert' in step.get('run', '')
        )
        publish = len(steps) - 1
        self.assertLess(aggregate, convert)
        self.assertLess(convert, publish)
        self.assertNotIn('if', steps[publish])
        download = next(
            step
            for step in steps
            if step.get('uses', '').endswith('download-artifact-with-retry')
        )
        self.assertEqual(
            download['with']['required-files'],
            '${{ steps.expected-lanes.outputs.required-files }}',
        )
        self.assertIn('${{ github.run_attempt }}', download['with']['pattern'])

    def test_upload_categories_matrix_and_sha_attribution_are_stable(self):
        upload = self.workflow()['jobs']['upload']
        self.assertEqual(
            upload['strategy']['matrix'],
            '${{ fromJSON(needs.inventory.outputs.matrix) }}',
        )
        config = upload['steps'][-1]['with']
        self.assertEqual(
            config['sha'], '${{ needs.inventory.outputs.source-sha }}'
        )
        self.assertEqual(config['ref'], 'refs/heads/release-2.18')
        self.assertEqual(config['checkout_path'], 'source')
        self.assertIn(
            'release-2.18-image-${{ matrix.name }}', config['category']
        )
        for unstable in (
            'source-sha',
            'run_attempt',
            'version',
            'request-id',
            'runner',
        ):
            self.assertNotIn(unstable, config['category'])

    def test_shared_scanner_installer_preserves_pins_and_checksum_verification(
        self,
    ):
        action = yaml.safe_load(
            (ROOT / '.github/actions/setup-trivy/action.yml').read_text()
        )
        script = action['runs']['steps'][0]['run']
        self.assertIn('trivy_0.76.0_Linux-64bit.tar.gz', script)
        self.assertIn('trivy_0.76.0_Linux-ARM64.tar.gz', script)
        self.assertIn(
            'e24b1c51293c1b15e4ba8d45bbc7891ba053f90278ec2698f3562c993e145fe7',
            script,
        )
        self.assertIn(
            'aab21e677a7b7071fc79f1711480840754c18a1b9d9cefa7ccd06af63a0e8f4c',
            script,
        )
        self.assertIn('sha256sum --check --strict', script)
        qualification = (
            ROOT / '.github/workflows/release-qualification.yml'
        ).read_text()
        self.assertIn(
            'uses: ./tooling/.github/actions/setup-trivy', qualification
        )

    def test_ci_runs_new_helper_and_workflow_tests(self):
        workflow = (ROOT / '.github/workflows/ci-scripts-tests.yml').read_text()
        self.assertIn(
            "'.github/workflows/release-branch-image-scanning.yml'", workflow
        )
        self.assertIn("'release/kfpr/branch_scan*.py'", workflow)
        self.assertIn(
            'python3 -m unittest kfpr.branch_scan_test kfpr.ci_images_runner_test',
            workflow,
        )


if __name__ == '__main__':
    unittest.main()
