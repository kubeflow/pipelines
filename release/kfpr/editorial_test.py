"""Tests for advisory release editorial guidance."""

import unittest

from kfpr.editorial import editorial_scaffold, validate_notes


class EditorialScaffoldTest(unittest.TestCase):
    def test_prompts_cover_editorial_decisions_without_release_claims(self):
        scaffold = editorial_scaffold()
        for heading in (
            'Highlights',
            'Action required',
            'Compatibility',
            'Known limitations',
        ):
            self.assertIn('## ' + heading, scaffold)
        self.assertIn('before/after', scaffold)
        self.assertIn('removal versions', scaffold)
        self.assertIn(
            'tested combinations from supported combinations', scaffold
        )
        self.assertGreaterEqual(scaffold.count('<!-- TODO:'), 4)
        self.assertTrue(scaffold.endswith('\n\n'))
        self.assertIn('unevaluated combinations', scaffold)
        warnings = validate_notes(scaffold)
        self.assertEqual(
            sum('placeholder' in warning for warning in warnings),
            scaffold.count('<!-- TODO:'),
        )
        self.assertEqual(
            sum('is empty' in warning for warning in warnings),
            scaffold.count('## '),
        )

    def test_product_templates_cover_distinct_editorial_decisions(self):
        backend = editorial_scaffold('backend')
        sdk = editorial_scaffold('sdk')
        upgrade = editorial_scaffold('upgrade')
        self.assertIn('## Bug fixes', backend)
        self.assertIn('## Security improvements', backend)
        self.assertIn('## Availability and resilience', backend)
        self.assertIn('## Installation', sdk)
        self.assertIn('target packaging layout', sdk)
        for heading in (
            'Prepare',
            'Roll out',
            'Verify',
            'Custom configurations',
        ):
            self.assertIn('## ' + heading, upgrade)
        self.assertIn('custom overlays', upgrade)
        self.assertIn(
            'only when the released implementation requires it', upgrade
        )
        self.assertNotIn('3.0', backend + sdk + upgrade)
        with self.assertRaises(ValueError):
            editorial_scaffold('invalid')


