#!/usr/bin/env python3
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
"""Keep download manifests complete and aligned with their artifact
producers."""

import json
import os
import re
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import shared_ci_images
import yaml
from arm64_smoke import IMAGES
from artifact_workflow_test_support import render

ROOT = Path(__file__).resolve().parents[3]
WRAPPER = './.github/actions/download-artifact-with-retry'
WRAPPERS = {WRAPPER, './tooling/.github/actions/download-artifact-with-retry'}
CALLERS = [
    ('.github/actions/shared-ci-images/action.yml', 'composite'),
    ('.github/workflows/build-tools-images.yml', 'compare-generated'),
    ('.github/workflows/build-tools-images.yml', 'publish-tools'),
    ('.github/workflows/create-manifest.yml', 'create-manifest'),
    ('.github/workflows/image-builds-master.yml', 'arm64-smoke'),
    ('.github/workflows/image-builds-release.yml', 'validate-release-images'),
    ('.github/workflows/image-builds.yml', 'runtime-base-images'),
    ('.github/workflows/release-qualification.yml', 'aggregate'),  # Images
    ('.github/workflows/release-qualification.yml', 'aggregate'),  # Source/SDK
    ('.github/workflows/release-qualification.yml', 'aggregate'),  # Third-party
    ('.github/workflows/third-party-image-scanning.yml', 'report'),  # Inventory
    ('.github/workflows/third-party-image-scanning.yml', 'report'),  # Scans
    ('.github/workflows/third-party-image-scanning.yml', 'upload'),  # SARIF
    ('.github/workflows/release-branch-image-scanning.yml', 'aggregate'),
    ('.github/workflows/release-branch-image-scanning.yml', 'upload'),
    ('.github/workflows/osv-scanner.yml', 'discover-images'),
]


def workflow(name):
    return yaml.safe_load((ROOT / '.github/workflows' / name).read_text())


