"""Supervise the installed API/worker and optional UI without a source
checkout."""

import contextlib
import fcntl
import os
import shutil
import signal
import socket
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import BinaryIO, Generator
from urllib.parse import quote

from kfp_local.assets import LocalRuntimeError, RuntimeAssets, _version


@dataclass(frozen=True)
class LocalConfig:
    """Persistent installation paths and loopback-only service ports."""

    data_dir: Path
    api_port: int = 3002
    rpc_port: int = 3003
    ui_port: int = 3000
    ui: bool = False
    startup_timeout: float = 60

    @property
    def api_url(self) -> str:
        """Return the endpoint printed for SDK submission."""
        return f'http://127.0.0.1:{self.api_port}'

    @property
    def ui_url(self) -> str:
        """Return the optional browser endpoint."""
        return f'http://127.0.0.1:{self.ui_port}'


@dataclass
class Child:
    """One owned child process and its diagnostic file."""

    name: str
    process: subprocess.Popen[bytes]
    log: BinaryIO
    path: Path


def default_data_dir() -> Path:
    """Keep metadata and artifacts between launches in a user-owned
    directory."""
    return (
        Path(
            os.environ.get(
                'XDG_DATA_HOME', str(Path.home() / '.local' / 'share')
            )
        ).expanduser()
        / 'kfp-local'
    )


def check_requirements(
    assets: RuntimeAssets, config: LocalConfig
) -> str | None:
    """Validate local prerequisites before starting either service."""
    ports = [config.api_port, config.rpc_port]
    if config.ui:
        ports.append(config.ui_port)
    if len(set(ports)) != len(ports):
        raise LocalRuntimeError('API, RPC, and UI ports must be distinct.')
    for port in ports:
        if not 1 <= port <= 65535:
            raise LocalRuntimeError('Ports must be between 1 and 65535.')
        with socket.socket() as listener:
            listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            try:
                listener.bind(('127.0.0.1', port))
            except OSError as exc:
                raise LocalRuntimeError(
                    f'Loopback port {port} is unavailable; choose another port.'
                ) from exc
    docker = shutil.which('docker')
    if docker is None:
        raise LocalRuntimeError(
            'Docker CLI and a running local Docker daemon are required.'
        )
    docker_host = os.environ.get('DOCKER_HOST', '')
    if docker_host and not docker_host.startswith('unix://'):
        raise LocalRuntimeError(
            'This proof of concept requires a local Unix-socket Docker daemon.'
        )
    try:
        subprocess.run(
            [docker, 'info', '--format', '{{.ServerVersion}}'],
            check=True,
            capture_output=True,
            timeout=15,
        )
    except (OSError, subprocess.SubprocessError) as exc:
        raise LocalRuntimeError(
            'Cannot connect to the local Docker daemon.'
        ) from exc
    if not config.ui:
        return None
    node = shutil.which('node')
    if node is None:
        raise LocalRuntimeError(
            'Node.js is required for "kfp local ui"; use "kfp local server" '
            'for headless operation.'
        )
    try:
        result = subprocess.run(
            [node, '--version'],
            check=True,
            capture_output=True,
            text=True,
            timeout=10,
        )
        version = _version(result.stdout.strip())
    except (OSError, ValueError, subprocess.SubprocessError) as exc:
        raise LocalRuntimeError(
            'Cannot determine the Node.js version.'
        ) from exc
    if version[0] != assets.node_version[0] or version < assets.node_version:
        minimum = '.'.join(map(str, assets.node_version))
        raise LocalRuntimeError(
            f'Use Node.js {minimum} or newer within major {assets.node_version[0]}.'
        )
    return node


def child_environment(config: LocalConfig) -> dict[str, str]:
    """Select standalone SQLite/file storage without inheriting cluster
    settings."""
    root = config.data_dir
    artifacts = root / 'artifacts'
    database = quote(str(root / 'metadata.sqlite'), safe='/')
    env = dict(os.environ)
    env.update(
        {
            'KUBECONFIG': str(root / 'kubeconfig'),
            'TMPDIR': str(root / 'tmp'),
            'POD_NAMESPACE': 'kubeflow',
            'MULTIUSER': 'false',
            'INITCONNECTIONTIMEOUT': '30s',
            'DBDRIVERNAME': 'sqlite',
            'DBCONFIG_SQLITECONFIG_DATASOURCENAME': f'file:{database}?_busy_timeout=30000&_journal_mode=WAL',
            'OBJECTSTORECONFIG_BUCKETURL': artifacts.as_uri(),
            'KFP_DEFAULT_PIPELINE_ROOT': (
                artifacts / 'v2' / 'artifacts'
            ).as_uri(),
            'KFP_V2_RUNTIME_MODE': 'coordinator-poc',
            'KFP_V2_RUNTIME_EXECUTOR': 'docker',
            'KFP_V2_RUNTIME_AUTO_EXECUTE': 'true',
            'KFP_V2_RUNTIME_WORKSPACE_ROOT': str(root / 'runtime'),
            'KFP_V2_LOCAL_SDK_ROOT': '',
            'LOCAL_API_SERVER_SDK_PACKAGE_PATH': '',
            'LOCAL_API_SERVER': 'true',
            'KFP_V2_LOCAL_API_SERVER_ADDRESS': '127.0.0.1',
            'KFP_V2_LOCAL_API_SERVER_PORT': str(config.rpc_port),
            'ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_HOST': '127.0.0.1',
            'ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_PORT': '8889',
            'ML_PIPELINE_SERVICE_HOST': '127.0.0.1',
            'ML_PIPELINE_SERVICE_PORT': str(config.api_port),
            'ML_PIPELINE_SERVICE_SCHEME': 'http',
            'UI_SERVER_HOST': '127.0.0.1',
            'ARTIFACTS_LOCAL_ROOT': str(artifacts),
            'NO_PROXY': ','.join(
                filter(
                    None,
                    [
                        os.environ.get('NO_PROXY', ''),
                        '127.0.0.1',
                        'localhost',
                    ],
                )
            ),
        }
    )
    return env


