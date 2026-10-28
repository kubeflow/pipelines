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

    it('does not expose the removed visualization configuration route', async () => {
      await request.get('/visualizations/allowed').expect(404);
    });

    it('serves v2 healthz status', async () => {
      const response = await request.get('/apis/v2/healthz').expect(200);

      expect(response.body).toMatchObject({
        apiServerReady: true,
        apiServerMultiUser: false,
        pipelineStore: 'database',
      });
      expect(response.body.frontendCommitHash).toBeDefined();
    });

    it('returns 404 for unknown v2 endpoints', async () => {
      await request.get('/apis/v2/does-not-exist').expect(404, 'Bad request endpoint.');
    });
  });

  describe('v2 fixture routes', () => {
    it('resolves sample graph artifact links and their producer/consumer lineage', async () => {
      const runId = 'sample-run-daily-success';
      const artifactId = `${runId}-prepare-data-dataset`;
      const artifact = await request.get(`/apis/v2/artifacts/${artifactId}`).expect(200);
      expect(artifact.body).toMatchObject({
        artifact_id: artifactId,
        type: 'Dataset',
        name: 'Prepared taxi features',
      });
      const lineage = await request
        .get('/apis/v2/artifact_tasks')
        .query({ artifact_ids: artifactId })
        .expect(200);
      expect(lineage.body.artifact_tasks).toEqual(
        expect.arrayContaining([
          expect.objectContaining({ task_id: `${runId}-prepare-data`, type: 'OUTPUT' }),
          expect.objectContaining({ task_id: `${runId}-train-model`, type: 'TASK_OUTPUT_INPUT' }),
        ]),
      );
      await request
        .get('/apis/v2/artifacts/sample-run-training-failed-train-model-model')
        .expect(404);
    });

    it('lists pipelines with v2 field names, pagination, and total size', async () => {
      const response = await request.get('/apis/v2/pipelines?page_token=0&page_size=3').expect(200);

      expect(response.body.next_page_token).toBe('3');
      expect(response.body.total_size).toBeGreaterThan(3);
      expect(response.body.pipelines).toHaveLength(3);
      expect(response.body.pipelines[0]).toMatchObject({
        pipeline_id: expect.any(String),
        display_name: expect.any(String),
      });
    });

    it('combines uploaded timestamps and name filters before pagination', async () => {
      const all = await request.get('/apis/v2/pipelines?page_size=100').expect(200);
      const pipeline = all.body.pipelines.find(
        (item: { display_name: string }) => item.display_name === 'XGBoost',
      );
      const predicates = [
        { key: 'name', operation: 'IS_SUBSTRING', string_value: 'XGBoost' },
        {
          key: 'created_at',
          operation: 'GREATER_THAN_EQUALS',
          timestamp_value: pipeline.created_at,
        },
      ];
      const inclusive = await request
        .get(
          `/apis/v2/pipelines${buildQuery({ filter: JSON.stringify({ predicates }), page_size: '1' })}`,
        )
        .expect(200);
      expect(inclusive.body.pipelines).toHaveLength(1);
      predicates[1].timestamp_value = new Date(
        new Date(pipeline.created_at).getTime() + 1000,
      ).toISOString();
      const excluded = await request
        .get(`/apis/v2/pipelines${buildQuery({ filter: JSON.stringify({ predicates }) })}`)
        .expect(200);
      expect(excluded.body.pipelines || []).toHaveLength(0);
    });

    it('filters v2 pipelines by name', async () => {
      const filter = JSON.stringify({
        predicates: [{ key: 'name', operation: 'IS_SUBSTRING', string_value: 'XGBoost' }],
      });
      const response = await request.get(`/apis/v2/pipelines${buildQuery({ filter })}`).expect(200);

      expect(response.body.pipelines).toHaveLength(1);
      expect(response.body.pipelines[0]).toMatchObject({
        pipeline_id: '8fbe3bd6-a01f-11e8-98d0-529269fb1462',
        display_name: 'XGBoost',
      });
    });

    it('fetches v2 pipeline versions for a pipeline', async () => {
      const response = await request
        .get('/apis/v2/pipelines/8fbe3bd6-a01f-11e8-98d0-529269fb1460/versions?page_size=1')
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
          '/apis/v2/pipelines/8fbe3bd6-a01f-11e8-98d0-529269fb1460/versions/9fbe3bd6-a01f-11e8-98d0-529269fb1460',
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
        .get(`/apis/v2/experiments${buildQuery({ filter, page_size: 2 })}`)
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
          `/apis/v2/runs${buildQuery({
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

    it('filters review runs by status and relative date before pagination', async () => {
      const query = {
        experiment_id: '275ea11d-ac63-4ce3-bc33-ec81981ed56b',
        filter: JSON.stringify({
          predicates: [
            { key: 'state', operation: 'EQUALS', string_value: 'FAILED' },
            { key: 'storage_state', operation: 'EQUALS', string_value: 'AVAILABLE' },
            {
              key: 'created_at',
              operation: 'GREATER_THAN_EQUALS',
              timestamp_value: new Date(Date.now() - 30 * 86_400_000).toISOString(),
            },
          ],
        }),
        sort_by: 'created_at desc',
        page_size: 1,
      };
      const first = await request.get(`/apis/v2/runs${buildQuery(query)}`).expect(200);
      expect(first.body.total_size).toBe(2);
      expect(first.body.runs[0].run_id).toBe('sample-run-training-failed');
      const next = await request
        .get(`/apis/v2/runs${buildQuery({ ...query, page_token: first.body.next_page_token })}`)
        .expect(200);
      expect(next.body.runs[0].run_id).toBe('sample-run-import-failed');
      expect(next.body.next_page_token).toBe('');
    });

    it('keeps archived samples separate and unfinished runs without finish times', async () => {
      const filter = JSON.stringify({
        predicates: [{ key: 'storage_state', operation: 'EQUALS', string_value: 'ARCHIVED' }],
      });
      const archived = await request.get(`/apis/v2/runs${buildQuery({ filter })}`).expect(200);
      expect(archived.body.total_size).toBe(2);
      expect(
        archived.body.runs.every(
          (run: { storage_state: string }) => run.storage_state === 'ARCHIVED',
        ),
      ).toBe(true);
      for (const id of ['training-running', 'validation-pending', 'refresh-canceling']) {
        const response = await request.get(`/apis/v2/runs/sample-run-${id}`).expect(200);
        expect(response.body.finished_at).toBeUndefined();
      }
    });

    it.each(['nested-conditions', 'parallel-folds', 'regional-training'])(
      'serves complex %s review runs and their saved versions',
      async (id) => {
        const run = (await request.get(`/apis/v2/runs/sample-run-${id}`).expect(200)).body;
        expect(run.state_history[0]).toMatchObject({
          state: 'RUNNING',
          update_time: expect.any(String),
        });
        const version = (
          await request.get(`/apis/v2/pipelines/${id}/versions/${id}-v1`).expect(200)
        ).body;
        expect(run.pipeline_spec).toEqual(version.pipeline_spec);
        const tasks = (await request.get(`/apis/v2/runs/sample-run-${id}/tasks`).expect(200)).body
          .tasks;
        expect(tasks.length).toBeGreaterThan(5);
        const leaf = tasks.find((task: { type: string }) => task.type === 'RUNTIME');
        const fetched = (
          await request.get(`/apis/v2/runs/sample-run-${id}/tasks/${leaf.task_id}`).expect(200)
        ).body;
        expect(fetched.task_id).toBe(leaf.task_id);
      },
    );

    it('fetches a v2 run by id', async () => {
      const response = await request
        .get('/apis/v2/runs/e0115ac1-0479-4194-a22d-01e65e09a32b')
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
        .get('/apis/v2/runs/e0115ac1-0479-4194-a22d-01e65e09a32b/tasks')
        .expect(200);

      expect(response.body.tasks).toHaveLength(4);
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
          expect.objectContaining({
            outputs: {
              artifacts: [
                expect.objectContaining({
                  artifact_key: 'model',
                  artifacts: [
                    expect.objectContaining({ artifact_id: 'mock-model-1', type: 'Model' }),
                  ],
                }),
              ],
            },
            task_id: 'mock-task-trainer',
            type: 'RUNTIME',
          }),
        ]),
      );
    });

    it('lists and fetches a native artifact by id', async () => {
      const listResponse = await request.get('/apis/v2/artifacts').expect(200);

      expect(listResponse.body).toMatchObject({
        total_size: 7,
        artifacts: expect.arrayContaining([
          expect.objectContaining({
            artifact_id: 'mock-artifact-1',
            name: 'mock-dataset',
            namespace: 'kubeflow-user-example-com',
            type: 'Dataset',
          }),
        ]),
      });

      const artifactResponse = await request.get('/apis/v2/artifacts/mock-artifact-1').expect(200);

      expect(artifactResponse.body).toMatchObject({
        artifact_id: 'mock-artifact-1',
        created_at: '2026-01-01T00:00:00.000Z',
        description: 'Representative native artifact for local frontend development.',
        name: 'mock-dataset',
        namespace: 'kubeflow-user-example-com',
        type: 'Dataset',
      });

      const modelResponse = await request.get('/apis/v2/artifacts/mock-model-1').expect(200);

      expect(modelResponse.body).toMatchObject({
        artifact_id: 'mock-model-1',
        metadata: { _kfp_preview: { version: 2, storage: 'sibling' } },
        name: 'mock-model',
        namespace: 'kubeflow-user-example-com',
        type: 'Model',
      });
    });

    it('combines artifact type, date and name filters before pagination', async () => {
      const predicates = [
        { key: 'type', operation: 'IN', int_values: { values: [2, 3] } },
        {
          key: 'created_at',
          operation: 'GREATER_THAN_EQUALS',
          timestamp_value: '2026-01-01T00:00:00Z',
        },
        { key: 'name', operation: 'IS_SUBSTRING', string_value: 'dataset' },
      ];
      const query = { filter: JSON.stringify({ predicates }), page_size: 1, sort_by: 'name desc' };
      const first = await request.get(`/apis/v2/artifacts${buildQuery(query)}`).expect(200);
      expect(first.body.total_size).toBe(3);
      expect(first.body.artifacts[0].artifact_id).toBe('mock-artifact-3');
      const next = await request
        .get(`/apis/v2/artifacts${buildQuery({ ...query, page_token: 1 })}`)
        .expect(200);
      expect(next.body.total_size).toBe(3);
      expect(next.body.artifacts[0].artifact_id).toBe('mock-artifact-2');
      expect(next.body.next_page_token).toBeTruthy();
    });

    it.each([
      { key: 'type', operation: 'IN', int_values: { values: [4] } },
      {
        key: 'created_at',
        operation: 'GREATER_THAN_EQUALS',
        timestamp_value: '2026-02-01T00:00:00Z',
      },
    ])('excludes artifacts that fail a predicate: %j', async (predicate) => {
      const response = await request
        .get(
          `/apis/v2/artifacts${buildQuery({ filter: JSON.stringify({ predicates: [predicate] }) })}`,
        )
        .expect(200);
      expect(response.body).toMatchObject({ total_size: 0, artifacts: [] });
    });

    it('filters experiments by their latest run date and excludes experiments without runs', async () => {
      const filter = JSON.stringify({
        predicates: [
          {
            key: 'last_run_created_at',
            operation: 'GREATER_THAN_EQUALS',
            timestamp_value: '2026-01-01T00:00:00Z',
          },
        ],
      });
      const response = await request
        .get(`/apis/v2/experiments${buildQuery({ filter })}`)
        .expect(200);
      expect(
        response.body.experiments.map(
          (experiment: { display_name: string }) => experiment.display_name,
        ),
      ).toEqual(['Default']);
      expect(Date.parse(response.body.experiments[0].last_run_created_at)).toBeGreaterThan(
        Date.now() - 2 * 86_400_000,
      );
    });

    it('does not assign the original dataset lineage to additional sample artifacts', async () => {
      const response = await request
        .get('/apis/v2/artifact_tasks?artifact_ids=mock-artifact-4')
        .expect(200);
      expect(response.body.artifact_tasks).toEqual([]);
      const original = await request
        .get('/apis/v2/artifact_tasks?artifact_ids=mock-artifact-1')
        .expect(200);
      expect(original.body.artifact_tasks).toHaveLength(2);
    });

    it('returns 404 for an unknown native artifact', async () => {
      await request
        .get('/apis/v2/artifacts/does-not-exist')
        .expect(404, 'No artifact was found with ID: does-not-exist');
    });

    it('serves representative native artifact-task relationships', async () => {
      const response = await request
        .get('/apis/v2/artifact_tasks')
        .query({ artifact_ids: 'mock-artifact-1' })
        .expect(200);

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
          `/apis/v2/recurringruns${buildQuery({
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
