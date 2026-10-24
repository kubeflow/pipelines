"""Validate installed payload shape and native compatibility diagnostics."""

import json
import platform
from pathlib import Path

import pytest
from kfp_local.assets import LocalRuntimeError, RuntimeAssets


def native_assets(root: Path) -> Path:
    """Make a small valid payload without pretending it is a real native
    binary."""
    libc, version = platform.libc_ver()
    manifest = {
        'version': 1,
        'system': platform.system(),
        'machine': platform.machine(),
        'libc': libc,
        'libc_version': version,
        'node_version': '24.22.1',
    }
    root.mkdir(exist_ok=True)
    (root / 'manifest.json').write_text(json.dumps(manifest))
    for relative in (
        'bin/apiserver',
        'config/config.json',
        'ui/server/dist/server.js',
        'ui/build/index.html',
        'ui/server/node_modules/express/package.json',
    ):
        path = root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text('{}')
    (root / 'bin/apiserver').chmod(0o755)
    return root


def test_native_assets_are_required(tmp_path: Path) -> None:
    """Editable/source installs must not masquerade as packaged servers."""
    with pytest.raises(LocalRuntimeError, match='assets are missing'):
        RuntimeAssets.load(tmp_path)


def test_platform_and_executable_validation(tmp_path: Path) -> None:
    """Reject wheels for another architecture and non-executable payloads."""
    native_assets(tmp_path)
    assets = RuntimeAssets.load(tmp_path)
    assert assets.api_binary == tmp_path / 'bin/apiserver'
    assets.api_binary.chmod(0o644)
    with pytest.raises(LocalRuntimeError, match='not executable'):
        RuntimeAssets.load(tmp_path)
    assets.api_binary.chmod(0o755)
    manifest = json.loads((tmp_path / 'manifest.json').read_text())
    manifest['machine'] = 'another-machine'
    (tmp_path / 'manifest.json').write_text(json.dumps(manifest))
    with pytest.raises(LocalRuntimeError, match='OS/architecture'):
        RuntimeAssets.load(tmp_path)


@pytest.mark.parametrize('content', ['[]', '{"version":1}', 'invalid'])
def test_malformed_manifest_is_actionable(tmp_path: Path, content: str) -> None:
    """Malformed installation metadata must produce a package error."""
    (tmp_path / 'manifest.json').write_text(content)
    with pytest.raises(LocalRuntimeError):
        RuntimeAssets.load(tmp_path)
