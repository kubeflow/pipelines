// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// https://www.apache.org/licenses/LICENSE-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mkdtemp, rename, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { execFileSync } from 'node:child_process';
import {
  ExternalArtifactStorageConfig,
  createExternalArtifactClient,
  getExternalArtifactStorageBinding,
  isAllowedExternalArtifactBucketName,
  parseExternalArtifactStorageConfig,
  readExternalArtifactCredentials,
} from './external-artifact-storage.js';

const config: ExternalArtifactStorageConfig = {
  namespaceUID: 'uid-a',
  configurationRevision: 'a'.repeat(64),
  storage: {
    source: 's3',
    bucket: 'external-artifacts',
    endpoint: 'https://store.example:9443',
    region: 'us-east-1',
  },
  identity: {
    kind: 'shared-credentials-file',
    principal: 'broker:team-a',
    credentialFile: '/var/run/broker/credentials',
    profile: 'artifacts',
  },
};

describe('external storage approval', () => {
  it('matches the independently verified controller binding contract', () => {
    expect(getExternalArtifactStorageBinding('team-a', config)).toBe(
      'd1b1915844609b2d71e35a33f6d8e9917f34237a0a9f61135cdbc8679b0fc9a2',
    );
  });
  it('accepts the bounded administrator-owned schema', () => {
    expect(parseExternalArtifactStorageConfig(config)).toEqual(config);
  });
  it.each([
    { namespaceUID: '../other' },
    { configurationRevision: '' },
    { storage: { ...config.storage, source: 'minio' } },
    ...[
      'https://user:secret@store.example',
      'https://store.example/path',
      'https://store.example?x=y',
      'https://store.example/',
      'file:///tmp/object',
      'http://[fd00:0:0:0:0:0:0:10]:8333',
      'http://[fd00::10]:8333/',
      'http://user:secret@[fd00::10]:8333',
    ].map((endpoint) => ({ storage: { ...config.storage, endpoint } })),
    ...['../credentials', '/tmp/../credentials', '/var/run/broker/'].map((credentialFile) => ({
      identity: { ...config.identity, credentialFile },
    })),
    { identity: { ...config.identity, kind: 'default-chain' } },
    { identity: { ...config.identity, profile: 'artifacts\ndefault' } },
  ])('rejects unbound or ambiguous configuration %j', (changed) => {
    expect(() => parseExternalArtifactStorageConfig({ ...config, ...changed })).toThrow(
      'Invalid approved external',
    );
  });
  it('uses the same dotted bucket grammar for approval and serving', () => {
    const dotted = { ...config, storage: { ...config.storage, bucket: 'team.artifacts' } };
    expect(isAllowedExternalArtifactBucketName(dotted.storage.bucket)).toBe(true);
    expect(parseExternalArtifactStorageConfig(dotted)).toEqual(dotted);
  });
  it.each(['', 'a', 'Uppercase', 'bucket/name', 'bucket:9000', 'a'.repeat(64)])(
    'rejects an invalid approved bucket %s in both paths',
    (bucket) => {
      expect(isAllowedExternalArtifactBucketName(bucket)).toBe(false);
      expect(() =>
        parseExternalArtifactStorageConfig({
          ...config,
          storage: { ...config.storage, bucket },
        }),
      ).toThrow('Invalid approved external');
    },
  );
  it('binds every approved destination and credential-source field', () => {
    const binding = getExternalArtifactStorageBinding('team-a', config);
    expect(getExternalArtifactStorageBinding('team-b', config)).not.toBe(binding);
    for (const changed of [
      { ...config, namespaceUID: 'replacement-uid' },
      { ...config, configurationRevision: 'b'.repeat(64) },
      ...Object.keys(config.storage)
        .filter((key) => key !== 'source')
        .map((key) => ({ ...config, storage: { ...config.storage, [key]: 'changed' } })),
      ...['principal', 'credentialFile', 'profile'].map((key) => ({
        ...config,
        identity: { ...config.identity, [key]: 'changed' },
      })),
    ])
      expect(getExternalArtifactStorageBinding('team-a', changed)).not.toBe(binding);
  });
});

