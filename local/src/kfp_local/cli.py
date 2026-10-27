"""Command entrypoints for the optional native-wheel proof of concept."""

import argparse
import importlib
import sys
from pathlib import Path
from typing import Sequence

from kfp_local.assets import LocalRuntimeError, RuntimeAssets
from kfp_local.runtime import LocalConfig, cleanup, default_data_dir, run

CLI_PROTOCOL_VERSION = 1


def main(argv: Sequence[str] | None = None) -> int:
    """Start local services or reconcile a stopped installation."""
    parser = argparse.ArgumentParser(prog='kfp local')
    commands = parser.add_subparsers(dest='command', required=True)
    for name in ('server', 'ui'):
        command = commands.add_parser(name)
        command.add_argument(
            '--data-dir', type=Path, default=default_data_dir()
        )
        command.add_argument('--api-port', type=int, default=3002)
        command.add_argument('--rpc-port', type=int, default=3003)
        if name == 'ui':
            command.add_argument('--ui-port', type=int, default=3000)
        command.add_argument('--startup-timeout', type=float, default=60)
    command = commands.add_parser(
        'cleanup', help='Clean terminal resources without starting services.'
    )
    command.add_argument('--data-dir', type=Path, default=default_data_dir())
    command.add_argument(
        '--keep-workspaces',
        action='store_true',
        help='Retain pipeline workspaces while cleaning execution resources.',
    )
    args = parser.parse_args(argv)
    if getattr(args, 'startup_timeout', 60) <= 0:
        parser.error('--startup-timeout must be positive')
    config = LocalConfig(
        data_dir=args.data_dir.expanduser().resolve(),
        api_port=getattr(args, 'api_port', 3002),
        rpc_port=getattr(args, 'rpc_port', 3003),
        ui_port=getattr(args, 'ui_port', 3000),
        ui=args.command == 'ui',
        startup_timeout=getattr(args, 'startup_timeout', 60),
    )
    try:
        try:
            sdk = importlib.import_module('kfp.cli.local')
        except ModuleNotFoundError as exc:
            if exc.name not in ('kfp', 'kfp.cli.local'):
                raise
            raise LocalRuntimeError(
                'Install the matching SDK wheel built from this checkout; '
                'the previously published SDK does not have the local CLI.'
            ) from exc
        if (
            getattr(sdk, 'LOCAL_CLI_PROTOCOL_VERSION', None)
            != CLI_PROTOCOL_VERSION
        ):
            raise LocalRuntimeError(
                'The installed SDK/local CLI versions are incompatible.'
            )
        assets = RuntimeAssets.load()
        if args.command == 'cleanup':
            return cleanup(config, assets, keep_workspaces=args.keep_workspaces)
        return run(config, assets)
    except (LocalRuntimeError, OSError) as exc:
        print(f'Error: {exc}', file=sys.stderr)
        return 1


def entrypoint() -> None:
    """Expose ``kfp-local`` for direct invocation and installation
    diagnostics."""
    raise SystemExit(main())
