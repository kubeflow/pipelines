"""Source-derived deployment, runtime-default and prerequisite image
inventory."""

import hashlib
import json
import re
import subprocess
import urllib.request
from pathlib import Path

import yaml

SOURCE_PATHS = (
    'manifests/kustomize',
    '.github/resources/scripts/deploy-kfp.sh',
    'backend/Makefile',
    'backend/src/apiserver/config/config.json',
    'backend/src/crd/controller/viewer/reconciler/reconciler.go',
    'frontend/src/components/viewers/Tensorboard.tsx',
)
LEGACY = 'manifests/kustomize/base/metadata/overlays/'
PREREQUISITES = {
    'istio': 'common/istio/istio-install/base',
    'profiles': 'applications/dashboard/upstream/profile-controller/overlays/kubeflow',
}


def repository_name(image):
    name = image.split('@', 1)[0]
    return name.rsplit(':', 1)[0] if ':' in name.rsplit('/', 1)[-1] else name


def extract_images(rendered):
    """Read pod images and explicit executor, cache and ACME image settings."""
    images = set()

    def visit(node):
        if isinstance(node, dict):
            for key, value in node.items():
                if key in ('containers', 'initContainers') and isinstance(
                    value, list
                ):
                    for container in value:
                        if not isinstance(container, dict):
                            continue
                        if container.get('image'):
                            images.add(container['image'])
                        args = container.get('args', [])
                        if not isinstance(args, list):
                            raise ValueError(
                                'Invalid container arguments; provide a YAML list'
                            )
                        for index, arg in enumerate(args):
                            for flag in (
                                '--executor-image',
                                '--acme-http01-solver-image',
                            ):
                                if arg == flag:
                                    if (
                                        index + 1 >= len(args)
                                        or not args[index + 1]
                                        or args[index + 1].startswith('-')
                                    ):
                                        raise ValueError(
                                            'Missing image argument; set '
                                            + flag
                                        )
                                    images.add(args[index + 1])
                                elif isinstance(arg, str) and arg.startswith(
                                    flag + '='
                                ):
                                    images.add(arg.split('=', 1)[1])
                if key == 'cacheImage' and isinstance(value, str) and value:
                    images.add(value)
                visit(value)
        elif isinstance(node, list):
            for value in node:
                visit(value)

    for document in yaml.safe_load_all(rendered):
        visit(document)
    if any(
        not isinstance(image, str) or not image or '{{' in image
        for image in images
    ):
        raise ValueError('Unresolved deployed image; render all image settings')
    return images


def profile_paths(source):
    source = Path(source)
    return sorted(
        {
            str(path.parent.relative_to(source))
            for root in ('manifests/kustomize/env', LEGACY.rstrip('/'))
            for filename in (
                'kustomization.yaml',
                'kustomization.yml',
                'Kustomization',
            )
            for path in (source / root).rglob(filename)
        }
    )


def _match(pattern, text, label):
    matches = re.findall(pattern, text, re.MULTILINE)
    if len(matches) != 1:
        raise ValueError(
            'Cannot identify ' + label + '; update the inventory parser'
        )
    return matches[0]


def _pins(sources):
    sha = _match(
        r'^KUBEFLOW_MANIFESTS_SHA=["\']([0-9a-f]{40})["\']',
        sources[SOURCE_PATHS[1]],
        'prerequisite commit',
    )
    version = _match(
        r'^CERT_MANAGER_VERSION\s*\?=\s*(v[0-9]+\.[0-9]+\.[0-9]+)\s*$',
        sources[SOURCE_PATHS[2]],
        'cert-manager version',
    )
    installer = sources[SOURCE_PATHS[1]]
    for path in PREREQUISITES.values():
        expected = (
            'https://github.com/kubeflow/community-distribution/'
            + path
            + '?ref=${KUBEFLOW_MANIFESTS_SHA}'
        )
        if expected not in installer:
            raise ValueError(
                'Prerequisite installer location changed; update the inventory parser'
            )
    certificate_url = (
        'https://github.com/cert-manager/cert-manager/releases/download/'
        '$(CERT_MANAGER_VERSION)/cert-manager.yaml'
    )
    if certificate_url not in sources[SOURCE_PATHS[2]]:
        raise ValueError(
            'Certificate installer location changed; update the inventory parser'
        )
    return {
        **{
            name: 'https://github.com/kubeflow/community-distribution/'
            + path
            + '?ref='
            + sha
            for name, path in PREREQUISITES.items()
        },
        'cert-manager': 'https://github.com/cert-manager/cert-manager/releases/download/'
        + version
        + '/cert-manager.yaml',
    }


def _digest(data):
    return hashlib.sha256(data).hexdigest()


