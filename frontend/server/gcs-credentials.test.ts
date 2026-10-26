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

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { generateKeyPairSync, verify } from 'node:crypto';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { getGCSClient } from './gcs-helper.js';

const scope = 'https://www.googleapis.com/auth/devstorage.read_write';
const { privateKey, publicKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
const pem = privateKey.export({ type: 'pkcs8', format: 'pem' }).toString();
const credentials = {
  type: 'service_account',
  project_id: 'test-project',
  client_email: 'kfp@example.iam.gserviceaccount.com',
  private_key: pem,
};
let directory: string | undefined;

function setApplicationCredentials(contents: string, extension = 'json') {
  directory = mkdtempSync(path.join(tmpdir(), 'kfp-gcs-credentials-'));
  const filename = path.join(directory, `credentials.${extension}`);
  writeFileSync(filename, contents);
  vi.stubEnv('GOOGLE_APPLICATION_CREDENTIALS', filename);
}

beforeEach(() => {
  vi.stubEnv('GOOGLE_CLOUD_PROJECT', 'test-project');
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
  if (directory) rmSync(directory, { recursive: true, force: true });
  directory = undefined;
});

describe('GCS credential compatibility with the real Google auth library', () => {
  it.each(['secret', 'ADC'])('signs OAuth assertions from %s JSON credentials', async (source) => {
    if (source === 'ADC') setApplicationCredentials(JSON.stringify(credentials));
    const client = await getGCSClient(source === 'secret' ? credentials : undefined);
    // Intercept only transport: credential loading and JWT signing remain real.
    const request = vi.spyOn(client.transporter, 'request').mockResolvedValue({
      data: { access_token: 'test-access-token', expires_in: 3600, token_type: 'Bearer' },
      headers: {},
      status: 200,
      statusText: 'OK',
      config: {},
    });

    const headers = await client.getRequestHeaders(
      'https://storage.googleapis.com/storage/v1/b/test/o',
    );

    expect(headers.Authorization).toBe('Bearer test-access-token');
    expect(request).toHaveBeenCalledTimes(1);
    const options = request.mock.calls[0][0];
    expect(options.url).toBe('https://www.googleapis.com/oauth2/v4/token');
    const data = options.data as { grant_type: string; assertion: string };
    expect(data.grant_type).toBe('urn:ietf:params:oauth:grant-type:jwt-bearer');
    const [header, payload, signature] = data.assertion.split('.');
    expect(
      verify(
        'RSA-SHA256',
        Buffer.from(`${header}.${payload}`),
        publicKey,
        Buffer.from(signature, 'base64url'),
      ),
    ).toBe(true);
    expect(JSON.parse(Buffer.from(payload, 'base64url').toString())).toMatchObject({
      iss: credentials.client_email,
      scope,
    });
  });

  it('preserves authorized-user ADC refresh credentials', async () => {
    setApplicationCredentials(
      JSON.stringify({
        type: 'authorized_user',
        client_id: 'test-client',
        client_secret: 'test-secret',
        refresh_token: 'test-refresh-token',
      }),
    );
    const client = await getGCSClient();
    expect(client.credentials.refresh_token).toBe('test-refresh-token');
  });

  it.each(['pem', 'p12'])(
    'rejects unsupported %s ADC files instead of treating them as JSON',
    async (extension) => {
      setApplicationCredentials(extension === 'pem' ? pem : 'not-json-p12', extension);
      await expect(getGCSClient()).rejects.toThrow(/JSON|Unexpected token/);
    },
  );
});
