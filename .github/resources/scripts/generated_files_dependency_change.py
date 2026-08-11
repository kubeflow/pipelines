#!/usr/bin/env python3
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
"""Detect dependency-only changes that require generated-file validation."""

import argparse
from pathlib import PurePosixPath
import subprocess
import sys


TRACKED_MODULES = {
    'go.mod': (
        'github.com/grpc-ecosystem/grpc-gateway/v2',
        'google.golang.org/grpc/cmd/protoc-gen-go-grpc',
        'google.golang.org/protobuf',
    ),
    'backend/api/tools/go.mod': (
        'github.com/go-swagger/go-swagger',
    ),
}

GENERATED_PATH_PREFIXES = (
    'api/v2alpha1/go/',
    'backend/api/v1beta1/go_client/',
    'backend/api/v1beta1/go_http_client/',
    'backend/api/v1beta1/python_http_client/',
    'backend/api/v1beta1/swagger/',
    'backend/api/v2beta1/go_client/',
    'backend/api/v2beta1/go_http_client/',
    'backend/api/v2beta1/python_http_client/',
    'backend/api/v2beta1/swagger/',
    'kubernetes_platform/go/',
    'kubernetes_platform/python/kfp/kubernetes/',
)


def module_version(go_mod: str, module: str) -> str | None:
    """Return a required module version from go.mod text."""
    for line in go_mod.splitlines():
        fields = line.split()
        if len(fields) >= 2 and fields[0] == module:
            return fields[1]
        if len(fields) >= 3 and fields[:2] == ['require', module]:
            return fields[2]
    return None


def is_go_module_metadata(path: str) -> bool:
    return PurePosixPath(path).name in {'go.mod', 'go.sum'}


def requires_validation(
    changed_paths: list[str],
    base_manifests: dict[str, str],
    head_manifests: dict[str, str],
) -> bool:
    """Return whether generator inputs or tracked tool versions changed."""
    if any(not is_go_module_metadata(path) for path in changed_paths):
        return True

    for manifest, modules in TRACKED_MODULES.items():
        for module in modules:
            if module_version(base_manifests[manifest], module) != module_version(
                head_manifests[manifest], module
            ):
                return True
    return False


def supports_auto_regeneration(
    changed_paths: list[str],
    base_manifests: dict[str, str],
    head_manifests: dict[str, str],
) -> bool:
    """Return whether a module-only change safely supports auto-regeneration."""
    if not changed_paths or any(
        not is_go_module_metadata(path) for path in changed_paths
    ):
        return False

    for manifest, modules in TRACKED_MODULES.items():
        for module in modules:
            base_version = module_version(base_manifests[manifest], module)
            head_version = module_version(head_manifests[manifest], module)
            if (
                base_version is not None
                and head_version is not None
                and base_version != head_version
            ):
                return True
    return False


def generated_paths_are_allowed(paths: list[str]) -> bool:
    """Return whether every path is a known committed generator output."""
    return bool(paths) and all(
        path.startswith(GENERATED_PATH_PREFIXES) for path in paths
    )


def git_output(*args: str) -> str:
    return subprocess.run(
        ('git', *args),
        check=True,
        stdout=subprocess.PIPE,
        text=True,
    ).stdout


def git_file_at_ref(ref: str, path: str) -> str:
    result = subprocess.run(
        ('git', 'show', f'{ref}:{path}'),
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
    )
    return result.stdout if result.returncode == 0 else ''


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--base')
    parser.add_argument('--head')
    parser.add_argument(
        '--mode',
        choices=('validation', 'auto-regeneration'),
        default='validation',
    )
    parser.add_argument(
        '--check-generated-paths',
        help='Validate newline-delimited generated paths from this file.',
    )
    args = parser.parse_args()

    if args.check_generated_paths:
        with open(args.check_generated_paths, encoding='utf-8') as path_file:
            paths = path_file.read().splitlines()
        if generated_paths_are_allowed(paths):
            return 0
        print(
            'Generated patch contains an empty or non-allowlisted path set:',
            *paths,
            sep='\n',
            file=sys.stderr,
        )
        return 1

    if not args.base or not args.head:
        parser.error('--base and --head are required for change detection')

    changed_paths = git_output(
        'diff', '--name-only', args.base, args.head
    ).splitlines()
    base_manifests = {
        path: git_file_at_ref(args.base, path) for path in TRACKED_MODULES
    }
    head_manifests = {
        path: git_file_at_ref(args.head, path) for path in TRACKED_MODULES
    }
    if args.mode == 'auto-regeneration':
        result = supports_auto_regeneration(
            changed_paths, base_manifests, head_manifests
        )
    else:
        result = requires_validation(
            changed_paths, base_manifests, head_manifests
        )
    print(str(result).lower())
    return 0


if __name__ == '__main__':
    sys.exit(main())
