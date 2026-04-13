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
import { buildQuery } from './test-helper.js';

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

function asText(test: requests.Test): requests.Test {
  return test.buffer(true).parse((response, callback) => {
    response.setEncoding('utf8');
    let text = '';
    response.on('data', (chunk) => {
      text += chunk;
    });
    response.on('end', () => {
      callback(null, text);
    });
  });
}

beforeAll(() => {
  process.chdir(frontendRoot);
});

afterAll(() => {
  process.chdir(originalCwd);
});

describe('mock backend routes', () => {
  let request: ReturnType<typeof requests>;

  beforeEach(async () => {
    vi.restoreAllMocks();
    vi.spyOn(console, 'info').mockImplementation(() => {});
    vi.spyOn(console, 'log').mockImplementation(() => {});
    request = await createRequest();
  });

  describe('basic endpoints', () => {
    it('serves healthz status', async () => {
      const response = await request.get('/apis/v1beta1/healthz').expect(200);

      expect(response.body).toMatchObject({
        apiServerReady: true,
        apiServerCommitHash: 'd3c4add0a95e930c70a330466d0923827784eb9a',
      });
      expect(response.body.frontendCommitHash).toBeDefined();
    });

    it('serves the hub endpoint', async () => {
      await request.get('/hub/').expect(200);
    });

    it('serves cluster and project metadata', async () => {
      await request.get('/system/cluster-name').expect(200, 'mock-cluster-name');
      await request.get('/system/project-id').expect(200, 'mock-project-id');
    });

    it('reports visualizations as allowed', async () => {
      await request.get('/visualizations/allowed').expect(200, 'true');
    });

    it('returns 404 for unknown v1beta1 endpoints', async () => {
      await request.get('/apis/v1beta1/does-not-exist').expect(404, 'Bad request endpoint.');
    });
  });

  describe('experiments', () => {
    it('lists experiments using current camelCase pagination parameters', async () => {
      const response = await request
        .get('/apis/v1beta1/experiments?pageToken=2&pageSize=2')
        .expect(200);

      expect(response.body.next_page_token).toBe('4');
      expect(response.body.experiments.map((experiment: { id: string }) => experiment.id)).toEqual([
        '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
        '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
      ]);
    });

    it('filters experiments by name', async () => {
      const filter = JSON.stringify({
        predicates: [{ key: 'name', op: 'EQUALS', string_value: 'No Runs' }],
      });
      const response = await request
        .get(`/apis/v1beta1/experiments${buildQuery({ filter })}`)
        .expect(200);

      expect(response.body.experiments).toHaveLength(1);
      expect(response.body.experiments[0]).toMatchObject({
        id: '7fc01714-4a13-4c05-5902-a8a72c14253b',
        name: 'No Runs',
      });
    });

    it('creates and fetches an experiment by id', async () => {
      const createResponse = await request
        .post('/apis/v1beta1/experiments')
        .send({ name: 'Mock Experiment', description: 'Created from tests' })
        .expect(200);

      expect(createResponse.body).toMatchObject({
        id: 'new-experiment-6',
        name: 'Mock Experiment',
      });

      await request
        .get('/apis/v1beta1/experiments/new-experiment-6')
        .expect(200)
        .expect(({ body }) => {
          expect(body.description).toBe('Created from tests');
        });
    });

    it('rejects duplicate experiment names', async () => {
      await request
        .post('/apis/v1beta1/experiments')
        .send({ name: 'No Runs' })
        .expect(404, 'An experiment with the same name already exists');
    });
  });

  describe('jobs', () => {
    it('lists jobs with current snake_case pagination parameters', async () => {
      const response = await request.get('/apis/v1beta1/jobs?page_token=0&page_size=2').expect(200);

      expect(response.body.next_page_token).toBe('2');
      expect(response.body.jobs.map((job: { id: string }) => job.id)).toEqual([
        'Some-job-id-4',
        'Some-job-id-5',
      ]);
    });

    it('fetches a job by id', async () => {
      const response = await request.get('/apis/v1beta1/jobs/Some-job-id-4').expect(200);
      expect(response.body).toMatchObject({
        id: 'Some-job-id-4',
        name: 'Job#4',
      });
    });

    it('creates a job', async () => {
      const response = await request
        .post('/apis/v1beta1/jobs')
        .send({ name: 'route-test-job', trigger: { cron_schedule: { cron: '* * * * *' } } })
        .expect(200);

      expect(response.body).toMatchObject({
        id: 'new-job-24',
        name: 'route-test-job',
        enabled: true,
      });
    });

    it('enables and disables a job', async () => {
      await request.post('/apis/v1beta1/jobs/Some-job-id-4/enable').expect(200, {});
      await request
        .get('/apis/v1beta1/jobs/Some-job-id-4')
        .expect(200)
        .expect(({ body }) => {
          expect(body.enabled).toBe(true);
        });

      await request.post('/apis/v1beta1/jobs/Some-job-id-4/disable').expect(200, {});
      await request
        .get('/apis/v1beta1/jobs/Some-job-id-4')
        .expect(200)
        .expect(({ body }) => {
          expect(body.enabled).toBe(false);
        });
    });

    it('rejects deleting protected jobs', async () => {
      const response = await asText(
        request.delete('/apis/v1beta1/jobs/7fc01714-4a13-4c05-7186-a8a72c14253b'),
      ).expect(502);
      expect(response.body).toBe("Deletion failed for job: 'Cannot be deleted - 1'");
    });

    it('returns 405 for unsupported job methods', async () => {
      const response = await asText(request.put('/apis/v1beta1/jobs/Some-job-id-4')).expect(405);
      expect(response.body).toBe('Unsupported request type: PUT');
    });
  });

  describe('runs', () => {
    it('lists runs filtered by experiment ownership', async () => {
      const response = await request
        .get(
          `/apis/v1beta1/runs${buildQuery({
            'resource_reference_key.type': 'EXPERIMENT',
            'resource_reference_key.id': '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
            page_size: 5,
          })}`,
        )
        .expect(200);

      expect(response.body.runs).toHaveLength(5);
      expect(response.body.next_page_token).toBe('5');
      response.body.runs.forEach(
        (run: { resource_references?: Array<{ key?: { id?: string } }> }) => {
          expect(
            run.resource_references?.some(
              (reference) => reference.key?.id === '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
            ),
          ).toBe(true);
        },
      );
    });

    it('fetches a run by id', async () => {
      const response = await request
        .get('/apis/v1beta1/runs/3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7')
        .expect(200);

      expect(response.body.run).toMatchObject({
        id: '3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7',
        name: 'coinflip-recursive-run-lknlfs3',
      });
    });

    it('archives and unarchives an existing run', async () => {
      await request
        .post('/apis/v1beta1/runs/3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7:archive')
        .expect(200, {});
      await request
        .get('/apis/v1beta1/runs/3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7')
        .expect(200)
        .expect(({ body }) => {
          expect(body.run.storage_state).toBe('STORAGESTATE_ARCHIVED');
        });

      await request
        .post('/apis/v1beta1/runs/3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7:unarchive')
        .expect(200, {});
      await request
        .get('/apis/v1beta1/runs/3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7')
        .expect(200)
        .expect(({ body }) => {
          expect(body.run.storage_state).toBe('STORAGESTATE_AVAILABLE');
        });
    });

    it('rejects unsupported run methods', async () => {
      await request
        .post('/apis/v1beta1/runs/3308d0ec-f1b3-4488-a2d3-8ad0f91e88e7:retry')
        .expect(500, 'Bad method');
    });

    it.todo('creates runs with a route response that matches the created run detail');
  });

  describe('pipelines', () => {
    it('lists pipelines with pagination', async () => {
      const response = await request
        .get('/apis/v1beta1/pipelines?page_token=0&page_size=3')
        .expect(200);

      expect(response.body.next_page_token).toBe('3');
      expect(response.body.pipelines.map((pipeline: { id: string }) => pipeline.id)).toEqual([
        'Some-pipeline-id-12',
        'Some-pipeline-id-13',
        'Some-pipeline-id-14',
      ]);
    });

    it('fetches a pipeline by id', async () => {
      const response = await request
        .get('/apis/v1beta1/pipelines/8fbe3bd6-a01f-11e8-98d0-529269fb1459')
        .expect(200);
      expect(response.body).toMatchObject({
        id: '8fbe3bd6-a01f-11e8-98d0-529269fb1459',
        name: 'Unstructured text',
      });
    });

    it('filters pipelines by name', async () => {
      const filter = JSON.stringify({
        predicates: [{ key: 'name', op: 'IS_SUBSTRING', string_value: 'Markdown' }],
      });
      const response = await request
        .get(`/apis/v1beta1/pipelines${buildQuery({ filter })}`)
        .expect(200);

      expect(response.body.pipelines).toHaveLength(1);
      expect(response.body.pipelines[0]).toMatchObject({
        id: '8fbe3bd6-a01f-11e8-98d0-529269fb1499',
        name: 'Markdown description',
      });
    });

    it('returns template content for a pipeline', async () => {
      const response = await request
        .get('/apis/v1beta1/pipelines/8fbe3bd6-a01f-11e8-98d0-529269fb1460/templates')
        .expect(200);

      const payload = JSON.parse(response.text) as { template: string };
      expect(payload.template).toContain('"components"');
      expect(payload.template).toContain('"executorLabel"');
    });

    it('creates and deletes a pipeline', async () => {
      const createResponse = await request
        .post('/apis/v1beta1/pipelines')
        .send({ name: 'Created by route tests' })
        .expect(200);

      expect(createResponse.body).toMatchObject({
        id: 'new-pipeline-42',
        name: 'Created by route tests',
      });

      await request.delete('/apis/v1beta1/pipelines/new-pipeline-42').expect(200, {});
      const response = await asText(request.get('/apis/v1beta1/pipelines/new-pipeline-42')).expect(
        404,
      );
      expect(response.body).toBe('No pipeline was found with ID: new-pipeline-42');
    });

    it('rejects deleting protected pipelines', async () => {
      const response = await asText(
        request.delete('/apis/v1beta1/pipelines/8fbe3bd6-a01f-11e8-98d0-529269f77777'),
      ).expect(502);
      expect(response.body).toBe("Deletion failed for pipeline: 'Cannot be deleted'");
    });

    it('uploads pipelines and decodes the query name', async () => {
      const response = await request
        .post('/apis/v1beta1/pipelines/upload?name=My%20Uploaded%20Pipeline')
        .send({ uploaded: true })
        .expect(200);

      expect(response.body).toMatchObject({
        id: 'new-pipeline-42',
        name: 'My Uploaded Pipeline',
      });
    });

    it('rejects duplicate uploaded pipeline names', async () => {
      const response = await asText(
        request.post('/apis/v1beta1/pipelines/upload?name=Unstructured%20text').send({}),
      ).expect(502);
      expect(response.body).toBe(
        'A Pipeline named: "Unstructured text" already exists. Please choose a different name.',
      );
    });
  });

  describe('pipeline versions', () => {
    it('lists explicit pipeline versions with pagination', async () => {
      const response = await request
        .get(
          `/apis/v1beta1/pipeline_versions${buildQuery({
            'resource_key.type': 'PIPELINE',
            'resource_key.id': '8fbe3bd6-a01f-11e8-98d0-529269fb1460',
            page_size: 1,
            page_token: 0,
          })}`,
        )
        .expect(200);

      expect(response.body.next_page_token).toBe('1');
      expect(response.body.versions).toHaveLength(1);
      expect(response.body.versions[0]).toMatchObject({
        id: '8fbe3bd6-a01f-11e8-98d0-529269fb1460',
        name: 'default version',
      });
    });

    it('falls back to a pipeline default version when no explicit version list exists', async () => {
      const response = await request
        .get(
          `/apis/v1beta1/pipeline_versions${buildQuery({
            'resource_key.type': 'PIPELINE',
            'resource_key.id': '8fbe3bd6-a01f-11e8-98d0-529269fb1459',
            page_size: 1,
          })}`,
        )
        .expect(200);

      expect(response.body.total_size).toBe(1);
      expect(response.body.versions).toHaveLength(1);
      expect(response.body.versions[0]).toMatchObject({
        id: '8fbe3bd6-a01f-11e8-98d0-529269fb1459',
        name: 'Unstructured text',
      });
    });

    it('fetches a pipeline version by id', async () => {
      const response = await request
        .get('/apis/v1beta1/pipeline_versions/9fbe3bd6-a01f-11e8-98d0-529269fb1460')
        .expect(200);

      expect(response.body).toMatchObject({
        id: '9fbe3bd6-a01f-11e8-98d0-529269fb1460',
        name: 'revision',
      });
    });

    it('returns template content for a v2 pipeline version', async () => {
      const response = await request
        .get('/apis/v1beta1/pipeline_versions/9fbe3bd6-a01f-11e8-98d0-529269fb1460/templates')
        .expect(200);

      const payload = JSON.parse(response.text) as { template: string };
      expect(payload.template).toContain('components:');
      expect(payload.template).toContain('defaultPipelineRoot: minio://dummy_root');
    });

    it.todo('serves v1 pipeline version templates from a stable checked-in path');
  });

  describe('file and pod endpoints', () => {
    it('serves artifact files based on decoded keys', async () => {
      const response = await asText(request.get('/artifacts/get?key=folder%2Froc.csv')).expect(200);
      expect(response.body).toContain('0.0,0.00265957446809,0.999972701073');
    });

    it('returns a dummy artifact payload for unknown keys', async () => {
      const response = await asText(request.get('/artifacts/get?key=unknown-file')).expect(200);
      expect(response.body).toBe('dummy file for key: unknown-file');
    });

    it('tracks tensorboard state within the current app instance', async () => {
      await request.get('/apps/tensorboard').expect(200, '');
      await request.post('/apps/tensorboard').expect(200, 'ok');
      await request.get('/apps/tensorboard').expect(200, 'http://tensorboardserver:port');
    });

    it('returns the expected pod log error paths', async () => {
      await request.get('/k8s/pod/logs?podname=json-12abc').expect(404, 'pod not found');
      await request
        .get('/k8s/pod/logs?podname=coinflip-recursive-q7dqb-3721646052')
        .expect(500, 'Failed to retrieve log');
    });

    it('returns short pod logs for standard pods', async () => {
      const response = await request.get('/k8s/pod/logs?podname=hello-world-7sm94').expect(200);
      expect(response.text).toContain('< hello world >');
    });
  });
});
