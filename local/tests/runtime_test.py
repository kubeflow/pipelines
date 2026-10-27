"""Exercise local orchestration without requiring Docker or native assets."""

import contextlib
import io
import socket
import subprocess
import threading
from pathlib import Path
from unittest import mock

import pytest
from kfp_local.assets import LocalRuntimeError, RuntimeAssets
from kfp_local.runtime import (
    Child,
    LocalConfig,
    _installation_lock,
    _stop,
    _wait_ready,
    check_requirements,
    child_environment,
    cleanup,
    run,
)


def test_environment_selects_persistent_local_storage(tmp_path: Path) -> None:
    """Ignore conflicting database/backend settings and bind both APIs
    locally."""
    config = LocalConfig(tmp_path / 'space and ? question')
    with mock.patch.dict(
        'os.environ',
        {
            'DBDRIVERNAME': 'mysql',
            'SCHEDULED_WORKFLOW_CONTROLLER_ENABLED': 'true',
        },
    ):
        env = child_environment(config)
    assert env['DBDRIVERNAME'] == 'sqlite'
    assert env['KFP_V2_RUNTIME_EXECUTOR'] == 'docker'
    assert env['SCHEDULED_WORKFLOW_CONTROLLER_ENABLED'] == 'false'
    assert env['UI_SERVER_HOST'] == '127.0.0.1'
    assert '%3F' in env['DBCONFIG_SQLITECONFIG_DATASOURCENAME']
    assert env['OBJECTSTORECONFIG_BUCKETURL'].startswith('file:///')
    assert Path(env['TMPDIR']).parent == config.data_dir


def test_headless_does_not_require_node(tmp_path: Path) -> None:
    """The server command needs Docker, not a frontend build/runtime
    toolchain."""
    assets = RuntimeAssets(tmp_path, (24, 22, 1))
    with (
        mock.patch(
            'kfp_local.runtime.shutil.which',
            side_effect=lambda name: '/docker' if name == 'docker' else None,
        ),
        mock.patch('kfp_local.runtime.subprocess.run'),
        mock.patch('kfp_local.runtime.socket.socket'),
    ):
        assert check_requirements(assets, LocalConfig(tmp_path)) is None
        with pytest.raises(LocalRuntimeError, match='Node.js is required'):
            check_requirements(assets, LocalConfig(tmp_path, ui=True))


def test_ports_must_be_distinct(tmp_path: Path) -> None:
    """Reject incompatible endpoint configuration before starting a child."""
    with pytest.raises(LocalRuntimeError, match='distinct'):
        check_requirements(
            RuntimeAssets(tmp_path, (24, 22, 1)),
            LocalConfig(tmp_path, api_port=3002, rpc_port=3002),
        )


def test_recent_tcp_connections_allow_immediate_restart(tmp_path: Path) -> None:
    """Match the real servers' reuse semantics without accepting live
    listeners."""
    with socket.socket() as server:
        server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        server.bind(('127.0.0.1', 0))
        port = server.getsockname()[1]
        server.listen()
        with socket.create_connection(('127.0.0.1', port)) as client:
            accepted, _ = server.accept()
            accepted.shutdown(socket.SHUT_WR)
            assert client.recv(1) == b''
            accepted.close()
    with socket.socket() as other:
        other.bind(('127.0.0.1', 0))
        rpc_port = other.getsockname()[1]
    with (
        mock.patch('kfp_local.runtime.shutil.which', return_value='/docker'),
        mock.patch('kfp_local.runtime.subprocess.run'),
    ):
        assert (
            check_requirements(
                RuntimeAssets(tmp_path, (24, 22, 1)),
                LocalConfig(tmp_path, api_port=port, rpc_port=rpc_port),
            )
            is None
        )


def test_existing_tcp_listener_is_not_replaced(tmp_path: Path) -> None:
    """The reuse check still rejects another process's actual service port."""
    with socket.socket() as server:
        server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        server.bind(('127.0.0.1', 0))
        server.listen()
        port = server.getsockname()[1]
        with pytest.raises(LocalRuntimeError, match='unavailable'):
            check_requirements(
                RuntimeAssets(tmp_path, (24, 22, 1)),
                LocalConfig(tmp_path, api_port=port),
            )


def test_installation_lock_is_released(tmp_path: Path) -> None:
    """A second launcher cannot share mutable metadata or workspace state."""
    with _installation_lock(tmp_path):
        with pytest.raises(LocalRuntimeError, match='already uses'):
            with _installation_lock(tmp_path):
                pytest.fail('a second launcher acquired the installation')
    with _installation_lock(tmp_path):
        pass


