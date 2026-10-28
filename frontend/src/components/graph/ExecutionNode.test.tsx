/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { ThemeProvider } from 'src/components/shell/ThemeProvider';
import * as React from 'react';
import { render, screen } from '@testing-library/react';
import ExecutionNode from './ExecutionNode';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { PipelineTaskTaskState } from 'src/apisv2/run';
import { ReactFlowProvider } from '@xyflow/react';

beforeEach(() => {
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() })),
  );
});
afterEach(() => vi.unstubAllGlobals());

describe('ExecutionNode', () => {
  const renderWithProvider = (component: React.ReactElement) => {
    return render(
      <ThemeProvider>
        <ReactFlowProvider>{component}</ReactFlowProvider>
      </ThemeProvider>,
    );
  };

  it('keeps the blue ring for selection without requiring focus', () => {
    const node = (selected: boolean) => (
      <ThemeProvider>
        <ReactFlowProvider>
          <ExecutionNode id='exec-1' data={{ label: 'train-step' }} selected={selected} />
        </ReactFlowProvider>
      </ThemeProvider>
    );
    const { rerender } = render(node(false));
    const button = screen.getByRole('button', { name: 'train-step' });
    expect(button).not.toHaveAttribute('data-selected');
    rerender(node(true));
    expect(button).not.toHaveFocus();
    expect(button).toHaveAttribute('data-selected', 'true');
    rerender(node(false));
    expect(button).not.toHaveAttribute('data-selected');
  });

  it('renders the execution label', () => {
    renderWithProvider(
      <ExecutionNode id='exec-1' data={{ label: 'train-step', state: undefined }} />,
    );
    expect(screen.getByText('train-step')).toBeInTheDocument();
    const reserved = screen
      .getByRole('button', { name: 'train-step' })
      .querySelector('.kfp-graph-node-status');
    expect(reserved).toBeEmptyDOMElement();
    expect(reserved).toHaveAttribute('aria-hidden', 'true');
  });

  it('renders with SUCCEEDED state and correct icon', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'completed-step', state: PipelineTaskTaskState.SUCCEEDED }}
      />,
    );
    expect(screen.getByText('completed-step')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'completed-step' })).toHaveAccessibleDescription(
      'Succeeded',
    );
  });

  it('renders with RUNNING state and correct icon', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'running-step', state: PipelineTaskTaskState.RUNNING }}
      />,
    );
    expect(screen.getByText('running-step')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'running-step' })).toHaveAccessibleDescription(
      'Running',
    );
  });

  it('renders with FAILED state and correct icon', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'failed-step', state: PipelineTaskTaskState.FAILED }}
      />,
    );
    expect(screen.getByText('failed-step')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'failed-step' })).toHaveAccessibleDescription(
      'Failed',
    );
  });

  it('shows a warning icon instead of the running icon when a lifecycle message is present', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{
          label: 'stuck-step',
          state: PipelineTaskTaskState.RUNNING,
          lifecycleMessage: 'ImagePullBackOff',
        }}
      />,
    );
    expect(document.querySelector('[data-state="WARNING"]')).toBeInTheDocument();
    expect(document.querySelector('[data-state="RUNNING"]')).not.toBeInTheDocument();
    expect(document.querySelector('[data-state="FAILED"]')).not.toBeInTheDocument();
  });

  it('announces the warning in the node name', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{
          label: 'stuck-step',
          state: PipelineTaskTaskState.RUNNING,
          lifecycleMessage: 'ImagePullBackOff',
        }}
      />,
    );
    expect(
      screen.getByRole('button', { name: 'stuck-step, pod lifecycle warning' }),
    ).toBeInTheDocument();
  });

  it('shows the failed icon, not the warning, once the task has failed', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{
          label: 'oom-step',
          state: PipelineTaskTaskState.FAILED,
          lifecycleMessage: 'OOMKilled',
        }}
      />,
    );
    expect(document.querySelector('[data-state="FAILED"]')).toBeInTheDocument();
    expect(document.querySelector('[data-state="WARNING"]')).not.toBeInTheDocument();
  });

  it('shows no warning when the lifecycle message is empty', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'healthy-step', state: PipelineTaskTaskState.RUNNING, lifecycleMessage: '' }}
      />,
    );
    expect(document.querySelector('[data-state="RUNNING"]')).toBeInTheDocument();
    expect(document.querySelector('[data-state="WARNING"]')).not.toBeInTheDocument();
  });

  it('sets the title attribute', () => {
    renderWithProvider(
      <ExecutionNode id='exec-1' data={{ label: 'titled-step', state: undefined }} />,
    );
    expect(screen.getByTitle('titled-step')).toBeInTheDocument();
  });

  it('renders hidden, non-connectable edge anchors', () => {
    const { container } = renderWithProvider(
      <ExecutionNode id='exec-1' data={{ label: 'anchored-step', state: undefined }} />,
    );
    const handles = container.querySelectorAll('.react-flow__handle');

    expect(handles).toHaveLength(2);
    handles.forEach((handle) => {
      expect(handle).toHaveStyle({
        height: '1px',
        minHeight: '1px',
        minWidth: '1px',
        opacity: '0',
        pointerEvents: 'none',
        width: '1px',
      });
      expect(handle).not.toHaveClass('connectable');
      expect(handle).not.toHaveClass('connectablestart');
      expect(handle).not.toHaveClass('connectableend');
      expect(handle).not.toHaveClass('connectionindicator');
    });
  });
});

