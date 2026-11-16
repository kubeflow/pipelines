from __future__ import annotations

import importlib.util
import pathlib
import sys
import unittest
from contextlib import ExitStack
from datetime import datetime, timezone
from unittest import mock

MODULE_PATH = pathlib.Path(__file__).with_name('contributor-report.py')
SPEC = importlib.util.spec_from_file_location('contributor_report', MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


def comment_node(owner='kubeflow', repo='pipelines', kind='pull'):
    return {
        'issue': {
            'url': f'https://github.com/{owner}/{repo}/{kind}/123',
            'repository': {'owner': {'login': owner}, 'name': repo},
        }
    }


def stats_response(nodes, cursor=None, errors=()):
    response = {
        'data': {
            'user': {
                'createdAt': '2020-01-01T00:00:00Z',
                'issueComments': {
                    'nodes': nodes,
                    'pageInfo': {
                        'hasNextPage': cursor is not None,
                        'endCursor': cursor,
                    },
                },
            },
            'issuesOpened': {'issueCount': 1},
            'mergedPrs': {'issueCount': 2},
        }
    }
    if errors:
        response['errors'] = list(errors)
    return response


def denied_comment_error():
    return {
        'type': 'FORBIDDEN',
        'path': ['user', 'issueComments', 'nodes', 0],
        'message': 'Resource not accessible by integration',
    }


class ContributorReportTest(unittest.TestCase):
    def test_pagination_queries_static_metrics_only_on_the_first_page(self):
        second_page = stats_response([comment_node()])
        del second_page['data']['issuesOpened']
        del second_page['data']['mergedPrs']
        del second_page['data']['user']['createdAt']
        with mock.patch.object(
            MODULE,
            'github_request',
            side_effect=[
                stats_response([comment_node()], cursor='next'),
                second_page,
            ],
        ) as request:
            stats = MODULE.fetch_contributor_stats('alice')
        self.assertEqual(stats.pr_comments, 2)
        bodies = [call.kwargs['body'] for call in request.call_args_list]
        self.assertEqual(len(bodies), 2)
        self.assertEqual(sum('search(' in body['query'] for body in bodies), 1)
        self.assertNotIn('createdAt', bodies[1]['query'])
        self.assertEqual(
            bodies[1]['variables'],
            {'username': 'alice', 'issueCommentCursor': 'next'},
        )

    def test_identical_report_does_not_write_a_comment(self):
        body = MODULE.REPORT_MARKER + '\nunchanged report'
        existing = {'id': 12, 'body': body, 'user': {'type': 'Bot'}}
        with (
            mock.patch.object(
                MODULE, 'list_issue_comments', return_value=[existing]
            ),
            mock.patch.object(MODULE, 'github_request') as request,
        ):
            result = MODULE.upsert_comment('kubeflow', 'pipelines', 123, body)
        self.assertEqual(result, 'unchanged')
        request.assert_not_called()

    def test_changed_report_is_updated(self):
        body = MODULE.REPORT_MARKER + '\nnew report'
        existing = {
            'id': 12,
            'body': MODULE.REPORT_MARKER + '\nold report',
            'user': {'type': 'Bot'},
        }
        with (
            mock.patch.object(
                MODULE, 'list_issue_comments', return_value=[existing]
            ),
            mock.patch.object(MODULE, 'github_request') as request,
        ):
            result = MODULE.upsert_comment('kubeflow', 'pipelines', 123, body)
        self.assertEqual(result, 'updated')
        request.assert_called_once_with(
            '/repos/kubeflow/pipelines/issues/comments/12',
            method='PATCH',
            body={'body': body},
        )

    def test_report_identifies_a_dated_informational_snapshot(self):
        with mock.patch.object(MODULE, 'datetime') as clock:
            clock.now.return_value = datetime(
                2026, 11, 14, 23, 59, tzinfo=timezone.utc
            )
            first = MODULE.build_comment('alice', [])
            clock.now.return_value = datetime(
                2026, 11, 14, 23, 59, 59, tzinfo=timezone.utc
            )
            same_day = MODULE.build_comment('alice', [])
            clock.now.return_value = datetime(2026, 11, 15, tzinfo=timezone.utc)
            next_day = MODULE.build_comment('alice', [])
        self.assertIn('Snapshot as of 2026-11-14 (UTC)', first)
        self.assertIn('not an authorization decision', first)
        self.assertEqual(first, same_day)
        self.assertIn('Snapshot as of 2026-11-15 (UTC)', next_day)

    def test_stats_paginate_and_count_only_target_repository_pr_comments(self):
        pages = [
            stats_response(
                [
                    comment_node(),
                    comment_node(kind='issues'),
                    comment_node(repo='other'),
                    comment_node(owner='other'),
                    {'issue': None},
                ],
                cursor='next',
            ),
            stats_response([comment_node(), comment_node()]),
        ]
        with mock.patch.object(
            MODULE, 'github_request', side_effect=pages
        ) as request:
            stats = MODULE.fetch_contributor_stats('alice')
        self.assertEqual(
            stats, MODULE.ContributorStats('2020-01-01T00:00:00Z', 1, 2, 3)
        )
        self.assertEqual(
            [
                call.kwargs['body']['variables']['issueCommentCursor']
                for call in request.call_args_list
            ],
            [None, 'next'],
        )

    def test_denied_comment_page_discards_partial_count_and_stops_pagination(
        self,
    ):
        for denied_page in (0, 1):
            with self.subTest(denied_page=denied_page):
                pages = [
                    stats_response([comment_node()], cursor='next')
                ] * denied_page
                pages.append(
                    stats_response(
                        [None, comment_node(), None],
                        cursor='more',
                        errors=[
                            denied_comment_error(),
                            {
                                **denied_comment_error(),
                                'path': ['user', 'issueComments', 'nodes', 2],
                            },
                        ],
                    )
                )
                with (
                    mock.patch.object(
                        MODULE, 'github_request', side_effect=pages
                    ) as request,
                    mock.patch('builtins.print'),
                ):
                    stats = MODULE.fetch_contributor_stats('alice')
                self.assertEqual(
                    stats,
                    MODULE.ContributorStats('2020-01-01T00:00:00Z', 1, 2, None),
                )
                self.assertEqual(request.call_count, denied_page + 1)
                rows = MODULE.build_user_rows(True, stats)
                values = {row.metric: row.value for row in rows}
                self.assertEqual(values['Kubeflow org member'], 'Yes')
                self.assertEqual(
                    values['Issues opened in kubeflow/pipelines'], '1'
                )
                self.assertEqual(
                    values['Merged PRs in kubeflow/pipelines'], '2'
                )
                self.assertEqual(
                    values['PR comments in kubeflow/pipelines'],
                    'Unavailable (GitHub access restrictions)',
                )

    def test_other_or_mixed_graphql_errors_are_not_partial_success(self):
        denied = denied_comment_error()
        other_errors = [
            {**denied, 'type': 'RATE_LIMITED'},
            {**denied, 'path': ['issuesOpened']},
            {**denied, 'path': ['user', 'issueComments']},
            {**denied, 'path': ['user', 'issueComments', 'nodes']},
            {**denied, 'path': ['user', 'issueComments', 'nodes', 0, 'issue']},
            {**denied, 'path': ['user', 'issueComments', 'nodes', '0']},
            {**denied, 'path': ['user', 'issueComments', 'nodes', True]},
            {**denied, 'path': ['user', 'issueComments', 'nodes', -1]},
        ]
        for error in other_errors:
            for errors in ([error], [denied, error]):
                with (
                    self.subTest(errors=errors),
                    mock.patch.object(
                        MODULE,
                        'github_request',
                        return_value=stats_response([None], errors=errors),
                    ),
                ):
                    with self.assertRaisesRegex(
                        RuntimeError, 'GitHub GraphQL failed'
                    ):
                        MODULE.fetch_contributor_stats('alice')

    def test_null_comment_nodes_without_access_error_are_rejected(self):
        for nodes in (None, [None]):
            with (
                self.subTest(nodes=nodes),
                mock.patch.object(
                    MODULE, 'github_request', return_value=stats_response(nodes)
                ),
            ):
                with self.assertRaises(
                    (RuntimeError, TypeError, AttributeError)
                ):
                    MODULE.fetch_contributor_stats('alice')

    def test_partial_response_requires_valid_independent_metrics(self):
        invalid_fields = [
            ('user', None),
            ('user', {'createdAt': None}),
            ('user', {'createdAt': ''}),
            ('user', {'createdAt': 1}),
            ('user', {'createdAt': 'not-a-date'}),
            ('user', {}),
            ('issuesOpened', None),
            ('issuesOpened', {}),
            ('issuesOpened', {'issueCount': None}),
            ('issuesOpened', {'issueCount': '1'}),
            ('mergedPrs', {'issueCount': -1}),
            ('mergedPrs', {'issueCount': True}),
            ('mergedPrs', {}),
        ]
        for field, value in invalid_fields:
            response = stats_response([None], errors=[denied_comment_error()])
            response['data'][field] = value
            with (
                self.subTest(field=field, value=value),
                mock.patch.object(
                    MODULE, 'github_request', return_value=response
                ),
                mock.patch('builtins.print'),
            ):
                with self.assertRaises(
                    (
                        RuntimeError,
                        KeyError,
                        TypeError,
                        AssertionError,
                        ValueError,
                    )
                ):
                    stats = MODULE.fetch_contributor_stats('alice')
                    MODULE.build_user_rows(True, stats)

    def test_membership_lookup_failure_does_not_publish_a_report(self):
        event = '{"pull_request": {"number": 123, "user": {"login": "alice", "type": "User"}}}'
        with ExitStack() as stack:
            stack.enter_context(
                mock.patch.dict(
                    MODULE.os.environ,
                    {
                        'GITHUB_EVENT_PATH': 'event.json',
                        'GITHUB_REPOSITORY': 'kubeflow/pipelines',
                        'CONTRIBUTOR_REPORT_DRY_RUN': 'false',
                    },
                )
            )
            stack.enter_context(
                mock.patch('builtins.open', mock.mock_open(read_data=event))
            )
            stack.enter_context(
                mock.patch.object(
                    MODULE,
                    'is_kubeflow_member',
                    side_effect=RuntimeError('ACL lookup failed'),
                )
            )
            fetch = stack.enter_context(
                mock.patch.object(MODULE, 'fetch_contributor_stats')
            )
            publish = stack.enter_context(
                mock.patch.object(MODULE, 'upsert_comment')
            )
            with self.assertRaisesRegex(RuntimeError, 'ACL lookup failed'):
                MODULE.main()
            fetch.assert_not_called()
            publish.assert_not_called()

    def test_workflow_skips_dependency_and_sync_bot_authors_at_job_level(self):
        workflow_path = (
            MODULE_PATH.parent.parent / 'workflows' / 'contributor-report.yml'
        )
        workflow = workflow_path.read_text()
        self.assertIn(
            '  contributor-report:\n'
            '    if: >-\n'
            "      github.event.pull_request.user.login != 'dependabot[bot]' &&\n"
            "      github.event.pull_request.user.login != 'copybara-service[bot]'\n",
            workflow,
        )

    def test_report_uses_shared_membership_lookup(self):
        event = (
            '{"pull_request": {"user": {"login": "JerT33", "type": "User"}}}'
        )
        stats = MODULE.ContributorStats('2020-01-01T00:00:00Z', 1, 2, 3)
        for is_member in (True, False):
            with self.subTest(is_member=is_member), ExitStack() as stack:
                stack.enter_context(
                    mock.patch.dict(
                        MODULE.os.environ,
                        {
                            'GITHUB_EVENT_PATH': 'event.json',
                            'CONTRIBUTOR_REPORT_DRY_RUN': 'true',
                        },
                    )
                )
                stack.enter_context(
                    mock.patch('builtins.open', mock.mock_open(read_data=event))
                )
                lookup = stack.enter_context(
                    mock.patch.object(
                        MODULE, 'is_kubeflow_member', return_value=is_member
                    )
                )
                stack.enter_context(
                    mock.patch.object(
                        MODULE, 'fetch_contributor_stats', return_value=stats
                    )
                )
                rows = stack.enter_context(
                    mock.patch.object(
                        MODULE, 'build_user_rows', return_value=[]
                    )
                )
                stack.enter_context(mock.patch('builtins.print'))
                self.assertEqual(MODULE.main(), 0)
                lookup.assert_called_once_with('JerT33')
                rows.assert_called_once_with(is_member, stats)

    def test_workflow_checks_out_shared_membership_module(self):
        workflow_path = (
            MODULE_PATH.parent.parent / 'workflows' / 'contributor-report.yml'
        )
        workflow = workflow_path.read_text()
        self.assertIn('ref: ${{ github.workflow_sha }}', workflow)
        self.assertIn(
            '            .github/scripts/kubeflow_membership.py\n', workflow
        )

    def test_non_user_rows_report_not_applicable(self):
        rows = MODULE.build_non_user_rows('Bot')
        self.assertEqual(rows[0].metric, 'GitHub author type')
        self.assertEqual(rows[0].value, 'Bot')
        self.assertTrue(
            all(
                row.value == 'N/A (non-human GitHub author)' for row in rows[1:]
            )
        )

    def test_build_comment_renders_markdown_table(self):
        rows = [
            MODULE.MarkdownRow(metric='Metric A', value='1'),
            MODULE.MarkdownRow(metric='Metric B', value='2'),
        ]
        comment = MODULE.build_comment('alice', rows)
        self.assertIn('#### Contributor Report for @alice', comment)
        self.assertIn('| Metric | Value |', comment)
        self.assertIn('| Metric A | 1 |', comment)
        self.assertIn('| Metric B | 2 |', comment)
        self.assertNotIn('| User | @alice |', comment)
        self.assertNotIn(
            'This report is generated by a repo-local workflow', comment
        )

    def test_list_issue_comments_paginates_until_short_page(self):
        calls = []
        pages = {
            1: [{'id': i} for i in range(100)],
            2: [{'id': 100}, {'id': 101}],
        }

        def fake_request(path, method='GET', body=None):
            del method, body
            calls.append(path)
            page = int(path.split('page=')[-1])
            return pages[page]

        original = MODULE.github_request
        try:
            MODULE.github_request = fake_request
            comments = MODULE.list_issue_comments('kubeflow', 'pipelines', 123)
        finally:
            MODULE.github_request = original

        self.assertEqual(len(comments), 102)
        self.assertEqual(len(calls), 2)
        self.assertTrue(calls[0].endswith('page=1'))
        self.assertTrue(calls[1].endswith('page=2'))

    def test_build_user_rows_uses_total_pr_comment_count_only(self):
        stats = MODULE.ContributorStats(
            created_at='2020-01-01T00:00:00Z',
            issues_opened=1,
            merged_prs=2,
            pr_comments=7,
        )
        rows = MODULE.build_user_rows(True, stats)
        pr_comments_row = next(
            row for row in rows if row.metric.startswith('PR comments in ')
        )
        self.assertEqual(pr_comments_row.value, '7')

    def test_is_report_comment_requires_marker_and_bot_author(self):
        self.assertTrue(
            MODULE.is_report_comment(
                {
                    'body': 'prefix <!-- kfp-contributor-report --> suffix',
                    'user': {'type': 'Bot'},
                }
            )
        )
        self.assertFalse(
            MODULE.is_report_comment(
                {
                    'body': 'prefix <!-- kfp-contributor-report --> suffix',
                    'user': {'type': 'User'},
                }
            )
        )
        self.assertFalse(
            MODULE.is_report_comment(
                {
                    'body': 'no marker here',
                    'user': {'type': 'Bot'},
                }
            )
        )

    def test_is_human_user_checks_github_type(self):
        self.assertTrue(MODULE.is_human_user({'type': 'User'}))
        self.assertFalse(MODULE.is_human_user({'type': 'Bot'}))
        self.assertFalse(MODULE.is_human_user({'type': 'Organization'}))


if __name__ == '__main__':
    unittest.main()
