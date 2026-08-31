# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Format-aware metadata discovery for repository Go version policy."""

from functools import lru_cache
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
from typing import Dict, Iterable, List, Optional, Tuple

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
HELPER_TEMP_DIRECTORY = tempfile.TemporaryDirectory(
    prefix='kfp-go-version-metadata-')
GO_TEXT_REFERENCE_PATTERN = re.compile(
    r'(?:\bgolang(?=[:@])|(?:dl\.google\.com/go/|go\.dev/dl/)go)',
    re.IGNORECASE,
)
SETUP_GO_USE_PREFIX = 'actions/setup-go@'
DOCKER_USE_PREFIX = 'docker://'
INTENTIONALLY_INVALID_YAML_PATHS = {
    Path('backend/src/crd/samples/scheduledworkflow/invalid.yaml'),
}
METADATA_BUILD_TIMEOUT_SECONDS = 120
METADATA_INSPECTION_TIMEOUT_SECONDS = 10
MAX_METADATA_INPUT_BYTES = 4 * 1024 * 1024
DOCKER_CLASSIFICATIONS = {'managed', 'unsupported', 'irrelevant', 'invalid'}


def is_container_recipe(relative_path: Path) -> bool:
    return (relative_path.name.startswith('Dockerfile') or
            relative_path.name.startswith('Containerfile'))


@lru_cache(maxsize=1)
def _helper_binary() -> Path:
    binary = Path(HELPER_TEMP_DIRECTORY.name) / 'go-version-metadata'
    build_environment = os.environ.copy()
    build_environment.setdefault(
        'GOCACHE', str(Path(tempfile.gettempdir()) / 'kfp-go-build-cache'))
    try:
        subprocess.run(
            ('go', 'build', '-o', str(binary), './tools/go-version-metadata'),
            cwd=REPOSITORY_ROOT,
            env=build_environment,
            check=True,
            capture_output=True,
            text=True,
            timeout=METADATA_BUILD_TIMEOUT_SECONDS,
        )
    except subprocess.TimeoutExpired as error:
        raise RuntimeError(
            'timed out building the Go metadata helper after '
            f'{METADATA_BUILD_TIMEOUT_SECONDS} seconds') from error
    except (FileNotFoundError, subprocess.CalledProcessError) as error:
        detail = getattr(error, 'stderr', '') or str(error)
        raise RuntimeError(
            f'could not build Go metadata helper: {detail.strip()}') from error
    return binary


@lru_cache(maxsize=512)
def inspect_metadata(relative_path: Path, contents: str) -> Dict:
    if len(contents.encode('utf-8')) > MAX_METADATA_INPUT_BYTES:
        raise RuntimeError(
            f'{relative_path} exceeds the {MAX_METADATA_INPUT_BYTES}-byte '
            'metadata input limit')
    try:
        result = subprocess.run(
            (str(_helper_binary()),),
            input=json.dumps({
                'path': str(relative_path),
                'contents': contents,
            }),
            check=True,
            capture_output=True,
            text=True,
            timeout=METADATA_INSPECTION_TIMEOUT_SECONDS,
        )
    except subprocess.TimeoutExpired as error:
        raise RuntimeError(
            f'Go metadata inspection for {relative_path} timed out after '
            f'{METADATA_INSPECTION_TIMEOUT_SECONDS} seconds') from error
    except subprocess.CalledProcessError as error:
        detail = error.stderr.strip()
        if detail.startswith('resource limit:'):
            raise RuntimeError(
                f'Go metadata inspection for {relative_path} exceeded a '
                f'{detail}') from error
        raise ValueError(detail or f'could not parse {relative_path}') \
            from error
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as error:
        raise RuntimeError(
            f'Go metadata helper returned invalid JSON for {relative_path}') \
            from error


def module_versions(contents: str,
                    relative_path: Path) -> Tuple[str, Optional[str]]:
    try:
        module = inspect_metadata(relative_path, contents).get('module')
    except ValueError as error:
        message = str(error)
        directive = ('toolchain' if 'toolchain' in message.lower() else 'go')
        raise ValueError(
            f'{relative_path} contains an invalid {directive} directive: '
            f'{message}') from error
    if module is None:
        raise ValueError(f'{relative_path} is not a Go module')
    return module['go'], module.get('toolchain')


def yaml_mapping_values(contents: str,
                        target_keys: Iterable[str]) -> Dict[str, List[str]]:
    values = inspect_metadata(Path('metadata.yaml'), contents).get(
        'yamlValues', {})
    return {key: list(values.get(key, [])) for key in target_keys}


def docker_runtime_classification(contents: str) -> Dict:
    metadata = inspect_metadata(Path('Dockerfile'), contents)
    classification = metadata.get('dockerClassification')
    if classification not in DOCKER_CLASSIFICATIONS:
        raise RuntimeError(
            f'Go metadata helper returned invalid Docker classification '
            f'{classification!r}')
    candidates = list(metadata.get('dockerCandidates', []))
    for candidate in candidates:
        candidate.setdefault('flavor', '')
    return {
        'classification': classification,
        'candidates': candidates,
        'error': metadata.get('dockerError', ''),
    }


def has_go_runtime_reference(relative_path: Path, contents: str) -> bool:
    if relative_path.suffix.lower() in {'.yaml', '.yml'}:
        if relative_path in INTENTIONALLY_INVALID_YAML_PATHS:
            return False
        metadata = inspect_metadata(relative_path, contents)
        values = metadata.get('yamlValues', {})
        if any(_is_golang_image(value)
               for key in ('container', 'image')
               for value in values.get(key, [])):
            return True
        if any(value.casefold().startswith(DOCKER_USE_PREFIX) and
               _is_golang_image(value)
               for value in values.get('uses', [])):
            return True
        return bool(metadata.get('hasGoDownload'))

    if is_container_recipe(relative_path):
        return docker_runtime_classification(
            contents)['classification'] != 'irrelevant'

    active_text = '\n'.join(
        line for line in contents.splitlines()
        if not line.lstrip().startswith('#'))
    return GO_TEXT_REFERENCE_PATTERN.search(active_text) is not None


def has_setup_go_use(contents: str) -> bool:
    values = yaml_mapping_values(contents, ('uses',))['uses']
    return any(value.casefold().startswith(SETUP_GO_USE_PREFIX)
               for value in values)


def _is_golang_image(value: str) -> bool:
    image = value.strip()
    if image.casefold().startswith('docker://'):
        image = image[len('docker://'):]
    name = image.rsplit('/', 1)[-1]
    return re.match(r'^golang(?=[:@]|$)', name, re.IGNORECASE) is not None
