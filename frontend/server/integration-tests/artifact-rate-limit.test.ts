// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import requests from 'supertest';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import { commonSetup } from './test-helper.js';

describe('artifact request rate limiting', () => {
  const { argv } = commonSetup();

  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] });
    vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('shares a user budget across previews, downloads, namespaces, and base paths before authorization', async () => {
    const fetchArtifact = vi.fn(async () => new Response('Forbidden', { status: 403 }));
    vi.stubGlobal('fetch', fetchArtifact);
    const options = loadConfigs(argv, {
      ENABLE_AUTHZ: 'true',
      KUBEFLOW_USERID_HEADER: 'x-kubeflow-user',
    });
    options.server.basePath = '/custom';
    const app = new UIServer(options);
    const request = requests(app.app);
    const query = '?source=s3&bucket=bucket&key=data&namespace=team-a';
    const paths = [
      `/artifacts/get${query}`,
      `/custom/artifacts/get${query}`,
      '/artifacts/artifact-1/preview?namespace=team-a',
      '/custom/artifacts/artifact-1/preview?namespace=team-a',
    ];

    for (let index = 0; index < 300; index++) {
      const response = await request
        .get(paths[index % paths.length])
        .set('x-kubeflow-user', 'reader');
      expect(response.status, `Artifact request ${index}: ${response.text}`).toBe(403);
    }
    expect(fetchArtifact).toHaveBeenCalledTimes(300);

    for (const path of paths) {
      const response = await request
        .get(path.replace('team-a', 'team-b'))
        .set('x-kubeflow-user', 'reader')
        .expect(429);
      expect(response.headers['retry-after']).toBe('60');
      expect(response.headers['ratelimit']).toContain('r=0');
      expect(response.headers['cache-control']).toBe('no-store');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
    }
    expect(fetchArtifact).toHaveBeenCalledTimes(300);

    await request.get(paths[0]).set('x-kubeflow-user', 'another-reader').expect(403);
    expect(fetchArtifact).toHaveBeenCalledTimes(301);
    await request.get('/').expect(200);

    await vi.advanceTimersByTimeAsync(60_000);
    await request.get(paths[0]).set('x-kubeflow-user', 'reader').expect(403);
    expect(fetchArtifact).toHaveBeenCalledTimes(302);
  });

  it('uses the client IP without trusting identity or forwarding headers when auth is disabled', async () => {
    const options = loadConfigs(argv, {
      ENABLE_AUTHZ: 'false',
      KUBEFLOW_USERID_HEADER: 'x-kubeflow-user',
    });
    const app = new UIServer(options);
    const request = requests(app.app);

    for (let index = 0; index < 300; index++) {
      await request
        .get('/artifacts/get?source=s3&source=gs')
        .set('x-kubeflow-user', `reader-${index}`)
        .set('X-Forwarded-For', `192.0.2.${index % 250}`)
        .expect(400);
    }
    await request
      .get('/artifacts/artifact-1/preview')
      .set('x-kubeflow-user', 'another-reader')
      .set('X-Forwarded-For', '198.51.100.1')
      .expect(429);
  });

  it('bounds unauthenticated requests before invoking the authorization service', async () => {
    const fetchArtifact = vi.fn();
    vi.stubGlobal('fetch', fetchArtifact);
    const options = loadConfigs(argv, { ENABLE_AUTHZ: 'true' });
    const app = new UIServer(options);
    const request = requests(app.app);

    for (let index = 0; index < 300; index++) {
      const response = await request.get('/artifacts/get');
      expect(response.status, `Unauthenticated request ${index}: ${response.text}`).toBe(401);
    }
    await request.get('/artifacts/artifact-1/preview?namespace=team-a').expect(429);
    expect(fetchArtifact).not.toHaveBeenCalled();
  });
});
