/*
 * Copyright 2018-2019 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { parseStoragePath, StorageService } from './StoragePath';
describe('parseStoragePath', () => {
  it('throws for unsupported protocol', () => {
    expect(() => parseStoragePath('unsupported://path')).toThrowError(
      'Unsupported storage path: unsupported://path',
    );
  });

  it('handles GCS bucket without key', () => {
    expect(parseStoragePath('gs://testbucket/')).toEqual({
      bucket: 'testbucket',
      key: '',
      source: StorageService.GCS,
    });
  });

  it('handles GCS bucket and key', () => {
    expect(parseStoragePath('gs://testbucket/testkey')).toEqual({
      bucket: 'testbucket',
      key: 'testkey',
      source: StorageService.GCS,
    });
  });

  it('handles GCS bucket and multi-part key', () => {
    expect(parseStoragePath('gs://testbucket/test/key/path')).toEqual({
      bucket: 'testbucket',
      key: 'test/key/path',
      source: StorageService.GCS,
    });
  });

  it('handles Minio bucket without key', () => {
    expect(parseStoragePath('minio://testbucket/')).toEqual({
      bucket: 'testbucket',
      key: '',
      source: StorageService.MINIO,
    });
  });

  it('handles Minio bucket and key', () => {
    expect(parseStoragePath('minio://testbucket/testkey')).toEqual({
      bucket: 'testbucket',
      key: 'testkey',
      source: StorageService.MINIO,
    });
  });

  it('handles Minio bucket and multi-part key', () => {
    expect(parseStoragePath('minio://testbucket/test/key/path')).toEqual({
      bucket: 'testbucket',
      key: 'test/key/path',
      source: StorageService.MINIO,
    });
  });

  it('handles S3 bucket without key', () => {
    expect(parseStoragePath('s3://testbucket/')).toEqual({
      bucket: 'testbucket',
      key: '',
      source: StorageService.S3,
    });
  });

  it('handles S3 bucket and key', () => {
    expect(parseStoragePath('s3://testbucket/testkey')).toEqual({
      bucket: 'testbucket',
      key: 'testkey',
      source: StorageService.S3,
    });
  });

  it('handles S3 bucket and multi-part key', () => {
    expect(parseStoragePath('s3://testbucket/test/key/path')).toEqual({
      bucket: 'testbucket',
      key: 'test/key/path',
      source: StorageService.S3,
    });
  });

  it('handles OCI Object Storage bucket and namespace without key', () => {
    expect(parseStoragePath('oci://testbucket@testnamespace/')).toEqual({
      bucket: 'testbucket@testnamespace',
      key: '',
      source: StorageService.OCI,
    });
  });

  it('handles OCI Object Storage bucket, namespace and multi-part key', () => {
    expect(parseStoragePath('oci://testbucket@testnamespace/test/key/path')).toEqual({
      bucket: 'testbucket@testnamespace',
      key: 'test/key/path',
      source: StorageService.OCI,
    });
  });

  it('rejects Modelcar container image references as storage paths', () => {
    expect(() => parseStoragePath('oci://registry.domain.local/org/repo:v1.0')).toThrowError(
      'Unsupported storage path: oci://registry.domain.local/org/repo:v1.0',
    );
  });

  it('handles HTTP URL without path', () => {
    expect(parseStoragePath('http://host:port')).toEqual({
      bucket: 'host:port',
      key: '',
      source: StorageService.HTTP,
    });
  });

  it('handles HTTP URL with path', () => {
    expect(parseStoragePath('http://host:port/path/foo/bar')).toEqual({
      bucket: 'host:port',
      key: 'path/foo/bar',
      source: StorageService.HTTP,
    });
  });

  it('handles HTTPS URL without path', () => {
    expect(parseStoragePath('https://host:port')).toEqual({
      bucket: 'host:port',
      key: '',
      source: StorageService.HTTPS,
    });
  });

  it('handles HTTPS URL with path', () => {
    expect(parseStoragePath('https://host:port/path/foo/bar')).toEqual({
      bucket: 'host:port',
      key: 'path/foo/bar',
      source: StorageService.HTTPS,
    });
  });

  it('handles volume file without path', () => {
    expect(parseStoragePath('volume://output')).toEqual({
      bucket: 'output',
      key: '',
      source: StorageService.VOLUME,
    });
  });

  it('handles volume file with path', () => {
    expect(parseStoragePath('volume://output/path/foo/bar')).toEqual({
      bucket: 'output',
      key: 'path/foo/bar',
      source: StorageService.VOLUME,
    });
  });
});
