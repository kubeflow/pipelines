"""Locate and validate native runtime assets installed from a platform
wheel."""

import json
import os
import platform
from dataclasses import dataclass
from pathlib import Path


class LocalRuntimeError(Exception):
    """An actionable local installation or runtime failure."""


@dataclass(frozen=True)
class RuntimeAssets:
    """Validated files and compatibility metadata from the installed wheel."""

    root: Path
    node_version: tuple[int, int, int]

    @property
    def api_binary(self) -> Path:
        """Return the native API server executable."""
        return self.root / 'bin' / 'apiserver'

    @property
    def config(self) -> Path:
        """Return the bundled API configuration directory."""
        return self.root / 'config'

    @property
    def ui_server(self) -> Path:
        """Return the compiled Node.js server entrypoint."""
        return self.root / 'ui' / 'server' / 'dist' / 'server.js'

    @property
    def ui_static(self) -> Path:
        """Return the compiled browser application."""
        return self.root / 'ui' / 'build'

    @classmethod
    def load(cls, root: Path | None = None) -> 'RuntimeAssets':
        """Reject incomplete or incompatible native installations
        explicitly."""
        root = root or Path(__file__).resolve().parent / '_assets'
        try:
            manifest = json.loads((root / 'manifest.json').read_text())
        except (OSError, ValueError) as exc:
            raise LocalRuntimeError(
                'Packaged runtime assets are missing or invalid. Build a native '
                'wheel with "python local/build.py" from the checkout; do not '
                'install this proof of concept as an editable package.'
            ) from exc
        if not isinstance(manifest, dict) or not all(
            isinstance(manifest.get(key), str)
            for key in (
                'system',
                'machine',
                'libc',
                'libc_version',
                'node_version',
            )
        ):
            raise LocalRuntimeError('Invalid runtime compatibility manifest.')
        if (
            manifest.get('version') != 1
            or manifest.get('system') != platform.system()
            or manifest.get('machine') != platform.machine()
        ):
            raise LocalRuntimeError(
                'This runtime wheel does not match the current OS/architecture.'
            )
        libc, libc_version = platform.libc_ver()
        try:
            compatible_libc = manifest['libc'] == libc and _version(
                libc_version
            ) >= _version(manifest['libc_version'])
            node_version = _version(manifest['node_version'])
        except ValueError as exc:
            raise LocalRuntimeError(
                'Invalid runtime compatibility version.'
            ) from exc
        if not compatible_libc:
            raise LocalRuntimeError(
                'This experimental native wheel requires the builder system '
                f'library: {manifest.get("libc")} '
                f'{manifest.get("libc_version")} or newer.'
            )
        assets = cls(root=root, node_version=node_version)
        for path in (
            assets.api_binary,
            assets.config / 'config.json',
            assets.ui_server,
            assets.ui_static / 'index.html',
            root
            / 'ui'
            / 'server'
            / 'node_modules'
            / 'express'
            / 'package.json',
        ):
            if not path.is_file():
                raise LocalRuntimeError(
                    f'Packaged runtime asset is missing: {path}'
                )
        if not os.access(assets.api_binary, os.X_OK):
            raise LocalRuntimeError(
                'The packaged API server is not executable.'
            )
        return assets


def _version(value: str) -> tuple[int, int, int]:
    parts = value.removeprefix('v').split('.')
    if (
        not value
        or len(parts) > 3
        or not all(part.isdecimal() for part in parts)
    ):
        raise ValueError(f'Invalid numeric version: {value!r}')
    numbers = [int(part) for part in parts]
    numbers.extend([0] * (3 - len(numbers)))
    return numbers[0], numbers[1], numbers[2]
