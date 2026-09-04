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
"""Focused tests for the bounded repository Go-version updater."""

from pathlib import Path
import subprocess
import tempfile
import unittest

import update_go_version as updater


ACTION_SHA = '1' * 40
OLD_DIGEST = 'sha256:' + '0' * 64
NEW_DIGESTS = {
    '1.27.1': 'sha256:' + '1' * 64,
    '1.27.1-alpine': 'sha256:' + 'a' * 64,
    '1.27.1-bookworm': 'sha256:' + 'b' * 64,
}


def _git(repo_root, *arguments):
    return subprocess.run(
        ('git', *arguments),
        cwd=repo_root,
        check=True,
        capture_output=True,
        text=True,
    )


def _dockerfile(version, pin, digest=OLD_DIGEST):
    return (
        '# syntax owned by this fixture\n'
        f'FROM golang:{version}{pin.flavor}@{digest} AS {pin.stage}\n'
        'RUN true\n'
    )


def _setup_action():
    return f"""name: Set up Go
runs:
  using: composite
  steps:
    - name: Set up Go
      uses: actions/setup-go@{ACTION_SHA}
      with:
        go-version-file: go.mod
"""


class RepositoryFixture:

    def __init__(self, test_case, docker_pins=None):
        self._temporary = tempfile.TemporaryDirectory()
        test_case.addCleanup(self._temporary.cleanup)
        self.root = Path(self._temporary.name)
        self.docker_pins = tuple(docker_pins or (
            updater.DockerPin(Path('Dockerfile'), '-alpine', 'builder'),
        ))
        self.setup_actions = (Path('.github/actions/setup-go/action.yml'),)

        self.write(
            Path('go.mod'),
            'module example.com/root\n\n'
            'go 1.26.5\n\n'
            'toolchain go1.26.6\n\n'
            'require example.com/dependency v1.0.0\n',
        )
        self.write(
            Path('nested/go.mod'),
            'module example.com/nested\n\n'
            'go 1.25.0\n',
        )
        for pin in self.docker_pins:
            self.write(pin.path, _dockerfile('1.26.6', pin))
        self.write(self.setup_actions[0], _setup_action())
        self.write(Path('README.md'), 'fixture\n')

        _git(self.root, 'init', '-q')
        self.commit('initial fixture')

    def write(self, relative_path, contents):
        path = self.root / relative_path
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(contents, encoding='utf-8')

    def read(self, relative_path):
        return (self.root / relative_path).read_text(encoding='utf-8')

    def commit(self, message='fixture update'):
        _git(self.root, 'add', '-A')
        _git(
            self.root,
            '-c',
            'user.name=KFP Test',
            '-c',
            'user.email=kfp-test@example.invalid',
            'commit',
            '-qm',
            message,
        )

    def snapshot(self):
        paths = _git(self.root, 'ls-files', '-z').stdout.split('\0')
        return {
            Path(path): (self.root / path).read_bytes()
            for path in paths if path
        }

    def plan(self, target, resolver):
        return updater.plan_update(
            self.root,
            target,
            resolver,
            self.docker_pins,
            self.setup_actions,
        )

    def update(self, target, resolver):
        return updater.update_repository(
            self.root,
            target,
            resolver,
            self.docker_pins,
            self.setup_actions,
        )

    def check(self):
        updater.check_repository(
            self.root,
            self.docker_pins,
            self.setup_actions,
        )


