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
export const DEFAULT_GCS_UNIVERSE_DOMAIN = 'googleapis.com';

export type GCSClient = Awaited<ReturnType<GoogleAuth['getClient']>>;

interface GCSListResponse {
  items?: Array<{ name?: string }>;
  nextPageToken?: string;
}

export async function getGCSClient(credentials?: CredentialBody): Promise<GCSClient> {
  const auth = new GoogleAuth({
    credentials,
    scopes: GCS_SCOPE,
  });
  return auth.getClient();
}

function getGCSApiBase(universeDomain?: string): string {
  const domain = universeDomain || DEFAULT_GCS_UNIVERSE_DOMAIN;
  if (!/^[a-z0-9.-]+$/i.test(domain) || domain.startsWith('.') || domain.endsWith('.')) {
    throw new Error(`Invalid GCS universe_domain: ${domain}`);
  }
  return `https://storage.${domain}/storage/v1`;
}

function getListObjectsUrl(
  bucket: string,
  prefix: string,
  pageToken?: string,
  universeDomain?: string,
): string {
  const url = new URL(`${getGCSApiBase(universeDomain)}/b/${encodeURIComponent(bucket)}/o`);
  url.searchParams.set('prefix', prefix);
  if (pageToken) {
    url.searchParams.set('pageToken', pageToken);
  }
  return url.toString();
}

function getDownloadObjectUrl(bucket: string, objectName: string, universeDomain?: string): string {
  const url = new URL(
    `${getGCSApiBase(universeDomain)}/b/${encodeURIComponent(bucket)}/o/${encodeURIComponent(objectName)}`,
  );
  url.searchParams.set('alt', 'media');
  return url.toString();
}

export async function listGCSObjectNames(options: {
  anonymous?: boolean;
  bucket: string;
  prefix: string;
  credentials?: CredentialBody;
  client?: GCSClient;
  universeDomain?: string;
}): Promise<string[]> {
  const { anonymous, bucket, prefix, credentials, client, universeDomain } = options;
  const resolvedClient = anonymous ? undefined : (client ?? (await getGCSClient(credentials)));
  const objectNames: string[] = [];

  let pageToken: string | undefined;
  do {
    const url = getListObjectsUrl(bucket, prefix, pageToken, universeDomain);
    let data: GCSListResponse;
    if (anonymous) {
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Anonymous GCS list request failed with HTTP ${response.status}.`);
      }
      data = (await response.json()) as GCSListResponse;
    } else {
      const response = await resolvedClient!.request<GCSListResponse>({ url });
      data = response.data;
    }
    objectNames.push(
      ...(data.items ?? [])
        .map((item) => item.name)
        .filter((name): name is string => typeof name === 'string' && name.length > 0),
    );
    pageToken = data.nextPageToken;
  } while (pageToken);

  return objectNames;
}

export async function downloadGCSObjectStream(options: {
  anonymous?: boolean;
  bucket: string;
  objectName: string;
  credentials?: CredentialBody;
  client?: GCSClient;
  universeDomain?: string;
}): Promise<Readable> {
  const { anonymous, bucket, objectName, credentials, client, universeDomain } = options;
  const url = getDownloadObjectUrl(bucket, objectName, universeDomain);
  if (anonymous) {
    const response = await fetch(url);
    if (!response.ok || !response.body) {
      throw new Error(`Anonymous GCS download request failed with HTTP ${response.status}.`);
    }
    return Readable.fromWeb(response.body);
  }
  const resolvedClient = client ?? (await getGCSClient(credentials));
  const response = await resolvedClient.request<Readable>({
    responseType: 'stream',
    url,
  });
  return response.data;
}
