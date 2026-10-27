#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Qualify same-major PostgreSQL image patches, not PostgreSQL KFP support."""

import argparse
import copy
from dataclasses import dataclass
import json
from pathlib import Path
import re
import socket
import subprocess
import sys
import time

import yaml

ROOT = Path(__file__).resolve().parents[3]
FIXTURES = ROOT / '.github/resources/postgresql-upgrade'


@dataclass(frozen=True)
class Profile:
    """Manifest and immutable predecessor qualified by an upgrade lane."""
    kustomize_path: str
    deployment: str
    claim: str
    db_user: str
    old_image: str


PROFILES = {
    '18':
        Profile(
            'third-party/postgresql/base', 'postgres', 'postgres-pvc', 'user',
            'docker.io/postgres:18.3-alpine3.22@sha256:af27ebd3413839ccfdcdf88e91ca76ad82117e52192067b76de9bb5be7ea6e04'
        ),
    '14':
        Profile(
            'base/metadata/overlays/postgres', 'metadata-postgres-db',
            'metadata-postgres', 'root',
            'docker.io/postgres:14.7-alpine3.17@sha256:07985c5c5e01ee550b30c67a2cf806f8b08b32f74e3f24c4cbef31305ef5776d'
        ),
}

DIGEST_QUERY = """SELECT md5(string_agg(id||name||payload::text||
created::text,',' ORDER BY id)) FROM patch_acceptance.records"""
SCHEMA_QUERY = """SELECT string_agg(table_name, ',' ORDER BY table_name)
FROM information_schema.tables WHERE table_schema='public'
AND table_type='BASE TABLE'"""
SYSTEM_ID_QUERY = 'SELECT system_identifier FROM pg_control_system()'


def run(args, **kwargs):
    """Fail at the actual command, including commands whose output is
    captured."""
    return subprocess.run(args, check=True, text=True, **kwargs)


def select_resources(documents, lane: str,
                     namespace: str) -> tuple[list, str, int]:
    """Keep database dependencies and the actual legacy MLMD deployment."""
    profile = PROFILES[lane]
    deployment, claim = profile.deployment, profile.claim
    selected = []
    for original in documents:
        if not original:
            continue
        kind, name = original['kind'], original['metadata']['name']
        keep = (
            kind in ('Secret', 'ConfigMap', 'ServiceAccount') or
            (kind == 'PersistentVolumeClaim' and name == claim) or
            (kind == 'Service' and name in (deployment, 'postgres-service')) or
            (kind == 'Deployment' and
             name in (deployment, 'metadata-grpc-deployment')))
        if keep:
            resource = copy.deepcopy(original)
            resource['metadata']['namespace'] = namespace
            selected.append(resource)
    database = next(
        r for r in selected
        if r['kind'] == 'Deployment' and r['metadata']['name'] == deployment)
    image = database['spec']['template']['spec']['containers'][0]['image']
    # Require an immutable same-major candidate. A major change requires a
    # migration test, not reuse of this retained-directory acceptance lane.
    match = re.search(r':(\d+)\.(\d+)[^@]*@sha256:[0-9a-f]{64}$', image)
    if not match or match[1] != lane:
        raise ValueError('Candidate must be a digest-pinned same-major '
                         'PostgreSQL image; update the manifest or design a '
                         'major-version migration test.')
    return selected, image, int(match[1]) * 10000 + int(match[2])