def _start(
    name: str, args: list[str], config: LocalConfig, env: dict[str, str]
) -> Child:
    path = config.data_dir / 'logs' / f'{name}.log'
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
    log = os.fdopen(descriptor, 'ab')
    try:
        process = subprocess.Popen(
            args,
            cwd=config.data_dir,
            env=env,
            stdout=log,
            stderr=subprocess.STDOUT,
        )
    except OSError as exc:
        log.close()
        raise LocalRuntimeError(f'Cannot start {name}: {exc}') from exc
    return Child(name, process, log, path)


def _wait_ready(
    child: Child, url: str, timeout: float, stop: threading.Event
) -> None:
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    deadline = time.monotonic() + timeout
    last_error = 'no response'
    while time.monotonic() < deadline:
        if stop.is_set():
            raise LocalRuntimeError('Startup interrupted.')
        status = child.process.poll()
        if status is not None:
            raise LocalRuntimeError(
                f'{child.name} exited during startup ({status}); see {child.path}'
            )
        try:
            with opener.open(url, timeout=1) as response:
                if response.status == 200:
                    return
        except (urllib.error.URLError, TimeoutError, OSError) as exc:
            last_error = str(exc)
        stop.wait(0.1)
    raise LocalRuntimeError(
        f'{child.name} did not become ready: {last_error}; see {child.path}'
    )


def _stop(children: list[Child]) -> None:
    errors = []
    for child in reversed(children):
        try:
            if child.process.poll() is None:
                child.process.terminate()
        except ProcessLookupError:
            pass
    deadline = time.monotonic() + 10
    for child in reversed(children):
        try:
            child.process.wait(timeout=max(0, deadline - time.monotonic()))
        except subprocess.TimeoutExpired:
            print(
                f'{child.name} did not exit after SIGTERM; stopping owned '
                f'process {child.process.pid}',
                file=sys.stderr,
            )
            try:
                child.process.kill()
            except ProcessLookupError:
                pass
            try:
                child.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                errors.append(
                    f'cannot reap {child.name} PID {child.process.pid}'
                )
        finally:
            child.log.close()
    if errors:
        raise LocalRuntimeError('; '.join(errors))


@contextlib.contextmanager
def _installation_lock(root: Path) -> Generator[None, None, None]:
    with (root / '.launcher.lock').open('a') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise LocalRuntimeError(
                f'Another local launcher already uses {root}.'
            ) from exc
        yield


@contextlib.contextmanager
def _signals(stop: threading.Event) -> Generator[None, None, None]:
    previous = {}
    for number in (signal.SIGINT, signal.SIGTERM):
        previous[number] = signal.signal(number, lambda *_: stop.set())
    try:
        yield
    finally:
        for number, handler in previous.items():
            signal.signal(number, handler)


def _initialize_data(root: Path) -> None:
    """Write launcher-owned files only while holding the installation lock."""
    for name in ('artifacts', 'runtime', 'logs', 'tmp'):
        (root / name).mkdir(mode=0o700, exist_ok=True)
    (root / 'kubeconfig').write_text(
        'apiVersion: v1\nkind: Config\nclusters:\n'
        '- name: local\n  cluster:\n'
        '    server: https://127.0.0.1:65535\n'
        'contexts:\n- name: local\n  context:\n'
        '    cluster: local\n    user: local\n    namespace: kubeflow\n'
        'current-context: local\nusers:\n'
        '- name: local\n  user:\n    token: unused-local-placeholder\n'
    )


def run(config: LocalConfig, assets: RuntimeAssets) -> int:
    """Run in the foreground, retain data, and stop all owned children on
    exit."""
    node = check_requirements(assets, config)
    root = config.data_dir
    root.mkdir(mode=0o700, parents=True, exist_ok=True)
    stop = threading.Event()
    children: list[Child] = []
    with _installation_lock(root), _signals(stop):
        _initialize_data(root)
        try:
            env = child_environment(config)
            api = _start(
                'apiserver',
                [
                    str(assets.api_binary),
                    '--config',
                    str(assets.config),
                    '-logtostderr=true',
                    f'--httpPortFlag=127.0.0.1:{config.api_port}',
                    f'--rpcPortFlag=127.0.0.1:{config.rpc_port}',
                ],
                config,
                env,
            )
            children.append(api)
            _wait_ready(
                api,
                config.api_url + '/apis/v2/experiments?page_size=1',
                config.startup_timeout,
                stop,
            )
            if config.ui:
                assert node is not None
                ui = _start(
                    'ui',
                    [
                        node,
                        str(assets.ui_server),
                        str(assets.ui_static),
                        str(config.ui_port),
                    ],
                    config,
                    env,
                )
                children.append(ui)
                _wait_ready(
                    ui, config.ui_url + '/', config.startup_timeout, stop
                )
            print(f'KFP API: {config.api_url}', flush=True)
            if config.ui:
                print(f'KFP UI: {config.ui_url}', flush=True)
            print(f'Data and logs: {root}\nPress Ctrl+C to stop.', flush=True)
            while not stop.wait(0.2):
                for child in children:
                    status = child.process.poll()
                    if status is not None:
                        raise LocalRuntimeError(
                            f'{child.name} exited unexpectedly ({status}); '
                            f'see {child.path}'
                        )
        finally:
            _stop(children)
    return 0
