"""Build only native wheels containing explicitly prepared runtime assets."""

import json
import sysconfig
from pathlib import Path
from typing import Any

from hatchling.builders.hooks.plugin.interface import BuildHookInterface


class CustomBuildHook(BuildHookInterface):
    """Reject source-only builds and avoid incorrectly universal wheel tags."""

    def initialize(self, version: str, build_data: dict[str, Any]) -> None:
        """Include built Go, Node, and browser payloads as platform assets."""
        if self.target_name != 'wheel' or version == 'editable':
            raise RuntimeError(
                'This native-wheel proof must be built with '
                '"python local/build.py" from a complete checkout.'
            )
        assets = Path(self.root) / 'src' / 'kfp_local' / '_assets'
        manifest_path = assets / 'manifest.json'
        if not manifest_path.is_file():
            raise RuntimeError(
                'Native assets are missing. Run "python local/build.py"; '
                'install the resulting wheel instead of this source directory.'
            )
        manifest = json.loads(manifest_path.read_text())
        platform_tag = (
            sysconfig.get_platform().replace('-', '_').replace('.', '_')
        )
        tag = f'py3-none-{platform_tag}'
        if manifest.get('wheel_tag') != tag:
            raise RuntimeError(
                'Prepared assets do not match the build platform.'
            )
        build_data['pure_python'] = False
        build_data['tag'] = tag
        build_data['force_include'][str(assets)] = 'kfp_local/_assets'
