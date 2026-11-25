#!/usr/bin/env python3
"""Tests for Read the Docs API automation."""

import unittest
from unittest import mock

from kfpr import rtd


class ReadTheDocsClientTest(unittest.TestCase):
    def test_request_sends_token_and_json_body(self):
        client = rtd.ReadTheDocsClient('secret-token')
        captured = {}

        def fake_urlopen(request, timeout):
            captured['url'] = request.full_url
            captured['method'] = request.get_method()
            captured['headers'] = dict(request.header_items())
            captured['body'] = request.data
            return mock.Mock(status=204, read=lambda: b'')

        with mock.patch('urllib.request.urlopen', side_effect=fake_urlopen):
            client.patch_project(
                'kubeflow-pipelines', {'default_version': 'sdk-3.2.1'}
            )

        self.assertEqual(
            captured['url'],
            'https://app.readthedocs.org/api/v3/projects/kubeflow-pipelines/',
        )
        self.assertEqual(captured['method'], 'PATCH')
        self.assertEqual(
            captured['headers']['Authorization'], 'Token secret-token'
        )
        self.assertEqual(
            captured['headers']['Content-type'], 'application/json'
        )
        self.assertEqual(captured['body'], b'{"default_version": "sdk-3.2.1"}')

    def test_backend_tag_build_uses_qualified_commit_and_public_url(self):
        tag = '2.18.2'
        project = 'kubeflow-pipelines'
        url = f'https://{project}.readthedocs.io/en/{tag}/'
        sha = 'a' * 40
        build = {
            'id': 12,
            'state': {'code': 'finished'},
            'success': True,
            'version': tag,
            'commit': sha,
        }
        for token in ('', 'rtd-token'):
            with (
                self.subTest(authenticated=bool(token)),
                mock.patch.object(rtd, 'ReadTheDocsClient') as client_type,
                mock.patch('urllib.request.urlopen') as get_public_docs,
            ):
                client = client_type.return_value
                client.trigger_build.return_value = 12
                client.get_build.return_value = build
                client.request.return_value = {'results': [build]}
                response = get_public_docs.return_value.__enter__.return_value
                response.status = 200
                response.geturl.return_value = url
                rtd.publish_backend_docs(tag, sha, token)
                client_type.assert_called_once_with(token)
                if token:
                    client.sync_versions.assert_called_once_with(project)
                    client.activate_version.assert_called_once_with(
                        project, tag
                    )
                    client.trigger_build.assert_called_once_with(project, tag)
                    client.get_build.assert_called_once_with(project, 12)
                else:
                    client.request.assert_called_once_with(
                        'GET',
                        f'/projects/{project}/versions/{tag}/builds/?limit=1',
                    )
                    client.activate_version.assert_not_called()
                    client.trigger_build.assert_not_called()
                client.patch_project.assert_not_called()
                request = get_public_docs.call_args.args[0]
                self.assertEqual(request.full_url, url)
                self.assertEqual(request.get_method(), 'HEAD')
                self.assertTrue(
                    request.get_header('User-agent').startswith('kfpr')
                )
                self.assertFalse(request.has_header('Authorization'))
                response.geturl.return_value = url.replace(tag, 'latest')
                with self.assertRaisesRegex(
                    rtd.ReadTheDocsError, 'tagged docs'
                ):
                    rtd.publish_backend_docs(tag, sha, token)

    def test_backend_tag_rejects_missing_or_mismatched_build(self):
        sha = 'a' * 40
        build = {
            'state': {'code': 'finished'},
            'success': True,
            'version': '2.18.2',
            'commit': sha,
        }
        for results in (
            [],
            [dict(build, version='sdk-2.18.2')],
            [dict(build, commit='b' * 40)],
            [dict(build, success=False)],
            [dict(build, state={'code': 'building'})],
        ):
            with (
                self.subTest(results=results),
                mock.patch.object(rtd, 'ReadTheDocsClient') as client_type,
                mock.patch('urllib.request.urlopen') as get_public_docs,
            ):
                client_type.return_value.request.return_value = {
                    'results': results
                }
                with self.assertRaisesRegex(
                    rtd.ReadTheDocsError, 'No successful'
                ):
                    rtd.publish_backend_docs('2.18.2', sha, '')
                get_public_docs.assert_not_called()

    def test_update_release_docs_syncs_activates_builds_and_sets_defaults(self):
        client = mock.Mock()
        client.get_version.side_effect = [
            {'slug': 'release-3.2'},
            {'slug': 'kfp-kubernetes-3.2'},
        ]
        client.get_build.side_effect = [
            {'state': {'code': 'finished'}, 'success': True},
            {'state': {'code': 'finished'}, 'success': True},
        ]
        client.trigger_build.side_effect = [101, 202]

        rtd.update_release_docs(
            client,
            '3.2.1',
            'release-3.2',
            sleep=lambda _: None,
            consolidated=True,
        )

        client.sync_versions.assert_called_once_with('kubeflow-pipelines')
        client.get_version.assert_called_once_with(
            'kubeflow-pipelines', 'release-3.2'
        )
        client.activate_version.assert_called_once_with(
            'kubeflow-pipelines', 'release-3.2'
        )
        client.trigger_build.assert_called_once_with(
            'kubeflow-pipelines', 'release-3.2'
        )
        client.patch_project.assert_called_once_with(
            'kubeflow-pipelines',
            {'default_version': 'release-3.2', 'default_branch': 'release-3.2'},
        )

    def test_split_release_updates_both_projects(self) -> None:
        """Preserve each legacy project's branch and defaults."""
        client = mock.Mock()
        client.get_build.return_value = {
            'state': {'code': 'finished'},
            'success': True,
        }
        client.trigger_build.side_effect = [101, 202]

        rtd.update_release_docs(
            client,
            '2.18.1',
            'release-2.18',
            sleep=lambda _: None,
            consolidated=False,
        )

        targets = [
            ('kubeflow-pipelines', 'release-2.18'),
            ('kfp-kubernetes', 'kfp-kubernetes-2.18'),
        ]
        self.assertEqual(
            client.sync_versions.call_args_list,
            [mock.call(project) for project, _ in targets],
        )
        for method in (
            client.get_version,
            client.activate_version,
            client.trigger_build,
        ):
            self.assertEqual(
                method.call_args_list,
                [mock.call(project, branch) for project, branch in targets],
            )
        self.assertEqual(
            client.get_build.call_args_list,
            [
                mock.call('kubeflow-pipelines', 101),
                mock.call('kfp-kubernetes', 202),
            ],
        )
        self.assertEqual(
            client.patch_project.call_args_list,
            [
                mock.call(
                    'kubeflow-pipelines',
                    {
                        'default_version': 'release-2.18',
                        'default_branch': 'release-2.18',
                    },
                ),
                mock.call(
                    'kfp-kubernetes',
                    {
                        'default_version': 'kfp-kubernetes-2.18',
                    },
                ),
            ],
        )

    def test_update_release_docs_prints_polling_progress(self):
        client = mock.Mock()
        client.get_version.side_effect = [
            {'slug': 'release-3.2'},
            {'slug': 'kfp-kubernetes-3.2'},
        ]
        client.get_build.side_effect = [
            {'state': {'code': 'building'}, 'success': False},
            {'state': {'code': 'finished'}, 'success': True},
            {'state': {'code': 'finished'}, 'success': True},
        ]
        client.trigger_build.side_effect = [101, 202]

        with mock.patch('builtins.print') as print_mock:
            rtd.update_release_docs(
                client,
                '3.2.1',
                'release-3.2',
                sleep=lambda _: None,
                consolidated=True,
            )

        output = '\n'.join(
            str(call.args[0]) for call in print_mock.call_args_list
        )
        self.assertIn('Syncing ReadTheDocs versions...', output)
        self.assertIn(
            'Waiting for kubeflow-pipelines release-3.2 to sync...', output
        )
        self.assertIn(
            'Waiting for kubeflow-pipelines build 101 to finish...', output
        )
        self.assertIn('kubeflow-pipelines build 101 is building.', output)
        self.assertIn('Setting ReadTheDocs defaults...', output)

    def test_split_kubernetes_build_failure_preserves_defaults(self) -> None:
        """Do not complete a split release with failed Kubernetes docs."""
        client = mock.Mock()
        client.trigger_build.side_effect = [101, 202]
        client.get_build.side_effect = [
            {'state': {'code': 'finished'}, 'success': True},
            {
                'state': {'code': 'finished'},
                'success': False,
                'error': 'Kubernetes docs failed',
            },
        ]

        with self.assertRaisesRegex(
            rtd.ReadTheDocsError, 'Kubernetes docs failed'
        ):
            rtd.update_release_docs(
                client,
                '2.18.1',
                'release-2.18',
                sleep=lambda _: None,
                consolidated=False,
            )

        client.get_build.assert_called_with('kfp-kubernetes', 202)
        client.patch_project.assert_not_called()

    def test_update_release_docs_raises_when_build_fails(self):
        client = mock.Mock()
        client.get_version.return_value = {'slug': 'release-3.2'}
        client.trigger_build.return_value = 101
        client.get_build.return_value = {
            'state': {'code': 'finished'},
            'success': False,
            'error': 'bad docs',
        }

        with self.assertRaisesRegex(rtd.ReadTheDocsError, 'bad docs'):
            rtd.update_release_docs(
                client,
                '3.2.1',
                'release-3.2',
                sleep=lambda _: None,
                consolidated=True,
            )

    def test_update_release_docs_waits_for_synced_version(self):
        client = mock.Mock()
        client.get_version.side_effect = [
            rtd.ReadTheDocsError('missing version', status_code=404),
            {'slug': 'release-3.2'},
            {'slug': 'kfp-kubernetes-3.2'},
        ]
        client.get_build.side_effect = [
            {'state': {'code': 'finished'}, 'success': True},
            {'state': {'code': 'finished'}, 'success': True},
        ]
        client.trigger_build.side_effect = [101, 202]

        rtd.update_release_docs(
            client,
            '3.2.1',
            'release-3.2',
            sleep=lambda _: None,
            consolidated=True,
        )

        self.assertEqual(
            client.get_version.call_args_list[:2],
            [
                mock.call('kubeflow-pipelines', 'release-3.2'),
                mock.call('kubeflow-pipelines', 'release-3.2'),
            ],
        )


if __name__ == '__main__':
    unittest.main()
