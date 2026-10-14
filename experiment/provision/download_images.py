#!/usr/bin/env python3
"""Download pinned GitHub image artifacts for the disposable hosted experiment."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import zipfile

import provision


def fetch(endpoint, destination):
    """gh consumes GH_TOKEN from its environment; never print response/error bodies."""
    partial = destination.with_suffix(destination.suffix + '.partial')
    with partial.open('wb') as out:
        try:
            result = subprocess.run(['gh', 'api', '--allow-escape-sequences', endpoint],
                                    stdout=out, stderr=subprocess.PIPE,
                                    timeout=240, check=False)
        except subprocess.TimeoutExpired:
            raise RuntimeError('GitHub download timed out: ' + destination.name) from None
    if result.returncode:
        status = re.search(rb'HTTP (\d{3})', result.stderr or b'')
        suffix = ' (HTTP ' + status[1].decode() + ')' if status else ''
        raise RuntimeError('GitHub download failed: ' + destination.name + suffix)
    partial.replace(destination)


def safe_member(name):
    return isinstance(name, str) and re.fullmatch(r'[a-z0-9-]+\.tar', name) is not None


def download(manifest_path, output):
    manifest = json.loads(Path(manifest_path).read_text())
    if (manifest.get('repository') != 'kubeflow/pipelines'
            or manifest.get('source_sha') != provision.SOURCE
            or manifest.get('head_sha') != provision.HEAD
            or set(manifest.get('images', {})) != set(provision.COMPONENTS)):
        raise RuntimeError('manifest repository/source/image inventory mismatch')
    for name, image in manifest['images'].items():
        if (not safe_member(image.get('archive_member'))
                or image['archive_member'] != name + '.tar'
                or type(image.get('artifact_id')) is not int or image['artifact_id'] <= 0
                or type(image.get('producer_job_id')) is not int or image['producer_job_id'] <= 0):
            raise RuntimeError('invalid artifact/member identity: ' + name)
    if type(manifest.get('run_id')) is not int or type(manifest.get('run_attempt')) is not int:
        raise RuntimeError('exact run and attempt required')
    output = Path(output).resolve()
    output.mkdir(parents=True, exist_ok=False, mode=0o700)
    if shutil.disk_usage(output).free < 3 * 1024**3:
        raise RuntimeError('at least 3 GiB free required for image ZIPs and archives')
    repo = 'repos/kubeflow/pipelines'
    run_file, commit_file = output / 'run.json', output / 'source-commit.json'
    fetch(f"{repo}/actions/runs/{manifest['run_id']}/attempts/{manifest['run_attempt']}", run_file)
    fetch(f"{repo}/git/commits/{provision.SOURCE}", commit_file)
    run = json.loads(run_file.read_text())
    if run.get('run_attempt') != manifest['run_attempt']:
        raise RuntimeError('run attempt mismatch')
    inputs = {}
    for name, item in manifest['images'].items():
        producer_run = item.get('producer_run_id', manifest['run_id'])
        producer_attempt = item.get('producer_run_attempt', manifest['run_attempt'])
        if type(producer_run) is not int or type(producer_attempt) is not int or producer_run <= 0 or producer_attempt <= 0:
            raise RuntimeError('invalid producer run identity: ' + name)
        image_run_file = output / f'run-{producer_run}-attempt-{producer_attempt}.json'
        if not image_run_file.exists():
            fetch(f'{repo}/actions/runs/{producer_run}/attempts/{producer_attempt}', image_run_file)
        image_run = json.loads(image_run_file.read_text())
        if (image_run.get('id') != producer_run or image_run.get('run_attempt') != producer_attempt
                or (item.get('producer_workflow_path') and image_run.get('path') != item['producer_workflow_path'])):
            raise RuntimeError('producer run/attempt mismatch: ' + name)
        metadata, job, log = (output / (name + suffix) for suffix in ('-artifact.json', '-job.json', '.log'))
        zipped, archive = output / (name + '.zip'), output / item['archive_member']
        fetch(f"{repo}/actions/artifacts/{item['artifact_id']}", metadata)
        artifact = json.loads(metadata.read_text())
        if artifact.get('id') != item['artifact_id'] or artifact.get('digest') != item['artifact_digest']:
            raise RuntimeError('artifact ID/digest changed: ' + name)
        fetch(f"{repo}/actions/jobs/{item['producer_job_id']}", job)
        job_data = json.loads(job.read_text())
        if job_data.get('id') != item['producer_job_id'] or job_data.get('run_attempt') != producer_attempt:
            raise RuntimeError('producer job ID/attempt mismatch: ' + name)
        fetch(f"{repo}/actions/jobs/{item['producer_job_id']}/logs", log)
        fetch(f"{repo}/actions/artifacts/{item['artifact_id']}/zip", zipped)
        image = {'component': name, 'source_sha': provision.SOURCE,
                 'archive': str(archive), 'archive_sha256': item['archive_sha256'],
                 'image_id': item['image_id'], 'artifact_zip': str(zipped),
                 'archive_member': item['archive_member'], 'artifact_metadata': str(metadata),
                 'producer_run_metadata': str(image_run_file), 'producer_job_metadata': str(job),
                 'producer_job_log': str(log), 'source_commit_metadata': str(commit_file)}
        provision.verify_image(image)
        # Stream a single verified member to a fixed filename, never extractall.
        with zipfile.ZipFile(zipped) as source:
            info = source.getinfo(item['archive_member'])
            if info.file_size > 1024**3:
                raise RuntimeError('archive exceeds bounded 1 GiB per-image budget: ' + name)
            with source.open(info) as src, archive.open('xb') as dst:
                shutil.copyfileobj(src, dst, length=1024 * 1024)
        if provision.sha(archive) != item['archive_sha256']:
            raise RuntimeError('extracted archive digest mismatch: ' + name)
        inputs[name] = image
        print('Verified image: ' + name, flush=True)
    provision.save(output / 'image-inputs.json', inputs)
    hashes = {p.name: provision.sha(p) for p in output.iterdir() if p.is_file()}
    provision.save(output / 'evidence-hashes.json', hashes)


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--manifest', required=True)
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    try:
        download(args.manifest, args.output)
    except RuntimeError as error:
        print(str(error), file=sys.stderr)
        return 1
    except Exception:
        print('Image preparation failed; inspect saved metadata locally; response bodies suppressed.', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
