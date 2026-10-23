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

from contextlib import contextmanager
from contextlib import nullcontext
from http.server import BaseHTTPRequestHandler
from http.server import ThreadingHTTPServer
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
import threading
import time
from typing import Any
import webbrowser

import click
from google.protobuf.json_format import ParseError
from kfp.cli import visualization
import yaml


def _output_paths(source: Path, output: Path, format_: str) -> dict[str, Path]:
    if format_ == 'both':
        paths = {
            'html': output.with_suffix('.html').resolve(),
            'markdown': output.with_suffix('.md').resolve(),
        }
    else:
        paths = {format_: output.resolve()}
    if len(set(paths.values())) != len(paths):
        raise ValueError('HTML and Markdown outputs must be distinct files.')
    for path in paths.values():
        if path == source.resolve():
            raise ValueError('Output must not overwrite the input file.')
        if path.is_dir():
            raise ValueError(f'Output is a directory: {path}. Choose a file.')
        if not path.parent.is_dir():
            raise ValueError(
                'Output directory does not exist. Create it first.')
    return paths


def _write_exports(exports: dict[Path, str]) -> None:
    temporary = []
    try:
        # Stage every artifact before replacing any existing output. Each rename
        # is atomic, but multiple files cannot form a filesystem transaction.
        for output, content in exports.items():
            with tempfile.NamedTemporaryFile(
                    mode='w',
                    encoding='utf-8',
                    dir=output.parent,
                    prefix=f'.{output.name}.',
                    delete=False,
            ) as stream:
                staged = Path(stream.name)
                temporary.append((staged, output))
                stream.write(content)
        for staged, output in temporary:
            staged.replace(output)
    finally:
        for staged, _ in temporary:
            staged.unlink(missing_ok=True)


def _fingerprint(source: Path) -> tuple:
    if source.suffix.lower() == '.py':
        paths = []
        for directory, folders, files in os.walk(source.parent):
            folders[:] = [
                name for name in folders
                if not name.startswith('.') and name not in ('__pycache__',
                                                             'node_modules',
                                                             'venv')
            ]
            paths.extend(
                Path(directory) / name
                for name in files
                if name.endswith('.py'))
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
def _live_server(state: dict[str, Any], *, live: bool = True):
    token = secrets.token_urlsafe(24)
    prefix = f'/{token}/'

    class Handler(BaseHTTPRequestHandler):

        def do_GET(self):
            if (self.headers.get('Host')
                    != f'127.0.0.1:{self.server.server_port}'):
                self.send_error(403)
                return
            if self.path == prefix:
                content = visualization.render_html(state['current'], live=live)
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


@contextmanager
def _launch_browser(url: str):
    """Launch without blocking watch updates or waiting for browser exit."""
    stopped = threading.Event()
    opener = webbrowser.open_new

    def launch():
        if stopped.is_set():
            return
        try:
            opened = opener(url)
        except (webbrowser.Error, OSError):
            opened = False
        if not opened and not stopped.is_set():
            click.echo(
                'Could not open a browser automatically. '
                'Open the URL above manually.',
                err=True,
            )

    try:
        threading.Thread(
            target=launch, name='kfp-preview-browser', daemon=True).start()
        yield
    finally:
        stopped.set()


def _export(
    source: Path,
    function: str | None,
    output: Path,
    format_: str,
    state: dict[str, Any],
) -> None:
    spec = visualization.load_spec(source, function)
    current = {
        'spec': spec,
        'error': None,
        'revision': state['current']['revision'] + 1,
    }
    paths = _output_paths(source, output, format_)
    exports = {}
    if 'markdown' in paths:
        exports[paths['markdown']] = visualization.render_markdown(
            visualization.build_view(spec))
    else:
        visualization.validate_spec(spec)
    if 'html' in paths:
        exports[paths['html']] = visualization.render_html(current)
    _write_exports(exports)
    state['current'] = current
    for path in paths.values():
        click.echo(f'Wrote {path}')


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
            'Keeping the last successful preview state; waiting for changes.',
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
    'source', type=click.Path(exists=True, dir_okay=False, path_type=Path))
@click.option(
    '-o',
    '--output',
    type=click.Path(dir_okay=False, path_type=Path),
    help='Export path; with --format both, a stem or .html/.md path for '
    'the pair. Defaults to the input stem plus .html or .md.',
)
@click.option(
    '--format',
    'format_',
    type=click.Choice(['html', 'both', 'markdown']),
    help='HTML (React Flow), Markdown (Mermaid), or both. Inferred from '
    '--output when omitted; defaults to HTML.',
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
    help='Re-export on saved changes without starting a server.',
)
@click.option(
    '-s',
    '--serve',
    is_flag=True,
    help='Serve an HTML preview and open it in a browser. Add --watch for '
    'live updates. Available with any export format.',
)
@click.option(
    '-n',
    '--no-open',
    is_flag=True,
    help='Do not open a browser when serving; the preview URL is still printed.',
)
def visualize(
    source: Path,
    output: Path | None,
    format_: str | None,
    function: str | None,
    watch: bool,
    serve: bool,
    no_open: bool,
) -> None:
    """Visualize a local .py, .yaml, .yml, or .json pipeline.

    HTML exports are offline, interactive React Flow diagrams. Markdown exports
    contain Mermaid diagrams. Neither requires a cluster. Python compilation
    executes module-level code: only visualize trusted Python sources.
    """
    source = source.resolve()
    if source.suffix.lower() not in ('.py', '.yaml', '.yml', '.json'):
        raise click.UsageError(
            'Input must be a .py, .yaml, .yml, or .json file.')
    if function and source.suffix.lower() != '.py':
        raise click.UsageError('--function is only valid for Python inputs.')
    suffixes = {'.html': 'html', '.md': 'markdown'}
    if output:
        inferred = suffixes.get(output.suffix.lower())
        if inferred is None and not (format_ == 'both' and not output.suffix):
            raise click.UsageError(
                'Output must end in .html or .md, or be a stem with --format both.'
            )
        if format_ and format_ not in (inferred, 'both'):
            raise click.UsageError(
                '--format conflicts with the output extension.')
        format_ = format_ or inferred
    format_ = format_ or 'html'
    output = output or source.with_suffix('.md' if format_ ==
                                          'markdown' else '.html')
    try:
        _output_paths(source, output, format_)
    except ValueError as error:
        raise click.UsageError(str(error)) from error

    if source.suffix.lower() == '.py':
        click.echo(
            'Compiling trusted Python source (module-level code executes).')
    state = {'current': {'spec': None, 'error': None, 'revision': 0}}
    previous = _fingerprint(source) if watch else ()
    _update(source, function, output, format_, state, watch)
    if watch:
        click.echo('Watching saved changes. Press Ctrl+C to stop.')
    if watch or serve:
        try:
            with (_live_server(state, live=watch)
                  if serve else nullcontext()) as url:
                if url:
                    label = 'Live viewer' if watch else 'Viewer'
                    click.echo(f'{label}: {url}')
                    click.echo('Press Ctrl+C to stop serving.')
                with (_launch_browser(url)
                      if url and not no_open else nullcontext()):
                    if watch:
                        _watch(source, function, output, format_, state,
                               previous)
                    else:
                        while True:
                            time.sleep(1)
        except KeyboardInterrupt:
            click.echo('\nStopped watching.' if watch else '\nStopped serving.')
