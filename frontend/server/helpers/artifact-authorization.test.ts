// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getK8sSecret } from '../k8s-helper.js';
import { getExternalArtifactStorageBinding } from './external-artifact-storage.js';
import {
  getArtifactCredentialBinding,
  getArtifactStorageAuthorization,
} from './artifact-authorization.js';

vi.mock('../k8s-helper.js', () => ({ getK8sSecret: vi.fn() }));

const minio = {
  endPoint: 'seaweedfs.kubeflow',
  port: 8333,
  useSSL: false,
  accessKey: 'team-a-key',
  secretKey: 'team-a-secret',
};
const record = {
  version: 1,
  namespace: 'team-a',
  namespaceUID: 'uid-a',
  state: 'ready',
  bucket: 'mlpipeline',
  credentialBinding: getArtifactCredentialBinding('team-a', 'uid-a', minio),
};

beforeEach(() => vi.resetAllMocks());

describe('controller storage authorization registry', () => {
  const external = {
    namespaceUID: 'uid-a',
    configurationRevision: 'a'.repeat(64),
    storage: {
      source: 's3' as const,
      bucket: 'external-artifacts',
      endpoint: 'https://store.example',
      region: 'us-east-1',
    },
    identity: {
      kind: 'shared-credentials-file' as const,
      principal: 'broker:team-a',
      credentialFile: '/var/run/broker/credentials',
      profile: 'artifacts',
    },
  };
  const externalRecord = {
    ...external,
    version: 2,
    namespace: 'team-a',
    state: 'ready',
    credentialBinding: getExternalArtifactStorageBinding('team-a', external),
  };
  it('accepts an approved external identity and its pending tombstone', async () => {
    for (const value of [
      externalRecord,
      { version: 2, namespace: 'team-a', namespaceUID: 'uid-a', state: 'pending' },
    ]) {
      vi.mocked(getK8sSecret).mockResolvedValue(JSON.stringify(value));
      expect(await getArtifactStorageAuthorization('team-a', 'install')).toEqual(value);
    }
  });
  it.each([
    { namespace: 'team-b' },
    { namespaceUID: 'replacement-uid' },
    { configurationRevision: 'b'.repeat(64) },
    { storage: { ...external.storage, bucket: 'other-bucket' } },
    { storage: { ...external.storage, endpoint: 'https://other.example' } },
    { identity: { ...external.identity, principal: 'broker:team-b' } },
    { identity: { ...external.identity, profile: 'default' } },
    { state: 'unsupported' },
    { credentialBinding: undefined },
  ])('rejects an external record with mismatched binding %j', async (changed) => {
    vi.mocked(getK8sSecret).mockResolvedValue(JSON.stringify({ ...externalRecord, ...changed }));
    await expect(getArtifactStorageAuthorization('team-a', 'install')).rejects.toThrow(
      'Invalid artifact storage authorization record',
    );
  });
  it('reads only the installation namespace record', async () => {
    vi.mocked(getK8sSecret).mockResolvedValue(JSON.stringify(record));
    expect(await getArtifactStorageAuthorization('team-a', 'install')).toEqual(record);
    expect(getK8sSecret).toHaveBeenCalledExactlyOnceWith(
      'kfp-artifact-storage-team-a',
      'authorization.json',
      'install',
    );
  });

  it.each([
    null,
    [],
    {},
    { ...record, version: 2 },
    { ...record, namespace: 'team-b' },
    { ...record, namespaceUID: '' },
    { ...record, namespaceUID: '../team-b' },
    { ...record, state: 'yes' },
    { ...record, bucket: 'other-bucket' },
    { ...record, credentialBinding: undefined },
    { ...record, credentialBinding: 'invalid' },
    { ...record, credentialBinding: 'F'.repeat(64) },
  ])('rejects malformed or mismatched approval %j', async (value) => {
    vi.mocked(getK8sSecret).mockResolvedValue(JSON.stringify(value));
    await expect(getArtifactStorageAuthorization('team-a', 'install')).rejects.toThrow(
      'Invalid artifact storage authorization record',
    );
  });

  it('classifies a successful Secret read with a missing data key as malformed', async () => {
    // getK8sSecret returns an empty string when authorization.json is absent.
    vi.mocked(getK8sSecret).mockResolvedValue('');
    await expect(getArtifactStorageAuthorization('team-a', 'install')).rejects.toMatchObject({
      reason: 'invalid-record',
    });
  });

  it('rejects malformed JSON without logging record contents', async () => {
    vi.mocked(getK8sSecret).mockResolvedValue('secret-invalid-json');
    await expect(getArtifactStorageAuthorization('team-a', 'install')).rejects.toThrow(
      'Invalid artifact storage authorization record',
    );
  });

  it.each(['pending', 'unsupported'])('accepts %s without a credential binding', async (state) => {
    vi.mocked(getK8sSecret).mockResolvedValue(
      JSON.stringify({ ...record, state, credentialBinding: undefined }),
    );
    expect(await getArtifactStorageAuthorization('team-a', 'install')).toMatchObject({ state });
  });

  it.each([
    { code: 404 },
    { statusCode: 404 },
    { response: { status: 404 } },
    { body: { code: 404 } },
  ])('recognizes a missing record %j', async (error) => {
    vi.mocked(getK8sSecret).mockRejectedValue(error);
    expect(await getArtifactStorageAuthorization('team-a', 'install')).toBeUndefined();
  });

  it.each([
    { code: 401 },
    { statusCode: 403 },
    { response: { status: 403 } },
    { body: { code: 401 } },
  ])('classifies record access failures without exposing API response data', async (error) => {
    vi.mocked(getK8sSecret).mockRejectedValue({ ...error, message: 'secret-data' });
    await expect(getArtifactStorageAuthorization('team-a', 'install')).rejects.toMatchObject({
      reason: 'access-denied',
      message: 'Artifact storage authorization record access is denied',
    });
  });

  it.each([{ code: 500 }, { code: 'ETIMEDOUT' }, new Error('secret-data')])(
    'does not downgrade API failure to an absent record',
    async (error) => {
      vi.mocked(getK8sSecret).mockRejectedValue(error);
      await expect(getArtifactStorageAuthorization('team-a', 'install')).rejects.toThrow(
        'Artifact storage authorization is temporarily unavailable',
      );
    },
  );
});

describe('receiver credential binding', () => {
  it('uses the documented UTF8 null-separated controller contract', () => {
    // Independently computed by Python hashlib.sha256 on the controller inputs.
    expect(record.credentialBinding).toBe(
      'cb05f03246885822f64c9a8136c6b7a060567b98d17eb8b6590095afbca16dc4',
    );
  });

  it('binds namespace, namespace UID, endpoint, key ID and secret', () => {
    expect(getArtifactCredentialBinding('team-b', 'uid-a', minio)).not.toBe(
      record.credentialBinding,
    );
    expect(getArtifactCredentialBinding('team-a', 'uid-b', minio)).not.toBe(
      record.credentialBinding,
    );
    for (const changed of [
      { endPoint: 'other' },
      { port: 9000 },
      { useSSL: true },
      { accessKey: 'other' },
      { secretKey: 'other' },
    ]) {
      expect(getArtifactCredentialBinding('team-a', 'uid-a', { ...minio, ...changed })).not.toBe(
        record.credentialBinding,
      );
    }
  });
});
