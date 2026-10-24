"""Tests for shared changelog classification and Markdown rendering."""

import unittest

from kfpr.notes import Commit, render_notes


class TestRenderNotes(unittest.TestCase):
    def test_user_facing_types_and_stable_group_order(self):
        messages = [
            'fix: handle empty inputs',
            'deprecate: retire legacy syntax',
            'perf: reduce startup time',
            'feat: support retries',
            'security: validate paths',
            'refactor!: remove legacy parser',
        ]
        notes = render_notes(Commit('abc123456', msg) for msg in messages)
        self.assertEqual(
            [line for line in notes.splitlines() if line.startswith('## ')],
            [
                '## Breaking changes',
                '## Security fixes',
                '## Features',
                '## Performance',
                '## Deprecations',
                '## Bug fixes',
            ],
        )

    def test_routine_changes_and_nonconventional_commits_are_excluded(self):
        messages = [
            f'{kind}(sdk): maintenance'
            for kind in ('chore', 'test', 'docs', 'ci', 'refactor', 'style')
        ] + ['Merge pull request #123', '']
        self.assertEqual(
            render_notes(Commit('abc123456', msg) for msg in messages), ''
        )

    def test_breaking_footers_include_routine_changes(self):
        for marker in ('BREAKING CHANGE', 'BREAKING-CHANGE'):
            with self.subTest(marker=marker):
                notes = render_notes(
                    [
                        Commit(
                            'abc123456',
                            f'chore(sdk): remove old interface\n\n{marker}: migrate to v2',
                        )
                    ],
                    'sdk',
                )
                self.assertIn('## Breaking changes', notes)
                self.assertIn('remove old interface', notes)
                self.assertNotIn('## Features', notes)

    def test_security_scope_and_explicit_deprecations_override_type(self):
        messages = [
            'chore(backend/security): rebuild vulnerable image',
            'fix(sdk): retire old argument\n\nDEPRECATION: use the new argument',
            'docs(deprecation): document retirement',
        ]
        notes = render_notes(Commit('abc123456', msg) for msg in messages)
        self.assertIn('## Security fixes', notes)
        self.assertIn('## Deprecations', notes)
        self.assertNotIn('## Bug fixes', notes)
        self.assertNotIn('## Features', notes)

    def test_breaking_migration_paragraphs_stop_at_next_trailer(self):
        notes = render_notes(
            [
                Commit(
                    'abc123456',
                    'chore(sdk): remove legacy configuration\n\n'
                    'BREAKING CHANGE: Back up your configuration.\n'
                    'Then run the migration command.\n\n'
                    'Restart the service after migrating.\n\n'
                    'Signed-off-by: Contributor <person@example.com>\n'
                    'Refs: #123',
                )
            ]
        )
        self.assertIn('## Breaking changes', notes)
        self.assertIn(
            '\n\n  Back up your configuration.\n'
            '  Then run the migration command.\n\n'
            '  Restart the service after migrating.',
            notes,
        )
        self.assertNotIn('Signed-off-by', notes)
        self.assertNotIn('person@example.com', notes)
        self.assertNotIn('Refs:', notes)

    def test_migration_url_and_following_paragraph_are_retained(self):
        notes = render_notes(
            [
                Commit(
                    'abc123456',
                    'fix(sdk)!: update configuration\n\n'
                    'BREAKING CHANGE: Follow this migration guide:\n'
                    'https://example.com/migration\n\n'
                    'Restart the service after migrating.\n\n'
                    'Signed-off-by: Contributor <person@example.com>',
                )
            ]
        )
        self.assertIn('https://example.com/migration', notes)
        self.assertIn('Restart the service after migrating.', notes)
        self.assertNotIn('Signed-off-by', notes)

    def test_fenced_migration_examples_do_not_end_guidance(self):
        for opening, closing in (('```yaml', '```'), ('~~~~yaml', '~~~~')):
            with self.subTest(opening=opening):
                notes = render_notes(
                    [
                        Commit(
                            'abc123456',
                            'fix(sdk)!: update configuration\n\n'
                            'BREAKING CHANGE: Use the following configuration:\n'
                            f'{opening}\n'
                            'apiVersion: pipelines.kubeflow.org/v2\n'
                            'Signed-off-by: this is example data\n'
                            '~~~\n'
                            f'{closing}\n\n'
                            'Apply the configuration, then restart the service.\n\n'
                            'Signed-off-by: Contributor <person@example.com>',
                        )
                    ]
                )
                self.assertIn('apiVersion: pipelines.kubeflow.org/v2', notes)
                self.assertIn('Signed-off-by: this is example data', notes)
                self.assertIn(f'  {closing}\n\n', notes)
                self.assertIn(
                    'Apply the configuration, then restart the service.', notes
                )
                self.assertNotIn('Contributor <person@example.com>', notes)

    def test_empty_trailer_line_supports_continued_guidance(self):
        notes = render_notes(
            [
                Commit(
                    'abc123456',
                    'fix(sdk)!: update configuration\n\n'
                    'BREAKING CHANGE:\n'
                    'Follow the migration guide.\n\n'
                    'Signed-off-by:\nContributor <person@example.com>',
                )
            ]
        )
        self.assertIn('Follow the migration guide.', notes)
        self.assertNotIn('Signed-off-by', notes)
        self.assertNotIn('Contributor <person@example.com>', notes)

    def test_deprecation_guidance_is_retained(self):
        notes = render_notes(
            [
                Commit(
                    'abc123456',
                    'fix(sdk): retire old option\n\n'
                    'DEPRECATION: Use the replacement option.\n'
                    'Reviewed-by: Reviewer <reviewer@example.com>',
                )
            ]
        )
        self.assertIn('## Deprecations', notes)
        self.assertIn('\n\n  Use the replacement option.', notes)
        self.assertNotIn('Reviewed-by', notes)

    def test_cve_scope_is_security_even_for_maintenance(self):
        notes = render_notes(
            [Commit('abc123456', 'chore(backend/cve): rebuild patched image')],
            'backend',
        )
        self.assertIn('## Security fixes', notes)
        self.assertIn('rebuild patched image', notes)

    def test_sdk_package_aliases_do_not_disappear(self):
        for scope in (
            'sdk',
            'kfp',
            'kfp-kubernetes',
            'kubernetes',
            'kubernetes-platform',
            'kubernetes_platform',
            'kfp-server-api',
            'server-api',
            'sdk/client',
        ):
            with self.subTest(scope=scope):
                commits = [
                    Commit('abc123456', f'fix({scope}): correct packaging')
                ]
                self.assertIn('correct packaging', render_notes(commits, 'sdk'))
                self.assertEqual(render_notes(commits, 'backend'), '')

    def test_backend_scopes_are_filtered(self):
        for scope in ('backend', 'runtime', 'frontend', 'manifests'):
            with self.subTest(scope=scope):
                commits = [
                    Commit('abc123456', f'fix({scope}): correct behavior')
                ]
                self.assertEqual(render_notes(commits, 'sdk'), '')
                self.assertIn(
                    'correct behavior', render_notes(commits, 'backend')
                )

    def test_shared_cross_product_unknown_and_unscoped_changes_reach_both(self):
        for scope in (
            'api',
            'pipeline-spec',
            'kfp-pipeline-spec',
            'backend/sdk',
            'sdk, frontend',
            'new-component',
            'SDK/RUNTIME',
            '',
        ):
            for product in ('sdk', 'backend', 'all'):
                with self.subTest(scope=scope, product=product):
                    prefix = f'({scope})' if scope else ''
                    self.assertIn(
                        'correct behavior',
                        render_notes(
                            [
                                Commit(
                                    'abc123456',
                                    f'fix{prefix}: correct behavior',
                                )
                            ],
                            product,
                        ),
                    )

    def test_documented_component_aliases_route_only_to_backend(self):
        aliases = {
            'deployment': 'manifests',
            'cache': 'backend/cache',
            'metadata': 'backend/metadata',
            'swf': 'backend/scheduled-workflow',
            'viewer': 'frontend/viewer',
        }
        for scope, canonical in aliases.items():
            for spelling in (scope, scope.upper()):
                with self.subTest(scope=spelling):
                    commits = [
                        Commit(
                            'abc123456', f'fix({spelling}): correct behavior'
                        )
                    ]
                    self.assertEqual(render_notes(commits, 'sdk'), '')
                    for product in ('backend', 'all'):
                        self.assertIn(
                            f'**{canonical}**: correct behavior',
                            render_notes(commits, product),
                        )

    def test_aliases_preserve_cross_product_and_shared_scope_routing(self):
        for scope, canonical in (
            ('sdk/client, deployment', 'sdk/client, manifests'),
            ('sdk/cache', 'sdk/backend/cache'),
            ('viewer/api', 'frontend/viewer/api'),
        ):
            with self.subTest(scope=scope):
                commits = [
                    Commit('abc123456', f'fix({scope}): repair behavior')
                ]
                for product in ('sdk', 'backend', 'all'):
                    self.assertIn(
                        f'**{canonical}**: repair behavior',
                        render_notes(commits, product),
                    )

    def test_existing_canonical_scope_paths_do_not_duplicate_components(self):
        for scope in ('backend/cache', 'backend/metadata', 'frontend/viewer'):
            with self.subTest(scope=scope):
                commits = [
                    Commit('abc123456', f'fix({scope}): repair behavior')
                ]
                self.assertIn(f'**{scope}**:', render_notes(commits, 'backend'))
                self.assertEqual(render_notes(commits, 'sdk'), '')

    def test_alias_matching_does_not_classify_unknown_substrings(self):
        for scope in (
            'cacheutils',
            'metadata-writer',
            'swf-extra',
            'viewer-ui',
            'not-deployment',
        ):
            with self.subTest(scope=scope):
                commits = [
                    Commit('abc123456', f'fix({scope}): repair behavior')
                ]
                for product in ('sdk', 'backend'):
                    self.assertIn(
                        f'**{scope}**:', render_notes(commits, product)
                    )

    def test_scope_matching_is_not_substring_matching(self):
        commits = [Commit('abc123456', 'fix(not-sdk): correct behavior')]
        self.assertIn('correct behavior', render_notes(commits, 'backend'))

    def test_markdown_keeps_scope_and_links_pr_or_commit(self):
        notes = render_notes(
            [
                Commit('abcdef123456789', 'feat(sdk): add retries (#123)'),
                Commit('123456789abcdef', 'fix(api): handle empty inputs'),
            ]
        )
        self.assertIn(
            '* **sdk**: add retries ([#123](https://github.com/kubeflow/pipelines/pull/123))',
            notes,
        )
        self.assertIn(
            '* **api**: handle empty inputs ([1234567](https://github.com/kubeflow/pipelines/commit/123456789abcdef))',
            notes,
        )
        self.assertNotIn('/commit/abcdef123456789', notes)

    def test_invalid_product_is_rejected(self):
        with self.assertRaisesRegex(ValueError, 'choose all, sdk, or backend'):
            render_notes([], 'other')


if __name__ == '__main__':
    unittest.main()
