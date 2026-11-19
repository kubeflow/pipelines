// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { afterEach, expect, it, vi } from 'vitest';
import { setArgoToken } from './Argo';
import {
  createResource,
  deleteExperiment,
  setScheduleSuspended,
  submitRun,
  workflowAction,
  type NativeResource,
} from './ArgoResources';

const version: NativeResource = {
  apiVersion: 'argoproj.io/v1alpha1',
  kind: 'WorkflowTemplate',
  metadata: {
    name: 'pipeline-abc123',
    labels: {
      'pipelines.kubeflow.org/pipeline': 'pipeline',
      'pipelines.kubeflow.org/version': 'abc123',
    },
  },
  spec: {},
};
const mock = () =>
  vi.stubGlobal(
    'fetch',
    vi.fn(async () => new Response('{}')),
  );
afterEach(() => {
  vi.unstubAllGlobals();
  setArgoToken('');
});

it('creates versions without an update/overwrite fallback', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn(async () => new Response('{}', { status: 409 })),
  );
  await expect(createResource('team', 'workflow-templates', version)).rejects.toThrow('409');
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(vi.mocked(fetch).mock.calls[0][1]?.method).toBe('POST');
});

it('rejects manifests targeting another namespace', async () => {
  mock();
  await expect(
    createResource('team', 'workflow-templates', {
      ...version,
      metadata: { ...version.metadata, namespace: 'someone-else' },
    }),
  ).rejects.toThrow('namespace team');
  expect(fetch).not.toHaveBeenCalled();
});

it('submits native workflows with version and experiment labels', async () => {
  mock();
  setArgoToken('test-namespace-token');
  await submitRun('team', version, { factor: '1.2' }, 'experiment-a');
  const [url, options] = vi.mocked(fetch).mock.calls[0];
  expect(url).toBe('/argo/api/v1/workflows/team');
  expect(options?.headers).toMatchObject({ Authorization: 'Bearer test-namespace-token' });
  const body = JSON.parse(options?.body as string);
  expect(body.workflow.spec.workflowTemplateRef.name).toBe(version.metadata.name);
  expect(body.workflow.metadata.labels['pipelines.kubeflow.org/experiment']).toBe('experiment-a');
});

it('deletes only the experiment ConfigMap, using its UID precondition', async () => {
  mock();
  await expect(
    deleteExperiment('team', {
      apiVersion: 'v1',
      kind: 'ConfigMap',
      metadata: { name: 'experiment-a' },
    }),
  ).rejects.toThrow('UID is missing');
  expect(fetch).not.toHaveBeenCalled();
  await deleteExperiment('team', {
    apiVersion: 'v1',
    kind: 'ConfigMap',
    metadata: { name: 'experiment-a', uid: 'uid-a' },
  });
  expect(fetch).toHaveBeenCalledTimes(1);
  const [url, options] = vi.mocked(fetch).mock.calls[0];
  expect(url).toBe('/kubernetes/api/v1/namespaces/team/configmaps/experiment-a');
  expect(JSON.parse(options?.body as string).preconditions).toEqual({ uid: 'uid-a' });
});

it('preserves resourceVersion when changing a CronWorkflow', async () => {
  mock();
  await setScheduleSuspended(
    'team',
    {
      ...version,
      kind: 'CronWorkflow',
      metadata: { name: 'schedule', resourceVersion: '123' },
      spec: { schedule: '0 9 * * *', suspend: true },
    },
    false,
  );
  const body = JSON.parse(vi.mocked(fetch).mock.calls[0][1]?.body as string);
  expect(body.cronWorkflow.metadata.resourceVersion).toBe('123');
  expect(body.cronWorkflow.spec.suspend).toBe(false);
});

it('refuses a same-name replacement before a workflow mutation', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async () =>
        new Response(
          JSON.stringify({
            metadata: { name: 'run', namespace: 'team', uid: 'replacement' },
            spec: {},
          }),
        ),
    ),
  );
  await expect(
    workflowAction(
      { metadata: { name: 'run', namespace: 'team', uid: 'original' }, spec: {} },
      'resume',
    ),
  ).rejects.toThrow('reused');
  expect(vi.mocked(fetch).mock.calls.every(([, options]) => !options?.method)).toBe(true);
});

it('reports denied workflow operations without retrying through another backend', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn(async () => new Response('{}', { status: 403 })),
  );
  await expect(
    workflowAction(
      { metadata: { name: 'run', namespace: 'team', uid: 'uid' }, spec: {} },
      'resume',
    ),
  ).rejects.toThrow('403');
  expect(fetch).toHaveBeenCalledTimes(1);
});
