// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, waitFor, cleanup } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ArgoRunDetails from './ArgoRunDetails';

vi.mock('src/components/graph/PipelineGraph', () => ({
  default: ({ elements, onSelectionChange }: any) => (
    <div>
      {elements
        .filter((element: any) => element.data)
        .map((element: any) => (
          <button key={element.id} onClick={() => onSelectionChange({ layers: ['root'], element })}>
            {element.data.label}
          </button>
        ))}
    </div>
  ),
}));
vi.mock('src/components/LogViewer', () => ({
  default: ({ logLines }: { logLines: string[] }) => <pre>{logLines.join('\n')}</pre>,
}));

const run = {
  metadata: {
    name: 'run-one',
    namespace: 'argo',
    uid: 'uid-one',
    labels: { 'pipelines.kubeflow.org/experiment': 'training' },
  },
  spec: {},
  status: {
    phase: 'Succeeded',
    nodes: {
      prepare: {
        id: 'prepare',
        displayName: 'prepare',
        type: 'Pod',
        phase: 'Succeeded',
        memoizationStatus: { hit: true },
        outputs: { artifacts: [{ name: 'kfp-metadata' }, { name: 'dataset' }] },
      },
    },
  },
};
const producer = { run: 'original-run', task: 'original-prepare', namespace: 'argo' };
const metadata = {
  schema: 'kfp-argo/v1',
  producer,
  inputs: [],
  metrics: { score: 0.75 },
  artifacts: [
    {
      port: 'dataset',
      type: 'system.Dataset@0.0.1',
      uri: 's3://bucket/original/dataset',
      producer,
      metadata: { rows: 4 },
    },
  ],
};

function setup() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string) => {
      if (url.includes('/artifacts/')) return new Response(JSON.stringify(metadata));
      if (url.includes('/log?'))
        return new Response('{"result":{"podName":"train","content":"training complete"}}\n');
      if (url.includes('/run-one')) return new Response(JSON.stringify(run));
      return new Response(JSON.stringify({ items: [run] }));
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ArgoRunDetails namespace='argo' />
    </QueryClientProvider>,
  );
  return client;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe('Argo run details', () => {
  it('filters experiment membership labels and displays ConfigMap names', async () => {
    const other = {
      ...run,
      metadata: {
        ...run.metadata,
        uid: 'other',
        name: 'other-run',
        labels: { 'pipelines.kubeflow.org/experiment': 'another' },
      },
    };
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response(JSON.stringify({ items: [run, other] }))),
    );
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const clear = vi.fn();
    render(
      <QueryClientProvider client={client}>
        <ArgoRunDetails
          namespace='argo'
          experiment='training'
          experimentNames={{ training: 'Human training' }}
          onClearExperiment={clear}
        />
      </QueryClientProvider>,
    );
    await screen.findByRole('button', { name: 'run-one' });
    expect(screen.queryByRole('button', { name: 'other-run' })).not.toBeInTheDocument();
    expect(screen.getByText(/Succeeded.*Human training/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'All experiments' }));
    expect(clear).toHaveBeenCalledTimes(1);
    client.clear();
  });
  it('shows native artifacts and cached producer, preserving task selection on refresh', async () => {
    const client = setup();
    fireEvent.click(await screen.findByRole('button', { name: 'run-one' }));
    fireEvent.click(await screen.findByRole('button', { name: 'prepare' }));
    expect(
      await screen.findByText('Producer: original-run / original-prepare'),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'dataset' })).toHaveAttribute(
      'href',
      '/argo/artifacts/argo/run-one/prepare/dataset',
    );
    await client.invalidateQueries({ queryKey: ['argo', 'uid-one'] });
    expect(screen.getByText('Producer: original-run / original-prepare')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Show run logs' }));
    expect(await screen.findByText('train: training complete')).toBeInTheDocument();
    expect(vi.mocked(fetch).mock.calls.every(([url]) => String(url).startsWith('/argo/'))).toBe(
      true,
    );
    client.clear();
  });
  it('loads comparison only for explicitly selected runs', async () => {
    const client = setup();
    const checkbox = await screen.findByRole('checkbox', { name: 'Compare' });
    expect(
      vi
        .mocked(fetch)
        .mock.calls.every(
          ([url]) =>
            String(url).startsWith('/argo/api/v1/workflows/argo?') ||
            String(url).startsWith('/argo/api/v1/archived-workflows?'),
        ),
    ).toBe(true);
    fireEvent.click(checkbox);
    expect(await screen.findByText('prepare.score: 0.75')).toBeInTheDocument();
    fireEvent.click(checkbox);
    expect(screen.queryByText('Selected-run comparison')).not.toBeInTheDocument();
    client.clear();
  });
  it('displays denied access without trying a KFP backend', async () => {
    const client = setup();
    await screen.findByRole('button', { name: 'run-one' });
    vi.mocked(fetch).mockResolvedValue(new Response('{}', { status: 403 }));
    fireEvent.click(screen.getByRole('button', { name: 'run-one' }));
    await waitFor(() =>
      expect(screen.getByText(/Argo request failed \(403\)/)).toBeInTheDocument(),
    );
    expect(vi.mocked(fetch).mock.calls.every(([url]) => String(url).startsWith('/argo/'))).toBe(
      true,
    );
    client.clear();
  });
});
