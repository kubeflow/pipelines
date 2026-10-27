"""Lazily delegate local server commands to the optional native package."""

import importlib

import click

LOCAL_CLI_PROTOCOL_VERSION = 1


@click.group()
def local() -> None:
    """Run or clean up the optional local KFP installation."""


def _delegate(ctx: click.Context, command: str) -> None:
    try:
        module = importlib.import_module('kfp_local.cli')
    except ModuleNotFoundError as exc:
        if exc.name != 'kfp_local':
            raise
        raise click.ClickException(
            'The optional kfp-local package is not installed. Install it with '
            '"pip install kfp-local". For this unreleased proof of concept, '
            'build and install the matching SDK and native wheels using '
            'local/README.md in the source checkout.'
        ) from exc
    entrypoint = getattr(module, 'main', None)
    if getattr(
        module, 'CLI_PROTOCOL_VERSION', None
    ) != LOCAL_CLI_PROTOCOL_VERSION or not callable(entrypoint):
        raise click.ClickException(
            'The installed kfp-local package is incompatible with this SDK. '
            'Install the matching wheels from the same checkout.'
        )
    ctx.exit(entrypoint([command, *ctx.args]))


@local.command(
    context_settings={
        'ignore_unknown_options': True,
        'allow_extra_args': True,
    }
)
@click.pass_context
def server(ctx: click.Context) -> None:
    """Start the headless API server and worker.

    --api-port INTEGER: HTTP API port (default: 3002).

    --rpc-port INTEGER: RPC API port (default: 3003).

    Example: kfp local server --api-port 3102 --rpc-port 3103

    Runtime options are forwarded unchanged. Use kfp-local server --help
    for the optional package's complete option list.
    """
    _delegate(ctx, 'server')


@local.command(
    context_settings={
        'ignore_unknown_options': True,
        'allow_extra_args': True,
    }
)
@click.pass_context
def ui(ctx: click.Context) -> None:
    """Start the API server, worker, and browser UI.

    --ui-port INTEGER: Browser UI port (default: 3000).

    --api-port INTEGER: HTTP API port (default: 3002).

    --rpc-port INTEGER: RPC API port (default: 3003).

    Example: kfp local ui --ui-port 3100

    Runtime options are forwarded unchanged. Use kfp-local ui --help
    for the optional package's complete option list.
    """
    _delegate(ctx, 'ui')


@local.command(
    context_settings={
        'ignore_unknown_options': True,
        'allow_extra_args': True,
    }
)
@click.pass_context
def cleanup(ctx: click.Context) -> None:
    """Clean terminal resources in a stopped local installation.

    Retains published artifacts, logs, and history. Use
    ``kfp-local cleanup --help`` for retention options.
    """
    _delegate(ctx, 'cleanup')
