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
import { randomBytes } from 'node:crypto';
import { pathToFileURL } from 'node:url';
import { setTimeout } from 'node:timers/promises';
import { CoreV1Api, KubeConfig } from '@kubernetes/client-node';

function statusCode(error: unknown): number | undefined {
  if (typeof error === 'object' && error !== null && 'code' in error) {
    return typeof error.code === 'number' ? error.code : undefined;
  }
  return undefined;
}

// Updating the pre-created Secret avoids granting this Job permission to create arbitrary Secrets.
export async function initializeTensorboardSigningKey(
  api: Pick<CoreV1Api, 'readNamespacedSecret' | 'replaceNamespacedSecret'>,
  namespace: string,
  name: string,
  key: string,
  wait: () => Promise<unknown> = () => setTimeout(1000),
): Promise<void> {
  for (let attempt = 0; attempt < 60; attempt++) {
    try {
      const secret = await api.readNamespacedSecret({ namespace, name });
      const existing = secret.data?.[key];
      if (existing !== undefined) {
        const decoded = Buffer.from(existing, 'base64');
        if (decoded.length < 32 || decoded.toString('base64') !== existing) {
          throw new Error('invalid signing key');
        }
        return;
      }
      if (!secret.metadata?.resourceVersion) {
        throw new Error('missing resource version');
      }
      secret.data = {
        ...secret.data,
        [key]: Buffer.from(randomBytes(32).toString('base64url')).toString('base64'),
      };
      // resourceVersion makes concurrent initializers compare-and-swap: the loser rereads the winner.
      await api.replaceNamespacedSecret({ namespace, name, body: secret });
      return;
    } catch (error) {
      const code = statusCode(error);
      if (code !== undefined && [404, 409, 429, 500, 502, 503, 504].includes(code)) {
        if (attempt < 59) {
          await wait();
          continue;
        }
      }
      // Kubernetes errors may contain Secret data. Never include their message/body in logs.
      // eslint-disable-next-line preserve-caught-error -- API error causes may disclose Secret data.
      throw new Error(
        'Unable to initialize the TensorBoard signing Secret. Check that it exists, the Job has get/update permission, and any existing signing key is at least 32 bytes. Existing keys are never replaced.',
      );
    }
  }
}

async function main(): Promise<void> {
  const namespace = process.env.POD_NAMESPACE;
  const name = process.env.SIGNING_SECRET_NAME;
  const key = process.env.SIGNING_SECRET_KEY;
  if (!namespace || !name || !key) {
    throw new Error('Set POD_NAMESPACE, SIGNING_SECRET_NAME and SIGNING_SECRET_KEY.');
  }
  const config = new KubeConfig();
  config.loadFromCluster();
  await initializeTensorboardSigningKey(config.makeApiClient(CoreV1Api), namespace, name, key);
  console.info('TensorBoard signing Secret is ready.');
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch(() => {
    console.error(
      'TensorBoard signing Secret initialization failed. Check Job configuration, Secret key length, and get/update permission. Existing keys are never replaced.',
    );
    process.exitCode = 1;
  });
}
