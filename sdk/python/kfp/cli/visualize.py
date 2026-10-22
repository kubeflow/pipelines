# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Export and watch local pipelines without a KFP server."""

import json
import os
import secrets
import subprocess
import tempfile
import threading
import time
from contextlib import contextmanager
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

import click
import yaml
from google.protobuf.json_format import ParseError

from kfp.cli import visualization


def _atomic_write(output: Path, content: str) -> None:
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(
            mode='w',
            encoding='utf-8',
            dir=output.parent,
            prefix=f'.{output.name}.',
            delete=False,
        ) as stream:
            temporary = Path(stream.name)
            stream.write(content)
        temporary.replace(output)
    finally:
        if temporary:
            temporary.unlink(missing_ok=True)


def _fingerprint(source: Path) -> tuple:
    if source.suffix.lower() == '.py':
        paths = []
        for directory, folders, files in os.walk(source.parent):
            folders[:] = [
                name
                for name in folders
                if not name.startswith('.')
                and name not in ('__pycache__', 'node_modules', 'venv')
            ]
            paths.extend(
                Path(directory) / name for name in files if name.endswith('.py')
            )
    else:
        paths = [source]
    fingerprint = []
    for path in sorted(paths):
        try:
            stat = path.stat()
            fingerprint.append((str(path), stat.st_mtime_ns, stat.st_size))
        except FileNotFoundError:
            pass  # Atomic editor saves can briefly remove a watched path.
    return tuple(fingerprint)


@contextmanager
def _live_server(state: dict[str, Any]):
    token = secrets.token_urlsafe(24)
    prefix = f'/{token}/'

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            if (
                self.headers.get('Host')
                != f'127.0.0.1:{self.server.server_port}'
            ):
                self.send_error(403)
                return
            if self.path == prefix:
                content = visualization.render_html(state['current'], live=True)
                content_type = 'text/html; charset=utf-8'
            elif self.path == prefix + 'state':
                content = json.dumps(state['current'])
                content_type = 'application/json'
            else:
                self.send_error(404)
                return
            body = content.encode('utf-8')
            self.send_response(200)
            self.send_header('Content-Type', content_type)
            self.send_header('Content-Length', str(len(body)))
            self.send_header('Cache-Control', 'no-store')
            self.send_header('X-Content-Type-Options', 'nosniff')
            self.send_header('Referrer-Policy', 'no-referrer')
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *_args):
            pass

    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield f'http://127.0.0.1:{server.server_port}{prefix}'
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


def _export(
    source: Path,
    function: str | None,
    output: Path,
    format_: str,
    state: dict[str, Any],
) -> None:
    spec = visualization.load_spec(source, function)
    view = visualization.build_view(spec)
    current = {
        'spec': spec,
        'error': None,
        'revision': state['current']['revision'] + 1,
    }
    content = (
        visualization.render_html(current)
        if format_ == 'html'
        else visualization.render_markdown(view)
    )
    _atomic_write(output, content)
    state['current'] = current
    click.echo(f'Wrote {output}')


def _update(
    source: Path,
    function: str | None,
    output: Path,
    format_: str,
    state: dict[str, Any],
    watch: bool,
) -> None:
    try:
        _export(source, function, output, format_, state)
    except (
        OSError,
        ValueError,
        ParseError,
        yaml.YAMLError,
        subprocess.TimeoutExpired,
        RecursionError,
    ) as error:
        message = str(error)
        if not watch:
            raise click.ClickException(message) from error
        state['current'] = {
            **state['current'],
            'error': message,
            'revision': state['current']['revision'] + 1,
        }
        click.echo(
            f'Preview not updated: {message}\n'
            'Keeping the last successful export; waiting for changes.',
            err=True,
        )


def _watch(
    source: Path,
    function: str | None,
    output: Path,
    format_: str,
    state: dict[str, Any],
    previous: tuple,
) -> None:
    pending = previous
    while True:
        time.sleep(0.3)
        current = _fingerprint(source)
        # Wait for a quiet poll before compiling an editor's burst of writes.
        if current != previous and current == pending:
            _update(source, function, output, format_, state, watch=True)
            previous = current
        pending = current


@click.command()
@click.argument(
    'source', type=click.Path(exists=True, dir_okay=False, path_type=Path)
)
@click.option(
    '-o',
    '--output',
    type=click.Path(dir_okay=False, path_type=Path),
    help='Export path. Defaults to the input stem plus .html or .md.',
)
@click.option(
    '--format',
    'format_',
    type=click.Choice(['html', 'markdown']),
    help='HTML (React Flow) or Markdown (Mermaid). Inferred from '
    '--output; defaults to HTML.',
)
@click.option(
    '--function',
    help='Pipeline function to compile for Python inputs '
    'containing more than one pipeline.',
)
@click.option(
    '-w',
    '--watch',
    is_flag=True,
    help='Re-export on saved changes. HTML '
    'also serves a live viewer on a loopback URL.',
)
def visualize(
    source: Path,
    output: Path | None,
    format_: str | None,
    function: str | None,
    watch: bool,
) -> None:
    """Visualize a local .py, .yaml, .yml, or .json pipeline.

    HTML exports are offline, interactive React Flow diagrams. Markdown exports
    contain Mermaid diagrams. Neither requires a cluster. Python compilation
    executes module-level code: only visualize trusted Python sources.
    """
    source = source.resolve()
    if source.suffix.lower() not in ('.py', '.yaml', '.yml', '.json'):
        raise click.UsageError(
            'Input must be a .py, .yaml, .yml, or .json file.'
        )
    if function and source.suffix.lower() != '.py':
        raise click.UsageError('--function is only valid for Python inputs.')
    suffixes = {'.html': 'html', '.md': 'markdown'}
    if output:
        inferred = suffixes.get(output.suffix.lower())
        if inferred is None:
            raise click.UsageError('Output must end in .html or .md.')
        if format_ and inferred != format_:
            raise click.UsageError(
                '--format conflicts with the output extension.'
            )
        format_ = inferred
    format_ = format_ or 'html'
    output = (
        output or source.with_suffix('.html' if format_ == 'html' else '.md')
    ).resolve()
    if output == source:
        raise click.UsageError('Output must not overwrite the input file.')
    if not output.parent.is_dir():
        raise click.UsageError(
            'Output directory does not exist. Create it first.'
        )

    if source.suffix.lower() == '.py':
        click.echo(
            'Compiling trusted Python source (module-level code executes).'
        )
    state = {'current': {'spec': None, 'error': None, 'revision': 0}}
    previous = _fingerprint(source)
    _update(source, function, output, format_, state, watch)
    if watch:
        click.echo('Watching saved changes. Press Ctrl+C to stop.')
        try:
            if format_ == 'html':
                with _live_server(state) as url:
                    click.echo(f'Live viewer: {url}')
                    _watch(source, function, output, format_, state, previous)
            else:
                _watch(source, function, output, format_, state, previous)
        except KeyboardInterrupt:
            click.echo('\nStopped watching.')
