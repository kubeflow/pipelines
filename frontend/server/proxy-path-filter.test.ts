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

import { describe, expect, it, vi } from 'vitest';
import { createProxyMiddleware } from 'http-proxy-middleware';
import { UIServer } from './app.js';
import { loadConfigs } from './configs.js';
import { commonSetup } from './integration-tests/test-helper.js';

vi.mock('http-proxy-middleware', async (importOriginal) => {
  const original = await importOriginal<typeof import('http-proxy-middleware')>();
  return { ...original, createProxyMiddleware: vi.fn(original.createProxyMiddleware) };
});

describe('proxy path filters', () => {
  const { argv } = commonSetup();

  it('registers every proxy without glob pattern filters', async () => {
    vi.mocked(createProxyMiddleware).mockClear();
    const app = new UIServer(
      loadConfigs(argv, {
        ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
        STREAM_LOGS_FROM_SERVER_API: 'true',
      }),
    );
    try {
      // Two artifact routes, TensorBoard, pod logs, and two API routes.
      expect(createProxyMiddleware).toHaveBeenCalledTimes(6);
      for (const [options] of vi.mocked(createProxyMiddleware).mock.calls) {
        // GHSA-vfj7-8cjw-p6xm: request data must not become brace patterns.
        expect(['undefined', 'function']).toContain(typeof options.pathFilter);
      }
    } finally {
      await app.close();
    }
  });
});
