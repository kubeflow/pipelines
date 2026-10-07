"""Require generated SDK bindings when building distributable artifacts."""

from pathlib import Path
from typing import Any

from hatchling.builders.hooks.plugin.interface import BuildHookInterface


class CustomBuildHook(BuildHookInterface):
    """Require checked-in bindings without changing editable CI setup."""

    def initialize(self, version: str, build_data: dict[str, Any]) -> None:
        """Fail incomplete builds and include bindings in wheels and sdists."""
        # CI installs the editable SDK before running the explicit generators.
        if self.target_name == 'wheel' and version == 'editable':
            return
        generated = (
            'kfp/pipeline_spec/pipeline_spec_pb2.py',
            'kfp/kubernetes/kubernetes_executor_config_pb2.py',
            'kfp/server_api/__init__.py',
        )
        for relative_path in generated:
            source = Path(self.root) / relative_path
            if not source.is_file():
                raise RuntimeError(
                    f'Missing generated SDK binding: {relative_path}. '
                    'Run "make -C sdk python" from the repository root before '
                    'building distributions.')
            build_data['force_include'][str(source)] = relative_path
