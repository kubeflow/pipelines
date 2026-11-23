"""Review the rendered, editable release pages together before publishing."""

import contextlib
import io
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from typer.testing import CliRunner

from kfpr import joint_review
from kfpr.advisory_coverage import ReleaseAdvisory
from kfpr.cli import app
from kfpr.context import ReleaseContext
from kfpr.metadata import ReleaseMetadata
from kfpr.runner import CommandRunner
from kfpr.state import ReleaseState


class JointReviewTest(unittest.TestCase):
    def setUp(self):
        root = Path(self.enterContext(tempfile.TemporaryDirectory()))
        self.context = ReleaseContext(
            root,
            ReleaseState(root / 'state.json'),
            CommandRunner(),
            ReleaseMetadata.from_version('patch', '2.18.1'),
            '',
            True,
            True,
        )
        self.paths = {name: root / f'{name}.md' for name in ('backend', 'sdk')}
        for name, path in self.paths.items():
            path.write_text(f'## Highlights\n\n{name} **feature**\n')
        self.fetch = self.enterContext(
            mock.patch.object(
                joint_review,
                'fetch_release_advisories',
                return_value=[],
            )
        )

    def review(self, width):
        output = io.StringIO()
        with (
            mock.patch.dict('os.environ', {'COLUMNS': str(width)}),
            contextlib.redirect_stdout(output),
        ):
            issues = joint_review.show_review(self.context, self.paths)
        return issues, output.getvalue()

    def test_wide_terminal_renders_both_pages_side_by_side_and_counts_coverage(
        self,
    ):
        issues, output = self.review(160)
        self.assertEqual(issues, [])
        self.assertTrue(
            any(
                'backend feature' in line and 'sdk feature' in line
                for line in output.splitlines()
            ),
            output,
        )
        self.assertNotIn('**feature**', output)
        self.assertIn('0 published advisories', output)
        self.fetch.assert_called_once_with(self.context.runner, '2.18.1')

    def test_narrow_terminal_stacks_pages_and_reloads_edits(self):
        issues, output = self.review(70)
        self.assertEqual(issues, [])
        self.assertLess(
            output.index('backend feature'), output.index('sdk feature')
        )
        self.assertFalse(
            any(
                'backend feature' in line and 'sdk feature' in line
                for line in output.splitlines()
            )
        )
        updated = '## Highlights\n\n<!-- TODO: finish -->\n\n```\nTODO\n```\n'
        self.paths['sdk'].write_text(updated)
        issues, _ = self.review(70)
        self.assertEqual(len(issues), 1, issues)
        self.assertIn('sdk', issues[0].lower())
        self.assertIn('placeholder', issues[0])
        self.assertEqual(self.paths['sdk'].read_text(), updated)
        self.assertEqual(self.fetch.call_count, 2)

    def test_publication_blockers_distinct_from_editorial_suggestions(self):
        self.paths['sdk'].write_text('  \n')
        self.paths['backend'].write_text('<<<<<<< HEAD\nConflict here\n')
        issues, _ = self.review(70)
        self.assertEqual(len(issues), 2, issues)
        self.assertIn('conflict', ' '.join(issues).lower())
        self.assertIn('empty', ' '.join(issues).lower())

    def test_read_published_sibling_on_resume_and_check_current_product_locally(
        self,
    ):
        advisory = ReleaseAdvisory(
            'GHSA-2222-3333-4444', 'SDK fix', frozenset({'sdk'})
        )
        self.fetch.return_value = [advisory]
        # The local SDK was corrected after publication. Its public page still misses the link.
        for path in self.paths.values():
            path.write_text(f'[Patched SDK vulnerability]({advisory.url})')
        with mock.patch.object(
            self.context.runner,
            'capture',
            return_value='Previously published SDK notes',
        ) as capture:
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                issues = joint_review.show_review(
                    self.context,
                    {**self.paths, 'sdk': None},
                    display=False,
                )
        self.assertEqual(len(issues), 1, issues)
        self.assertIn(advisory.url, issues[0])
        self.assertIn('releases/tag/sdk-2.18.1', issues[0])
        self.assertNotIn('Backend release notes', output.getvalue())
        self.assertIn('sdk-2.18.1', capture.call_args.args[0])
        self.assertEqual(capture.call_count, 1)

    def test_sdk_only_summary_includes_applicable_advisories(self):
        del self.paths['backend']
        sdk = ReleaseAdvisory(
            'GHSA-2222-3333-4444', 'SDK fix', frozenset({'sdk'})
        )
        backend = ReleaseAdvisory(
            'GHSA-2222-3333-5555', 'Backend fix', frozenset({'backend'})
        )
        self.fetch.return_value = [sdk, backend]
        issues, output = self.review(70)
        self.assertIn('1 published advisor', output)
        self.assertIn('0 backend, 1 SDK', output)
        self.assertIn(sdk.summary, output)
        self.assertIn(sdk.url, issues[0])
        self.assertNotIn(backend.url, output)

    def test_review_command_preserves_checkpoint_and_signals_unfinished_notes(
        self,
    ):
        checkpoint = self.context.state.path
        original = json.dumps(
            {
                'answers': {
                    'release_type': 'patch',
                    'version': '2.18.1',
                    'fork_remote': 'https://github.com/example/pipelines',
                    'patch_prs': 'not-needed',
                    'previous_release': '2.18.0',
                    'previous_sdk_release': 'sdk-2.18.0',
                    'include_backend': True,
                    'include_sdk': True,
                }
            }
        )
        checkpoint.write_text(original)
        with mock.patch(
            'kfpr.steps.review_release_notes', return_value=['Missing SDK link']
        ) as review:
            result = CliRunner().invoke(
                app, ['review-notes', '--state-file', str(checkpoint)]
            )
        self.assertEqual(result.exit_code, 1, (result.output, result.exception))
        self.assertIn('Missing SDK link', result.output)
        context = review.call_args.args[0]
        self.assertTrue(context.include_backend and context.include_sdk)
        self.assertFalse(context.runner.dry_run)
        self.assertEqual(checkpoint.read_text(), original)


if __name__ == '__main__':
    unittest.main()
