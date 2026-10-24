"""Contracts for lazy optional-package delegation without changing
kfp.local."""

import types
import unittest
from unittest import mock

from click.testing import CliRunner

from kfp.cli import cli


class LocalCommandTest(unittest.TestCase):
    """Keep optional native dependencies out of ordinary SDK command
    loading."""

    def invoke(self, args: list[str]):
        """Invoke the public command tree without constructing an API
        client."""
        return CliRunner().invoke(cli.cli, args, obj={})

    @mock.patch('kfp.cli.cli.client.Client')
    @mock.patch('kfp.cli.local.importlib.import_module')
    def test_help_does_not_import_local_package(self, importer, client):
        for args in (['--help'], ['local', '--help']):
            result = self.invoke(args)
            self.assertEqual(result.exit_code, 0, result.output)
        self.assertNotIn(mock.call('kfp_local.cli'), importer.call_args_list)
        client.assert_not_called()

    @mock.patch('kfp.cli.cli.client.Client')
    @mock.patch('kfp.cli.local.importlib.import_module')
    def test_missing_optional_package_is_actionable(self, importer, client):
        importer.side_effect = ModuleNotFoundError(
            'No module named kfp_local', name='kfp_local'
        )
        result = self.invoke(['local', 'server'])
        self.assertEqual(result.exit_code, 1)
        self.assertIn('pip install kfp-local', result.output)
        client.assert_not_called()

    @mock.patch('kfp.cli.local.importlib.import_module')
    def test_delegation_preserves_options_and_exit_status(self, importer):
        entrypoint = mock.Mock(return_value=7)
        importer.return_value = types.SimpleNamespace(
            CLI_PROTOCOL_VERSION=1, main=entrypoint
        )
        result = self.invoke(
            ['local', 'ui', '--data-dir', '/tmp/example', '--ui-port', '3100']
        )
        self.assertEqual(result.exit_code, 7)
        entrypoint.assert_called_once_with(
            ['ui', '--data-dir', '/tmp/example', '--ui-port', '3100']
        )

    @mock.patch('kfp.cli.local.importlib.import_module')
    def test_incompatible_package_is_rejected(self, importer):
        importer.return_value = types.SimpleNamespace(
            CLI_PROTOCOL_VERSION=2, main=mock.Mock()
        )
        result = self.invoke(['local', 'server'])
        self.assertEqual(result.exit_code, 1)
        self.assertIn('incompatible', result.output)

    @mock.patch('kfp.cli.local.importlib.import_module')
    def test_internal_missing_dependency_is_not_misreported(self, importer):
        importer.side_effect = ModuleNotFoundError(
            'missing internal dependency', name='internal_dependency'
        )
        result = self.invoke(['local', 'server'])
        self.assertIsInstance(result.exception, ModuleNotFoundError)
        self.assertNotIn('pip install kfp-local', result.output)

    def test_existing_sdk_local_namespace_is_unchanged(self):
        from kfp import local

        self.assertTrue(callable(local.init))


if __name__ == '__main__':
    unittest.main()
