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
"""Unit tests for structure-aware Go metadata discovery."""

from pathlib import Path
import subprocess
import unittest
from unittest import mock

import go_version_metadata
from go_version_metadata import docker_buildkit_metadata
from go_version_metadata import docker_runtime_classification
from go_version_metadata import has_go_runtime_reference
from go_version_metadata import has_setup_go_use
from go_version_metadata import is_container_recipe
from go_version_metadata import is_runtime_metadata_path
from go_version_metadata import is_yaml_metadata_path
from go_version_metadata import yaml_mapping_values


class GoVersionMetadataTest(unittest.TestCase):

    def tearDown(self):
        go_version_metadata.inspect_metadata.cache_clear()

    def test_malformed_flow_collection_is_rejected(self):
        with self.assertRaises(ValueError):
            yaml_mapping_values('steps: [}\n', ('uses',))

    def test_docker_classification_is_explicit(self):
        digest = 'a' * 64
        for contents, expected in (
            (f'FROM golang:1.27.0@sha256:{digest} AS builder\n',
             'managed'), ('FROM golang:latest AS builder\n', 'unsupported'),
            ('FROM alpine\nCOPY --exclude=ignored . /app\n',
             'irrelevant'), ('FROM golang:latest\nRUN <<EOF\n', 'invalid')):
            with self.subTest(expected=expected):
                self.assertEqual(
                    docker_runtime_classification(contents)['classification'],
                    expected,
                )

    def test_frontend_selector_is_always_runtime_metadata(self):
        for contents, value, line, exact in (
                ('# syntax=example.com/golang:latest\ncustom payload\n',
                 'example.com/golang:latest', 1, True),
                ('// syntax=example.com/golang:latest\n',
                 'example.com/golang:latest', 1, True),
                ('{"syntax":"example.com/golang:latest"}\n',
                 'example.com/golang:latest', 1, True),
                ('#\u00a0check=   \n'
                 '#\u00a0syntax=example.com/golang:latest\ncustom payload\n',
                 '', 2, False),
                ('#\u00a0escape=   \n'
                 '#\u00a0syntax=example.com/golang:latest\ncustom payload\n',
                 '', 2, False),
                ('#\u00a0syntax=   \ncustom payload\n', '', 1, False)):
            with self.subTest(contents=contents):
                result = docker_runtime_classification(contents)
                self.assertEqual(result['classification'], 'unsupported')
                self.assertEqual(
                    [candidate['kind'] for candidate in result['candidates']],
                    ['unsupported-frontend'],
                )
                if exact:
                    self.assertEqual(result['candidates'][0]['value'], value)
                else:
                    self.assertNotIn('value', result['candidates'][0])
                self.assertTrue(
                    has_go_runtime_reference(Path('Dockerfile'), contents))
                projection = docker_buildkit_metadata(contents)
                self.assertEqual(projection['instructions'], [])
                expected_directives = []
                if exact:
                    expected_directives = [{
                        'name': 'syntax',
                        'value': value,
                        'line': line,
                    }]
                self.assertEqual(projection['directives'],
                                 expected_directives)

    def test_ambiguous_compatibility_preambles_are_rejected(self):
        long_prefix = '#\u00a0check='
        long_check = long_prefix + 'x' * ((64 * 1024) - len(
            long_prefix.encode('utf-8')))
        for contents, expected_line in (
                ('#\u00a0check=first\n#\u00a0check=second\n'
                 '#\u00a0syntax=example.com/frontend:latest\nFROM scratch\n',
                 2),
                ('#\u00a0escape=\\\n#\u00a0escape=`\n'
                 '#\u00a0syntax=example.com/frontend:latest\nFROM scratch\n',
                 2),
                (long_check +
                 '\n#\u00a0syntax=example.com/frontend:latest\nFROM scratch\n',
                 1)):
            with self.subTest(expected_line=expected_line):
                result = docker_runtime_classification(contents)
                self.assertEqual(result['classification'], 'unsupported')
                self.assertEqual(result['error'], '')
                self.assertNotIn('value', result['candidates'][0])
                self.assertEqual(result['candidates'][0]['line'], expected_line)

    def test_whitespace_only_frontend_metadata_is_bounded(self):
        prefix = '#\u00a0syntax=' + ('\t' * (32 * 1024)) + '\n'
        contents = prefix + ('x' * (
            go_version_metadata.MAX_METADATA_INPUT_BYTES -
            len(prefix.encode('utf-8'))))
        result = docker_runtime_classification(contents)
        self.assertEqual(result['classification'], 'unsupported')
        self.assertNotIn('value', result['candidates'][0])
        projection = docker_buildkit_metadata(contents)
        self.assertEqual(projection['directives'], [])

    def test_yaml_block_scalars_are_structural(self):
        contents = ('steps:\n'
                    '  - uses: >-\n'
                    '      actions/setup-go@v7\n'
                    'container:\n'
                    '  image: |-\n'
                    '    golang:1.27.0\n')
        self.assertTrue(has_setup_go_use(contents))
        self.assertTrue(has_go_runtime_reference(Path('test.yaml'), contents))

        command = ('run: |\n'
                   '  uses: actions/setup-go@v7\n'
                   '  image: golang:1.27.0\n')
        self.assertFalse(has_setup_go_use(command))
        self.assertFalse(has_go_runtime_reference(Path('test.yaml'), command))

    def test_yaml_decodes_escapes_and_aliases(self):
        contents = ('go-image: &go-image "gol\\u0061ng"\n'
                    'setup: &setup "actions/setup\\u002dgo@v7"\n'
                    'container: {image: *go-image}\n'
                    'steps: [{uses: *setup}]\n')
        self.assertTrue(has_setup_go_use(contents))
        self.assertTrue(has_go_runtime_reference(Path('test.yaml'), contents))

    def test_yaml_action_uses_are_semantic_and_case_insensitive(self):
        for value in ('docker://golang:1.20',
                      'Docker://docker.io/library/Golang:1.20'):
            with self.subTest(value=value):
                self.assertTrue(
                    has_go_runtime_reference(
                        Path('workflow.yaml'),
                        f'steps: [{{uses: {value}}}]\n',
                    ))
        self.assertTrue(
            has_setup_go_use('steps: [{uses: Actions/Setup-Go@v7}]\n'))
        for value in ('./.github/actions/golang', 'owner/golang@v1'):
            with self.subTest(value=value):
                self.assertFalse(
                    has_go_runtime_reference(
                        Path('workflow.yaml'),
                        f'steps: [{{uses: {value}}}]\n',
                    ))

    def test_malformed_escaped_yaml_candidates_fail_closed(self):
        for escape in (r'\x61', r'\u0061', r'\U00000061'):
            contents = f'container: {{image: "gol{escape}ng"\n'
            with self.subTest(kind='runtime', escape=escape):
                with self.assertRaises(ValueError):
                    has_go_runtime_reference(Path('workflow.yaml'), contents)
        for escape in (r'\x2d', r'\u002d', r'\U0000002d'):
            contents = f'steps: [{{uses: "actions/setup{escape}go@v7"}}\n'
            with self.subTest(kind='setup-go', escape=escape):
                with self.assertRaises(ValueError):
                    has_setup_go_use(contents)

    def test_all_malformed_yaml_fails_closed_without_a_fallback_lexer(self):
        for contents in ('steps: [}\n', 'steps: [{uses: "go\\\nlang:1.20"}\n',
                         'run: alpine\ninvalid: [}\n'):
            with self.subTest(contents=contents):
                with self.assertRaises(ValueError):
                    has_go_runtime_reference(Path('workflow.yaml'), contents)
                with self.assertRaises(ValueError):
                    has_setup_go_use(contents)

    def test_only_the_named_invalid_repository_fixture_is_excluded(self):
        contents = 'steps: [}\n'
        self.assertFalse(
            has_go_runtime_reference(
                Path('backend/src/crd/samples/scheduledworkflow/invalid.yaml'),
                contents,
            ))
        with self.assertRaises(ValueError):
            has_go_runtime_reference(Path('another/invalid.yaml'), contents)

    def test_yaml_colon_requires_mapping_separation(self):
        contents = ('image:golang\n'
                    'uses:actions/setup-go@v7\n'
                    'container:golang\n')
        self.assertEqual(
            yaml_mapping_values(contents, ('image', 'uses', 'container')),
            {
                'image': [],
                'uses': [],
                'container': []
            },
        )
        self.assertFalse(has_setup_go_use(contents))
        self.assertFalse(has_go_runtime_reference(Path('test.yaml'), contents))

    def test_docker_arg_values_are_not_evaluated(self):
        result = docker_runtime_classification(
            'ARG IMAGE=golang:1.27.0\nFROM ${IMAGE} AS builder\n')
        self.assertEqual(result['classification'], 'unsupported')

    def test_local_stage_named_golang_is_not_an_external_image(self):
        result = docker_runtime_classification(
            'FROM alpine AS golang\nFROM golang AS final\n')
        self.assertEqual(result['classification'], 'irrelevant')

    def test_container_recipe_names_are_discovered(self):
        for name in ('Dockerfile', 'Dockerfile.dev', 'Containerfile',
                     'Containerfile.dev'):
            with self.subTest(name=name):
                self.assertTrue(is_container_recipe(Path('nested') / name))
        self.assertFalse(is_container_recipe(Path('NotAContainerfile')))

    def test_metadata_suffixes_are_case_insensitive(self):
        for name in ('runtime.yaml', 'runtime.YAML', 'runtime.YmL',
                     'runtime.SH'):
            with self.subTest(name=name):
                self.assertTrue(is_runtime_metadata_path(Path(name)))
        for name in ('workflow.yaml', 'workflow.YAML', 'workflow.YmL'):
            with self.subTest(name=name):
                self.assertTrue(is_yaml_metadata_path(Path(name)))

    def test_discovers_external_copy_and_run_mount_images(self):
        result = docker_runtime_classification(
            'FROM alpine\n'
            'COPY --exclude=ignored --from=golang:1.27.0 /go /go\n'
            'RUN --mount=from=golang@sha256:' + ('a' * 64) +
            ',target=/go true\n')
        self.assertEqual(
            result['classification'],
            'unsupported',
        )
        self.assertEqual(
            [candidate['kind'] for candidate in result['candidates']],
            ['copy-from', 'run-mount-from'],
        )

    def test_metadata_input_size_is_bounded_before_subprocess(self):
        with mock.patch.object(go_version_metadata.subprocess, 'run') as run:
            with self.assertRaisesRegex(RuntimeError, 'input limit'):
                go_version_metadata.inspect_metadata(
                    Path('workflow.yaml'),
                    'x' * (go_version_metadata.MAX_METADATA_INPUT_BYTES + 1),
                )
        run.assert_not_called()

    def test_metadata_helper_invocation_is_bounded(self):
        completed = subprocess.CompletedProcess(
            args=('helper',), returncode=0, stdout='{}\n', stderr='')
        with mock.patch.object(
                go_version_metadata, '_helper_binary',
                return_value=Path('/helper')):
            with mock.patch.object(
                    go_version_metadata.subprocess, 'run',
                    return_value=completed) as run:
                go_version_metadata.inspect_metadata(
                    Path('workflow.yaml'), 'steps: []\n')
        self.assertEqual(
            run.call_args.kwargs['timeout'],
            go_version_metadata.METADATA_INSPECTION_TIMEOUT_SECONDS,
        )

    def test_public_deadline_bounds_distinct_deep_source_arg_defaults(self):
        depth = 9331
        nested_default = '${A:-' * depth + 'golang:latestx' + '}' * depth
        contents = 'FROM alpine\n' + ''.join(
            f'ARG A{index}={nested_default}{index}\n' for index in range(8))
        self.assertEqual(len(contents.encode('utf-8')), 448084)

        result = docker_runtime_classification(contents)

        self.assertEqual(result['classification'], 'invalid')
        self.assertIn('normalization input limit', result['error'])

    def test_metadata_helper_timeout_is_actionable_and_retried(self):
        completed = subprocess.CompletedProcess(
            args=('helper',), returncode=0, stdout='{}\n', stderr='')
        timeout = subprocess.TimeoutExpired(
            cmd=('helper',),
            timeout=go_version_metadata.METADATA_INSPECTION_TIMEOUT_SECONDS,
        )
        with mock.patch.object(
                go_version_metadata, '_helper_binary',
                return_value=Path('/helper')):
            with mock.patch.object(
                    go_version_metadata.subprocess,
                    'run',
                    side_effect=(timeout, completed)) as run:
                with self.assertRaisesRegex(RuntimeError,
                                            'workflow.yaml.*timed out'):
                    go_version_metadata.inspect_metadata(
                        Path('workflow.yaml'), 'steps: []\n')
                self.assertEqual(
                    go_version_metadata.inspect_metadata(
                        Path('workflow.yaml'), 'steps: []\n'), {})
        self.assertEqual(run.call_count, 2)

    def test_resource_limits_cannot_fail_open_escaped_candidates(self):
        image = ('---\nimage: "gol\\u0061ng"\n' + ('---\na: 1\n' * 64))
        with self.assertRaisesRegex(RuntimeError, 'resource limit'):
            has_go_runtime_reference(Path('workflow.yaml'), image)

        setup = ('---\nuses: "actions/setup\\u002dgo@v7"\n' +
                 ('---\na: 1\n' * 64))
        with self.assertRaisesRegex(RuntimeError, 'resource limit'):
            has_setup_go_use(setup)

        parser_depth = ('image: "gol\\u0061ng"\ndeep: ' + ('[' * 10001) + '0' +
                        (']' * 10001))
        with self.assertRaisesRegex(RuntimeError, 'exceeded max depth'):
            has_go_runtime_reference(Path('workflow.yaml'), parser_depth)


if __name__ == '__main__':
    unittest.main()
