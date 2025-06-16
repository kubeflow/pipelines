import logging

import click
from kfp.ui.server import start_ui_server


@click.command()
@click.option('--port', default=3000, help='Port to run the UI server on')
@click.option('--host', default='localhost', help='Host to bind the server to')
@click.option(
    '--api-server',
    help='ML Pipeline API server address (e.g., http://localhost:8888). If not provided, the UI will run in local execution mode where pipelines are executed locally using kfp.local.'
)
def ui(port, host, api_server):
    """Start the Kubeflow Pipelines UI server.

    The UI can operate in two modes:
    1. Remote mode: When --api-server is provided, pipeline runs are submitted to a remote KFP cluster
    2. Local mode: When --api-server is not provided, pipelines are executed locally using kfp.local
    """
    logging.warn(
        "\033[91mThe local `kfp ui` is in early development and still considered alpha/experimental. \x1b[0m"
    )
    logging.warn(
        "\033[93mPlease report any bugs if you encounter them https://github.com/kubeflow/pipelines/issues/new?template=BUG_FRONTEND.md \033[93m"
    )
    start_ui_server(host=host, port=port, api_server=api_server)
