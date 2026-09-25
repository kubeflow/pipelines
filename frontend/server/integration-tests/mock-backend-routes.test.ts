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

import { beforeEach, describe, expect, it } from 'vitest';
import {
  asText,
  buildQuery,
  createMockBackendRequest,
  setupMockBackendTest,
} from './test-helper.js';

setupMockBackendTest();

describe('mock backend routes', () => {
  let request: Awaited<ReturnType<typeof createMockBackendRequest>>;

  beforeEach(async () => {
    request = await createMockBackendRequest();
  });

  describe('basic endpoints', () => {
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

    it('serves v2 healthz status', async () => {
      const response = await request.get('/apis/v2beta1/healthz').expect(200);

      expect(response.body).toMatchObject({
        apiServerReady: true,
        apiServerMultiUser: false,
        pipelineStore: 'database',
      });
      expect(response.body.frontendCommitHash).toBeDefined();
    });

    it('returns 404 for unknown v2beta1 endpoints', async () => {
      await request.get('/apis/v2beta1/does-not-exist').expect(404, 'Bad request endpoint.');
    });
  });

  describe('v2 fixture routes', () => {
    it('lists pipelines with v2 field names, pagination, and total size', async () => {
      const response = await request
        .get('/apis/v2beta1/pipelines?page_token=0&page_size=3')
        .expect(200);

      expect(response.body.next_page_token).toBe('3');
      expect(response.body.total_size).toBeGreaterThan(3);
      expect(response.body.pipelines).toHaveLength(3);
      expect(response.body.pipelines[0]).toMatchObject({
        pipeline_id: expect.any(String),
        display_name: expect.any(String),
      });
    });

    it('filters v2 pipelines by name', async () => {
      const filter = JSON.stringify({
        predicates: [{ key: 'name', operation: 'IS_SUBSTRING', string_value: 'XGBoost' }],
      });
      const response = await request
        .get(`/apis/v2beta1/pipelines${buildQuery({ filter })}`)
        .expect(200);

      expect(response.body.pipelines).toHaveLength(1);
      expect(response.body.pipelines[0]).toMatchObject({
        pipeline_id: '8fbe3bd6-a01f-11e8-98d0-529269fb1462',
        display_name: 'XGBoost',
      });
    });

    it('fetches v2 pipeline versions for a pipeline', async () => {
      const response = await request
        .get('/apis/v2beta1/pipelines/8fbe3bd6-a01f-11e8-98d0-529269fb1460/versions?page_size=1')
        .expect(200);

      expect(response.body.next_page_token).toBe('1');
      expect(response.body.pipeline_versions).toHaveLength(1);
      expect(response.body.pipeline_versions[0]).toMatchObject({
        pipeline_id: '8fbe3bd6-a01f-11e8-98d0-529269fb1460',
        pipeline_version_id: '8fbe3bd6-a01f-11e8-98d0-529269fb1460',
      });
    });

    it('fetches a v2 pipeline version by id', async () => {
      const response = await request
        .get(
          '/apis/v2beta1/pipelines/8fbe3bd6-a01f-11e8-98d0-529269fb1460/versions/9fbe3bd6-a01f-11e8-98d0-529269fb1460',
        )
        .expect(200);

      expect(response.body).toMatchObject({
        pipeline_id: '8fbe3bd6-a01f-11e8-98d0-529269fb1460',
        pipeline_version_id: '9fbe3bd6-a01f-11e8-98d0-529269fb1460',
        display_name: 'revision',
      });
    });

    it('lists v2 experiments and applies storage-state filters', async () => {
      const filter = JSON.stringify({
        predicates: [{ key: 'storage_state', operation: 'NOT_EQUALS', string_value: 'ARCHIVED' }],
      });
      const response = await request
        .get(`/apis/v2beta1/experiments${buildQuery({ filter, page_size: 2 })}`)
        .expect(200);

      expect(response.body.next_page_token).toBe('2');
      expect(response.body.total_size).toBeGreaterThan(2);
      expect(response.body.experiments[0]).toMatchObject({
        experiment_id: expect.any(String),
        display_name: expect.any(String),
        storage_state: 'AVAILABLE',
      });
    });

    it('lists v2 runs filtered by experiment id', async () => {
      const response = await request
        .get(
          `/apis/v2beta1/runs${buildQuery({
            experiment_id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
            page_size: 3,
          })}`,
        )
        .expect(200);

      expect(response.body.runs).toHaveLength(3);
      expect(response.body.runs[0]).toMatchObject({
        run_id: expect.any(String),
        display_name: expect.any(String),
        experiment_id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
        storage_state: 'AVAILABLE',
      });
    });

    it('fetches a v2 run by id', async () => {
      const response = await request
        .get('/apis/v2beta1/runs/e0115ac1-0479-4194-a22d-01e65e09a32b')
        .expect(200);

      expect(response.body).toMatchObject({
        run_id: 'e0115ac1-0479-4194-a22d-01e65e09a32b',
        display_name: 'v2-xgboost-ilbo',
        state: 'SUCCEEDED',
      });
      expect(response.body.pipeline_version_reference).toBeUndefined();
    });

    it('serves representative native tasks for the v2 run details graph', async () => {
      const response = await request
        .get('/apis/v2beta1/runs/e0115ac1-0479-4194-a22d-01e65e09a32b/tasks')
        .expect(200);

      expect(response.body.tasks).toHaveLength(3);
      expect(response.body.tasks).toEqual(
        expect.arrayContaining([
          expect.objectContaining({ task_id: 'mock-task-root', type: 'ROOT' }),
          expect.objectContaining({
            outputs: expect.objectContaining({ artifacts: expect.any(Array) }),
            task_id: 'mock-task-producer',
            type: 'RUNTIME',
          }),
          expect.objectContaining({
            inputs: expect.objectContaining({ artifacts: expect.any(Array) }),
            pods: expect.arrayContaining([
              expect.objectContaining({ type: 'DRIVER' }),
              expect.objectContaining({ type: 'EXECUTOR' }),
            ]),
            state: 'SUCCEEDED',
            task_id: 'mock-task-consumer',
          }),
        ]),
      );
    });

    it('lists and fetches a native artifact by id', async () => {
      const listResponse = await request.get('/apis/v2beta1/artifacts').expect(200);

      expect(listResponse.body).toMatchObject({
        total_size: 1,
        artifacts: [
          {
            artifact_id: 'mock-artifact-1',
            name: 'mock-dataset',
            namespace: 'kubeflow-user-example-com',
            type: 'Dataset',
          },
        ],
      });

      const artifactResponse = await request
        .get('/apis/v2beta1/artifacts/mock-artifact-1')
        .expect(200);

      expect(artifactResponse.body).toMatchObject({
        artifact_id: 'mock-artifact-1',
        created_at: '2026-01-01T00:00:00.000Z',
        description: 'Representative native artifact for local frontend development.',
        name: 'mock-dataset',
        namespace: 'kubeflow-user-example-com',
        type: 'Dataset',
      });
    });

    it('returns 404 for an unknown native artifact', async () => {
      await request
        .get('/apis/v2beta1/artifacts/does-not-exist')
        .expect(404, 'No artifact was found with ID: does-not-exist');
    });

    it('serves representative native artifact-task relationships', async () => {
      const response = await request.get('/apis/v2beta1/artifact_tasks').expect(200);

      expect(response.body.artifact_tasks).toEqual([
        expect.objectContaining({
          artifact_id: 'mock-artifact-1',
          task_id: 'mock-task-producer',
          type: 'OUTPUT',
        }),
        expect.objectContaining({
          artifact_id: 'mock-artifact-1',
          task_id: 'mock-task-consumer',
          type: 'TASK_OUTPUT_INPUT',
        }),
      ]);
    });

    it('lists v2 recurring runs filtered by experiment id', async () => {
      const response = await request
        .get(
          `/apis/v2beta1/recurringruns${buildQuery({
            experiment_id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
            page_size: 2,
          })}`,
        )
        .expect(200);

      expect(response.body.recurringRuns).toHaveLength(2);
      expect(response.body.recurringRuns[0]).toMatchObject({
        recurring_run_id: expect.any(String),
        display_name: expect.any(String),
        experiment_id: '275ea11d-ac63-4ce3-bc33-ec81981ed56a',
      });
    });
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
      await request
        .get('/apps/tensorboard')
        .expect(200, { proxyPath: '', tfVersion: '', image: '' });
      await request.post('/apps/tensorboard').expect(200, 'apps/tensorboard/proxy/mock-token/');
      await request.get('/apps/tensorboard').expect(200, {
        proxyPath: 'apps/tensorboard/proxy/mock-token/',
        tfVersion: '',
        image: '',
      });
    });

    it('does not serve removed API endpoints', async () => {
      await request.get('/apis/v1beta1/_proxy/http%3A%2F%2Fviewer.test%2Fdata').expect(404);
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
