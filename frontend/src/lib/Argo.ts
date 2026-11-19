// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { PipelineTaskTaskState } from 'src/apisv2/run';
import type { PipelineFlowElement } from './v2/FlowTypes';
import { buildGraphLayout, NodeTypeNames } from './v2/StaticFlow';

export interface ArgoNode {
  id: string;
  displayName: string;
  type: string;
  phase?: string;
  message?: string;
  children?: string[];
  memoizationStatus?: { hit: boolean };
  outputs?: {
    artifacts?: Array<{ name: string }>;
    parameters?: Array<{ name: string; value?: string }>;
  };
}

export interface ArgoWorkflow {
  archived?: boolean;
  metadata: {
    name: string;
    namespace: string;
    uid: string;
    creationTimestamp?: string;
    labels?: Record<string, string>;
  };
  spec: { arguments?: { parameters?: Array<{ name: string; value?: string }> } };
  status?: {
    phase?: string;
    message?: string;
    nodes?: Record<string, ArgoNode>;
    compressedNodes?: string;
    offloadNodeStatusVersion?: string;
  };
}

interface Producer {
  run: string;
  task: string;
  namespace: string;
}

export interface ArgoMetadata {
  schema: 'kfp-argo/v1';
  producer: Producer;
  inputs: Array<{ port: string; producer: Producer; uri: string }>;
  artifacts: Array<{
    port: string;
    type: string;
    uri: string;
    producer: Producer;
    metadata: Record<string, unknown>;
  }>;
  metrics: Record<string, number>;
}

let bearerToken = '';

/** Keep namespace-scoped credentials in memory, never in URLs or web storage. */
export function setArgoToken(token: string) {
  bearerToken = token.trim().replace(/^Bearer\s+/i, '');
}

export const hasArgoMetadata = (node: ArgoNode) =>
  node.outputs?.artifacts?.some((artifact) => artifact.name === 'kfp-metadata') ||
  node.outputs?.parameters?.some((parameter) => parameter.name === 'kfp-metadata');

const isPersisted = (run: ArgoWorkflow) =>
  run.metadata.labels?.['workflows.argoproj.io/workflow-archiving-status'] === 'Persisted';
const segment = encodeURIComponent;
const workflowPath = (namespace: string, name: string) =>
  `/argo/api/v1/workflows/${segment(namespace)}/${segment(name)}`;

export async function argoRequest(
  path: string,
  signal?: AbortSignal,
  init: RequestInit = {},
): Promise<Response> {
  const response = await fetch(path, {
    ...init,
    signal,
    credentials: 'same-origin',
    headers: {
      ...(bearerToken ? { Authorization: `Bearer ${bearerToken}` } : {}),
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...init.headers,
    },
  });
  if (!response.ok)
    throw new Error(
      `Argo request failed (${response.status}). Check your namespace and access permissions.`,
    );
  return response;
}

export async function listArgoWorkflows(
  namespace: string,
  signal?: AbortSignal,
): Promise<ArgoWorkflow[]> {
  const query = new URLSearchParams({
    'listOptions.labelSelector': 'pipelines.kubeflow.org/pipeline',
    'listOptions.limit': '100',
  });
  const result = await (
    await argoRequest(`/argo/api/v1/workflows/${segment(namespace)}?${query}`, signal)
  ).json();
  const archived = await (
    await argoRequest(
      `/argo/api/v1/archived-workflows?namespace=${segment(namespace)}&${query}`,
      signal,
    )
  ).json();
  const runs = new Map<string, ArgoWorkflow>();
  for (const run of archived.items || []) runs.set(run.metadata.uid, { ...run, archived: true });
  // Argo 4 includes SQL rows in its workflow list, even after live CR deletion.
  for (const run of result.items || [])
    runs.set(run.metadata.uid, isPersisted(run) ? { ...run, archived: true } : run);
  return [...runs.values()].sort((a, b) =>
    (b.metadata.creationTimestamp || '').localeCompare(a.metadata.creationTimestamp || ''),
  );
}

export async function getArgoWorkflow(
  run: ArgoWorkflow,
  signal?: AbortSignal,
): Promise<ArgoWorkflow> {
  const workflow: ArgoWorkflow = await (
    await argoRequest(
      run.archived
        ? `/argo/api/v1/archived-workflows/${segment(run.metadata.uid)}?namespace=${segment(run.metadata.namespace)}`
        : workflowPath(run.metadata.namespace, run.metadata.name),
      signal,
    )
  ).json();
  if (workflow.metadata.uid !== run.metadata.uid)
    throw new Error('This workflow name was reused. Refresh the run list and select the new run.');
  if (
    !workflow.status?.nodes &&
    (workflow.status?.compressedNodes || workflow.status?.offloadNodeStatusVersion)
  ) {
    throw new Error('Argo did not return hydrated nodes. Check Argo Server node-status storage.');
  }
  return { ...workflow, archived: run.archived || isPersisted(workflow) || undefined };
}