class GoVersionUpdaterTest(unittest.TestCase):

    def test_exact_target_version(self):
        self.assertEqual(updater._parse_exact_version('1.27.0'), (1, 27, 0))
        for invalid in (
                '1.27', 'go1.27.0', '2.27.0', '1.027.0', '1.27.00',
                '1.27.0-rc1', ''):
            with self.subTest(invalid=invalid):
                with self.assertRaises(updater.PolicyError):
                    updater._parse_exact_version(invalid)

    def test_render_module_sets_and_removes_toolchain(self):
        fixture = RepositoryFixture(self)
        original = fixture.read(Path('go.mod'))

        patch_release = updater._render_module(
            fixture.root, original, (1, 26, 5), (1, 26, 7))
        self.assertIn('\ngo 1.26.5\n', patch_release)
        self.assertIn('\ntoolchain go1.26.7\n', patch_release)
        self.assertIn('require example.com/dependency v1.0.0', patch_release)

        minor_release = updater._render_module(
            fixture.root, original, (1, 27, 0), None)
        self.assertIn('\ngo 1.27.0\n', minor_release)
        self.assertNotIn('toolchain ', minor_release)
        self.assertIn('require example.com/dependency v1.0.0', minor_release)

    def test_root_module_requires_setup_go_compatible_directives(self):
        for invalid in ('  go 1.26.5', 'go  1.26.5'):
            with self.subTest(invalid=invalid):
                fixture = RepositoryFixture(self)
                contents = fixture.read(Path('go.mod')).replace(
                    'go 1.26.5', invalid)
                fixture.write(Path('go.mod'), contents)
                with self.assertRaisesRegex(updater.PolicyError,
                                            'exact line go 1.X.Y'):
                    fixture.plan('1.27.1', lambda tag: NEW_DIGESTS[tag])

    def test_plan_preserves_floor_for_patch_and_advances_it_for_minor(self):
        fixture = RepositoryFixture(self)

        patch = fixture.plan(
            '1.26.7', lambda _tag: 'sha256:' + '7' * 64)
        self.assertIn('\ngo 1.26.5\n', patch.expected[Path('go.mod')])
        self.assertIn(
            '\ntoolchain go1.26.7\n', patch.expected[Path('go.mod')])
        self.assertIn('\ngo 1.26.0\n', patch.expected[Path('nested/go.mod')])

        minor = fixture.plan(
            '1.27.0', lambda _tag: 'sha256:' + '8' * 64)
        for path in (Path('go.mod'), Path('nested/go.mod')):
            self.assertIn('\ngo 1.27.0\n', minor.expected[path])
            self.assertNotIn('toolchain ', minor.expected[path])

    def test_plan_rejects_module_floor_newer_than_target(self):
        fixture = RepositoryFixture(self)
        fixture.write(
            Path('nested/go.mod'),
            'module example.com/nested\n\ngo 1.28.0\n',
        )
        before = fixture.snapshot()
        with self.assertRaisesRegex(updater.PolicyError, 'newer than target'):
            fixture.plan('1.27.1', lambda tag: NEW_DIGESTS[tag])
        self.assertEqual(fixture.snapshot(), before)

    def test_docker_pin_validation_and_update(self):
        pin = updater.DockerPin(Path('Dockerfile'), '-alpine', 'builder')
        original = _dockerfile('1.26.6', pin)
        metadata = updater._docker_metadata(original, pin)
        self.assertEqual(metadata.version, '1.26.6')
        self.assertEqual(metadata.flavor, '-alpine')

        updated = updater._updated_dockerfile(
            original, metadata, '1.27.1', NEW_DIGESTS['1.27.1-alpine'])
        self.assertIn(
            'FROM golang:1.27.1-alpine@'
            f'{NEW_DIGESTS["1.27.1-alpine"]} AS builder',
            updated,
        )
        self.assertTrue(updated.endswith('RUN true\n'))

    def test_docker_pin_rejects_noncanonical_or_extra_sources(self):
        pin = updater.DockerPin(Path('Dockerfile'), '-alpine', 'builder')
        canonical = _dockerfile('1.26.6', pin)
        cases = {
            'lowercase AS': canonical.replace(' AS builder', ' as builder'),
            'missing digest': canonical.replace(f'@{OLD_DIGEST}', ''),
            'duplicate pin': canonical + canonical,
            'wrong flavor': canonical.replace('-alpine', '-bookworm'),
            'wrong stage': canonical.replace('AS builder', 'AS compiler'),
            'extra literal source': canonical + 'RUN echo golang:latest\n',
        }
        for name, contents in cases.items():
            with self.subTest(name=name):
                with self.assertRaises(updater.PolicyError):
                    updater._docker_metadata(contents, pin)

    def test_plan_resolves_each_distinct_flavor_once_without_writing(self):
        pins = (
            updater.DockerPin(Path('Dockerfile.default'), '', 'generator'),
            updater.DockerPin(Path('Dockerfile.alpine-a'), '-alpine', 'builder'),
            updater.DockerPin(Path('Dockerfile.alpine-b'), '-alpine', 'builder'),
            updater.DockerPin(Path('Dockerfile.bookworm'), '-bookworm', 'builder'),
        )
        fixture = RepositoryFixture(self, pins)
        before = fixture.snapshot()
        calls = []

        def resolve(tag):
            calls.append(tag)
            return NEW_DIGESTS[tag]

        plan = fixture.plan('1.27.1', resolve)

        self.assertEqual(
            calls, ['1.27.1', '1.27.1-alpine', '1.27.1-bookworm'])
        self.assertEqual(fixture.snapshot(), before)
        for pin in pins:
            expected = plan.expected[pin.path]
            self.assertIn(NEW_DIGESTS['1.27.1' + pin.flavor], expected)

    def test_planning_failure_does_not_write(self):
        fixture = RepositoryFixture(self)
        before = fixture.snapshot()

        def fail(_tag):
            raise updater.PolicyError('registry unavailable')

        with self.assertRaisesRegex(updater.PolicyError, 'registry unavailable'):
            fixture.plan('1.27.1', fail)
        self.assertEqual(fixture.snapshot(), before)

    def test_update_is_consistent_and_idempotent(self):
        fixture = RepositoryFixture(self)
        resolver = lambda _tag: NEW_DIGESTS['1.27.1-alpine']

        changed = fixture.update('1.27.1', resolver)
        self.assertEqual(
            changed,
            [Path('Dockerfile'), Path('go.mod'), Path('nested/go.mod')],
        )
        fixture.check()

        self.assertEqual(fixture.update('1.27.1', resolver), [])
        fixture.check()

    def test_update_rejects_dirty_managed_paths_but_allows_unrelated_changes(self):
        dirty_managed = RepositoryFixture(self)
        dirty_managed.write(
            Path('go.mod'), dirty_managed.read(Path('go.mod')) + '\n')
        resolver_called = []

        with self.assertRaisesRegex(updater.PolicyError,
                                    'managed files must be clean'):
            dirty_managed.update(
                '1.27.1', lambda tag: resolver_called.append(tag) or OLD_DIGEST)
        self.assertEqual(resolver_called, ['1.27.1-alpine'])

        dirty_unrelated = RepositoryFixture(self)
        dirty_unrelated.write(Path('README.md'), 'unrelated local edit\n')
        dirty_unrelated.update(
            '1.27.1', lambda _tag: NEW_DIGESTS['1.27.1-alpine'])
        self.assertEqual(
            dirty_unrelated.read(Path('README.md')), 'unrelated local edit\n')

    def test_update_rechecks_managed_paths_after_digest_resolution(self):
        fixture = RepositoryFixture(self)
        before = fixture.snapshot()

        def edit_during_resolution(_tag):
            (fixture.root / 'nested/go.mod').unlink()
            return NEW_DIGESTS['1.27.1-alpine']

        with self.assertRaisesRegex(updater.PolicyError,
                                    'managed files must be clean'):
            fixture.update('1.27.1', edit_during_resolution)
        self.assertFalse((fixture.root / 'nested/go.mod').exists())
        self.assertEqual(fixture.read(Path('Dockerfile')),
                         before[Path('Dockerfile')].decode())

    def test_verify_image_digests_checks_the_registry_value(self):
        fixture = RepositoryFixture(self)
        fixture.update(
            '1.27.1', lambda _tag: NEW_DIGESTS['1.27.1-alpine'])

        updater.verify_image_digests(
            fixture.root,
            lambda _tag: NEW_DIGESTS['1.27.1-alpine'],
            fixture.docker_pins,
            fixture.setup_actions,
        )
        with self.assertRaisesRegex(updater.PolicyError, 'not pinned digest'):
            updater.verify_image_digests(
                fixture.root,
                lambda _tag: 'sha256:' + 'f' * 64,
                fixture.docker_pins,
                fixture.setup_actions,
            )

    def test_inventory_rejects_unregistered_literal_docker_source(self):
        fixture = RepositoryFixture(self)
        extra = updater.DockerPin(
            Path('Dockerfile.unregistered'), '-bookworm', 'builder')
        fixture.write(extra.path, _dockerfile('1.26.6', extra))
        fixture.commit('add unregistered Dockerfile')

        with self.assertRaisesRegex(updater.PolicyError,
                                    'register literal Go sources'):
            fixture.plan('1.27.1', lambda tag: NEW_DIGESTS[tag])

    def test_inventory_rejects_unregistered_or_malformed_setup_action(self):
        unregistered = RepositoryFixture(self)
        unregistered.write(
            Path('.github/workflows/direct-setup.yml'), _setup_action())
        unregistered.commit('add direct setup-go caller')
        with self.assertRaisesRegex(updater.PolicyError,
                                    'route setup-go callers'):
            unregistered.plan('1.27.1', lambda tag: NEW_DIGESTS[tag])

        malformed = RepositoryFixture(self)
        path = malformed.setup_actions[0]
        malformed.write(
            path,
            _setup_action().replace(
                'go-version-file: go.mod', 'go-version: 1.26.6'),
        )
        with self.assertRaisesRegex(updater.PolicyError,
                                    'go-version-file: go.mod'):
            malformed.plan('1.27.1', lambda tag: NEW_DIGESTS[tag])

        conflicting = RepositoryFixture(self)
        path = conflicting.setup_actions[0]
        conflicting.write(
            path,
            _setup_action().replace(
                'go-version-file: go.mod',
                'go-version-file: go.mod\n'
                '        go-version: ${{ inputs.version }}',
            ),
        )
        with self.assertRaisesRegex(updater.PolicyError, 'must use only'):
            conflicting.plan('1.27.1', lambda tag: NEW_DIGESTS[tag])

    def test_current_repository_satisfies_the_contract(self):
        updater.check_repository(updater.REPOSITORY_ROOT)


if __name__ == '__main__':
    unittest.main()
