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
      client.patch_project('kubeflow-pipelines', {'default_version': 'sdk-3.2.1'})

    self.assertEqual(captured['url'], 'https://app.readthedocs.org/api/v3/projects/kubeflow-pipelines/')
    self.assertEqual(captured['method'], 'PATCH')
    self.assertEqual(captured['headers']['Authorization'], 'Token secret-token')
    self.assertEqual(captured['headers']['Content-type'], 'application/json')
    self.assertEqual(captured['body'], b'{"default_version": "sdk-3.2.1"}')

  def test_update_release_docs_syncs_activates_builds_and_sets_defaults(self):
    client = mock.Mock()
    client.get_version.side_effect = [
        {'slug': 'sdk-3.2.1'},
        {'slug': 'kfp-kubernetes-3.2.1'},
    ]
    client.get_build.side_effect = [
        {'state': {'code': 'finished'}, 'success': True},
        {'state': {'code': 'finished'}, 'success': True},
    ]
    client.trigger_build.side_effect = [101, 202]

    rtd.update_release_docs(client, '3.2.1', 'release-3.2', sleep=lambda _: None)

    client.sync_versions.assert_has_calls([
        mock.call('kubeflow-pipelines'),
        mock.call('kfp-kubernetes'),
    ])
    client.get_version.assert_has_calls([
        mock.call('kubeflow-pipelines', 'sdk-3.2.1'),
        mock.call('kfp-kubernetes', 'kfp-kubernetes-3.2.1'),
    ])
    client.activate_version.assert_has_calls([
        mock.call('kubeflow-pipelines', 'sdk-3.2.1'),
        mock.call('kfp-kubernetes', 'kfp-kubernetes-3.2.1'),
    ])
    client.trigger_build.assert_has_calls([
        mock.call('kubeflow-pipelines', 'sdk-3.2.1'),
        mock.call('kfp-kubernetes', 'kfp-kubernetes-3.2.1'),
    ])
    client.patch_project.assert_has_calls([
        mock.call('kubeflow-pipelines', {'default_version': 'sdk-3.2.1', 'default_branch': 'release-3.2'}),
        mock.call('kfp-kubernetes', {'default_version': 'kfp-kubernetes-3.2.1'}),
    ])

  def test_update_release_docs_raises_when_build_fails(self):
    client = mock.Mock()
    client.get_version.return_value = {'slug': 'sdk-3.2.1'}
    client.trigger_build.return_value = 101
    client.get_build.return_value = {'state': {'code': 'finished'}, 'success': False, 'error': 'bad docs'}

    with self.assertRaisesRegex(rtd.ReadTheDocsError, 'bad docs'):
      rtd.update_release_docs(client, '3.2.1', 'release-3.2', sleep=lambda _: None)

  def test_update_release_docs_waits_for_synced_version(self):
    client = mock.Mock()
    client.get_version.side_effect = [
        rtd.ReadTheDocsError('missing version', status_code=404),
        {'slug': 'sdk-3.2.1'},
        {'slug': 'kfp-kubernetes-3.2.1'},
    ]
    client.get_build.side_effect = [
        {'state': {'code': 'finished'}, 'success': True},
        {'state': {'code': 'finished'}, 'success': True},
    ]
    client.trigger_build.side_effect = [101, 202]

    rtd.update_release_docs(client, '3.2.1', 'release-3.2', sleep=lambda _: None)

    self.assertEqual(client.get_version.call_args_list[:2], [
        mock.call('kubeflow-pipelines', 'sdk-3.2.1'),
        mock.call('kubeflow-pipelines', 'sdk-3.2.1'),
    ])


if __name__ == '__main__':
  unittest.main()
