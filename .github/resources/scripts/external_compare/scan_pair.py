"""Scan immutable old/new images with the same trusted OSV implementation."""

import argparse
import hashlib
import json
import subprocess
from pathlib import Path

import kfpr.osv_image_scan as osv_module
import yaml
from kfpr.osv_image_scan import scan_image, validate_report
from kfpr.osv_reports import findings
from third_party_upgrade_scan import compare_reports


def read_baseline(source, target):
    paths = {
        'seaweedfs': (
            'manifests/kustomize/third-party/seaweedfs/base/seaweedfs/seaweedfs-deployment.yaml',
            'seaweedfs',
        ),
        'metacontroller': (
            'manifests/kustomize/third-party/metacontroller/base/stateful-set.yaml',
            'metacontroller',
        ),
    }
    path, name = paths[target]
    document = yaml.safe_load((source / path).read_text())
    return next(
        c['image']
        for c in document['spec']['template']['spec']['containers']
        if c['name'] == name
    )


def capture(reference, output):
    output.mkdir(parents=True)
    subprocess.run(
        ['docker', 'pull', '--platform', 'linux/amd64', reference], check=True
    )
    metadata = json.loads(
        subprocess.check_output(
            ['docker', 'image', 'inspect', reference], text=True
        )
    )[0]
    assert metadata['Os'] == 'linux' and metadata['Architecture'] == 'amd64'
    assert metadata['RepoDigests']
    expected_digest = reference.split('@', 1)[1]
    assert any(
        value.endswith('@' + expected_digest)
        for value in metadata['RepoDigests']
    )
    subject = {
        'reference': reference,
        'image_id': metadata['Id'],
        'repo_digests': metadata['RepoDigests'],
        'platform': 'linux/amd64',
    }
    (output / 'subject.json').write_text(json.dumps(subject, indent=2) + '\n')
    scan_image(metadata['Id'], 'linux/amd64', output, report_format='json')
    envelope = json.loads((output / 'scan.json').read_text())
    validate_report(envelope, metadata['Id'], 'json')
    subject['raw_sha256'] = {
        p.name: hashlib.sha256(p.read_bytes()).hexdigest()
        for p in [output / 'scan.json', output / 'osv-report.json']
    }
    return subject, envelope['report']


def main():
    parser = argparse.ArgumentParser()
    for key in ['source', 'pairs', 'output']:
        parser.add_argument('--' + key, type=Path, required=True)
    parser.add_argument(
        '--target', choices=['seaweedfs', 'metacontroller'], required=True
    )
    args = parser.parse_args()
    config = json.loads(args.pairs.read_text())
    tooling = Path(osv_module.__file__).resolve().parents[2]
    actual_tooling = subprocess.check_output(
        ['git', '-C', str(tooling), 'rev-parse', 'HEAD'], text=True
    ).strip()
    assert actual_tooling == config['tooling_sha']
    source_sha = subprocess.check_output(
        ['git', '-C', str(args.source), 'rev-parse', 'HEAD'], text=True
    ).strip()
    assert source_sha == config['source_sha']
    pair = config['pairs'][args.target]
    assert read_baseline(args.source, args.target) == pair['baseline']
    before, old = capture(pair['baseline'], args.output / 'baseline')
    after, new = capture(pair['candidate'], args.output / 'candidate')
    result = {
        'source_sha': source_sha,
        'tooling_sha': config['tooling_sha'],
        'target': args.target,
        'baseline': before,
        'candidate': after,
        'comparison': compare_reports(old, new),
    }
    for label, report in [('baseline', old), ('candidate', new)]:
        rows = findings(report)
        result[label]['fixable'] = sum(row['fixable'] for row in rows)
        result[label]['residual'] = sum(not row['fixable'] for row in rows)
        result[label]['findings'] = rows
    (args.output / 'comparison.json').write_text(
        json.dumps(result, indent=2) + '\n'
    )
    print(
        json.dumps(
            {
                label: {
                    k: result[label][k]
                    for k in ['fixable', 'residual', 'image_id']
                }
                for label in ['baseline', 'candidate']
            },
            indent=2,
        )
    )


if __name__ == '__main__':
    main()
