"""Verify offline cleanup routing through the installed command parser."""

import types
from pathlib import Path
from unittest import mock

from kfp_local.cli import main


def test_ui_routes_custom_ports(tmp_path: Path) -> None:
    """Port overrides reach the runtime without being replaced by defaults."""
    with (
        mock.patch('kfp_local.cli.RuntimeAssets.load'),
        mock.patch('kfp_local.cli.run', return_value=0) as server,
        mock.patch(
            'kfp_local.cli.importlib.import_module',
            return_value=types.SimpleNamespace(LOCAL_CLI_PROTOCOL_VERSION=1),
        ),
    ):
        assert (
            main(
                [
                    'ui',
                    '--data-dir',
                    str(tmp_path),
                    '--ui-port',
                    '3100',
                    '--api-port',
                    '3102',
                    '--rpc-port',
                    '3103',
                ]
            )
            == 0
        )
    config = server.call_args.args[0]
    assert config.ui
    assert config.ui_port == 3100
    assert config.api_port == 3102
    assert config.rpc_port == 3103


def test_cleanup_routes_retention_without_server_options(
    tmp_path: Path,
) -> None:
    """The cleanup command accepts retention but does not launch services."""
    with (
        mock.patch('kfp_local.cli.RuntimeAssets.load'),
        mock.patch('kfp_local.cli.run') as server,
        mock.patch('kfp_local.cli.cleanup', return_value=5) as cleanup,
        mock.patch(
            'kfp_local.cli.importlib.import_module',
            return_value=types.SimpleNamespace(LOCAL_CLI_PROTOCOL_VERSION=1),
        ),
    ):
        assert (
            main(['cleanup', '--data-dir', str(tmp_path), '--keep-workspaces'])
            == 5
        )
    server.assert_not_called()
    config = cleanup.call_args.args[0]
    assert config.data_dir == tmp_path
    assert cleanup.call_args.kwargs['keep_workspaces']
