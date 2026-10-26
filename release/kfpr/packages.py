"""Validate legacy release wheels before committing version updates."""

import re
import sys
import zipfile
from email.parser import BytesParser
from pathlib import Path

LEGACY_PACKAGES = (
    (
        'api/v2alpha1/python',
        'kfp-pipeline-spec',
        ('kfp/pipeline_spec/pipeline_spec_pb2.py',),
    ),
    (
        'backend/api/v2beta1/python_http_client',
        'kfp-server-api',
        ('kfp_server_api/api_client.py', 'kfp_server_api/configuration.py'),
    ),
    ('sdk/python', 'kfp', ('kfp/version.py', 'kfp/compiler/compiler.py')),
    (
        'kubernetes_platform/python',
        'kfp-kubernetes',
        ('kfp/kubernetes/kubernetes_executor_config_pb2.py',),
    ),
)


def clean_legacy_distributions(root: Path) -> None:
    """Remove only these packages' old wheels and sdists before wildcard
    builds."""
    for directory, name, _ in LEGACY_PACKAGES:
        names = '|'.join(
            re.escape(value) for value in {name, name.replace('-', '_')}
        )
        artifact = re.compile(rf'(?:{names})-[0-9].*\.(?:whl|tar\.gz|zip)')
        dist = root / directory / 'dist'
        if not dist.is_dir():
            continue
        for path in dist.iterdir():
            if artifact.fullmatch(path.name) and path.is_file():
                path.unlink()


def validate_legacy_wheels(root: Path, version: str) -> None:
    """Reject missing or incomplete release artifacts, even if a build
    succeeded."""
    for directory, name, modules in LEGACY_PACKAGES:
        normalized = name.replace('-', '_')
        wheels = list(
            (root / directory / 'dist').glob(f'{normalized}-{version}-*.whl')
        )
        if len(wheels) != 1:
            raise ValueError(
                f'Expected one {name} {version} wheel; found {len(wheels)}'
            )
        with zipfile.ZipFile(wheels[0]) as wheel:
            metadata_path = f'{normalized}-{version}.dist-info/METADATA'
            metadata = BytesParser().parsebytes(wheel.read(metadata_path))
            if metadata['Name'] != name or metadata['Version'] != version:
                raise ValueError(f'Incorrect release metadata in {wheels[0]}')
            for module in modules:
                if module not in wheel.namelist():
                    raise ValueError(
                        f'{wheels[0]} is missing {module}; regenerate bindings and rebuild'
                    )
                compile(wheel.read(module), module, 'exec')


if __name__ == '__main__':
    if sys.argv[1] == 'clean':
        clean_legacy_distributions(Path(sys.argv[2]))
    else:
        validate_legacy_wheels(Path(sys.argv[1]), sys.argv[2])