export function argoArtifactUrl(run: ArgoWorkflow, node: ArgoNode, artifact: string): string {
  if (run.archived)
    return `/argo/artifact-files/${segment(run.metadata.namespace)}/archived-workflows/${segment(run.metadata.uid)}/${segment(node.id)}/outputs/${segment(artifact)}`;
  return `/argo/artifacts/${segment(run.metadata.namespace)}/${segment(run.metadata.name)}/${segment(node.id)}/${segment(artifact)}`;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}

function isProducer(value: unknown): boolean {
  return (
    isRecord(value) && ['run', 'task', 'namespace'].every((key) => typeof value[key] === 'string')
  );
}

function isArtifactReference(value: unknown): boolean {
  return (
    isRecord(value) &&
    typeof value.port === 'string' &&
    typeof value.uri === 'string' &&
    isProducer(value.producer)
  );
}

export async function getArgoMetadata(
  run: ArgoWorkflow,
  node: ArgoNode,
  signal?: AbortSignal,
): Promise<ArgoMetadata> {
  const inline = node.outputs?.parameters?.find(
    (parameter) => parameter.name === 'kfp-metadata',
  )?.value;
  const text =
    inline ??
    (await (await argoRequest(argoArtifactUrl(run, node, 'kfp-metadata'), signal)).text());
  if (text.length > 1024 * 1024)
    throw new Error(
      'Task metadata exceeds the 1 MiB preview limit. Download the artifact instead.',
    );
  const data = JSON.parse(text);
  if (
    !isRecord(data) ||
    data.schema !== 'kfp-argo/v1' ||
    !Array.isArray(data.artifacts) ||
    !data.artifacts.every(
      (artifact: unknown) =>
        isRecord(artifact) &&
        isArtifactReference(artifact) &&
        typeof artifact.type === 'string' &&
        isRecord(artifact.metadata),
    ) ||
    !Array.isArray(data.inputs) ||
    !data.inputs.every(isArtifactReference) ||
    !isProducer(data.producer) ||
    !isRecord(data.metrics) ||
    Object.values(data.metrics).some(
      (value) => typeof value !== 'number' || !Number.isFinite(value),
    )
  ) {
    throw new Error('Unsupported task metadata. Use the kfp-argo/v1 runtime contract.');
  }
  return data as unknown as ArgoMetadata;
}

export async function getArgoLogs(run: ArgoWorkflow, signal?: AbortSignal): Promise<string[]> {
  if (run.archived) {
    const workflow = await getArgoWorkflow(run, signal);
    const logs = await Promise.all(
      Object.values(workflow.status?.nodes || {})
        .filter((node) =>
          node.outputs?.artifacts?.some((artifact) => artifact.name === 'main-logs'),
        )
        .map(async (node) => {
          const text = await (
            await argoRequest(argoArtifactUrl(run, node, 'main-logs'), signal)
          ).text();
          return text
            .split('\n')
            .slice(-200)
            .map((line) => `${node.displayName}: ${line}`);
        }),
    );
    return logs.flat();
  }
  const query = new URLSearchParams({
    'logOptions.container': 'main',
    'logOptions.follow': 'false',
    'logOptions.tailLines': '200',
  });
  const text = await (
    await argoRequest(
      `${workflowPath(run.metadata.namespace, run.metadata.name)}/log?${query}`,
      signal,
    )
  ).text();
  return text
    .split('\n')
    .filter(Boolean)
    .map((line) => {
      const entry = JSON.parse(line);
      if (entry.error) throw new Error(entry.error.message || 'Argo log stream failed.');
      return `${entry.result.podName}: ${entry.result.content}`;
    });
}

/** Render Argo's node status directly; the KFP enum is only a widget icon input. */
export function argoGraph(run: ArgoWorkflow): PipelineFlowElement[] {
  const nodes = Object.values(run.status?.nodes || {});
  const phases: Record<string, PipelineTaskTaskState> = {
    Pending: PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED,
    Running: PipelineTaskTaskState.RUNNING,
    Succeeded: PipelineTaskTaskState.SUCCEEDED,
    Failed: PipelineTaskTaskState.FAILED,
    Error: PipelineTaskTaskState.FAILED,
    Skipped: PipelineTaskTaskState.SKIPPED,
    Omitted: PipelineTaskTaskState.SKIPPED,
  };
  const elements: PipelineFlowElement[] = nodes.map((node) => ({
    id: node.id,
    type: NodeTypeNames.EXECUTION,
    position: { x: 0, y: 0 },
    data: {
      label: node.displayName,
      taskId: node.id,
      state: node.memoizationStatus?.hit
        ? PipelineTaskTaskState.CACHED
        : phases[node.phase || 'Pending'],
      lifecycleMessage: node.message,
    },
  }));
  const ids = new Set(nodes.map((node) => node.id));
  for (const node of nodes) {
    for (const child of node.children || []) {
      if (ids.has(child))
        elements.push({ id: `${node.id}-${child}`, source: node.id, target: child });
    }
  }
  return buildGraphLayout(elements);
}
