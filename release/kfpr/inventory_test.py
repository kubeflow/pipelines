"""Editorial inventory tests against an exact, real Git release range."""

import contextlib
import copy
import json
import re
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from typer.testing import CliRunner

from kfpr import inventory
from kfpr.cli import app
from kfpr.generated import restore_generated_sections
from kfpr.ranges import resolve_range


class InventoryTest(unittest.TestCase):
    def setUp(self):
        identity = {}
        for key in ('user.name', 'user.email'):
            result = subprocess.run(
                ['git', 'config', '--get', key],
                cwd=Path(__file__).resolve().parents[2],
                capture_output=True,
                text=True,
                check=False,
            )
            if result.returncode or not result.stdout.strip():
                self.skipTest(f'Git {key} must be configured for history tests')
            identity[key] = result.stdout.strip()
        temporary = TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.git('init', '-q')
        for key, value in identity.items():
            self.git('config', key, value)
        self.commit('Initial release')
        self.git('tag', '2.17.3')
        self.plain = self.commit('Restore support for existing installations')
        self.squash = self.commit(
            'Backport reviewed release changes\n\n'
            '* feat(sdk): support embedded artifacts (#101)\n'
            '- fix(controller): recover recurring runs (#102)\n'
            '```text\nfix(sdk): example only\n```\n'
            '~~~\nfeat(api): another example\n~~~\n'
            'fix(api): preserve existing access (#103)'
        )
        self.git('tag', '2.18.0')
        self.later = self.commit('feat(sdk): master-only future feature')
        self.selected = resolve_range(self.root, '2.17.3', '2.18.0')
        self.data = inventory.build_inventory(
            self.root, self.selected, '2.18.0'
        )

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
        return self.git('rev-parse', 'HEAD')

    @staticmethod
    def note(text, **extra):
        return {
            'section': 'compatibility',
            'products': list(inventory.PRODUCTS),
            'text': text,
            **extra,
        }

    def test_nested_or_crossed_generated_markers_cannot_swallow_editorial_prose(
        self,
    ):
        baseline = inventory.release_scaffolding(self.data, 'backend')
        for malformed in (
            '<!-- kfpr:generated:credits:start --><!-- kfpr:generated:assets:start -->'
            '<!-- kfpr:generated:assets:end -->Important operator warning<!-- kfpr:generated:credits:end -->',
            '<!-- kfpr:generated:credits:start --><!-- kfpr:generated:assets:start -->'
            '<!-- kfpr:generated:credits:end --><!-- kfpr:generated:assets:end -->',
        ):
            with self.subTest(markers=malformed):
                with self.assertRaisesRegex(ValueError, 'Malformed'):
                    restore_generated_sections(malformed, baseline)

    def test_invalid_fact_types_produce_actionable_validation_errors(self):
        for fact in (
            self.note('Bad section.', section=[]),
            self.note('Bad evidence.', commit=[]),
        ):
            data = copy.deepcopy(self.data)
            data['facts'] = [fact]
            with self.subTest(fact=fact), self.assertRaises(ValueError):
                inventory.validate_inventory(data)

    def test_only_actual_coauthor_trailers_are_credited(self):
        self.commit(
            'docs: explain attribution\n\n```text\nCo-authored-by: Fenced Example <fenced@example.org>\n```\n\n'
            'Co-authored-by: Midbody Example <middle@example.org>\n\nThis is explanatory prose.\n\n'
            'Co-authored-by: Actual Collaborator <actual@example.org>'
        )
        selected = resolve_range(self.root, '2.18.0', 'HEAD')
        data = inventory.build_inventory(self.root, selected, '2.18.1')
        self.assertEqual(
            data['changes'][-1]['coauthors'], ['Actual Collaborator']
        )
        text = inventory.release_scaffolding(data, 'backend')
        self.assertIn('Actual Collaborator', text)
        self.assertNotIn('Fenced Example', text)
        self.assertNotIn('Midbody Example', text)

    def test_cli_bundle_and_render_preserve_review_and_reject_overwrite(self):
        initial = self.root / 'editorial'
        rendered = self.root / 'reviewed'
        runner = CliRunner()
        with contextlib.chdir(self.root):
            result = runner.invoke(
                app,
                [
                    'prepare-editorial',
                    '--version',
                    '2.18.0',
                    '--base',
                    '2.17.3',
                    '--target',
                    '2.18.0',
                    '--output',
                    str(initial),
                ],
            )
            self.assertEqual(result.exit_code, 0, result.output)
            path = initial / 'inventory.json'
            data = json.loads(path.read_text())
            data['facts'].append(
                self.note(
                    'Python 3.9 remains supported.',
                    commit=self.selected.target_sha,
                )
            )
            data['changes'][0]['decision'] = 'include'
            data['changes'][0]['notes'] = [
                self.note(
                    'Preserve supported imports.',
                    section='bug_fixes',
                    products=['sdk'],
                )
            ]
            path.write_text(json.dumps(data))
            result = runner.invoke(
                app, ['render-editorial', str(path), '--output', str(rendered)]
            )
            self.assertEqual(result.exit_code, 0, result.output)
            for product in inventory.PRODUCTS:
                self.assertIn(
                    'Python 3.9 remains supported.',
                    (rendered / f'{product}.md').read_text(),
                )
            self.assertIn(
                'Preserve supported imports.', (rendered / 'sdk.md').read_text()
            )
            self.assertNotIn(
                'Preserve supported imports.',
                (rendered / 'backend.md').read_text(),
            )
            before = (rendered / 'sdk.md').read_bytes()
            result = runner.invoke(
                app, ['render-editorial', str(path), '--output', str(rendered)]
            )
            self.assertNotEqual(result.exit_code, 0)
            self.assertEqual((rendered / 'sdk.md').read_bytes(), before)
            self.assertFalse((self.root / '.kfpr-state.json').exists())
            bad = runner.invoke(
                app,
                [
                    'validate-notes',
                    str(rendered / 'sdk.md'),
                    '--product',
                    'invalid',
                ],
            )
            self.assertNotEqual(bad.exit_code, 0)

    def test_full_range_retains_nonconventional_and_nested_candidates(self):
        changes = self.data['changes']
        self.assertEqual(
            [change['sha'] for change in changes], [self.plain, self.squash]
        )
        self.assertEqual(
            changes[0]['subject'], 'Restore support for existing installations'
        )
        self.assertEqual(
            changes[1]['nested_subjects'],
            [
                {
                    'line': 3,
                    'subject': 'feat(sdk): support embedded artifacts (#101)',
                },
                {
                    'line': 4,
                    'subject': 'fix(controller): recover recurring runs (#102)',
                },
                {
                    'line': 11,
                    'subject': 'fix(api): preserve existing access (#103)',
                },
            ],
        )
        self.assertIn('example only', changes[1]['message'])
        self.assertNotIn(self.later, json.dumps(self.data))
        inventory.verify_evidence(self.root, self.data)

    def test_forged_or_missing_evidence_is_rejected(self):
        for field in ('sha', 'author', 'subject', 'message', 'nested_subjects'):
            with self.subTest(field=field):
                changed = copy.deepcopy(self.data)
                changed['changes'][0][field] = (
                    self.later
                    if field == 'sha'
                    else ([] if field == 'nested_subjects' else 'forged')
                )
                if field == 'nested_subjects':
                    changed['changes'][0][field] = [
                        {'line': 2, 'subject': 'feat: forged'}
                    ]
                with self.assertRaisesRegex(
                    ValueError, 'evidence differs.*fresh inventory'
                ):
                    inventory.verify_evidence(self.root, changed)
        self.data['changes'].pop()
        with self.assertRaisesRegex(ValueError, 'evidence differs'):
            inventory.verify_evidence(self.root, self.data)

    def test_reviewed_facts_are_shared_and_pending_notes_are_not_emitted(self):
        fact = 'Python 3.9 through 3.13 are supported.'
        self.data['facts'] = [self.note(fact, commit=self.squash)]
        self.data['changes'][0]['notes'] = [
            self.note('Unreviewed assertion must stay private.')
        ]
        self.data['changes'][1].update(
            decision='include', notes=[self.note('Reviewed release benefit.')]
        )
        for product in inventory.PRODUCTS:
            with self.subTest(product=product):
                text = inventory.render_document(self.data, product)
                self.assertEqual(text.count(fact), 1)
                self.assertNotIn('Unreviewed assertion', text)
                self.assertIn('Reviewed release benefit.', text)
                self.assertIn('/commit/' + self.squash, text)
        self.assertTrue(
            any(
                '1 commits' in warning
                for warning in inventory.review_warnings(self.data)
            )
        )

    def test_second_write_preserves_entire_previous_review_bundle(self):
        output = self.root / 'review'
        inventory.write_bundle(self.root, self.data, output)
        (output / 'backend.md').write_text('Human editorial work\n')
        before = {path.name: path.read_bytes() for path in output.iterdir()}
        with self.assertRaises(FileExistsError):
            inventory.write_bundle(self.root, self.data, output)
        self.assertEqual(
            before, {path.name: path.read_bytes() for path in output.iterdir()}
        )
        self.assertEqual(json.loads(before['inventory.json']), self.data)

    def test_assets_and_credits_are_grounded_in_tag_and_commit_evidence(self):
        for product, tag in (('backend', '2.18.0'), ('sdk', 'sdk-2.18.0')):
            with self.subTest(product=product):
                text = inventory.release_scaffolding(self.data, product)
                self.assertIn(f'/archive/refs/tags/{tag}.zip', text)
                self.assertIn(f'/releases/tag/{tag}', text)
                self.assertIn('/commit/' + self.plain, text)
                self.assertNotIn('pypi.org', text)
                self.assertNotIn('/releases/download/', text)
                self.assertNotRegex(
                    text, r'https://github.com/(?!kubeflow/pipelines/)'
                )
                self.assertEqual(
                    text.count('](' + inventory.REPOSITORY + '/commit/'), 1
                )

    def test_malformed_inventory_has_actionable_errors(self):
        cases = [
            ('schema_version', 2, 'generate an inventory'),
            ('version', 'latest', 'release version'),
            ('range', {}, 'resolved release range'),
            ('changes', {}, 'changes must be a list'),
            ('facts', None, 'must be lists'),
            (
                'facts',
                [self.note('Unsupported claim.', commit=self.later)],
                'commit from this inventory',
            ),
            ('coverage', {}, 'coverage checklist'),
        ]
        for field, value, message in cases:
            with self.subTest(field=field, value=value):
                changed = copy.deepcopy(self.data)
                changed[field] = value
                with self.assertRaisesRegex(ValueError, message):
                    inventory.validate_inventory(changed)
        for update, message in [
            ({'decision': 'omit'}, 'Explain why'),
            ({'decision': 'include'}, 'at least one reviewed note'),
            ({'decision': 'approved'}, 'Choose pending, include, or omit'),
            (
                {'notes': [self.note('## Injected heading')]},
                'without section headings',
            ),
            (
                {'notes': [self.note('Claim', products=['unknown'])]},
                'backend, sdk, or upgrade',
            ),
        ]:
            with self.subTest(update=update):
                changed = copy.deepcopy(self.data)
                changed['changes'][0].update(update)
                with self.assertRaisesRegex(ValueError, message):
                    inventory.validate_inventory(changed)

    def test_deleted_or_edited_generated_sections_are_restored(self):
        baseline = inventory.release_scaffolding(self.data, 'sdk')
        prose = '## Highlights\n\nHuman-written benefit.\n\n'
        for name in ('credits', 'assets'):
            pattern = rf'<!-- kfpr:generated:{name}:start -->.*?<!-- kfpr:generated:{name}:end -->'
            for replacement in (
                '',
                f'<!-- kfpr:generated:{name}:start -->\nInvented download or contributor\n<!-- kfpr:generated:{name}:end -->',
            ):
                with self.subTest(name=name, replacement=replacement):
                    edited = prose + re.sub(
                        pattern,
                        lambda _: replacement,
                        baseline,
                        flags=re.DOTALL,
                    )
                    restored = restore_generated_sections(edited, baseline)
                    self.assertTrue(restored.startswith(prose))
                    self.assertNotIn('Invented download', restored)
                    for section in ('credits', 'assets'):
                        section_pattern = rf'<!-- kfpr:generated:{section}:start -->.*?<!-- kfpr:generated:{section}:end -->'
                        self.assertEqual(
                            re.findall(section_pattern, restored, re.DOTALL),
                            re.findall(section_pattern, baseline, re.DOTALL),
                        )

    def test_malformed_edited_markers_are_rejected_and_legacy_prose_is_preserved(
        self,
    ):
        baseline = inventory.release_scaffolding(self.data, 'backend')
        for name in ('credits', 'assets'):
            start = f'<!-- kfpr:generated:{name}:start -->'
            end = f'<!-- kfpr:generated:{name}:end -->'
            for edited in (
                baseline.replace(end, ''),
                baseline + start,
                end + start,
            ):
                with self.subTest(name=name, edited=edited):
                    with self.assertRaisesRegex(ValueError, '(?i)malformed'):
                        restore_generated_sections(edited, baseline)
        self.assertEqual(
            restore_generated_sections(
                'Legacy editorial prose', 'Old unmarked baseline'
            ),
            'Legacy editorial prose',
        )

    def test_malformed_generated_baseline_is_rejected(self):
        baseline = inventory.release_scaffolding(self.data, 'backend')
        for name in ('credits', 'assets'):
            with self.subTest(name=name):
                malformed = baseline.replace(
                    f'<!-- kfpr:generated:{name}:end -->', ''
                )
                with self.assertRaisesRegex(ValueError, '(?i)malformed'):
                    restore_generated_sections(
                        'Human editorial prose', malformed
                    )


if __name__ == '__main__':
    unittest.main()
