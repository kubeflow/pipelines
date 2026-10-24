import express from 'express';
import request from 'supertest';
import { vi } from 'vitest';
import { getHealthzHandler } from './healthz.js';

describe('healthz capability forwarding', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('preserves explicit false and its reason without changing health or legacy fields', async () => {
    const capabilities = {
      recurring_runs: { supported: false, reason: 'Controller unavailable' },
    };
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          new Response(JSON.stringify({ commit_sha: 'sha', multi_user: false, capabilities })),
        ),
    );
    const app = express().get(
      '/healthz',
      getHealthzHandler({ healthzStats: {}, healthzEndpoint: 'http://api/healthz' }),
    );
    const response = await request(app).get('/healthz');
    expect(response.status).toBe(200);
    expect(response.body).toMatchObject({
      apiServerReady: true,
      apiServerCommitHash: 'sha',
      multi_user: false,
      capabilities,
    });
  });

  it('removes stale capabilities when a server no longer advertises them', async () => {
    const fetch = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ capabilities: { recurring_runs: { supported: false } } })),
      )
      .mockResolvedValueOnce(new Response('{}'));
    vi.stubGlobal('fetch', fetch);
    const app = express().get(
      '/healthz',
      getHealthzHandler({ healthzStats: {}, healthzEndpoint: 'http://api/healthz' }),
    );
    await request(app).get('/healthz');
    const response = await request(app).get('/healthz');
    expect(response.body.apiServerReady).toBe(true);
    expect(response.body).not.toHaveProperty('capabilities');
  });
});
