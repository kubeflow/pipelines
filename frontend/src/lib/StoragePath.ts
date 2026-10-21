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

export enum StorageService {
  GCS = 'gcs',
  HTTP = 'http',
  HTTPS = 'https',
  MINIO = 'minio',
  S3 = 's3',
  VOLUME = 'volume',
}

export interface StoragePath {
  source: StorageService;
  bucket: string;
  key: string;
  /** Whether `key` is a decoded storage key or the canonical path from an artifact URI. */
  keyEncoding?: 'storage' | 'uri';
  /** Exact persisted URI path when reconstructing it from `key` would change its spelling. */
  uriKey?: string;
}

export function parseStoragePath(strPath: string): StoragePath {
  if (strPath.startsWith('gs://')) {
    const pathParts = strPath.substr('gs://'.length).split('/');
    return {
      bucket: pathParts[0],
      key: pathParts.slice(1).join('/'),
      source: StorageService.GCS,
    };
  } else if (strPath.startsWith('minio://')) {
    const pathParts = strPath.substr('minio://'.length).split('/');
    return {
      bucket: pathParts[0],
      key: pathParts.slice(1).join('/'),
      source: StorageService.MINIO,
    };
  } else if (strPath.startsWith('s3://')) {
    const pathParts = strPath.substr('s3://'.length).split('/');
    return {
      bucket: pathParts[0],
      key: pathParts.slice(1).join('/'),
      source: StorageService.S3,
    };
  } else if (strPath.startsWith('http://')) {
    const pathParts = strPath.substr('http://'.length).split('/');
    return {
      bucket: pathParts[0],
      key: pathParts.slice(1).join('/'),
      source: StorageService.HTTP,
    };
  } else if (strPath.startsWith('https://')) {
    const pathParts = strPath.substr('https://'.length).split('/');
    return {
      bucket: pathParts[0],
      key: pathParts.slice(1).join('/'),
      source: StorageService.HTTPS,
    };
  } else if (strPath.startsWith('volume://')) {
    const pathParts = strPath.substr('volume://'.length).split('/');
    return {
      bucket: pathParts[0],
      key: pathParts.slice(1).join('/'),
      source: StorageService.VOLUME,
    };
  } else {
    throw new Error('Unsupported storage path: ' + strPath);
  }
}
