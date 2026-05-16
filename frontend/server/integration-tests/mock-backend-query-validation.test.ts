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

import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
import requests from 'supertest';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const frontendRoot = path.resolve(__dirname, '..', '..');
const originalCwd = process.cwd();

async function createRequest(): Promise<ReturnType<typeof requests>> {
  vi.resetModules();
  const { default: mockApiMiddleware } = await import('../../mock-backend/mock-api-middleware.ts');
  const app = express();
  mockApiMiddleware(app as any);
  return requests(app);
}

beforeAll(() => {
  process.chdir(frontendRoot);
});

afterAll(() => {
  process.chdir(originalCwd);
});

describe('mock backend decoded query validation', () => {
  let request: ReturnType<typeof requests>;

  beforeEach(async () => {
    vi.restoreAllMocks();
    vi.spyOn(console, 'info').mockImplementation(() => {});
    vi.spyOn(console, 'log').mockImplementation(() => {});
    request = await createRequest();
  });

  it('rejects pipeline upload requests without a name query param', async () => {
    await request
      .post('/apis/v1beta1/pipelines/upload')
      .send({ uploaded: true })
      .expect(400)
      .expect('name argument is required');
  });

  it('rejects pipeline upload requests with invalid percent-encoding in the name query param', async () => {
    await request
      .post('/apis/v1beta1/pipelines/upload?name=%E0%A4%A')
      .send({ uploaded: true })
      .expect(400)
      .expect('name argument is invalid');
  });

  it('rejects artifact requests without a key query param', async () => {
    await request.get('/artifacts/get').expect(400).expect('key argument is required');
  });

  it('rejects artifact requests with invalid percent-encoding in the key query param', async () => {
    await request.get('/artifacts/get?key=%E0%A4%A').expect(400).expect('key argument is invalid');
  });

  it('rejects pod log requests without a podname query param', async () => {
    await request.get('/k8s/pod/logs').expect(400).expect('podname argument is required');
  });

  it('rejects pod log requests with invalid percent-encoding in the podname query param', async () => {
    await request
      .get('/k8s/pod/logs?podname=%E0%A4%A')
      .expect(400)
      .expect('podname argument is invalid');
  });
});
