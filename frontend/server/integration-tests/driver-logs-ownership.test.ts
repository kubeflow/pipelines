// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import express from 'express';
import requests from 'supertest';
import { afterEach, describe, expect, it, vi } from 'vitest';

describe('driver log artifact ownership', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
    vi.resetModules();
  });

  it.each(['artifact-only', 'mlmd-only'])(
    'requires a namespace-owned driver-log Artifact in %s mode',
    async (mode) => {
      vi.stubEnv('ARTIFACT_NAMESPACE_OWNERSHIP_MODE', mode);
      vi.resetModules();
      const uri = 's3://bucket/private-artifacts/team-a/run/task/driver-logs';
      let registered = false;
      const fetchArtifact = vi.fn(async (input: string | URL | Request) => {
        const url = new URL(String(input));
        expect(url.pathname).toContain('/artifacts');
        expect(url.searchParams.get('namespace')).toBe('team-a');
        const filter = JSON.parse(decodeURIComponent(url.searchParams.get('filter')!));
        expect(filter.predicates[0].stringValue).toBe(uri);
        return new Response(
          JSON.stringify({
            artifacts: registered
              ? [{ artifact_id: 'driver-log-id', name: 'driver-logs', uri, namespace: 'team-a' }]
              : [],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        );
      });
      vi.stubGlobal('fetch', fetchArtifact);
      const { getArtifactsAuthMiddleware } = await import('../handlers/artifacts.js');
      const app = express();
      app.get(
        '/artifacts/get',
        getArtifactsAuthMiddleware(
          async () => undefined,
          true,
          'x-kubeflow-user',
          'http://artifact-api.test',
        ),
        (_request, response) => response.status(200).send('authorized driver logs'),
      );
      const query = {
        source: 's3',
        bucket: 'bucket',
        key: 'private-artifacts/team-a/run/task/driver-logs',
        namespace: 'team-a',
      };
      await requests(app)
        .get('/artifacts/get')
        .set('x-kubeflow-user', 'user')
        .query(query)
        .expect(403);
      registered = true;
      await requests(app)
        .get('/artifacts/get')
        .set('x-kubeflow-user', 'user')
        .query(query)
        .expect(200);
      expect(fetchArtifact).toHaveBeenCalledTimes(2);
      await requests(app)
        .get('/artifacts/get')
        .set('x-kubeflow-user', 'other-user')
        .query({ ...query, namespace: 'team-b' })
        .expect(403);
      expect(fetchArtifact).toHaveBeenCalledTimes(2);
    },
  );
});
