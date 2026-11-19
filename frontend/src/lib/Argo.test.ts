// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { vi, describe, it, expect, afterEach } from 'vitest';
import { PipelineTaskTaskState } from 'src/apisv2/run';
import {
  argoGraph,
  argoArtifactUrl,
  getArgoWorkflow,
  getArgoMetadata,
  getArgoLogs,
  listArgoWorkflows,
  type ArgoWorkflow,
} from './Argo';

const run: ArgoWorkflow = {
  metadata: { namespace: 'argo', name: 'demo', uid: 'uid-1' },
  spec: {},
  status: {
    nodes: {
      first: {
        id: 'first',
        displayName: 'prepare',
        type: 'Pod',
        phase: 'Succeeded',
        memoizationStatus: { hit: true },
        children: ['second'],
      },
      second: { id: 'second', displayName: 'train', type: 'Pod', phase: 'Failed' },
    },
  },
};
const node = run.status!.nodes!.first;
const reply = (body: unknown, status = 200) =>
  vi.stubGlobal(
    'fetch',
    vi.fn().mockImplementation(async () => new Response(JSON.stringify(body), { status })),
  );
afterEach(() => vi.unstubAllGlobals());

describe('native Argo adapter', () => {
  it('renders native status and cache hits without a KFP run or IR', () => {
    const graph = argoGraph(run);
    expect(graph.find((item) => item.id === 'first')?.data?.state).toBe(
      PipelineTaskTaskState.CACHED,
    );
    expect(graph.find((item) => item.id === 'second')?.data?.state).toBe(
      PipelineTaskTaskState.FAILED,
    );
    expect(graph.find((item) => item.id === 'first-second')).toMatchObject({
      source: 'first',
      target: 'second',
    });
  });
  it('bounds live discovery and only uses the Argo same-origin route', async () => {
    reply({ items: [run] });
    expect(await listArgoWorkflows('team/a')).toEqual([run]);
    const [path, options] = vi.mocked(fetch).mock.calls[0];
    expect(path).toContain('/argo/api/v1/workflows/team%2Fa?');
    expect(path).toContain('listOptions.limit=100');
    expect(options?.credentials).toBe('same-origin');
  });
  it('deduplicates live/archive by UID and prefers the live workflow', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(new Response(JSON.stringify({ items: [run] })))
        .mockResolvedValueOnce(
          new Response(
            JSON.stringify({
              items: [
                run,
                { ...run, metadata: { ...run.metadata, uid: 'archived', name: 'older' } },
              ],
            }),
          ),
        ),
    );
    const runs = await listArgoWorkflows('argo');
    expect(runs).toHaveLength(2);
    expect(runs.find((item) => item.metadata.uid === run.metadata.uid)?.archived).toBeUndefined();
    expect(runs.find((item) => item.metadata.uid === 'archived')?.archived).toBe(true);
  });
  it('recognizes persisted archive rows returned by Argo 4 workflow listing', async () => {
    reply({
      items: [
        {
          ...run,
          metadata: {
            ...run.metadata,
            labels: { 'workflows.argoproj.io/workflow-archiving-status': 'Persisted' },
          },
        },
      ],
    });
    expect((await listArgoWorkflows('argo'))[0].archived).toBe(true);
  });
  it('reads archived artifact metadata by UID, not a reused workflow name', async () => {
    expect(argoArtifactUrl({ ...run, archived: true }, node, 'dataset')).toBe(
      '/argo/artifact-files/argo/archived-workflows/uid-1/first/outputs/dataset',
    );
  });
  it('reads native container inline metadata without an extra request', async () => {
    const metadata = {
      schema: 'kfp-argo/v1',
      producer: { run: 'r', task: 't', namespace: 'n' },
      artifacts: [],
      inputs: [],
      metrics: {},
    };
    vi.stubGlobal('fetch', vi.fn());
    expect(
      await getArgoMetadata(run, {
        ...node,
        outputs: { parameters: [{ name: 'kfp-metadata', value: JSON.stringify(metadata) }] },
      }),
    ).toEqual(metadata);
    expect(fetch).not.toHaveBeenCalled();
  });
  it('does not fall back to a KFP endpoint on denied access', async () => {
    reply({}, 403);
    await expect(getArgoWorkflow(run)).rejects.toThrow('403');
    expect(fetch).toHaveBeenCalledTimes(1);
  });
  it('rejects a same-name replacement workflow', async () => {
    reply({ ...run, metadata: { ...run.metadata, uid: 'replacement' } });
    await expect(getArgoWorkflow(run)).rejects.toThrow('reused');
  });
  it('requires hydrated workflow details', async () => {
    reply({ ...run, status: { offloadNodeStatusVersion: 'v1' } });
    await expect(getArgoWorkflow(run)).rejects.toThrow('hydrated');
  });
  it('escapes every artifact path segment', () => {
    expect(argoArtifactUrl(run, { ...node, id: 'node/path' }, 'a?b')).toBe(
      '/argo/artifacts/argo/demo/node%2Fpath/a%3Fb',
    );
  });
  it('preserves the original cached producer', async () => {
    const metadata = {
      schema: 'kfp-argo/v1',
      producer: { run: 'original', task: 'prepare', namespace: 'argo' },
      inputs: [],
      artifacts: [],
      metrics: { score: 1 },
    };
    reply(metadata);
    expect((await getArgoMetadata(run, node)).producer.run).toBe('original');
  });
  it('rejects unknown metadata rather than silently omitting it', async () => {
    reply({ schema: 'future' });
    await expect(getArgoMetadata(run, node)).rejects.toThrow('Unsupported task metadata');
  });
  it.each([null, { run: {} }, { run: 'r', task: 't', namespace: 1 }])(
    'rejects malformed producer metadata: %j',
    async (producer) => {
      reply({ schema: 'kfp-argo/v1', producer, inputs: [], artifacts: [], metrics: {} });
      await expect(getArgoMetadata(run, node)).rejects.toThrow('Unsupported task metadata');
    },
  );
  it('rejects malformed artifact declarations before rendering', async () => {
    reply({
      schema: 'kfp-argo/v1',
      producer: { run: 'r', task: 't', namespace: 'n' },
      inputs: [],
      artifacts: [null],
      metrics: {},
    });
    await expect(getArgoMetadata(run, node)).rejects.toThrow('Unsupported task metadata');
  });
  it('decodes finite Argo log streams and reports stream errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(new Response('{"result":{"podName":"train","content":"hello"}}\n'))
        .mockResolvedValueOnce(new Response('{"error":{"message":"permission denied"}}\n')),
    );
    expect(await getArgoLogs(run)).toEqual(['train: hello']);
    await expect(getArgoLogs(run)).rejects.toThrow('permission denied');
  });
});