describe('broker credentials', () => {
  let directory: string;
  let local: ExternalArtifactStorageConfig;
  const credentials = (key: string) =>
    `[artifacts]\naws_access_key_id=${key}\naws_secret_access_key=secret-${key}\naws_session_token=session-${key}\n`;
  beforeEach(async () => {
    directory = await mkdtemp(join(tmpdir(), 'kfp-broker-'));
    local = {
      ...config,
      identity: { ...config.identity, credentialFile: join(directory, 'credentials') },
    };
    vi.stubEnv('AWS_ACCESS_KEY_ID', 'broad-ambient-key');
    vi.stubEnv('AWS_SECRET_ACCESS_KEY', 'broad-ambient-secret');
  });
  afterEach(async () => {
    vi.unstubAllEnvs();
    await rm(directory, { recursive: true, force: true });
  });
  it.each([
    ['http://[fd00::10]:8333', 'fd00::10', 8333, 'http:'],
    ['https://[2001:db8::10]', '2001:db8::10', 443, 'https:'],
    ['http://192.0.2.10:8333', '192.0.2.10', 8333, 'http:'],
    ['https://store.example:9443', 'store.example', 9443, 'https:'],
  ])(
    'constructs a real client for approved endpoint %s without changing its binding',
    async (endpoint, host, port, protocol) => {
      const approved = parseExternalArtifactStorageConfig({
        ...local,
        storage: { ...local.storage, endpoint },
      });
      const binding = getExternalArtifactStorageBinding('team-a', approved);
      await writeFile(local.identity.credentialFile, credentials('namespace-key'));
      // Do not mock Minio: its constructor rejects bracketed IPv6 hostnames.
      const client = await createExternalArtifactClient(approved);
      expect(client).toMatchObject({ host, port, protocol });
      expect(approved.storage.endpoint).toBe(endpoint);
      expect(getExternalArtifactStorageBinding('team-a', approved)).toBe(binding);
    },
  );

  it('reads each atomically replaced profile without changing the stable binding', async () => {
    const binding = getExternalArtifactStorageBinding('team-a', local);
    for (const key of ['first', 'second', 'third']) {
      await writeFile(join(directory, 'next'), credentials(key));
      await rename(join(directory, 'next'), local.identity.credentialFile);
      expect(await readExternalArtifactCredentials(local)).toEqual({
        accessKey: key,
        secretKey: `secret-${key}`,
        sessionToken: `session-${key}`,
      });
      expect(getExternalArtifactStorageBinding('team-a', local)).toBe(binding);
    }
  });
  it.each(['aws_session_token', 'aws_security_token'])(
    'reads %s while retaining the approved region and identity binding',
    async (tokenKey) => {
      const binding = getExternalArtifactStorageBinding('team-a', local);
      await writeFile(
        local.identity.credentialFile,
        credentials('namespace-key').replace('aws_session_token', tokenKey) +
          'region=unapproved-region\n',
      );
      expect(await readExternalArtifactCredentials(local)).toEqual({
        accessKey: 'namespace-key',
        secretKey: 'secret-namespace-key',
        sessionToken: 'session-namespace-key',
      });
      expect(await createExternalArtifactClient(local)).toMatchObject({ region: 'us-east-1' });
      expect(getExternalArtifactStorageBinding('team-a', local)).toBe(binding);
    },
  );
  it.each([-1, 0, 1])(
    'checks credential expiration at the current instant (%ims)',
    async (offset) => {
      const now = Date.parse('2030-01-01T00:00:00Z');
      const clock = vi.spyOn(Date, 'now').mockReturnValue(now);
      try {
        await writeFile(
          local.identity.credentialFile,
          credentials('namespace-key') + `expiration=${new Date(now + offset).toISOString()}\n`,
        );
        if (offset <= 0) {
          await expect(readExternalArtifactCredentials(local)).rejects.toThrow(
            'Approved artifact credentials are unavailable',
          );
        } else {
          await expect(readExternalArtifactCredentials(local)).resolves.toMatchObject({
            accessKey: 'namespace-key',
          });
        }
      } finally {
        clock.mockRestore();
      }
    },
  );
  it.each([
    '',
    '[default]\naws_access_key_id=other\naws_secret_access_key=other\naws_session_token=other',
    '[artifacts]\naws_access_key_id=only-key',
    '[artifacts]\naws_access_key_id=key\naws_secret_access_key=secret',
    credentials('x') + 'credential_process=echo-secret',
    credentials('x') + 'source_profile=default',
    credentials('x') + 'role_arn=arn:aws:iam::123:role/admin',
    credentials('x') + 'aws_session_token=duplicate',
    credentials('x') + 'aws_security_token=conflicting',
    credentials('x') + 'aws_security_token=session-x',
    credentials('x').replace('aws_session_token', 'aws_security_token') +
      'aws_security_token=duplicate',
    credentials('x') + 'region=us-east-1\nregion=us-west-2',
    credentials('x') + '[artifacts]\naws_access_key_id=duplicate',
    credentials('x') + 'expiration=2000-01-01T00:00:00Z',
    credentials('x') + 'expiration=invalid',
    'x'.repeat(65537),
  ])('never falls back to ambient credentials for invalid broker input', async (input) => {
    await writeFile(local.identity.credentialFile, input);
    await expect(readExternalArtifactCredentials(local)).rejects.toThrow(
      'Approved artifact credentials are unavailable',
    );
  });
  it('fails closed on a missing file without exposing its contents', async () => {
    await expect(readExternalArtifactCredentials(local)).rejects.toThrow(
      'Approved artifact credentials are unavailable',
    );
  });
  it.skipIf(process.platform === 'win32')(
    'rejects a FIFO without waiting for a writer',
    async () => {
      execFileSync('mkfifo', [local.identity.credentialFile]);
      await expect(readExternalArtifactCredentials(local)).rejects.toThrow(
        'Approved artifact credentials are unavailable',
      );
    },
  );
});
