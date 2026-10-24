"""Release-note integration tests using real Git history."""

import contextlib
import io
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import core, steps


class ReleaseNotesIntegrationTest(unittest.TestCase):
    def setUp(self):
        for name, value in (
            ('qualification.require_qualification', None),
            ('_ensure_release_tag', None),
            ('_should_set_backend_latest', True),
            ('_sdk_release_is_consolidated', False),
        ):
            patcher = mock.patch('kfpr.steps.' + name, return_value=value)
            patcher.start()
            self.addCleanup(patcher.stop)
        source_root = Path(__file__).resolve().parents[2]
        identity = {}
        for key in ('user.name', 'user.email'):
            result = subprocess.run(
                ['git', 'config', '--get', key],
                cwd=source_root,
                capture_output=True,
                text=True,
                check=False,
            )
            if result.returncode or not result.stdout.strip():
                self.skipTest(f'Git {key} must be configured for history tests')
            identity[key] = result.stdout.strip()
        self.temporary = TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.git('init', '-q')
        for key, value in identity.items():
            self.git('config', key, value)
        sdk = self.root / 'sdk/python/kfp/version.py'
        sdk.parent.mkdir(parents=True)
        sdk.write_text('__version__ = "2.1.0"\n')
        self.git('add', 'sdk')
        self.commit('chore: initial commit')
        self.git('tag', '2.1.0')
        self.git('tag', 'sdk-2.1.0')
        self.commit(
            'chore(sdk): replace interface\n\nBREAKING CHANGE: migrate saved configs'
        )
        self.commit('feat(backend/sdk): support retries (#42)')
        self.commit('perf(api): improve response time')
        self.commit('chore(backend/security): rebuild vulnerable image')
        self.commit('docs: update contributor instructions')
        self.target = self.git('rev-parse', 'HEAD')
        self.git('tag', 'reviewed-target')
        self.commit('fix(sdk): unrelated future fix')
        self.git('tag', 'sdk-99.0.0')
        self.context = core.ReleaseContext(
            root=self.root,
            state=core.ReleaseState(path=self.root / 'release-state.json'),
            runner=core.CommandRunner(dry_run=True),
            metadata=core.ReleaseMetadata.from_version('patch', '2.1.1'),
            fork_remote='origin',
            include_backend=True,
            include_sdk=True,
            previous_release='2.1.0',
            previous_sdk_release='sdk-2.1.0',
            release_target='reviewed-target',
        )
        self.output = contextlib.redirect_stdout(io.StringIO())
        self.output.__enter__()
        self.addCleanup(self.output.__exit__, None, None, None)

    def git(self, *args):
        return subprocess.check_output(
            ['git', *args], cwd=self.root, text=True, stderr=subprocess.PIPE
        ).strip()

    def commit(self, message):
        self.git(
            '-c',
            'commit.gpgsign=false',
            'commit',
            '--allow-empty',
            '-q',
            '-s',
            '-m',
            message,
        )

    def test_generators_use_explicit_range_and_full_commit_bodies(self):
        sdk_range = steps._notes_range(self.context, 'sdk')
        backend_range = steps._notes_range(self.context, 'backend')
        sdk = steps._generate_sdk_release_notes(self.root, sdk_range)
        backend = steps._generate_backend_release_notes(
            self.root, backend_range
        )
        self.assertEqual(sdk_range.target_sha, self.target)
        self.assertIn('## Breaking changes', sdk)
        self.assertIn('replace interface', sdk)
        self.assertNotIn('replace interface', backend)
        self.assertIn('## Security fixes', backend)
        self.assertNotIn('rebuild vulnerable image', sdk)
        for notes in (sdk, backend):
            self.assertIn('support retries', notes)
            self.assertIn('/pull/42', notes)
            self.assertIn('## Performance', notes)
            self.assertNotIn('unrelated future fix', notes)
            self.assertNotIn('contributor instructions', notes)

    def test_publication_rejects_range_different_from_qualified_commit(self):
        self.context.state.answers['release_sha'] = self.git(
            'rev-parse', 'HEAD'
        )
        for product in ('sdk', 'backend'):
            with self.subTest(product=product):
                with self.assertRaisesRegex(
                    ValueError, 'qualified release commit'
                ):
                    steps._notes_range(self.context, product, publishing=True)
        self.context.state.answers['release_sha'] = self.target
        self.assertEqual(
            steps._notes_range(self.context, 'sdk', publishing=True).target_sha,
            self.target,
        )

    def test_preparation_rejects_target_behind_checked_out_commit(self):
        for product in ('sdk', 'backend'):
            with self.subTest(product=product):
                with self.assertRaisesRegex(
                    ValueError, 'target differs from the checked-out'
                ):
                    steps._preparation_range(self.context, product)

    def test_preparation_accepts_checked_out_target(self):
        self.git('checkout', '--detach', '-q', self.target)
        for product in ('sdk', 'backend'):
            with self.subTest(product=product):
                selected = steps._preparation_range(self.context, product)
                self.assertEqual(selected.target_sha, self.target)

    def test_changelog_and_curated_sdk_notes_share_classification(self):
        selected = steps._notes_range(self.context, 'sdk')
        sdk = steps._generate_sdk_release_notes(self.root, selected)
        path = self.root / 'SDK-RELEASE.md'
        path.write_text(
            '# Current Version (in development)\n\n'
            '## Upgrade actions\n\nBack up existing configuration.\n'
            '\n# 2.1.0\n\nPrevious release.\n'
        )
        steps._update_sdk_release_notes(path, '2.1.1', sdk)
        self.git('checkout', '--detach', '-q', self.target)
        self.context.runner.dry_run = False
        steps._update_changelog(self.context)
        changelog = (self.root / 'CHANGELOG.md').read_text()
        released = path.read_text().split('\n# 2.1.1\n', 1)[1]
        self.assertIn('Back up existing configuration.', released)
        for text in (released, changelog):
            self.assertIn('### Breaking changes', text)
            self.assertIn('### Performance', text)
            self.assertIn('support retries', text)
            self.assertNotIn('unrelated future fix', text)
            self.assertNotIn('contributor instructions', text)
        self.assertIn(selected.compare_url, changelog)

    def test_changed_target_rejects_preview_without_losing_edits(self):
        selected = steps._notes_range(self.context, 'sdk')
        path = steps._release_notes_preview(
            self.context, 'sdk-2.1.1', lambda: 'Generated notes', selected
        )
        path.write_text('Reviewed migration instructions.\n')
        self.context.release_target = 'HEAD'
        changed = steps._notes_range(self.context, 'sdk')
        with self.assertRaisesRegex(ValueError, 'Release range changed'):
            steps._release_notes_preview(
                self.context, 'sdk-2.1.1', lambda: 'Replacement notes', changed
            )
        self.assertEqual(path.read_text(), 'Reviewed migration instructions.\n')

    def test_missing_provenance_rejects_existing_notes(self):
        selected = steps._notes_range(self.context, 'sdk')
        path = steps._release_notes_preview(
            self.context, 'sdk-2.1.1', lambda: 'Reviewed notes', selected
        )
        path.with_suffix('.json').unlink()
        with self.assertRaisesRegex(ValueError, 'provenance is missing'):
            steps._release_notes_preview(
                self.context, 'sdk-2.1.1', lambda: 'Replacement notes', selected
            )
        self.assertEqual(path.read_text(), 'Reviewed notes\n')

    def test_publication_uses_resolved_target_and_reviewed_file(self):
        for product, remote_sha, publish in (
            ('sdk', self.target, steps.step_create_sdk_release),
            ('backend', self.target, steps.step_create_backend_release),
            ('backend', '', steps.step_create_backend_release),
        ):
            with self.subTest(product=product, remote_sha=remote_sha):
                self.context.runner.dry_run = True
                publish(self.context)
                tag = 'sdk-2.1.1' if product == 'sdk' else '2.1.1'
                path = self.root / 'release-state-notes' / f'{tag}.md'
                path.write_text('Reviewed user-facing changes.\n')
                self.context.runner.dry_run = False
                with (
                    mock.patch.object(
                        self.context.runner,
                        'run',
                        return_value=subprocess.CompletedProcess([], 1),
                    ) as run,
                    mock.patch.object(steps, 'confirm'),
                    mock.patch.object(
                        steps, '_remote_release_sha', return_value=remote_sha
                    ),
                ):
                    publish(self.context)
                create = next(
                    call.args[0]
                    for call in run.call_args_list
                    if call.args[0][:3] == ['gh', 'release', 'create']
                )
                self.assertEqual(
                    create[create.index('--target') + 1], self.target
                )
                self.assertEqual(
                    Path(create[create.index('--notes-file') + 1]).resolve(),
                    path.resolve(),
                )
                self.assertEqual(
                    path.read_text(), 'Reviewed user-facing changes.\n'
                )

    def test_remote_tag_mismatch_or_missing_sdk_blocks_publication(self):
        for product, remote_sha, publish in (
            ('sdk', 'f' * 40, steps.step_create_sdk_release),
            ('sdk', '', steps.step_create_sdk_release),
            ('backend', 'f' * 40, steps.step_create_backend_release),
        ):
            with self.subTest(product=product, remote_sha=remote_sha):
                self.context.runner.dry_run = True
                publish(self.context)
                tag = 'sdk-2.1.1' if product == 'sdk' else '2.1.1'
                path = self.root / 'release-state-notes' / f'{tag}.md'
                reviewed = 'Reviewed migration instructions.\n'
                path.write_text(reviewed)
                self.context.runner.dry_run = False
                with (
                    mock.patch.object(
                        self.context.runner,
                        'run',
                        return_value=subprocess.CompletedProcess([], 0),
                    ) as run,
                    mock.patch.object(steps, 'confirm'),
                    mock.patch.object(steps, 'prompt_choice', return_value='r'),
                    mock.patch.object(
                        steps, '_remote_release_sha', return_value=remote_sha
                    ),
                ):
                    with self.assertRaises(ValueError):
                        publish(self.context)
                mutations = [
                    call.args[0]
                    for call in run.call_args_list
                    if call.args[0][:3]
                    in (
                        ['gh', 'release', 'create'],
                        ['gh', 'release', 'delete'],
                    )
                ]
                self.assertEqual(mutations, [])
                self.assertEqual(path.read_text(), reviewed)


class RemoteReleaseTagTest(unittest.TestCase):
    def test_annotated_tag_uses_peeled_commit(self):
        ref = 'refs/tags/sdk-2.1.1'
        tag_object = 'a' * 40
        commit = 'b' * 40
        output = f'{tag_object}\t{ref}\n{commit}\t{ref}^{{}}\n'
        with mock.patch.object(subprocess, 'check_output', return_value=output):
            self.assertEqual(
                steps._remote_release_sha(Path('/unused'), 'sdk-2.1.1'), commit
            )

    def test_lightweight_tag_uses_direct_commit(self):
        commit = 'b' * 40
        with mock.patch.object(
            subprocess,
            'check_output',
            return_value=f'{commit}\trefs/tags/2.1.1\n',
        ):
            self.assertEqual(
                steps._remote_release_sha(Path('/unused'), '2.1.1'), commit
            )

    def test_absent_tag_returns_empty_commit(self):
        with mock.patch.object(subprocess, 'check_output', return_value=''):
            self.assertEqual(
                steps._remote_release_sha(Path('/unused'), '2.1.1'), ''
            )


if __name__ == '__main__':
    unittest.main()
