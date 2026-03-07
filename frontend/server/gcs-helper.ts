// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { CredentialBody, GoogleAuth } from 'google-auth-library';
import { Readable } from 'stream';

const GCS_SCOPE = 'https://www.googleapis.com/auth/devstorage.read_write';
const GCS_API_BASE = 'https://storage.googleapis.com/storage/v1';

interface GCSListResponse {
  items?: Array<{ name?: string }>;
  nextPageToken?: string;
}

async function getGCSClient(credentials?: CredentialBody) {
  const auth = new GoogleAuth({
    credentials,
    scopes: GCS_SCOPE,
  });
  return auth.getClient();
}

function getListObjectsUrl(bucket: string, prefix: string, pageToken?: string): string {
  const url = new URL(`${GCS_API_BASE}/b/${encodeURIComponent(bucket)}/o`);
  url.searchParams.set('prefix', prefix);
  if (pageToken) {
    url.searchParams.set('pageToken', pageToken);
  }
  return url.toString();
}

function getDownloadObjectUrl(bucket: string, objectName: string): string {
  const url = new URL(
    `${GCS_API_BASE}/b/${encodeURIComponent(bucket)}/o/${encodeURIComponent(objectName)}`,
  );
  url.searchParams.set('alt', 'media');
  return url.toString();
}

export async function listGCSObjectNames(options: {
  bucket: string;
  prefix: string;
  credentials?: CredentialBody;
}): Promise<string[]> {
  const { bucket, prefix, credentials } = options;
  const client = await getGCSClient(credentials);
  const objectNames: string[] = [];

  let pageToken: string | undefined;
  do {
    const response = await client.request<GCSListResponse>({
      url: getListObjectsUrl(bucket, prefix, pageToken),
    });
    objectNames.push(
      ...(response.data.items ?? [])
        .map((item) => item.name)
        .filter((name): name is string => typeof name === 'string' && name.length > 0),
    );
    pageToken = response.data.nextPageToken;
  } while (pageToken);

  return objectNames;
}

export async function downloadGCSObjectStream(options: {
  bucket: string;
  objectName: string;
  credentials?: CredentialBody;
}): Promise<Readable> {
  const { bucket, objectName, credentials } = options;
  const client = await getGCSClient(credentials);
  const response = await client.request<Readable>({
    responseType: 'stream',
    url: getDownloadObjectUrl(bucket, objectName),
  });
  return response.data;
}
