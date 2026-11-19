// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { argoRequest, getArgoWorkflow, type ArgoWorkflow } from './Argo';

export interface NativeResource {
  apiVersion: string;
  kind: string;
  metadata: {
    name: string;
    uid?: string;
    namespace?: string;
    resourceVersion?: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
  };
  spec?: Record<string, any>;
  data?: Record<string, string>;
}
const encode = encodeURIComponent;
export type ResourceKind = 'workflow-templates' | 'cron-workflows' | 'experiments';

function resourcePath(namespace: string, kind: ResourceKind, name?: string) {
  const root =
    kind === 'experiments'
      ? `/kubernetes/api/v1/namespaces/${encode(namespace)}/configmaps`
      : `/argo/api/v1/${kind}/${encode(namespace)}`;
  return root + (name ? '/' + encode(name) : '');
}

export async function listResources(
  namespace: string,
  kind: ResourceKind,
  signal?: AbortSignal,
): Promise<NativeResource[]> {
  const query = new URLSearchParams(
    kind === 'experiments'
      ? { labelSelector: 'pipelines.kubeflow.org/resource=experiment', limit: '100' }
      : {
          'listOptions.labelSelector': 'pipelines.kubeflow.org/pipeline',
          'listOptions.limit': '100',
        },
  );
  const data = await (
    await argoRequest(`${resourcePath(namespace, kind)}?${query}`, signal)
  ).json();
  return data.items || [];
}

export async function createResource(
  namespace: string,
  kind: ResourceKind,
  resource: NativeResource,
) {
  const expected = {
    'workflow-templates': 'WorkflowTemplate',
    'cron-workflows': 'CronWorkflow',
    experiments: 'ConfigMap',
  }[kind];
  if (
    resource?.kind !== expected ||
    !resource.metadata?.name ||
    (resource.metadata.namespace && resource.metadata.namespace !== namespace)
  ) {
    throw new Error(`Expected a named ${expected} in namespace ${namespace}.`);
  }
  if (
    kind === 'workflow-templates' &&
    !resource.metadata.labels?.['pipelines.kubeflow.org/pipeline']
  )
    throw new Error(
      'WorkflowTemplate needs the pipelines.kubeflow.org/pipeline label. Compile with the native KFP compiler.',
    );
  const key = kind === 'workflow-templates' ? 'template' : 'cronWorkflow';
  const body = kind === 'experiments' ? resource : { namespace, [key]: resource };
  return (
    await argoRequest(resourcePath(namespace, kind), undefined, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  ).json();
}

export async function deleteExperiment(namespace: string, experiment: NativeResource) {
  if (!experiment.metadata.uid)
    throw new Error('Refresh the experiment before deleting it; its UID is missing.');
  // No owner references or run deletions: membership is only a workflow label.
  return argoRequest(resourcePath(namespace, 'experiments', experiment.metadata.name), undefined, {
    method: 'DELETE',
    body: JSON.stringify({
      apiVersion: 'v1',
      kind: 'DeleteOptions',
      preconditions: { uid: experiment.metadata.uid },
    }),
  });
}

export async function submitRun(
  namespace: string,
  version: NativeResource,
  parameters: Record<string, string>,
  experiment: string,
) {
  const labels = {
    ...version.metadata.labels,
    ...(experiment ? { 'pipelines.kubeflow.org/experiment': experiment } : {}),
  };
  const workflow = {
    apiVersion: 'argoproj.io/v1alpha1',
    kind: 'Workflow',
    metadata: { generateName: version.metadata.name.slice(0, 48) + '-', namespace, labels },
    spec: {
      workflowTemplateRef: { name: version.metadata.name },
      arguments: {
        parameters: Object.entries(parameters).map(([name, value]) => ({ name, value })),
      },
    },
  };
  return (
    await argoRequest(`/argo/api/v1/workflows/${encode(namespace)}`, undefined, {
      method: 'POST',
      body: JSON.stringify({ namespace, workflow }),
    })
  ).json();
}

export async function setScheduleSuspended(
  namespace: string,
  schedule: NativeResource,
  suspend: boolean,
) {
  // PUT retains the resourceVersion: stale edits conflict instead of overwriting.
  return (
    await argoRequest(
      resourcePath(namespace, 'cron-workflows', schedule.metadata.name),
      undefined,
      {
        method: 'PUT',
        body: JSON.stringify({
          namespace,
          name: schedule.metadata.name,
          cronWorkflow: { ...schedule, spec: { ...schedule.spec, suspend } },
        }),
      },
    )
  ).json();
}

export async function workflowAction(run: ArgoWorkflow, action: 'resume' | 'stop' | 'terminate') {
  const current = await getArgoWorkflow(run);
  if (current.archived) throw new Error('Archived workflows cannot be changed. Submit a new run.');
  return (
    await argoRequest(
      `/argo/api/v1/workflows/${encode(run.metadata.namespace)}/${encode(run.metadata.name)}/${action}`,
      undefined,
      {
        method: 'PUT',
        body: JSON.stringify({ namespace: run.metadata.namespace, name: run.metadata.name }),
      },
    )
  ).json();
}
