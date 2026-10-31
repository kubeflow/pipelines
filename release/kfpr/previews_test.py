"""Regression tests for editable release preview regeneration."""

import json
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import previews
from kfpr.ranges import ReleaseRange


class RegeneratePreviewTest(unittest.TestCase):
    def setUp(self):
        directory = TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        self.path = self.root / '3.0.0.md'
        self.sidecar = self.path.with_suffix('.json')
        self.old_range = ReleaseRange('2.0.0', 'old-target', 'a' * 40, 'b' * 40)
        self.new_range = ReleaseRange('2.0.0', 'new-target', 'a' * 40, 'c' * 40)
        self.baseline = (
            '# Release notes\n\nIntro.\n\n## Changes\n\n* Old feature.\n'
        )
        self.write_preview(self.baseline, self.baseline)

    def write_preview(self, current, baseline):
        self.path.write_text(current)
        self.sidecar.write_text(
            json.dumps(self.old_range.as_dict() | {'generated_notes': baseline})
        )

    def assert_unchanged(self, operation):
        original = self.path.read_bytes()
        provenance = self.sidecar.read_bytes()
        with self.assertRaises(ValueError):
            operation()
        self.assertEqual(self.path.read_bytes(), original)
        self.assertEqual(self.sidecar.read_bytes(), provenance)
        self.assertEqual(list(self.root.glob('*.backup-*')), [])

    def test_independent_edits_merge_with_exact_original_backups(self):
        current = self.baseline.replace(
            'Intro.', 'Handwritten upgrade guidance.'
        )
        self.write_preview(current, self.baseline)
        original = self.path.read_bytes()
        provenance = self.sidecar.read_bytes()
        generated = self.baseline.replace('Old feature.', 'New feature.')
        self.assertTrue(
            previews.regenerate_preview(self.path, self.new_range, generated)
        )
        self.assertEqual(
            self.path.read_text(),
            generated.replace('Intro.', 'Handwritten upgrade guidance.'),
        )
        self.assertEqual(
            json.loads(self.sidecar.read_text()),
            self.new_range.as_dict() | {'generated_notes': generated},
        )
        [backup] = self.root.glob('*.backup-*')
        self.assertEqual((backup / 'original.md').read_bytes(), original)
        self.assertEqual((backup / 'provenance.json').read_bytes(), provenance)

    def test_overlap_retains_both_versions_and_new_baseline(self):
        self.write_preview(
            self.baseline.replace('Old feature.', 'Reviewed feature.'),
            self.baseline,
        )
        generated = self.baseline.replace('Old feature.', 'New feature.')
        self.assertFalse(
            previews.regenerate_preview(self.path, self.new_range, generated)
        )
        merged = self.path.read_text()
        self.assertTrue(previews.has_conflicts(merged))
        self.assertIn('Reviewed feature.', merged)
        self.assertIn('New feature.', merged)
        self.assertIn('<<<<<<< reviewed notes', merged)
        self.assertIn('>>>>>>> new generated notes', merged)
        self.assertEqual(
            json.loads(self.sidecar.read_text())['generated_notes'], generated
        )
        snapshot = (self.path.read_bytes(), self.sidecar.read_bytes())
        with self.assertRaisesRegex(
            ValueError, 'Resolve existing conflict markers'
        ):
            previews.regenerate_preview(self.path, self.new_range, generated)
        self.assertEqual(
            (self.path.read_bytes(), self.sidecar.read_bytes()), snapshot
        )
        self.assertEqual(len(list(self.root.glob('*.backup-*'))), 1)

    def test_repeating_clean_regeneration_preserves_edits_and_unique_backups(
        self,
    ):
        self.write_preview(
            self.baseline.replace('Intro.', 'Reviewed intro.'), self.baseline
        )
        generated = self.baseline.replace('Old feature.', 'New feature.')
        self.assertTrue(
            previews.regenerate_preview(self.path, self.new_range, generated)
        )
        first = (self.path.read_bytes(), self.sidecar.read_bytes())
        self.assertTrue(
            previews.regenerate_preview(self.path, self.new_range, generated)
        )
        self.assertEqual(
            (self.path.read_bytes(), self.sidecar.read_bytes()), first
        )
        self.assertEqual(len(list(self.root.glob('*.backup-*'))), 2)

    def test_resolved_conflict_can_be_regenerated_again(self):
        self.write_preview(
            self.baseline.replace('Old feature.', 'Reviewed feature.'),
            self.baseline,
        )
        generated = self.baseline.replace('Old feature.', 'New feature.')
        self.assertFalse(
            previews.regenerate_preview(self.path, self.new_range, generated)
        )
        resolved = generated.replace('New feature.', 'Reviewed new feature.')
        self.path.write_text(resolved)
        self.assertTrue(
            previews.regenerate_preview(self.path, self.new_range, generated)
        )
        self.assertEqual(self.path.read_text(), resolved)

    def test_invalid_or_legacy_provenance_leaves_files_untouched(self):
        for provenance in (
            'not json',
            '[]',
            json.dumps(self.old_range.as_dict()),
            json.dumps(self.old_range.as_dict() | {'generated_notes': 7}),
            json.dumps(
                self.old_range.as_dict()
                | {'generated_notes': self.baseline, 'target_sha': 'invalid'}
            ),
        ):
            with self.subTest(provenance=provenance):
                self.sidecar.write_text(provenance)
                self.assert_unchanged(
                    lambda: previews.regenerate_preview(
                        self.path, self.new_range, self.baseline
                    )
                )

    def test_missing_preview_or_sidecar_does_not_create_files(self):
        self.sidecar.unlink()
        with self.assertRaisesRegex(ValueError, 'Cannot read'):
            previews.regenerate_preview(
                self.path, self.new_range, self.baseline
            )
        self.assertEqual(self.path.read_text(), self.baseline)
        self.assertFalse(self.sidecar.exists())
        self.path.unlink()
        with self.assertRaisesRegex(ValueError, 'Cannot read'):
            previews.regenerate_preview(
                self.path, self.new_range, self.baseline
            )
        self.assertEqual(list(self.root.iterdir()), [])

    def test_empty_notes_and_existing_conflicts_leave_files_untouched(self):
        for current, generated in (
            ('   \n', self.baseline),
            (self.baseline, '  '),
            ('<<<<<<< reviewed notes\n', self.baseline),
            (self.baseline, '>>>>>>> generated notes\n'),
        ):
            with self.subTest(current=current, generated=generated):
                self.write_preview(current, self.baseline)
                self.assert_unchanged(
                    lambda: previews.regenerate_preview(
                        self.path, self.new_range, generated
                    )
                )

    def test_merge_process_failure_leaves_files_untouched(self):
        for failure in (
            OSError('git unavailable'),
            subprocess.CompletedProcess([], 255, b'', b'failed'),
        ):
            with self.subTest(failure=failure):
                options = (
                    {'side_effect': failure}
                    if isinstance(failure, OSError)
                    else {'return_value': failure}
                )
                with mock.patch.object(previews.subprocess, 'run', **options):
                    self.assert_unchanged(
                        lambda: previews.regenerate_preview(
                            self.path, self.new_range, self.baseline
                        )
                    )

    def test_concurrent_edit_is_not_overwritten(self):
        original_run = subprocess.run

        def merge_and_edit(*args, **kwargs):
            result = original_run(*args, **kwargs)
            self.path.write_text('New handwritten edit.\n')
            return result

        provenance = self.sidecar.read_bytes()
        with mock.patch.object(
            previews.subprocess, 'run', side_effect=merge_and_edit
        ):
            with self.assertRaisesRegex(
                ValueError, 'changed during regeneration'
            ):
                previews.regenerate_preview(
                    self.path, self.new_range, self.baseline
                )
        self.assertEqual(self.path.read_text(), 'New handwritten edit.\n')
        self.assertEqual(self.sidecar.read_bytes(), provenance)
        self.assertEqual(list(self.root.glob('*.backup-*')), [])

    def test_sidecar_save_failure_restores_both_original_files(self):
        original = self.path.read_bytes()
        provenance = self.sidecar.read_bytes()
        original_replace = previews.os.replace
        failed_once = False

        def fail_sidecar_save(source, destination):
            nonlocal failed_once
            if destination == self.sidecar and not failed_once:
                failed_once = True
                raise OSError('simulated disk error')
            return original_replace(source, destination)

        generated = self.baseline.replace('Old feature.', 'New feature.')
        with mock.patch.object(
            previews.os, 'replace', side_effect=fail_sidecar_save
        ):
            with self.assertRaisesRegex(ValueError, 'Original files restored'):
                previews.regenerate_preview(
                    self.path, self.new_range, generated
                )
        self.assertTrue(failed_once)
        self.assertEqual(self.path.read_bytes(), original)
        self.assertEqual(self.sidecar.read_bytes(), provenance)
        [backup] = self.root.glob('*.backup-*')
        self.assertEqual((backup / 'original.md').read_bytes(), original)
        self.assertEqual((backup / 'provenance.json').read_bytes(), provenance)
        self.assertEqual(len(list(self.root.iterdir())), 3)

    def test_failed_rollback_reports_recovery_from_backups(self):
        original = self.path.read_bytes()
        provenance = self.sidecar.read_bytes()
        with mock.patch.object(
            previews.os, 'replace', side_effect=OSError('disk error')
        ):
            with self.assertRaisesRegex(ValueError, 'Restore originals from'):
                previews.regenerate_preview(
                    self.path, self.new_range, self.baseline
                )
        [backup] = self.root.glob('*.backup-*')
        self.assertEqual((backup / 'original.md').read_bytes(), original)
        self.assertEqual((backup / 'provenance.json').read_bytes(), provenance)

    def test_multiple_conflicts_are_saved_for_manual_resolution(self):
        baseline = 'first\n' + 'unchanged\n' * 20 + 'last\n'
        self.write_preview(
            baseline.replace('first', 'reviewed first').replace(
                'last', 'reviewed last'
            ),
            baseline,
        )
        generated = baseline.replace('first', 'new first').replace(
            'last', 'new last'
        )
        self.assertFalse(
            previews.regenerate_preview(self.path, self.new_range, generated)
        )
        self.assertEqual(
            self.path.read_text().count('<<<<<<< reviewed notes'), 2
        )

    def test_generated_baseline_matches_normalized_preview(self):
        generated = self.baseline.rstrip() + '\n\n '
        self.assertTrue(
            previews.regenerate_preview(self.path, self.new_range, generated)
        )
        self.assertEqual(self.path.read_text(), generated.rstrip() + '\n')
        self.assertEqual(
            json.loads(self.sidecar.read_text())['generated_notes'],
            self.path.read_text(),
        )


class ConflictMarkersTest(unittest.TestCase):
    def test_detects_open_close_and_ancestor_markers(self):
        for text in (
            '<<<<<<< reviewed\n',
            '>>>>>>> generated\n',
            '||||||| ancestor\n',
            '<<<<<<<< ours\n',
        ):
            self.assertTrue(previews.has_conflicts(text))

    def test_setext_markdown_is_not_a_conflict(self):
        self.assertFalse(
            previews.has_conflicts(
                'Release notes\n=======\n\nText with <<<<<<< inline.\n'
            )
        )


if __name__ == '__main__':
    unittest.main()