class Acceptance:
    """Run on a disposable Kind cluster with its real provisioned volumes."""

    def __init__(self, lane: str, output: Path, schema: Path):
        self.lane, self.output, self.schema = lane, output, schema
        self.namespace = 'postgres-upgrade-' + lane
        profile = PROFILES[lane]
        self.deployment = profile.deployment
        self.claim = profile.claim
        self.user = profile.db_user
        self.old = profile.old_image
        self.forwards = []
        self.receipt = {'lane': lane, 'old_image': self.old, 'phases': []}

    def kubectl(self, *args: str, **kwargs) -> subprocess.CompletedProcess:
        """Run kubectl within the disposable lane namespace."""
        return run(['kubectl', '-n', self.namespace, *args], **kwargs)

    def sql(self, statement: str, database: str = 'postgres') -> str:
        """Execute SQL through the deployed database with errors enabled."""
        return self.kubectl(
            'exec',
            '-i',
            'deployment/' + self.deployment,
            '--',
            'psql',
            '-v',
            'ON_ERROR_STOP=1',
            '-U',
            self.user,
            '-d',
            database,
            '-At',
            input=statement,
            capture_output=True).stdout.strip()

    def wait_database(self) -> None:
        """Wait for deployment readiness and a successful database query."""
        self.kubectl('rollout', 'status', 'deployment/' + self.deployment,
                     '--timeout=240s')
        deadline = time.monotonic() + 120
        while time.monotonic() < deadline:
            try:
                if self.sql('SELECT 1') == '1':
                    return
            except subprocess.CalledProcessError:
                pass
            time.sleep(2)
        raise RuntimeError('Database failed readiness; inspect saved pod logs '
                           'and volume events before changing the fixture.')

    def forward(self, deployment: str, local: int, remote: int) -> None:
        """Start a bounded, logged application port-forward."""
        log = (self.output / f'forward-{local}.log').open('a')
        process = subprocess.Popen([
            'kubectl', '-n', self.namespace, 'port-forward',
            'deployment/' + deployment, f'{local}:{remote}'
        ],
                                   stdout=log,
                                   stderr=subprocess.STDOUT)
        self.forwards.append((process, log))
        for _ in range(60):
            if process.poll() is not None:
                break
            try:
                with socket.create_connection(('127.0.0.1', local), timeout=1):
                    return
            except OSError:
                time.sleep(1)
        raise RuntimeError('Port forwarding failed; inspect the saved '
                           f'forward-{local}.log and deployment readiness.')

    def stop_forwards(self) -> None:
        """Close port-forward processes before replacing or restarting pods."""
        for process, log in self.forwards:
            process.terminate()
            process.wait(timeout=15)
            log.close()
        self.forwards.clear()

    def application(self, phase: str) -> None:
        """Check actual KFP or MLMD persistence for one upgrade phase."""
        if self.lane == '18':
            self.forward(self.deployment, 15432, 5432)
            run([
                str(self.schema), '--phase',
                'seed' if phase == 'old' else 'verify', '--host', '127.0.0.1',
                '--port', '15432', '--database', 'mlpipeline', '--user',
                self.user, '--password', 'password'
            ])
        else:
            self.kubectl('apply', '-f', str(self.output / 'mlmd.yaml'))
            self.kubectl('rollout', 'status',
                         'deployment/metadata-grpc-deployment',
                         '--timeout=240s')
            self.forward('metadata-grpc-deployment', 15433, 8080)
            run([
                sys.executable,
                str(FIXTURES / 'mlmd-client.py'), '--phase',
                'seed' if phase == 'old' else phase, '--host', '127.0.0.1',
                '--port', '15433'
            ])
        self.stop_forwards()
        self.receipt['phases'].append(phase)
        (self.output /
         'receipt.json').write_text(json.dumps(self.receipt, indent=2) + '\n')

    def stop_mlmd(self) -> None:
        """Stop the legacy client before changing its database image."""
        if self.lane == '14':
            self.kubectl('scale', 'deployment/metadata-grpc-deployment',
                         '--replicas=0')
            self.kubectl('wait', '--for=delete', 'pod', '-l',
                         'component=metadata-grpc-server', '--timeout=120s')

    def diagnostic(self) -> None:
        """Save deployment and volume diagnostics even after a failed phase."""
        self.stop_forwards()
        for resource in ('pods', 'pvc', 'events', 'deployments'):
            with (self.output / f'{resource}.yaml').open('w') as output:
                subprocess.run([
                    'kubectl', '-n', self.namespace, 'get', resource, '-o',
                    'yaml'
                ],
                               stdout=output,
                               stderr=subprocess.STDOUT,
                               check=False)
        for deployment in (self.deployment, 'metadata-grpc-deployment'):
            with (self.output / f'{deployment}.log').open('w') as output:
                subprocess.run([
                    'kubectl', '-n', self.namespace, 'logs',
                    'deployment/' + deployment, '--all-containers'
                ],
                               stdout=output,
                               stderr=subprocess.STDOUT,
                               check=False)

    def deploy_old(self) -> tuple[str, str, int]:
        """Render the candidate and deploy its pinned predecessor."""
        source = ROOT / 'manifests/kustomize' / PROFILES[
            self.lane].kustomize_path
        rendered = run(
            ['kubectl', 'kustomize', str(source)], capture_output=True).stdout
        resources, candidate, expected_version = select_resources(
            yaml.safe_load_all(rendered), self.lane, self.namespace)
        self.receipt['candidate_image'] = candidate
        (self.output / 'rendered.yaml').write_text(rendered)
        mlmd = [
            r for r in resources if r['kind'] == 'Deployment' and
            r['metadata']['name'] == 'metadata-grpc-deployment'
        ]
        (self.output / 'mlmd.yaml').write_text(yaml.safe_dump_all(mlmd))
        resources = [r for r in resources if r not in mlmd]
        database = next(r for r in resources if r['kind'] == 'Deployment')
        database['spec']['template']['spec']['containers'][0][
            'image'] = self.old
        run(['kubectl', 'create', 'namespace', self.namespace])
        self.kubectl('apply', '-f', '-', input=yaml.safe_dump_all(resources))
        self.wait_database()
        claim = json.loads(
            self.kubectl(
                'get', 'pvc', self.claim, '-o', 'json',
                capture_output=True).stdout)
        self.receipt['pvc_uid'] = claim['metadata']['uid']
        self.receipt['pv_name'] = claim['spec']['volumeName']
        container = database['spec']['template']['spec']['containers'][0][
            'name']
        return container, candidate, expected_version

    def seed(self) -> tuple[str, str, str]:
        """Seed data and capture identity and the exact KFP schema."""
        self.sql('''CREATE SCHEMA patch_acceptance;
CREATE TABLE patch_acceptance.records (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 name text UNIQUE NOT NULL, payload jsonb NOT NULL, created timestamptz NOT NULL);
CREATE INDEX records_payload ON patch_acceptance.records USING gin(payload);
INSERT INTO patch_acceptance.records(name,payload,created)
 SELECT 'record-'||i,jsonb_build_object('value',i),
 TIMESTAMPTZ '2026-01-01 12:00:00+00' FROM generate_series(1,1000) i;''')
        if self.lane == '14':
            self.sql('CREATE DATABASE mlmdpostgres')
        self.application('old')
        tables = self.sql(SCHEMA_QUERY,
                          'mlpipeline') if self.lane == '18' else ''
        return self.sql(DIGEST_QUERY), self.sql(SYSTEM_ID_QUERY), tables

    def upgrade(self, container: str, candidate: str,
                expected_version: int) -> None:
        """Replace only the image and require the intended server version."""
        self.stop_mlmd()
        self.kubectl('set', 'image', 'deployment/' + self.deployment,
                     container + '=' + candidate)
        self.wait_database()
        if self.sql('SHOW server_version_num') != str(expected_version):
            raise RuntimeError(
                'Wrong PostgreSQL version; inspect rendered image '
                'and runtime image ID before accepting the pin.')

    def verify_retained(self, before: str, system_id: str, tables: str) -> None:
        """Check retained data before migrations can repair the schema."""
        if (self.sql(DIGEST_QUERY) != before or
                self.sql(SYSTEM_ID_QUERY) != system_id):
            raise RuntimeError('Upgrade changed retained data or database '
                               'identity; investigate the mounted PVC.')
        if self.lane == '18' and self.sql(SCHEMA_QUERY, 'mlpipeline') != tables:
            raise RuntimeError('Upgrade changed the KFP table set; inspect the '
                               'retained database before running migrations.')
        self.sql('''DO $$ BEGIN
 IF (SELECT count(*) FROM patch_acceptance.records
     WHERE payload @> '{"value":500}'::jsonb) <> 1
 THEN RAISE EXCEPTION 'JSONB lookup failed'; END IF;
 IF (SELECT count(*) FROM pg_index
     WHERE indrelid='patch_acceptance.records'::regclass AND indisvalid) <> 3
 THEN RAISE EXCEPTION 'index lost'; END IF;
END $$;
INSERT INTO patch_acceptance.records(name,payload,created)
 VALUES ('after-upgrade','{"value":1001}',now());''')
        self.application('upgraded')

    def restart_and_verify(self) -> None:
        """Require post-upgrade writes and the original PVC after a restart."""
        self.stop_mlmd()
        self.kubectl('rollout', 'restart', 'deployment/' + self.deployment)
        self.wait_database()
        if self.sql('SELECT count(*) FROM patch_acceptance.records') != '1001':
            raise RuntimeError('Post-upgrade write was not retained; inspect '
                               'database and volume logs.')
        self.application('restarted')
        after = json.loads(
            self.kubectl(
                'get', 'pvc', self.claim, '-o', 'json',
                capture_output=True).stdout)
        if (after['metadata']['uid'] != self.receipt['pvc_uid'] or
                after['spec']['volumeName'] != self.receipt['pv_name']):
            raise RuntimeError('PVC changed during acceptance; repeat with the '
                               'original volume intact.')

    def execute(self) -> None:
        """Publish success only after every retained-volume phase passes."""
        container, candidate, expected_version = self.deploy_old()
        before, system_id, tables = self.seed()
        self.upgrade(container, candidate, expected_version)
        self.verify_retained(before, system_id, tables)
        self.restart_and_verify()
        self.receipt['status'] = 'passed'
        (self.output /
         'receipt.json').write_text(json.dumps(self.receipt, indent=2) + '\n')
        print(
            f'PASS PostgreSQL {self.lane} retained-PVC image upgrade',
            flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--lane', required=True, choices=PROFILES)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument(
        '--schema-helper', type=Path, default=Path('/tmp/kfp-schema'))
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    acceptance = Acceptance(args.lane, args.output, args.schema_helper)
    try:
        acceptance.execute()
    finally:
        acceptance.diagnostic()


if __name__ == '__main__':
    main()