class ValidateNotesTest(unittest.TestCase):
    def test_explicit_placeholders_are_advisory(self):
        warnings = validate_notes(
            '## Highlights\n\nTODO: finish summary.\n\n<!-- TBD: migration -->\nFIXME: verify compatibility.\n'
        )
        self.assertEqual(len(warnings), 3)
        self.assertTrue(all('placeholder' in warning for warning in warnings))
        self.assertEqual(validate_notes('The todo-list API was removed.'), [])

    def test_duplicate_bullets_ignore_spacing_case_and_marker(self):
        warnings = validate_notes(
            '* Fix   run retries.\n- fix run retries.\n+ Different behavior.\n'
        )
        self.assertEqual(len(warnings), 1)
        self.assertIn('Line 2: duplicate bullet', warnings[0])
        self.assertIn('line 1', warnings[0])

    def test_repeated_installation_code_and_fake_headings_are_ignored(self):
        text = """## Compatibility

Verified with the documented release environment.

```bash
pip install kfp==3.0.0
# TODO: example comment
## Breaking changes
* Repeated example
```

~~~markdown
## Known issues
* Repeated example
FIXME: example only
~~~
"""
        self.assertEqual(validate_notes(text), [])

    def test_fence_closer_must_match_opening_marker_and_length(self):
        text = """````markdown
```
## Known issues
* Duplicate
* Duplicate
TODO
````
"""
        self.assertEqual(validate_notes(text), [])

    def test_missing_breaking_and_deprecation_guidance_warns(self):
        warnings = validate_notes(
            '## Breaking changes\n\n* Remove legacy client ([#1](https://example.com/pull/1)).\n\n## Deprecations\n\n* Retire old parameter.\n'
        )
        self.assertEqual(len(warnings), 2)
        self.assertTrue(
            all(
                'may need migration guidance' in warning for warning in warnings
            )
        )
        self.assertIn('breaking changes', warnings[0])
        self.assertIn('deprecations', warnings[1])

    def test_indented_migration_explanation_satisfies_guidance(self):
        text = """## Breaking changes

* Replace the legacy client.

  Previously users called `old_client()`; use `new_client()` after upgrading.

## Deprecations

* Retire the old parameter.

  The parameter remains accepted in this release. Replace it with `mode`;
  removal is planned for version 4.0.
"""
        self.assertEqual(validate_notes(text), [])

    def test_migration_links_satisfy_guidance_but_pr_links_do_not(self):
        text = """## Breaking changes

* Replace the client; follow the [migration guide](https://example.com/guide).
* Change configuration; see [details](https://example.com/upgrade).
* Remove another field ([PR](https://example.com/pull/3)).
"""
        warnings = validate_notes(text)
        self.assertEqual(len(warnings), 1)
        self.assertIn('Line 5:', warnings[0])

    def test_indented_action_list_satisfies_guidance(self):
        text = """## Breaking changes

* Replace the client.
  - Install the new SDK release.
  - Replace `old_client()` calls with `new_client()`.
"""
        self.assertEqual(validate_notes(text), [])

    def test_plain_migration_url_satisfies_guidance(self):
        self.assertEqual(
            validate_notes(
                '## Breaking changes\n\n* Replace the client. https://example.com/migration\n'
            ),
            [],
        )

    def test_comments_and_fenced_examples_do_not_supply_migration_prose(self):
        text = """## Breaking changes

* Replace the client.

  <!-- Add migration details later. -->

  ```python
  old_client()
  ```
"""
        warnings = validate_notes(text)
        self.assertEqual(len(warnings), 1)
        self.assertIn('may need migration guidance', warnings[0])

    def test_category_ends_at_peer_heading_but_survives_subheadings(self):
        text = """## Breaking changes

### SDK

* Remove old client.

## Features

* Add retry support.
"""
        warnings = validate_notes(text)
        self.assertEqual(len(warnings), 1)
        self.assertIn('Line 5:', warnings[0])

    def test_empty_editorial_sections_include_comment_only_sections(self):
        text = """## Highlights

<!-- Reviewed: no summary written yet.
     Continue editing here. -->

## Action required

### Migration

## Compatibility

Requires the matching backend release.

## Known issues
"""
        warnings = validate_notes(text)
        self.assertEqual(len(warnings), 3)
        self.assertTrue(all('is empty' in warning for warning in warnings))
        self.assertFalse(
            any('compatibility' in warning for warning in warnings)
        )

    def test_code_can_fill_editorial_section_without_creating_headings(self):
        self.assertEqual(
            validate_notes(
                '## Compatibility\n\n```text\n## Known issues\nexample compatible-version output\n```\n'
            ),
            [],
        )

    def test_every_template_section_requires_content(self):
        for product in ('backend', 'sdk', 'upgrade'):
            scaffold = editorial_scaffold(product)
            headings = [
                line for line in scaffold.splitlines() if line.startswith('## ')
            ]
            for heading in headings:
                with self.subTest(product=product, heading=heading):
                    warnings = validate_notes(
                        heading + '\n<!-- Add verified facts. -->\n'
                    )
                    self.assertEqual(len(warnings), 1)
                    self.assertIn('is empty', warnings[0])
                    self.assertEqual(
                        validate_notes(heading + '\nVerified behavior.\n'), []
                    )

    def test_blank_notes_have_no_invented_warnings(self):
        self.assertEqual(validate_notes(''), [])

    def test_multiple_version_sections_warn_but_one_release_heading_does_not(
        self,
    ):
        self.assertEqual(
            validate_notes('## 2.18.0 (2026-10-27)\nRelease notes.'), []
        )
        warnings = validate_notes(
            '## 2.18.0\nNew release.\n## 2.17.3\nOld release.\n'
        )
        self.assertTrue(any('prior-release' in warning for warning in warnings))
        self.assertEqual(
            validate_notes('```markdown\n## 2.18.0\n## 2.17.3\n```'), []
        )
        self.assertEqual(
            validate_notes('## Python 3.9 compatibility\nSupported.'), []
        )

    def test_word_count_excludes_code_and_complete_generated_blocks(self):
        long_text = 'word ' * 1300
        self.assertTrue(
            any('1300 words' in w for w in validate_notes(long_text))
        )
        self.assertEqual(validate_notes('```text\n' + long_text + '\n```'), [])
        for kind in ('credits', 'contributors', 'assets'):
            text = (
                f'<!-- kfpr:generated:{kind}:start -->\n'
                + long_text
                + f'\n<!-- kfpr:generated:{kind}:end -->'
            )
            self.assertEqual(validate_notes(text), [])
        unfinished = '<!-- kfpr:generated:credits:start -->\n' + long_text
        self.assertTrue(any('words' in w for w in validate_notes(unfinished)))
        self.assertFalse(
            any(
                'words' in w
                for w in validate_notes(long_text, product='upgrade')
            )
        )

    def test_product_section_checks_are_explicit_and_advisory(self):
        self.assertEqual(validate_notes('Release text.'), [])
        for product in ('backend', 'sdk'):
            warnings = validate_notes('Release text.', product=product)
            self.assertEqual(len(warnings), 3)
            self.assertTrue(all('Missing' in w for w in warnings))
            self.assertEqual(
                validate_notes(
                    '## Bug fixes\nA fix.\n## New Contributors\nA contributor.\n## Assets\nA download.',
                    product=product,
                ),
                [],
            )
        self.assertEqual(validate_notes('Upgrade text.', product='upgrade'), [])
        with self.assertRaises(ValueError):
            validate_notes('text', product='invalid')


if __name__ == '__main__':
    unittest.main()
