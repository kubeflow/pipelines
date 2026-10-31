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

import { describe, expect, it } from 'vitest';
import requests from 'supertest';
import { createMockApiApp } from '../../mock-backend/mock-api-app.ts';
import { setupMockBackendTest } from './test-helper.js';

setupMockBackendTest();

describe('standalone mock API server', () => {
  it('serves task-specific output with failures, progress and cancellation', async () => {
    const request = requests(createMockApiApp());
    for (const [id, expected] of [
      ['training-failed', 'ValueError: Training data contains invalid feature values.'],
      ['training-running', 'Progress: 40 of 100 estimators fitted'],
      ['sweep-canceled', 'Execution stopped after the run was canceled.'],
    ]) {
      const { body } = await request.get(`/apis/v2/runs/sample-run-${id}/tasks`).expect(200);
      const task = body.tasks.find((task: { name: string }) => task.name === 'train-model');
      const response = await request
        .get('/k8s/pod/logs')
        .query({ podname: task.pods[0].name })
        .expect(200);
      expect(response.text).toContain(expected);
      expect(response.text).not.toContain('hello world');
      expect(response.text).not.toContain('Component completed successfully');
      const prepare = body.tasks.find((task: { name: string }) => task.name === 'prepare-data');
      const prepared = await request
        .get('/k8s/pod/logs')
        .query({ podname: prepare.pods[0].name })
        .expect(200);
      expect(prepared.text).toContain('Prepared 12480 rows');
      expect(prepared.text).toContain('Component completed successfully');
    }
    await request
      .get('/k8s/pod/logs')
      .query({ podname: 'sample-run-training-failed-evaluate-model-executor' })
      .expect(404);
  });

  it('exposes distinct failed automatic retry archives and pod logs', async () => {
    const request = requests(createMockApiApp());
    const { body } = await request.get('/apis/v2/runs/sample-run-import-failed/tasks').expect(200);
    const task = body.tasks.find((task: { name: string }) => task.name === 'train-model');
    expect(task.state).toBe('FAILED');
    expect(task.pods).toHaveLength(2);
    const archives = task.outputs.artifacts[0].artifacts;
    expect(archives).toHaveLength(2);
    for (const [index, expected] of [
      'TimeoutError: Timed out reading the training dataset',
      'ValueError: Training data contains invalid feature values.',
    ].entries()) {
      const uri = new URL(archives[index].uri);
      const archived = await request
        .get('/artifacts/get')
        .query({
          source: 's3',
          bucket: uri.hostname,
          key: uri.pathname.slice(1),
          namespace: 'kubeflow',
        })
        .expect(200);
      const pod = await request
        .get('/k8s/pod/logs')
        .query({ podname: task.pods[index].name })
        .expect(200);
      expect(archived.text).toContain(expected);
      expect(archived.text).toEqual(pod.text);
      expect(archived.text).not.toContain('Component completed successfully');
    }
    await request.get('/artifacts/get').query({ key: 'review-executor-logs/missing' }).expect(404);
  });

  it('serves varied recurring schedules with detail lookups', async () => {
    const request = requests(createMockApiApp());
    const { body } = await request.get('/apis/v2/recurringruns').expect(200);
    const samples = body.recurringRuns.filter((run: { recurring_run_id: string }) =>
      run.recurring_run_id.startsWith('sample-recurring-'),
    );
    expect(samples).toHaveLength(5);
    expect(
      samples.filter((run: { trigger: { cron_schedule?: unknown } }) => run.trigger.cron_schedule),
    ).toHaveLength(2);
    expect(samples.some((run: { status: string }) => run.status === 'DISABLED')).toBe(true);
    for (const sample of samples) {
      const detail = await request
        .get(`/apis/v2/recurringruns/${sample.recurring_run_id}`)
        .expect(200);
      expect(detail.body.trigger).toEqual(sample.trigger);
    }
    const upcoming = samples.find((run: { display_name: string }) =>
      run.display_name.startsWith('Upcoming'),
    );
    const expired = samples.find((run: { display_name: string }) =>
      run.display_name.startsWith('Completed'),
    );
    expect(new Date(upcoming.trigger.periodic_schedule.start_time).getTime()).toBeGreaterThan(
      Date.now(),
    );
    expect(new Date(expired.trigger.periodic_schedule.end_time).getTime()).toBeLessThan(Date.now());
  });

  it('serves driver logs linked from sample task metadata', async () => {
    const request = requests(createMockApiApp());
    const { body } = await request
      .get('/apis/v2/runs/sample-run-training-failed/tasks')
      .expect(200);
    const task = body.tasks.find((task: { name: string }) => task.name === 'train-model');
    const uri = new URL(task.status_metadata.custom_properties.driver_logs_uri);
    const response = await request
      .get('/artifacts/get')
      .query({ source: 's3', bucket: uri.hostname, key: uri.pathname.slice(1) })
      .expect(200);
    expect(response.headers['content-type']).toContain('text/plain');
    expect(response.text).toContain('Cache disabled globally at the server level.');
    expect(response.text).toContain('Creating task train-model in pod ');
    await request.get('/artifacts/get').query({ key: 'review-driver-logs/missing' }).expect(404);
  });

  it('resolves saved sample versions while retaining inline submissions', async () => {
    const request = requests(createMockApiApp());
    for (const [id, versionName] of [
      ['training-running', 'v1.1'],
      ['monthly-success', 'v1.0'],
    ]) {
      const { body: run } = await request.get(`/apis/v2/runs/sample-run-${id}`).expect(200);
      const ref = run.pipeline_version_reference;
      const { body: version } = await request
        .get(`/apis/v2/pipelines/${ref.pipeline_id}/versions/${ref.pipeline_version_id}`)
        .expect(200);
      expect(version.display_name).toBe(versionName);
      expect(version.pipeline_spec).toEqual(run.pipeline_spec);
      await request.get(`/apis/v2/pipelines/${ref.pipeline_id}`).expect(200);
    }
    for (const id of ['sweep-canceled', 'import-failed']) {
      const { body: run } = await request.get(`/apis/v2/runs/sample-run-${id}`).expect(200);
      expect(run.pipeline_version_reference).toBeUndefined();
      expect(run.pipeline_spec.root.dag.tasks).toHaveProperty('train-model');
    }
  });

  it('constructs the app with the mock backend dependency graph', async () => {
    const request = requests(createMockApiApp());

    await request.get('/apis/v2/healthz').expect(200);
    await request.get('/apis/v1beta1/healthz').expect(404);
    await request.get('/APIS/V2/not-found').expect(404, 'Bad request endpoint.');
    await request.get('/APPS/TENSORBOARD/PROXY/mock-token/index.js').expect(200);
  });
});
