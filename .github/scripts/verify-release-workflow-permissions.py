#!/usr/bin/env python3
"""Check the release image callers against their reusable workflow contracts."""

from pathlib import Path
import re

from ruamel.yaml import YAML

WORKFLOWS = Path(__file__).resolve().parents[1] / 'workflows'
YAML_PARSER = YAML(typ='safe')


def load_workflow(name):
    return YAML_PARSER.load((WORKFLOWS / name).read_text(encoding='utf-8'))


def expect_equal(description, actual, expected):
    if actual != expected:
        raise AssertionError(f'{description}: expected {expected!r}, got {actual!r}')


def main():
    release = load_workflow('image-builds-release.yml')
    contracts = (
        (
            'build-images-for-release',
            'build-and-push.yml',
            'build-and-push-images',
            {
                'contents': 'read',
                'packages': 'write',
                'attestations': 'write',
                'id-token': 'write',
            },
        ),
        (
            'create-manifests',
            'create-manifest.yml',
            'create-manifest',
            {'contents': 'read', 'packages': 'write'},
        ),
    )

    for caller_name, callee_file, callee_name, permissions in contracts:
        caller = release['jobs'][caller_name]
        callee = load_workflow(callee_file)['jobs'][callee_name]
        expect_equal(f'{caller_name} reusable workflow', caller['uses'],
                     f'./.github/workflows/{callee_file}')
        expect_equal(f'{caller_name} permissions', caller.get('permissions'),
                     permissions)
        expect_equal(f'{callee_name} permissions', callee.get('permissions'),
                     permissions)
        if 'secrets' in caller:
            raise AssertionError(f'{caller_name} must not inherit caller secrets')
        callee_text = (WORKFLOWS / callee_file).read_text(encoding='utf-8')
        referenced_secrets = set(re.findall(
            r'\bsecrets\.([A-Za-z_][A-Za-z_0-9]*)\b', callee_text))
        expect_equal(f'{callee_name} secrets', referenced_secrets,
                     {'GITHUB_TOKEN'})

    dispatch = release['on']['workflow_dispatch']['inputs']
    expect_equal('dry_run type', dispatch['dry_run']['type'], 'boolean')
    expect_equal('dry_run default', dispatch['dry_run']['default'], False)
    expect_equal('dry-run build push',
                 release['jobs']['build-images-for-release']['with']['push'],
                 '${{ !inputs.dry_run }}')
    expect_equal('dry-run manifest gate',
                 release['jobs']['create-manifests']['if'],
                 '${{ !inputs.dry_run }}')
    print('Release image workflow permissions and dry-run gates are valid.')


if __name__ == '__main__':
    main()