class ArtifactDownloadCallersTest(unittest.TestCase):
    def required_files(
        self, filename, job, context=None, cwd=ROOT, caller_index=0
    ):
        document = yaml.safe_load((ROOT / filename).read_text())
        steps = (
            document['runs']['steps']
            if job == 'composite'
            else document['jobs'][job]['steps']
        )
        caller = [step for step in steps if step.get('uses') in WRAPPERS][
            caller_index
        ]
        manifest = caller['with']['required-files']
        context = context or {}
        output = re.fullmatch(
            r'\$\{\{\s*steps\.([\w-]+)\.outputs\.required-files\s*\}\}',
            manifest,
        )
        if output:
            producer = next(
                step for step in steps if step.get('id') == output.group(1)
            )
            self.assertLess(steps.index(producer), steps.index(caller))
            environment = os.environ.copy()
            environment['PATH'] = (
                str(Path(sys.executable).parent)
                + os.pathsep
                + environment['PATH']
            )
            environment.update(
                {
                    key: render(value, context)
                    for key, value in producer.get('env', {}).items()
                }
            )
            with tempfile.TemporaryDirectory() as directory:
                output_path = Path(directory) / 'github-output'
                environment['GITHUB_OUTPUT'] = str(output_path)
                result = subprocess.run(
                    [
                        'bash',
                        '-e',
                        '-o',
                        'pipefail',
                        '-c',
                        render(producer['run'], context),
                    ],
                    cwd=cwd,
                    env=environment,
                    text=True,
                    capture_output=True,
                    check=False,
                )
                self.assertEqual(
                    result.returncode, 0, result.stdout + result.stderr
                )
                lines = output_path.read_text().splitlines()
            start = lines.index('required-files<<EOF') + 1
            manifest = '\n'.join(lines[start : lines.index('EOF', start)])
        else:
            manifest = render(manifest, context)
        files = manifest.splitlines()
        self.assertTrue(files, f'{filename}:{job} has an empty manifest')
        self.assertEqual(
            len(files), len(set(files)), 'Duplicate required files'
        )
        for filename in files:
            self.assertTrue(filename.strip())
            self.assertFalse(Path(filename).is_absolute(), filename)
            self.assertNotIn('..', Path(filename).parts)
        return set(files)

    def test_every_maintained_download_has_a_manifest_and_no_raw_bypass(self):
        callers = []
        for directory in (ROOT / '.github/workflows', ROOT / '.github/actions'):
            for path in sorted(directory.rglob('*')):
                if path.suffix not in ('.yml', '.yaml'):
                    continue
                if directory.name == 'actions' and path.name not in (
                    'action.yml',
                    'action.yaml',
                ):
                    continue
                # Agent workflow lockfiles are generated and use their own
                # artifact protocol; this contract covers maintained CI YAML.
                if path.name.endswith(('.lock.yml', '.lock.yaml')):
                    continue
                document = yaml.safe_load(path.read_text())
                if not isinstance(document, dict):
                    continue
                jobs = {'composite': document.get('runs', {})}
                jobs.update(document.get('jobs', {}))
                for job, definition in jobs.items():
                    for step in definition.get('steps', []):
                        action = step.get('uses', '')
                        location = (path.relative_to(ROOT).as_posix(), job)
                        if action.startswith('actions/download-artifact@'):
                            # The OIDC-only PyPI job cannot check out this local
                            # wrapper. Its single named same-run artifact has an
                            # exact checksum inventory, covered by
                            # publish_packages_workflow_test.py.
                            self.assertIn(
                                location,
                                {
                                    (
                                        '.github/actions/download-artifact-with-retry/action.yml',
                                        'composite',
                                    ),
                                    (
                                        '.github/workflows/publish-packages.yml',
                                        'publish',
                                    ),
                                },
                                f'Raw download bypasses completeness: {location}',
                            )
                        if action not in WRAPPERS:
                            continue
                        self.assertTrue(
                            step.get('with', {})
                            .get('required-files', '')
                            .strip(),
                            f'Download requires a manifest: {location}',
                        )
                        callers.append(location)
        self.assertCountEqual(callers, CALLERS)

    def test_release_qualification_manifest_matches_attempt_bound_lane_artifacts(
        self,
    ):
        document = workflow('release-qualification.yml')
        producer = next(
            step
            for step in document['jobs']['images']['steps']
            if step.get('uses', '').startswith('actions/upload-artifact@')
        )
        # Include an AMD64-only image to prove completeness follows the source
        # matrix rather than assuming the tool branch's image/platform pairs.
        lanes = [
            {'name': 'legacy-api', 'arch': 'amd64'},
            {'name': 'legacy-api', 'arch': 'arm64'},
            {'name': 'legacy-proxy', 'arch': 'amd64'},
        ]
        for attempt in ('1', '17'):
            with self.subTest(attempt=attempt):
                context = {
                    'inputs': {'request_id': 'a' * 32},
                    'github': {'run_attempt': attempt},
                    'needs': {
                        'inventory': {
                            'outputs': {
                                'matrix': json.dumps({'include': lanes}),
                            }
                        }
                    },
                }
                actual = self.required_files(
                    '.github/workflows/release-qualification.yml',
                    'aggregate',
                    context,
                )
                expected = {
                    render(
                        producer['with']['name'], {**context, 'matrix': lane}
                    )
                    + '/'
                    + filename
                    for lane in lanes
                    for filename in ('lane.json', 'scan.json')
                }
                self.assertEqual(actual, expected)
                steps = document['jobs']['aggregate']['steps']
                download = next(
                    step for step in steps if step.get('uses') in WRAPPERS
                )
                verify = next(
                    step
                    for step in steps
                    if 'ci_qualification_runner aggregate'
                    in step.get('run', '')
                )
                self.assertLess(steps.index(download), steps.index(verify))
                self.assertNotIn('continue-on-error', download)
                self.assertNotIn('if', verify)

    def shared_outputs(self, architecture, kind):
        images = (
            ['runtime-base-images']
            if kind == 'runtime'
            else sorted(shared_ci_images.IMAGES)
        )
        records = [
            {
                'artifact_id': index + 100,
                'provenance': image
                + ('-arm64' if architecture == 'arm64' else '')
                + '.provenance.json',
                'files': {
                    name: 'a' * 64
                    for name in shared_ci_images.filenames(image, architecture)
                },
            }
            for index, image in enumerate(images)
        ]
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'output'
            shared_ci_images.emit_outputs(
                output,
                {
                    'identity': {
                        'run_id': 1234,
                        'repository': 'upstream/pipelines',
                    },
                    'sets': {kind: records},
                },
                kind,
            )
            lines = output.read_text().splitlines()
        start = lines.index('required-files<<SHARED_CI_FILES') + 1
        files = set(lines[start : lines.index('SHARED_CI_FILES', start)])
        return lines, files, records

    def test_shared_manifest_matches_ci_producers_and_modelcar_selection(self):
        built_images = {
            item['image']
            for item in workflow('image-builds.yml')['jobs']['image-build'][
                'strategy'
            ]['matrix']['include']
        }
        self.assertEqual(set(shared_ci_images.IMAGES), built_images)
        _, runtime, _ = self.shared_outputs('amd64', 'runtime')
        _, branch, _ = self.shared_outputs('amd64', 'images')
        # Runtime archives are verified together; deploy still loads Modelcar only
        # when requested, and branches are normalized into nested paths afterwards.
        self.assertEqual(
            runtime,
            {
                'runtime-base-images.tar',
                'modelcar.tar',
                'runtime-base-images.provenance.json',
            },
        )
        self.assertEqual(
            branch,
            {
                f'{name}.{extension}'
                for name in built_images
                for extension in ('tar', 'provenance.json')
            },
        )
        self.assertFalse(runtime & branch)

    def test_shared_download_uses_only_resolved_ids_and_verifies_after_transfer(
        self,
    ):
        action = yaml.safe_load(
            (ROOT / '.github/actions/shared-ci-images/action.yml').read_text()
        )
        steps = action['runs']['steps']
        resolver = next(step for step in steps if step.get('id') == 'resolve')
        download = next(step for step in steps if step.get('uses') == WRAPPER)
        verify = next(
            step
            for step in steps
            if 'shared_ci_images.py verify' in step.get('run', '')
        )
        self.assertLess(steps.index(resolver), steps.index(download))
        self.assertLess(steps.index(download), steps.index(verify))
        for key in ('artifact-ids', 'run-id', 'repository', 'required-files'):
            self.assertEqual(
                download['with'][key],
                '${{ steps.resolve.outputs.' + key + ' }}',
            )
        self.assertEqual(
            download['with']['github-token'], '${{ github.token }}'
        )
        self.assertEqual(download['with']['merge-multiple'], 'true')
        self.assertNotIn('name', download['with'])
        self.assertNotIn('pattern', download['with'])
        for step in (resolver, download, verify):
            self.assertNotIn('continue-on-error', step)
            self.assertNotIn('if', step)
        self.assertEqual(
            resolver['env']['PRODUCER_STATE'], verify['env']['PRODUCER_STATE']
        )
        self.assertEqual(
            action['inputs']['state']['default'],
            '${{ runner.temp }}/kfp-shared-images.json',
        )
        for architecture, kind in (
            ('amd64', 'runtime'),
            ('amd64', 'images'),
            ('arm64', 'images'),
        ):
            lines, _, records = self.shared_outputs(architecture, kind)
            self.assertIn(
                'artifact-ids='
                + ','.join(str(record['artifact_id']) for record in records),
                lines,
            )
            self.assertIn('run-id=1234', lines)
            self.assertIn('repository=upstream/pipelines', lines)

    def test_arm_presubmit_requires_each_produced_archive_and_record(self):
        producer = workflow('image-builds.yml')['jobs']['image-build']
        images = {
            item['image'] for item in producer['strategy']['matrix']['include']
        }
        upload = next(
            step
            for step in producer['steps']
            if step.get('uses', '').startswith('actions/upload-artifact@')
        )
        self.assertEqual(
            upload['with']['path'].splitlines(),
            [
                '${{ env.ARTIFACTS_PATH }}/${{ env.ARTIFACT_NAME }}.tar',
                '${{ env.ARTIFACTS_PATH }}/${{ env.ARTIFACT_NAME }}.json',
                '${{ env.ARTIFACTS_PATH }}/${{ env.ARTIFACT_NAME }}.provenance.json',
            ],
        )
        smoke = workflow('arm64-presubmit.yml')['jobs']['smoke']
        download = next(
            step
            for step in smoke['steps']
            if step.get('uses') == './.github/actions/shared-ci-images'
        )
        self.assertEqual(download['with']['architecture'], 'arm64')
        self.assertEqual(download['with']['layout'], 'flat')
        _, actual, _ = self.shared_outputs('arm64', 'images')
        self.assertEqual(
            actual,
            {
                f'{name}-arm64.{extension}'
                for name in images
                for extension in ('tar', 'json', 'provenance.json')
            },
        )

    def test_osv_discovery_requires_complete_published_records(self):
        actual = self.required_files(
            '.github/workflows/osv-scanner.yml', 'discover-images'
        )
        self.assertEqual(actual, {f'{name}.json' for name in IMAGES})

    def test_published_record_manifests_follow_each_publication_matrix(self):
        for name, job in (
            ('image-builds-master.yml', 'arm64-smoke'),
            ('image-builds-release.yml', 'validate-release-images'),
        ):
            with self.subTest(workflow=name):
                images = {
                    item['image']
                    for item in workflow(name)['jobs']['create-manifests'][
                        'strategy'
                    ]['matrix']['component']
                }
                self.assertEqual(images, IMAGES)
                actual = self.required_files(f'.github/workflows/{name}', job)
                self.assertEqual(actual, {f'{name}.json' for name in images})

    def test_digest_manifest_respects_supported_platform_subsets(self):
        for platforms, expected in (
            ('linux/amd64,linux/arm64', {'amd64.json', 'arm64.json'}),
            ('linux/amd64', {'amd64.json'}),
            ('linux/arm64', {'arm64.json'}),
        ):
            with self.subTest(platforms=platforms):
                actual = self.required_files(
                    '.github/workflows/create-manifest.yml',
                    'create-manifest',
                    {'inputs': {'expected_platforms': platforms}},
                )
                self.assertEqual(actual, expected)
        producer = (ROOT / '.github/workflows/build-and-push.yml').read_text()
        self.assertIn('> "/tmp/digests/${ARCH}.json"', producer)

    def test_shared_runtime_download_requires_both_produced_archives(self):
        job = workflow('image-builds.yml')['jobs']['runtime-base-images']
        actual = self.required_files(
            '.github/workflows/image-builds.yml',
            'runtime-base-images',
            {'env': job['env']},
        )
        self.assertEqual(actual, {'runtime-base-images.tar', 'modelcar.tar'})
        producer = (
            ROOT / '.github/resources/scripts/build-runtime-base-images.sh'
        ).read_text()
        for filename in actual:
            self.assertIn(filename, producer)

    def inventory(self, *arguments):
        return subprocess.check_output(
            [
                'bash',
                str(ROOT / '.github/resources/scripts/ci-image-artifacts.sh'),
                *arguments,
            ],
            text=True,
        ).splitlines()

    def test_tool_output_manifest_matches_executed_producer(self):
        document = workflow('build-tools-images.yml')
        architectures = {
            item['arch']
            for item in document['jobs']['build-tools']['strategy']['matrix'][
                'include'
            ]
        }
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'tool-output'
            # Execute the producer's file-writing flow without native Docker
            # generation or archiving the checkout. Source hashing is covered
            # separately by maintainer_tools_smoke_test.
            script = r"""
source "$1"
validate_native_images() { :; }
snapshot_sources() { echo 'fixture source checksum'; }
git() {
  if [[ "$1" == rev-parse ]]; then
    printf '%s\n' "$PWD"
  else
    tar -cf - -T /dev/null
  fi
}
docker() {
  cat >/dev/null
  local argument directory
  for argument in "$@"; do
    if [[ "$argument" == type=bind,source=* ]]; then
      directory="${argument#type=bind,source=}"
      directory="${directory%%,target=*}"
      printf '## 3.0.0-smoke (2026-10-05)\nfixture\n' > "$directory/CHANGELOG.md"
    fi
  done
}
main amd64 "$2"
"""
            subprocess.run(
                [
                    'bash',
                    '-e',
                    '-o',
                    'pipefail',
                    '-c',
                    script,
                    'producer',
                    str(
                        ROOT
                        / '.github/resources/scripts/maintainer_tools_smoke.sh'
                    ),
                    str(output),
                ],
                cwd=ROOT,
                check=True,
                capture_output=True,
                text=True,
            )
            filenames = {path.name for path in output.iterdir()}
            self.assertEqual(
                filenames, set(self.inventory('tool-output-files'))
            )
            self.assertTrue(
                all(path.stat().st_size for path in output.iterdir())
            )
        for attempt in ('1', '17'):
            with self.subTest(attempt=attempt):
                actual = self.required_files(
                    '.github/workflows/build-tools-images.yml',
                    'compare-generated',
                    {'github': {'run_attempt': attempt}},
                )
                self.assertEqual(
                    actual,
                    {
                        f'tool-output-{arch}-{attempt}/{name}'
                        for arch in architectures
                        for name in filenames
                    },
                )

    def test_tool_digest_manifest_matches_executed_staging(self):
        producer = workflow('build-tools-images.yml')['jobs']['build-tools']
        architectures = {
            item['arch'] for item in producer['strategy']['matrix']['include']
        }
        stage = next(
            step['run']
            for step in producer['steps']
            if step.get('name')
            == 'Stage tested images without changing shared tags'
        )
        mocks = r"""
docker() {
  local previous='' argument
  for argument in "$@"; do
    if [[ "$previous" == --metadata-file ]]; then
      printf '{}\n' > "$argument"
    fi
    previous="$argument"
  done
}
jq() {
  if [[ "$1" == -er ]]; then
    printf 'sha256:%064d\n' 1
  else
    python3 -c 'import json, sys; args=sys.argv[1:]; print(json.dumps({args[i+1]:args[i+2] for i, value in enumerate(args) if value == "--arg"}))' "$@"
  fi
}
"""
        with tempfile.TemporaryDirectory() as directory:
            for arch in architectures:
                subprocess.run(
                    ['bash', '-e', '-o', 'pipefail', '-c', mocks + stage],
                    cwd=ROOT,
                    env={
                        **os.environ,
                        'RUNNER_TEMP': directory,
                        'ARCH': arch,
                        'PLATFORM': f'linux/{arch}',
                        'SOURCE_SHA': '1' * 40,
                        'IMAGE_REGISTRY': 'ghcr.io',
                        'IMAGE_ORG': 'kubeflow',
                        'RUN_TAG': 'run-123-17',
                    },
                    check=True,
                    capture_output=True,
                    text=True,
                )
            records = Path(directory) / 'tool-digests'
            produced = {
                path.relative_to(records).as_posix()
                for path in records.rglob('*.json')
            }
        actual = self.required_files(
            '.github/workflows/build-tools-images.yml', 'publish-tools'
        )
        self.assertEqual(actual, produced)

    def test_inventory_rejects_invalid_command_and_attempt(self):
        for arguments in (
            ('unknown',),
            ('tool-output-artifacts', '0'),
            ('tool-output-artifacts', '../1'),
        ):
            with self.subTest(arguments=arguments):
                with self.assertRaises(subprocess.CalledProcessError):
                    subprocess.check_output(
                        [
                            'bash',
                            str(
                                ROOT
                                / '.github/resources/scripts/ci-image-artifacts.sh'
                            ),
                            *arguments,
                        ],
                        stderr=subprocess.PIPE,
                        text=True,
                    )


if __name__ == '__main__':
    unittest.main()
