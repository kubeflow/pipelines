"""Pin deployment manifests to the already verified shared-image producer."""

import tempfile
import unittest
from pathlib import Path

import shared_ci_images as shared
import yaml
from shared_ci_images_test import identity

ROOT = Path(__file__).resolve().parents[3]


def read_yaml(path: str) -> dict:
    return yaml.load((ROOT / path).read_text(), Loader=yaml.BaseLoader)


class ArgoManifestWorkflowTest(unittest.TestCase):
    def test_prepare_and_publish_once_before_architecture_jobs(self) -> None:
        workflow = read_yaml('.github/workflows/ci-images.yml')
        source = workflow['jobs']['source']
        self.assertGreaterEqual(int(source['timeout-minutes']), 20)
        steps = source['steps']
        prepare = next(
            i
            for i, s in enumerate(steps)
            if 'argo_manifest_artifacts.py prepare' in s.get('run', '')
        )
        publish = next(
            i
            for i, s in enumerate(steps)
            if s.get('with', {}).get('name')
            == 'argo-manifests-attempt-${{ github.run_attempt }}'
        )
        self.assertLess(prepare, publish)
        self.assertEqual(steps[publish]['with']['if-no-files-found'], 'error')
        self.assertEqual(
            set(steps[publish]['with']['path'].split()),
            {
                'argo-manifests/argo-manifests.tar',
                'argo-manifests/argo-manifests.json',
            },
        )
        for architecture in ('amd64', 'arm64'):
            self.assertEqual(workflow['jobs'][architecture]['needs'], 'source')

    def test_consumer_download_uses_verified_producer_and_attempt(self) -> None:
        action = read_yaml('.github/actions/deploy/action.yml')
        steps = action['runs']['steps']
        shared_step = next(
            s
            for s in steps
            if s.get('uses') == './.github/actions/shared-ci-images'
        )
        self.assertEqual(shared_step['id'], 'shared-images')
        download = next(
            s
            for s in steps
            if s.get('with', {})
            .get('name', '')
            .startswith('argo-manifests-attempt-')
        )
        self.assertEqual(
            download['uses'], './.github/actions/download-artifact-with-retry'
        )
        for key, expected in {
            'name': 'argo-manifests-attempt-${{ steps.shared-images.outputs.run-attempt }}',
            'run-id': '${{ steps.shared-images.outputs.run-id }}',
            'repository': '${{ steps.shared-images.outputs.repository }}',
            'github-token': '${{ github.token }}',
        }.items():
            self.assertEqual(download['with'][key], expected)
        self.assertEqual(
            set(download['with']['required-files'].split()),
            {'argo-manifests.tar', 'argo-manifests.json'},
        )
        verify = next(s for s in steps if s.get('id') == 'argo-manifests')
        self.assertIn('argo_manifest_artifacts.py verify', verify['run'])
        self.assertEqual(
            verify['env']['PRODUCER_STATE'],
            '${{ steps.shared-images.outputs.state }}',
        )
        self.assertIn('--directory "$ARGO_MANIFESTS_DIRECTORY"', verify['run'])
        self.assertIn('--state "$PRODUCER_STATE"', verify['run'])
        deploy = next(s for s in steps if s.get('id') == 'deploy-kfp')
        self.assertEqual(
            deploy['env']['KFP_ARGO_MANIFESTS_DIRECTORY'],
            '${{ steps.argo-manifests.outputs.directory }}',
        )
        self.assertLess(steps.index(shared_step), steps.index(download))
        self.assertLess(steps.index(download), steps.index(verify))
        self.assertLess(steps.index(verify), steps.index(deploy))

    def test_shared_action_exposes_the_pinned_state(self) -> None:
        action = read_yaml('.github/actions/shared-ci-images/action.yml')
        for name in ('repository', 'run-id', 'run-attempt'):
            self.assertEqual(
                action['outputs'][name]['value'],
                '${{ steps.resolve.outputs.' + name + ' }}',
            )
        self.assertEqual(
            action['outputs']['state']['value'], '${{ inputs.state }}'
        )

    def test_emitted_attempt_is_producer_attempt_not_consumer_attempt(
        self,
    ) -> None:
        state = {
            'identity': dict(identity(10), run_attempt=3),
            'consumer': dict(identity(20), run_attempt=9),
            'sets': {'images': []},
        }
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / 'outputs'
            shared.emit_outputs(output, state, 'images')
            self.assertIn('run-attempt=3\n', output.read_text())
            self.assertNotIn('run-attempt=9\n', output.read_text())


if __name__ == '__main__':
    unittest.main()
