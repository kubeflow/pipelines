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
import { CoreV1Api, V1Secret } from '@kubernetes/client-node';
import { initializeTensorboardSigningKey } from './initialize-tensorboard-signing-key.js';

const namespace = 'kubeflow';
const name = 'ml-pipeline-ui-tensorboard-proxy';
const key = 'signing-secret';
const encode = (value: string) => Buffer.from(value).toString('base64');
const wait = async () => {};

function fakeApi(initial: V1Secret = { metadata: { name, resourceVersion: '1' } }) {
  let stored = structuredClone(initial);
  const readNamespacedSecret = vi.fn(async () => structuredClone(stored));
  const replaceNamespacedSecret = vi.fn(async ({ body }: { body: V1Secret }) => {
    if (body.metadata?.resourceVersion !== stored.metadata?.resourceVersion) {
      throw { code: 409 };
    }
    stored = structuredClone(body);
    stored.metadata!.resourceVersion = String(Number(stored.metadata!.resourceVersion) + 1);
    return structuredClone(stored);
  });
  const api: Pick<CoreV1Api, 'readNamespacedSecret' | 'replaceNamespacedSecret'> = {
    readNamespacedSecret,
    replaceNamespacedSecret,
  };
  return { api, readNamespacedSecret, replaceNamespacedSecret, stored: () => stored };
}

describe('initializeTensorboardSigningKey', () => {
  it('generates a dedicated random key and preserves unrelated Secret fields', async () => {
    const fake = fakeApi({
      metadata: { name, resourceVersion: '1', labels: { managed: 'externally' } },
      data: { unrelated: encode('keep') },
    });
    await initializeTensorboardSigningKey(fake.api, namespace, name, key, wait);
    const generated = Buffer.from(fake.stored().data![key], 'base64').toString();
    expect(Buffer.from(generated, 'base64url')).toHaveLength(32);
    expect(generated).toMatch(/^[A-Za-z0-9_-]{43}$/);
    expect(fake.stored().data!.unrelated).toBe(encode('keep'));
    expect(fake.stored().metadata!.labels).toEqual({ managed: 'externally' });
    expect(fake.readNamespacedSecret).toHaveBeenCalledWith({ namespace, name });
    await initializeTensorboardSigningKey(fake.api, namespace, name, key, wait);
    expect(fake.replaceNamespacedSecret).toHaveBeenCalledTimes(1);
  });

  it('preserves an operator-provided key without updating the Secret', async () => {
    const fake = fakeApi({ metadata: { name }, data: { [key]: encode('x'.repeat(32)) } });
    await initializeTensorboardSigningKey(fake.api, namespace, name, key, wait);
    expect(fake.replaceNamespacedSecret).not.toHaveBeenCalled();
  });

  it('converges on one key when two initializers race', async () => {
    const fake = fakeApi();
    await Promise.all([
      initializeTensorboardSigningKey(fake.api, namespace, name, key, wait),
      initializeTensorboardSigningKey(fake.api, namespace, name, key, wait),
    ]);
    expect(fake.replaceNamespacedSecret).toHaveBeenCalledTimes(2);
    expect(fake.readNamespacedSecret).toHaveBeenCalledTimes(3);
    expect(fake.stored().metadata!.resourceVersion).toBe('2');
    const winner = fake.stored().data![key];
    await initializeTensorboardSigningKey(fake.api, namespace, name, key, wait);
    expect(fake.stored().data![key]).toBe(winner);
  });

  it.each([404, 409, 429, 500, 502, 503, 504])(
    'retries HTTP %s and rereads the Secret',
    async (code) => {
      const fake = fakeApi();
      fake.readNamespacedSecret.mockRejectedValueOnce({ code });
      await initializeTensorboardSigningKey(fake.api, namespace, name, key, wait);
      expect(fake.readNamespacedSecret).toHaveBeenCalledTimes(2);
      expect(fake.replaceNamespacedSecret).toHaveBeenCalledTimes(1);
    },
  );

  it('preserves a committed key after an uncertain update response', async () => {
    const fake = fakeApi();
    const replace = fake.replaceNamespacedSecret.getMockImplementation()!;
    fake.replaceNamespacedSecret.mockImplementationOnce(async (request) => {
      await replace(request);
      throw { code: 504 };
    });
    await initializeTensorboardSigningKey(fake.api, namespace, name, key, wait);
    expect(fake.replaceNamespacedSecret).toHaveBeenCalledTimes(1);
    expect(fake.stored().metadata!.resourceVersion).toBe('2');
  });

  it.each(['', encode('too short'), 'not base64'.repeat(10)])(
    'rejects an invalid existing key without replacing it',
    async (existing) => {
      const fake = fakeApi({ metadata: { name, resourceVersion: '1' }, data: { [key]: existing } });
      await expect(
        initializeTensorboardSigningKey(fake.api, namespace, name, key, wait),
      ).rejects.toThrow('Existing keys are never replaced');
      expect(fake.stored().data![key]).toBe(existing);
      expect(fake.replaceNamespacedSecret).not.toHaveBeenCalled();
    },
  );

  it('never issues an unconditional update without resourceVersion', async () => {
    const fake = fakeApi({ metadata: { name } });
    await expect(
      initializeTensorboardSigningKey(fake.api, namespace, name, key, wait),
    ).rejects.toThrow();
    expect(fake.replaceNamespacedSecret).not.toHaveBeenCalled();
  });

  it('bounds retries and does not disclose API error bodies', async () => {
    const fake = fakeApi();
    fake.readNamespacedSecret.mockRejectedValue({ code: 503, body: 'sensitive-value' });
    await expect(
      initializeTensorboardSigningKey(fake.api, namespace, name, key, wait),
    ).rejects.toThrow(/^Unable to initialize the TensorBoard signing Secret/);
    expect(fake.readNamespacedSecret).toHaveBeenCalledTimes(60);
  });

  it('fails immediately on authorization errors without disclosing their contents', async () => {
    const fake = fakeApi();
    fake.readNamespacedSecret.mockRejectedValue({ code: 403, message: 'sensitive-value' });
    await expect(
      initializeTensorboardSigningKey(fake.api, namespace, name, key, wait),
    ).rejects.not.toThrow('sensitive-value');
    expect(fake.readNamespacedSecret).toHaveBeenCalledTimes(1);
  });
});
