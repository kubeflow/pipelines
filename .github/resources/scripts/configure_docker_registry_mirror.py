#!/usr/bin/env python3

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
"""Idempotently add a registry mirror to Docker daemon configuration."""

import argparse
import json
import os
import stat
import tempfile
from pathlib import Path


def configure_registry_mirror(config_path: Path, mirror: str) -> bool:
    """Put mirror first in registry-mirrors while preserving other settings."""
    existing_config = (
        config_path.read_text(encoding='utf-8') if config_path.exists() else ''
    )
    if existing_config.strip():
        config = json.loads(existing_config)
    else:
        config = {}
    if not isinstance(config, dict):
        raise ValueError('Docker daemon configuration must be a JSON object')

    existing_mirrors = config.get('registry-mirrors', [])
    if not isinstance(existing_mirrors, list) or not all(
        isinstance(existing_mirror, str)
        for existing_mirror in existing_mirrors
    ):
        raise ValueError('registry-mirrors must be a list of strings')

    normalized_mirror = mirror.rstrip('/')
    configured_mirror = next(
        (
            existing_mirror
            for existing_mirror in existing_mirrors
            if existing_mirror.rstrip('/') == normalized_mirror
        ),
        mirror,
    )
    mirrors = [configured_mirror]
    mirrors.extend(
        existing_mirror
        for existing_mirror in existing_mirrors
        if existing_mirror.rstrip('/') != normalized_mirror
    )
    if existing_mirrors == mirrors:
        return False

    config['registry-mirrors'] = mirrors
    config_path.parent.mkdir(parents=True, exist_ok=True)
    mode = (
        stat.S_IMODE(config_path.stat().st_mode)
        if config_path.exists()
        else 0o644
    )
    temporary_path = None
    try:
        with tempfile.NamedTemporaryFile(
            mode='w',
            encoding='utf-8',
            dir=config_path.parent,
            prefix=f'.{config_path.name}.',
            delete=False,
        ) as temporary_file:
            temporary_path = Path(temporary_file.name)
            json.dump(config, temporary_file, indent=2, sort_keys=True)
            temporary_file.write('\n')
        os.chmod(temporary_path, mode)
        os.replace(temporary_path, config_path)
    finally:
        if temporary_path is not None and temporary_path.exists():
            temporary_path.unlink()
    return True


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--mirror', required=True)
    args = parser.parse_args()

    changed = configure_registry_mirror(args.config, args.mirror)
    print('changed' if changed else 'unchanged')


if __name__ == '__main__':
    main()