describe('task state presentation', () => {
  it.each([
    [PipelineTaskTaskState.SUCCEEDED, 'Succeeded'],
    [PipelineTaskTaskState.RUNNING, 'Running'],
    [PipelineTaskTaskState.FAILED, 'Failed'],
    [PipelineTaskTaskState.SKIPPED, 'Skipped'],
    [PipelineTaskTaskState.CACHED, 'Cached'],
    [PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED, 'Unknown'],
    [undefined, 'Task'],
  ] as const)('announces %s without depending on color', (state, label) => {
    render(
      <ThemeProvider>
        <ReactFlowProvider>
          <ExecutionNode id='task' data={{ label: 'Train', state }} />
        </ReactFlowProvider>
      </ThemeProvider>,
    );
    expect(screen.getByRole('img', { name: 'Task' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Train' })).toHaveAccessibleDescription(label);
    expect(screen.getByRole('button', { name: 'Train' })).toHaveStyle({
      width: '200px',
      height: '56px',
    });
  });

  it('activates task inspection using the keyboard', async () => {
    const inspect = vi.fn();
    render(
      <ThemeProvider>
        <ReactFlowProvider>
          <div onClick={inspect}>
            <ExecutionNode id='task' data={{ label: 'Train' }} />
          </div>
        </ReactFlowProvider>
      </ThemeProvider>,
    );
    await userEvent.tab();
    await userEvent.keyboard('{Enter}');
    expect(inspect).toHaveBeenCalledTimes(1);
  });
});

it('shows status on the right without a subtitle and exposes it on keyboard focus', async () => {
  render(
    <ThemeProvider defaultTheme='dark' storageKey='graph-tooltip-test'>
      <ReactFlowProvider>
        <ExecutionNode
          id='compact'
          data={{ label: 'Train', state: PipelineTaskTaskState.SUCCEEDED }}
        />
      </ReactFlowProvider>
    </ThemeProvider>,
  );
  const node = screen.getByRole('button', { name: 'Train' });
  expect(node.querySelector('.kfp-graph-node-meta')).toBeNull();
  expect(node.querySelector('.kfp-graph-node-status [data-state="SUCCEEDED"]')).toBeInTheDocument();
  expect(node).toHaveAccessibleDescription('Succeeded');
  await userEvent.tab();
  const tooltip = await screen.findByRole('tooltip');
  expect(tooltip).toHaveTextContent('Train · Succeeded');
  expect(tooltip.closest('.kfp-theme')).toHaveClass('dark');
});

it('explains the status icon directly on hover and keyboard focus', async () => {
  const user = userEvent.setup();
  render(
    <ThemeProvider>
      <ReactFlowProvider>
        <ExecutionNode
          id='running'
          data={{ label: 'Train', state: PipelineTaskTaskState.RUNNING }}
        />
      </ReactFlowProvider>
    </ThemeProvider>,
  );
  const status = screen.getByRole('img', { name: 'Running' });
  await user.hover(status);
  expect(await screen.findByRole('tooltip')).toHaveTextContent(/^Running$/);
  await user.unhover(status);
  await user.tab();
  await user.tab();
  expect(status).toHaveFocus();
  expect(await screen.findByRole('tooltip')).toHaveTextContent(/^Running$/);
});