def capture(source, kustomize, output):
    """Preserve inputs before deriving targets; no image pulls or scans."""
    source, output = Path(source), Path(output)
    missing = [
        path for path in SOURCE_PATHS[1:] if not (source / path).is_file()
    ]
    if missing:
        raise ValueError(
            'Missing inventory source: '
            + ', '.join(missing)
            + '; fetch the complete source or update the inventory parser for the new source layout'
        )
    output.mkdir(parents=True, exist_ok=False)
    sources = {path: (source / path).read_text() for path in SOURCE_PATHS[1:]}
    profiles = profile_paths(source)
    kustomizations = {
        str(path.relative_to(source)): path.read_text()
        for profile in profiles
        for path in (source / profile).iterdir()
        if path.name
        in ('kustomization.yaml', 'kustomization.yml', 'Kustomization')
    }
    catalog = {
        'sources': sources,
        'kustomizations': kustomizations,
        'profiles': profiles,
        'prerequisites': _pins(sources),
    }
    (output / 'inputs.json').write_text(json.dumps(catalog, sort_keys=True))
    renders = {}
    for profile in profiles:
        renders[profile] = subprocess.check_output(
            [
                str(kustomize),
                'build',
                '--load-restrictor',
                'LoadRestrictionsNone',
                str(source / profile),
            ],
            text=True,
            timeout=300,
        )
    for name, url in catalog['prerequisites'].items():
        if name == 'cert-manager':
            with urllib.request.urlopen(url, timeout=60) as response:
                renders['prerequisite/' + name] = response.read().decode()
        else:
            renders['prerequisite/' + name] = subprocess.check_output(
                [str(kustomize), 'build', url], text=True, timeout=300
            )
    (output / 'rendered.json').write_text(json.dumps(renders, sort_keys=True))
    return discover(output)


def discover(output):
    """Recompute expected targets from preserved source and rendered inputs."""
    output = Path(output)
    if output.is_symlink() or {p.name for p in output.iterdir()} != {
        'inputs.json',
        'rendered.json',
    }:
        raise ValueError(
            'Invalid inventory evidence files; collect fresh inventory'
        )
    if any(p.is_symlink() or not p.is_file() for p in output.iterdir()):
        raise ValueError('Inventory evidence must be regular files')
    inputs = json.loads((output / 'inputs.json').read_text())
    renders = json.loads((output / 'rendered.json').read_text())
    sources = inputs['sources']
    profiles = inputs['profiles']
    if (
        not isinstance(profiles, list)
        or profiles != sorted(set(profiles))
        or not profiles
    ):
        raise ValueError('Missing profile inventory; collect fresh inventory')
    if any(
        not (p.startswith('manifests/kustomize/env/') or p.startswith(LEGACY))
        or '..' in Path(p).parts
        for p in profiles
    ):
        raise ValueError('Invalid profile path; collect fresh inventory')
    if not {
        'manifests/kustomize/env/platform-agnostic',
        'manifests/kustomize/env/platform-agnostic-multi-user',
    } <= set(profiles):
        raise ValueError('Missing standard profiles; collect fresh inventory')
    expected_profiles = sorted(
        {str(Path(path).parent) for path in inputs['kustomizations']}
    )
    if profiles != expected_profiles:
        raise ValueError(
            'Profile catalog differs from source kustomizations; collect fresh inventory'
        )
    if set(sources) != set(SOURCE_PATHS[1:]):
        raise ValueError(
            'Missing or unexpected inventory source; collect complete inputs or update the parser for the new source layout'
        )
    pins = _pins(sources)
    if inputs['prerequisites'] != pins or set(renders) != set(profiles) | {
        'prerequisite/' + name for name in pins
    }:
        raise ValueError(
            'Incomplete inventory renders; collect every profile and prerequisite'
        )
    targets = {}

    def add(image, scope, profile, role=None):
        repo = repository_name(image)
        target = (
            repo
            if scope == 'deployment'
            else scope + '/' + (role + '/' if role else '') + repo
        )
        key = (target, image)
        record = targets.setdefault(
            key,
            {'image': image, 'scope': scope, 'target': target, 'profiles': []},
        )
        record['profiles'].append(profile)

    for profile, rendered in sorted(renders.items()):
        scope = (
            'prerequisite'
            if profile.startswith('prerequisite/')
            else 'legacy'
            if profile.startswith(LEGACY)
            else 'deployment'
        )
        for image in sorted(extract_images(rendered)):
            add(image, scope, profile)
    config = json.loads(sources[SOURCE_PATHS[3]])
    if (
        not isinstance(config.get('CACHE_IMAGE'), str)
        or not config['CACHE_IMAGE']
    ):
        raise ValueError(
            'Missing cache image default; update the inventory parser for the new configuration'
        )
    add(config['CACHE_IMAGE'], 'runtime', SOURCE_PATHS[3], 'cache')
    for path, pattern, role in (
        (
            SOURCE_PATHS[4],
            r'const defaultTensorflowImage\s*=\s*"([^"]+)"',
            'viewer',
        ),
        (
            SOURCE_PATHS[5],
            r"const DEFAULT_TF_IMAGE\s*=\s*'([^']+)'",
            'tensorboard',
        ),
    ):
        add(
            _match(pattern, sources[path], role + ' default'),
            'runtime',
            path,
            role,
        )
    rows = sorted(
        targets.values(), key=lambda row: (row['target'], row['image'])
    )
    return {
        'profiles': {
            name: sorted(extract_images(data))
            for name, data in sorted(renders.items())
        },
        'targets': rows,
        'input_sha256': {
            name: _digest((output / name).read_bytes())
            for name in ('inputs.json', 'rendered.json')
        },
    }
