# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercise image publication boundaries without contacting a registry."""

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml
from artifact_workflow_test_support import evaluate, render

ROOT = Path(__file__).resolve().parents[3]
WORKFLOWS = ROOT / '.github/workflows'
DIGEST = 'sha256:' + 'a' * 64


def workflow(name):
    return yaml.safe_load((WORKFLOWS / name).read_text())


class ImagePublicationPermissionsTest(unittest.TestCase):
    def setUp(self):
        self.jobs = workflow('build-and-push.yml')['jobs']

    def build_job(self, push):
        jobs = [
            job
            for job in self.jobs.values()
            if evaluate(job['if'], {'inputs': {'push': push}})
        ]
        self.assertEqual(len(jobs), 1)
        return jobs[0]

    def test_dry_run_cannot_write_packages_attestations_or_request_oidc(self):
        job = self.build_job(False)
        self.assertEqual(
            job['permissions'],
            {
                'contents': 'read',
                'packages': 'read',
            },
        )
        context = {
            'inputs': {'push': False},
            'steps': {'check_tag': {'outcome': 'success'}},
        }
        for step in job['steps']:
            if (
                step.get('uses', '').startswith(
                    (
                        'actions/upload-artifact@',
                        'actions/attest-build-provenance@',
                    )
                )
                or step['name'] == 'Export digest'
            ):
                with self.subTest(step=step['name']):
                    self.assertFalse(evaluate(step['if'], context))

    def test_publication_retains_digest_record_and_provenance(self):
        job = self.build_job(True)
        self.assertEqual(
            job['permissions'],
            {
                'contents': 'read',
                'packages': 'write',
                'attestations': 'write',
                'id-token': 'write',
            },
        )
        steps = {step['name']: step for step in job['steps']}
        context = {
            'inputs': {'push': True},
            'steps': {'check_tag': {'outcome': 'success'}},
        }
        for name in (
            'Export digest',
            'Upload digest',
            'Generate artifact attestation',
        ):
            self.assertTrue(evaluate(steps[name]['if'], context))
        exported = steps['Export digest']['env']
        self.assertEqual(exported['DIGEST'], '${{ steps.push.outputs.digest }}')
        self.assertEqual(
            exported['SOURCE_SHA'], '${{ steps.get_commit.outputs.sha }}'
        )
        self.assertEqual(exported['PLATFORM'], '${{ inputs.platforms }}')
        self.assertEqual(
            steps['Generate artifact attestation']['with']['subject-digest'],
            exported['DIGEST'],
        )
        self.assertEqual(
            steps['Upload digest']['with']['name'],
            'digests-${{ inputs.app_to_build }}-${{ inputs.arch_tag }}-${{ github.run_attempt }}',
        )

    def test_reusable_callers_explicitly_allow_only_the_callee_contract(self):
        for name in ('image-builds-master.yml', 'image-builds-release.yml'):
            for job in workflow(name)['jobs'].values():
                if 'uses' not in job:
                    continue
                with self.subTest(workflow=name, callee=job['uses']):
                    self.assertNotIn('secrets', job)
                    callee = workflow(Path(job['uses']).name)
                    required = {}
                    rank = {'read': 1, 'write': 2}
                    for called_job in callee['jobs'].values():
                        for permission, level in called_job[
                            'permissions'
                        ].items():
                            if rank[level] > rank.get(
                                required.get(permission), 0
                            ):
                                required[permission] = level
                    self.assertEqual(job['permissions'], required)

    def test_manifest_publication_has_no_oidc_or_attestation_grant(self):
        job = workflow('create-manifest.yml')['jobs']['create-manifest']
        self.assertEqual(
            job['permissions'],
            {
                'contents': 'read',
                'packages': 'write',
            },
        )

    def run_build(self, push, digest):
        job = self.build_job(push)
        build = next(step for step in job['steps'] if step.get('id') == 'push')
        context = {
            'inputs': {
                'docker_file': 'docker files/Dockerfile',
                'image_context': 'build context',
                'platforms': 'linux/arm64',
                'push': push,
                'target_tag': '3.0.0-rc.1',
                'app_to_build': 'kfp-api-server',
            },
            'env': {'IMAGE_REGISTRY': 'ghcr.io', 'IMAGE_ORG': 'kubeflow'},
            'steps': {
                'meta': {'outputs': {'labels': 'example.name=literal value'}},
                'frontend_node': {'outputs': {'version': '24.4.0'}},
                'get_commit': {'outputs': {'sha': 'b' * 40}},
            },
        }
        with tempfile.TemporaryDirectory() as directory:
            temporary = Path(directory)
            wrapper = (
                temporary
                / '.github/resources/scripts/docker-buildx-build-with-retry.sh'
            )
            wrapper.parent.mkdir(parents=True)
            wrapper.write_text("""#!/usr/bin/env bash
set -euo pipefail
printf '%s\\0' "$@" > "$BUILD_ARGUMENTS"
while [ "$#" -gt 0 ]; do
  if [ "$1" = --metadata-file ]; then
    printf '{"containerimage.digest":"%s"}' "$BUILD_DIGEST" > "$2"
    break
  fi
  shift
done
""")
            arguments = temporary / 'arguments'
            output = temporary / 'output'
            environment = os.environ.copy()
            environment.update(
                {
                    key: render(value, context)
                    for key, value in build['env'].items()
                }
            )
            environment.update(
                {
                    'BUILD_ARGUMENTS': str(arguments),
                    'BUILD_DIGEST': digest,
                    'GITHUB_OUTPUT': str(output),
                }
            )
            result = subprocess.run(
                ['bash', '-e', '-o', 'pipefail', '-c', build['run']],
                cwd=temporary,
                env=environment,
                capture_output=True,
                text=True,
            )
            return (
                result,
                arguments.read_bytes().decode().split('\0')[:-1],
                output.read_text() if output.exists() else '',
            )

    def test_actual_build_command_preserves_dry_run_and_publication_outputs(
        self,
    ):
        for push in (False, True):
            with self.subTest(push=push):
                result, arguments, output = self.run_build(push, DIGEST)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn('--provenance=mode=min', arguments)
                self.assertIn(
                    'type=image,name=ghcr.io/kubeflow/kfp-api-server,'
                    f'push-by-digest=true,name-canonical=true,push={str(push).lower()}',
                    arguments,
                )
                self.assertEqual(
                    arguments[arguments.index('--platform') + 1], 'linux/arm64'
                )
                self.assertEqual(
                    arguments[arguments.index('--file') + 1],
                    'docker files/Dockerfile',
                )
                self.assertEqual(arguments[-1], 'build context')
                self.assertIn('COMMIT_SHA=' + 'b' * 40, arguments)
                self.assertIn('NODE_VERSION=24.4.0', arguments)
                self.assertEqual(output, f'digest={DIGEST}\n')

    def test_only_publication_requires_a_registry_digest(self):
        for push in (False, True):
            with self.subTest(push=push):
                result, _, output = self.run_build(push, '')
                self.assertEqual(result.returncode, 1 if push else 0)
                self.assertEqual(output, '')
                if push:
                    self.assertIn('without a container digest', result.stdout)


if __name__ == '__main__':
    unittest.main()
