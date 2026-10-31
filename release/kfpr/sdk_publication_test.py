"""Publish SDK dependencies before dependents and resume partial releases."""

import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import core, steps

PACKAGES = ['kfp-pipeline-spec', 'kfp-server-api', 'kfp', 'kfp-kubernetes']
SHA = 'a' * 40


class SDKPublicationTest(unittest.TestCase):
    def setUp(self):
        directory = TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        root = Path(directory.name)
        self.runner = mock.Mock(spec=core.CommandRunner, dry_run=False)
        self.context = core.ReleaseContext(
            root=root,
            state=core.ReleaseState(root / 'state.json'),
            runner=self.runner,
            metadata=core.ReleaseMetadata.from_version('patch', '2.17.3'),
            fork_remote='origin',
            include_backend=True,
            include_sdk=True,
        )
        self.context.state.answers['release_sha'] = SHA
        self.available = set()
        self.events = []
        for name in (
            'qualification.require_qualification',
            '_ensure_release_tag',
        ):
            patcher = mock.patch('kfpr.steps.' + name)
            patcher.start()
            self.addCleanup(patcher.stop)

        def dispatch(command):
            package = next(
                arg.removeprefix('packages=')
                for arg in command
                if arg.startswith('packages=')
            )
            self.events.append(('dispatch', package))
            self.assertNotIn(package, self.available)
            self.assertEqual(command[command.index('--ref') + 1], 'sdk-2.17.3')
            if package == 'kfp':
                self.assertTrue(set(PACKAGES[:2]) <= self.available)
            if package == 'kfp-kubernetes':
                self.assertIn('kfp', self.available)

        self.runner.run.side_effect = dispatch

    def run_publication(self, consolidated=False, wait=None, watch_error=None):
        def available(version, packages):
            self.assertEqual(version, '2.17.3')
            self.assertEqual(len(packages), 1)
            self.events.append(('available', packages[0]))
            self.available.update(packages)
            return []

        with (
            mock.patch.object(
                steps, '_sdk_release_is_consolidated', return_value=consolidated
            ),
            mock.patch.object(
                steps,
                '_is_pypi_version_published',
                side_effect=lambda package, version: package in self.available,
            ),
            mock.patch.object(
                steps, '_wait_for_pypi_packages', side_effect=wait or available
            ),
            mock.patch.object(
                steps, 'watch_latest_workflow_run', side_effect=watch_error
            ) as watch,
        ):
            steps.step_publish_sdks(self.context)
        for call in watch.call_args_list:
            self.assertEqual(call.kwargs, {'expected_sha': SHA})
            self.assertEqual(
                call.args, (self.runner, 'publish-packages.yml', 'sdk-2.17.3')
            )
        self.assertEqual(watch.call_count, self.runner.run.call_count)

    def test_split_dependencies_are_public_before_dependent_dispatch(self):
        self.run_publication()
        self.assertEqual(
            self.events,
            [
                (event, package)
                for package in PACKAGES
                for event in ('dispatch', 'available')
            ],
        )

    def test_partial_release_publishes_only_missing_kubernetes(self):
        self.available.update(PACKAGES[:3])
        self.run_publication()
        self.assertEqual(
            self.events,
            [('dispatch', 'kfp-kubernetes'), ('available', 'kfp-kubernetes')],
        )

    def test_complete_release_dispatches_nothing(self):
        self.available.update(PACKAGES)
        self.run_publication()
        self.runner.run.assert_not_called()

    def test_unified_release_publishes_only_kfp(self):
        self.runner.run.side_effect = lambda command: self.events.append(
            (
                'dispatch',
                next(arg for arg in command if arg.startswith('packages=')),
            )
        )
        self.run_publication(consolidated=True)
        self.assertEqual(
            self.events, [('dispatch', 'packages=kfp'), ('available', 'kfp')]
        )

    def test_missing_dependency_stops_before_dependent_dispatch(self):
        with self.assertRaisesRegex(
            RuntimeError, 'PyPI packages are still missing'
        ):
            self.run_publication(wait=lambda version, packages: packages)
        self.assertEqual(self.events, [('dispatch', 'kfp-pipeline-spec')])

    def test_workflow_failure_stops_and_resume_skips_published_packages(self):
        with self.assertRaisesRegex(RuntimeError, 'workflow failed'):
            self.run_publication(watch_error=RuntimeError('workflow failed'))
        self.assertEqual(self.events, [('dispatch', 'kfp-pipeline-spec')])
        # An upload may finish even when a subsequent workflow step fails.
        self.available.add('kfp-pipeline-spec')
        self.events.clear()
        self.runner.run.reset_mock()
        self.run_publication()
        self.assertEqual(
            self.events,
            [
                (event, package)
                for package in PACKAGES[1:]
                for event in ('dispatch', 'available')
            ],
        )
