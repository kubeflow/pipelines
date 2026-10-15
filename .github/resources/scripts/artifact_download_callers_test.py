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

import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest

from arm64_smoke import IMAGES
from artifact_workflow_test_support import render
import yaml

ROOT = Path(__file__).resolve().parents[3]
WRAPPER = './.github/actions/download-artifact-with-retry'
CALLERS = [
    ('.github/workflows/frontend.yml', 'browser-tests'),
    ('.github/actions/deploy/action.yml', 'composite'),
    ('.github/workflows/arm64-presubmit.yml', 'smoke'),
    ('.github/workflows/build-tools-images.yml', 'compare-generated'),
    ('.github/workflows/build-tools-images.yml', 'publish-tools'),
    ('.github/workflows/create-manifest.yml', 'create-manifest'),
    ('.github/workflows/image-builds-master.yml', 'arm64-smoke'),
    ('.github/workflows/image-builds-release.yml', 'validate-release-images'),
    ('.github/workflows/image-builds.yml', 'runtime-base-images'),
]


def workflow(name):
    return yaml.safe_load((ROOT / '.github/workflows' / name).read_text())


class ArtifactDownloadCallersTest(unittest.TestCase):

    def required_files(self, filename, job, context=None, cwd=ROOT):
        document = yaml.safe_load((ROOT / filename).read_text())
        steps = (
            document['runs']['steps']
            if job == 'composite' else document['jobs'][job]['steps'])
        caller = next(step for step in steps if step.get('uses') == WRAPPER)
        manifest = caller['with']['required-files']
        context = context or {}
        output = re.fullmatch(
            r'\$\{\{\s*steps\.([\w-]+)\.outputs\.required-files\s*\}\}',
            manifest)
        if output:
            producer = next(
                step for step in steps if step.get('id') == output.group(1))
            self.assertLess(steps.index(producer), steps.index(caller))
            environment = os.environ.copy()
            environment['PATH'] = (
                str(Path(sys.executable).parent) + os.pathsep +
                environment['PATH'])
            environment.update({
                key: render(value, context)
                for key, value in producer.get('env', {}).items()
            })
            with tempfile.TemporaryDirectory() as directory:
                output_path = Path(directory) / 'github-output'
                environment['GITHUB_OUTPUT'] = str(output_path)
                result = subprocess.run(
                    [
                        'bash', '-e', '-o', 'pipefail', '-c',
                        render(producer['run'], context)
                    ],
                    cwd=cwd,
                    env=environment,
                    text=True,
                    capture_output=True,
                    check=False,
                )
                self.assertEqual(result.returncode, 0,
                                 result.stdout + result.stderr)
                lines = output_path.read_text().splitlines()
            start = lines.index('required-files<<EOF') + 1
            manifest = '\n'.join(lines[start:lines.index('EOF', start)])
        else:
            manifest = render(manifest, context)
        files = manifest.splitlines()
        self.assertTrue(files, f'{filename}:{job} has an empty manifest')
        self.assertEqual(
            len(files), len(set(files)), 'Duplicate required files')
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
                        'action.yml', 'action.yaml'):
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
                            self.assertEqual(
                                location[0],
                                '.github/actions/download-artifact-with-retry/action.yml',
                                f'Raw download bypasses completeness: {location}'
                            )
                        if action != WRAPPER:
                            continue
                        self.assertTrue(
                            step.get('with', {}).get('required-files',
                                                     '').strip(),
                            f'Download requires a manifest: {location}')
                        callers.append(location)
        self.assertCountEqual(callers, CALLERS)

    def test_browser_bundle_uses_workspace_relative_paths_on_windows(self):
        steps = workflow('frontend.yml')['jobs']['browser-tests']['steps']
        download = next(step for step in steps if step.get('uses') == WRAPPER)
        self.assertEqual(download['with']['path'], 'frontend/.ci-bundle')
        self.assertEqual(download['with']['required-files'],
                         'frontend-bundle.tar.gz')
        extract = next(
            step for step in steps
            if step.get('name') == 'Extract complete production bundle')
        self.assertIn('.ci-bundle/frontend-bundle.tar.gz', extract['run'])
        self.assertNotIn('RUNNER_TEMP', extract['run'])

    def test_deploy_manifest_matches_ci_producers_and_modelcar_selection(self):
        built_images = {
            item['image'] for item in workflow('image-builds.yml')['jobs']
            ['image-build']['strategy']['matrix']['include']
        }
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            scripts = root / '.github/resources/scripts'
            scripts.mkdir(parents=True)
            inventory = ROOT / '.github/resources/scripts/ci-image-artifacts.sh'
            (scripts / inventory.name).write_text(inventory.read_text())
            waiter = scripts / 'wait-for-image-artifacts.sh'
            waiter.write_text('#!/usr/bin/env bash\nexit 0\n')
            waiter.chmod(0o755)
            for modelcar in ('false', 'true'):
                with self.subTest(modelcar=modelcar):
                    actual = self.required_files(
                        '.github/actions/deploy/action.yml',
                        'composite', {
                            'inputs': {
                                'load_modelcar_fixture': modelcar
                            },
                            'github': {
                                'token': 'unused'
                            }
                        },
                        cwd=root)
                    expected = {
                        f'{name}/{name}.tar' for name in built_images
                        | {'runtime-base-images'}
                    }
                    if modelcar == 'true':
                        expected.add('runtime-base-images/modelcar.tar')
                    self.assertEqual(actual, expected)

    def test_arm_presubmit_requires_each_produced_archive_and_record(self):
        producer = workflow('image-builds.yml')['jobs']['image-build']
        images = {
            item['image'] for item in producer['strategy']['matrix']['include']
        }
        upload = next(
            step for step in producer['steps']
            if step.get('uses', '').startswith('actions/upload-artifact@'))
        self.assertEqual(upload['with']['path'].splitlines(), [
            '${{ env.ARTIFACTS_PATH }}/${{ env.ARTIFACT_NAME }}.tar',
            '${{ env.ARTIFACTS_PATH }}/${{ env.ARTIFACT_NAME }}.json',
        ])
        actual = self.required_files('.github/workflows/arm64-presubmit.yml',
                                     'smoke')
        self.assertEqual(
            actual, {
                f'{name}-arm64.{extension}' for name in images
                for extension in ('tar', 'json')
            })

    def test_published_record_manifests_follow_each_publication_matrix(self):
        for name, job in (('image-builds-master.yml', 'arm64-smoke'),
                          ('image-builds-release.yml',
                           'validate-release-images')):
            with self.subTest(workflow=name):
                images = {
                    item['image'] for item in workflow(name)['jobs']
                    ['create-manifests']['strategy']['matrix']['component']
                }
                self.assertEqual(images, IMAGES)
                actual = self.required_files(f'.github/workflows/{name}', job)
                self.assertEqual(actual, {f'{name}.json' for name in images})

    def test_digest_manifest_respects_supported_platform_subsets(self):
        for platforms, expected in (('linux/amd64,linux/arm64', {
                'amd64.json', 'arm64.json'
        }), ('linux/amd64', {'amd64.json'}), ('linux/arm64', {'arm64.json'})):
            with self.subTest(platforms=platforms):
                actual = self.required_files(
                    '.github/workflows/create-manifest.yml', 'create-manifest',
                    {'inputs': {
                        'expected_platforms': platforms
                    }})
                self.assertEqual(actual, expected)
        producer = (ROOT / '.github/workflows/build-and-push.yml').read_text()
        self.assertIn('> "/tmp/digests/${ARCH}.json"', producer)

    def test_shared_runtime_download_requires_both_produced_archives(self):
        job = workflow('image-builds.yml')['jobs']['runtime-base-images']
        actual = self.required_files('.github/workflows/image-builds.yml',
                                     'runtime-base-images', {'env': job['env']})
        self.assertEqual(actual, {'runtime-base-images.tar', 'modelcar.tar'})
        producer = (ROOT /
                    '.github/resources/scripts/build-runtime-base-images.sh'
                   ).read_text()
        for filename in actual:
            self.assertIn(filename, producer)

    def inventory(self, *arguments):
        return subprocess.check_output([
            'bash',
            str(ROOT / '.github/resources/scripts/ci-image-artifacts.sh'),
            *arguments
        ],
                                       text=True).splitlines()

    def test_tool_output_manifest_matches_executed_producer(self):
        document = workflow('build-tools-images.yml')
        architectures = {
            item['arch'] for item in document['jobs']['build-tools']['strategy']
            ['matrix']['include']
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
            subprocess.run([
                'bash', '-e', '-o', 'pipefail', '-c', script, 'producer',
                str(ROOT /
                    '.github/resources/scripts/maintainer_tools_smoke.sh'),
                str(output)
            ],
                           cwd=ROOT,
                           check=True,
                           capture_output=True,
                           text=True)
            filenames = {path.name for path in output.iterdir()}
            self.assertEqual(filenames,
                             set(self.inventory('tool-output-files')))
            self.assertTrue(
                all(path.stat().st_size for path in output.iterdir()))
        for attempt in ('1', '17'):
            with self.subTest(attempt=attempt):
                actual = self.required_files(
                    '.github/workflows/build-tools-images.yml',
                    'compare-generated', {'github': {
                        'run_attempt': attempt
                    }})
                self.assertEqual(
                    actual, {
                        f'tool-output-{arch}-{attempt}/{name}'
                        for arch in architectures
                        for name in filenames
                    })

    def test_tool_digest_manifest_matches_executed_staging(self):
        producer = workflow('build-tools-images.yml')['jobs']['build-tools']
        architectures = {
            item['arch'] for item in producer['strategy']['matrix']['include']
        }
        stage = next(step['run'] for step in producer['steps'] if step.get(
            'name') == 'Stage tested images without changing shared tags')
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
                        **os.environ, 'RUNNER_TEMP': directory,
                        'ARCH': arch,
                        'PLATFORM': f'linux/{arch}',
                        'SOURCE_SHA': '1' * 40,
                        'IMAGE_REGISTRY': 'ghcr.io',
                        'IMAGE_ORG': 'kubeflow',
                        'RUN_TAG': 'run-123-17'
                    },
                    check=True,
                    capture_output=True,
                    text=True)
            records = Path(directory) / 'tool-digests'
            produced = {
                path.relative_to(records).as_posix()
                for path in records.rglob('*.json')
            }
        actual = self.required_files('.github/workflows/build-tools-images.yml',
                                     'publish-tools')
        self.assertEqual(actual, produced)

    def test_inventory_rejects_invalid_command_and_attempt(self):
        for arguments in (('unknown',), ('tool-output-artifacts', '0'),
                          ('tool-output-artifacts', '../1')):
            with self.subTest(arguments=arguments):
                with self.assertRaises(subprocess.CalledProcessError):
                    subprocess.check_output([
                        'bash',
                        str(ROOT /
                            '.github/resources/scripts/ci-image-artifacts.sh'),
                        *arguments
                    ],
                                            stderr=subprocess.PIPE,
                                            text=True)


if __name__ == '__main__':
    unittest.main()