def test_early_exit_is_not_readiness(tmp_path: Path) -> None:
    """Even exit zero is a startup failure for a required background
    service."""
    process = mock.Mock()
    process.poll.return_value = 0
    child = Child('api', process, io.BytesIO(), tmp_path / 'api.log')
    with pytest.raises(LocalRuntimeError, match='exited during startup'):
        _wait_ready(child, 'http://127.0.0.1:1', 1, threading.Event())


def test_shutdown_reaps_term_resistant_children(tmp_path: Path) -> None:
    """Force-stop only recorded PIDs after the bounded graceful interval."""
    process = mock.Mock()
    process.poll.return_value = None
    process.wait.side_effect = [subprocess.TimeoutExpired('api', 10), 0]
    child = Child('api', process, io.BytesIO(), tmp_path / 'api.log')
    _stop([child])
    process.terminate.assert_called_once()
    process.kill.assert_called_once()
    assert process.wait.call_count == 2
    assert child.log.closed


def test_partial_startup_stops_started_child(tmp_path: Path) -> None:
    """UI startup failure must not leave the API/worker running."""
    process = mock.Mock()
    process.poll.return_value = None
    process.wait.return_value = 0
    log = io.BytesIO()
    assets = RuntimeAssets(tmp_path, (24, 22, 1))
    with (
        mock.patch(
            'kfp_local.runtime.check_requirements', return_value='/node'
        ),
        mock.patch(
            'kfp_local.runtime._signals', return_value=contextlib.nullcontext()
        ),
        mock.patch(
            'kfp_local.runtime._start',
            side_effect=[
                Child('api', process, log, tmp_path / 'api.log'),
                LocalRuntimeError('UI did not start'),
            ],
        ),
        mock.patch('kfp_local.runtime._wait_ready'),
    ):
        with pytest.raises(LocalRuntimeError, match='UI did not start'):
            run(LocalConfig(tmp_path, ui=True), assets)
    process.terminate.assert_called_once()
    assert log.closed


def test_cleanup_is_offline_and_preserves_failure_status(
    tmp_path: Path,
) -> None:
    """Cleanup needs neither Node, service ports, nor a running HTTP server."""
    (tmp_path / 'metadata.sqlite').touch()
    assets = RuntimeAssets(tmp_path, (24, 22, 1))
    with (
        mock.patch('kfp_local.runtime.check_requirements') as requirements,
        mock.patch('kfp_local.runtime._start') as start,
        mock.patch('kfp_local.runtime.socket.socket') as socket_factory,
        mock.patch(
            'kfp_local.runtime.subprocess.run',
            return_value=subprocess.CompletedProcess([], 3),
        ) as command,
    ):
        assert cleanup(LocalConfig(tmp_path, ui=True), assets, True) == 3
    requirements.assert_not_called()
    start.assert_not_called()
    socket_factory.assert_not_called()
    args = command.call_args.args[0]
    assert '--runtimeCleanup' in args
    assert '--runtimeCleanupKeepWorkspaces' in args
    assert not any('PortFlag' in arg for arg in args)
    assert (
        command.call_args.kwargs['env']['KFP_V2_RUNTIME_AUTO_EXECUTE']
        == 'false'
    )
    assert (tmp_path / 'metadata.sqlite').exists()


def test_cleanup_refuses_live_installation(tmp_path: Path) -> None:
    """An existing server's installation lock prevents any cleanup process."""
    (tmp_path / 'metadata.sqlite').touch()
    with (
        _installation_lock(tmp_path),
        mock.patch('kfp_local.runtime.subprocess.run') as command,
    ):
        with pytest.raises(LocalRuntimeError, match='already uses'):
            cleanup(LocalConfig(tmp_path), RuntimeAssets(tmp_path, (24, 22, 1)))
    command.assert_not_called()


def test_cleanup_does_not_create_an_installation(tmp_path: Path) -> None:
    """A mistyped data path must not initialize a new metadata database."""
    root = tmp_path / 'missing'
    with pytest.raises(LocalRuntimeError, match='metadata.sqlite'):
        cleanup(LocalConfig(root), RuntimeAssets(tmp_path, (24, 22, 1)))
    assert not root.exists()


def test_cleanup_rejects_remote_docker(tmp_path: Path) -> None:
    """Local paths cannot be pruned against a daemon on another host."""
    (tmp_path / 'metadata.sqlite').touch()
    with (
        mock.patch.dict('os.environ', {'DOCKER_HOST': 'tcp://remote:2376'}),
        mock.patch('kfp_local.runtime.subprocess.run') as command,
        pytest.raises(LocalRuntimeError, match='local Unix-socket'),
    ):
        cleanup(LocalConfig(tmp_path), RuntimeAssets(tmp_path, (24, 22, 1)))
    command.assert_not_called()
