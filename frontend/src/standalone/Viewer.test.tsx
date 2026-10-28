// Copyright 2026 The Kubeflow Authors. Licensed under the Apache License, Version 2.0.
import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { CommonTestWrapper } from 'src/TestWrapper';
import { mockResizeObserver } from 'src/TestUtils';
import { nestedArtifactSpec } from 'src/data/test/groupedFlow';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import { Viewer, type ViewerState } from './Viewer';

const initial: ViewerState = {
  revision: 1,
  error: null,
  spec: PipelineSpec.toJSON(nestedArtifactSpec) as Record<string, unknown>,
};

beforeEach(() => {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  );
});

beforeEach(() => mockResizeObserver());
afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe('standalone viewer using the KFP pipeline graph', () => {
  it('uses KFP task/artifact nodes, inline DAG groups, controls, and inspector offline', () => {
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);
    render(
      <CommonTestWrapper>
        <Viewer initial={initial} live={false} />
      </CommonTestWrapper>,
    );
    expect(screen.getByTestId('DagCanvas')).toBeInTheDocument();
    expect(document.querySelector('.react-flow__node-EXECUTION')).not.toBeNull();
    expect(document.querySelector('.react-flow__node-ARTIFACT')).not.toBeNull();
    expect(screen.getByRole('group', { name: 'Graph controls' })).toBeInTheDocument();
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse Training pipeline' }));
    expect(screen.queryByText('Train model')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Expand Training pipeline' }));
    fireEvent.click(screen.getByText('Train model'));
    expect(screen.getByText('Output Artifacts')).toBeInTheDocument();
    expect(screen.getByText('Image')).toBeInTheDocument();
    expect(screen.getByText('python:3.11')).toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalled();
  });

  it('preserves inspector selection and expansion through live updates and compile errors', async () => {
    vi.useFakeTimers();
    let next = initial;
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({ ok: true, json: async () => next })),
    );
    render(
      <CommonTestWrapper>
        <Viewer initial={initial} live />
      </CommonTestWrapper>,
    );
    await act(async () => {});
    fireEvent.click(screen.getByRole('button', { name: 'Collapse Training pipeline' }));
    fireEvent.click(screen.getByText('Deploy model'));
    const updated = PipelineSpec.fromJSON(initial.spec);
    updated.deploymentSpec!.executors['exec-evaluate'].container.image = 'python:3.13';
    updated.root!.dag!.tasks.deploy.taskInfo!.name = 'Deploy updated model';
    next = {
      ...initial,
      revision: 2,
      spec: PipelineSpec.toJSON(updated) as Record<string, unknown>,
    };
    await act(async () => {
      await vi.advanceTimersByTimeAsync(750);
    });
    expect(screen.getByText('python:3.13')).toBeInTheDocument();
    expect(screen.getAllByText('Deploy updated model')).toHaveLength(2);
    expect(screen.getByRole('button', { name: 'Expand Training pipeline' })).toBeInTheDocument();
    next = { ...next, revision: 3, error: 'SyntaxError: incomplete source' };
    await act(async () => {
      await vi.advanceTimersByTimeAsync(750);
    });
    expect(screen.getByRole('alert')).toHaveTextContent('SyntaxError');
    expect(screen.getByText('python:3.13')).toBeInTheDocument();
    next = { ...next, revision: 4, error: null };
    await act(async () => {
      await vi.advanceTimersByTimeAsync(750);
    });
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('closes the inspector when the selected task is removed', async () => {
    vi.useFakeTimers();
    let next = initial;
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({ ok: true, json: async () => next })),
    );
    render(
      <CommonTestWrapper>
        <Viewer initial={initial} live />
      </CommonTestWrapper>,
    );
    await act(async () => {});
    fireEvent.click(screen.getByText('Deploy model'));
    expect(screen.getByRole('button', { name: 'close' })).toBeInTheDocument();
    const updated = PipelineSpec.fromJSON(initial.spec);
    delete updated.root!.dag!.tasks.deploy;
    next = {
      ...initial,
      revision: 2,
      spec: PipelineSpec.toJSON(updated) as Record<string, unknown>,
    };
    await act(async () => {
      await vi.advanceTimersByTimeAsync(750);
    });
    expect(screen.queryByRole('button', { name: 'close' })).not.toBeInTheDocument();
  });

  it('reports disconnection, retries, and stops polling on unmount', async () => {
    vi.useFakeTimers();
    const fetch = vi.fn().mockRejectedValue(new Error('offline'));
    vi.stubGlobal('fetch', fetch);
    const { unmount } = render(
      <CommonTestWrapper>
        <Viewer initial={initial} live />
      </CommonTestWrapper>,
    );
    await act(async () => {});
    expect(screen.getByRole('alert')).toHaveTextContent('Disconnected');
    fetch.mockResolvedValue({ ok: true, json: async () => initial });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(750);
    });
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    unmount();
    const calls = fetch.mock.calls.length;
    await vi.advanceTimersByTimeAsync(2000);
    expect(fetch).toHaveBeenCalledTimes(calls);
    expect(fetch.mock.calls[0][1].signal.aborted).toBe(true);
  });
});
