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

import * as React from 'react';
import { render, screen } from '@testing-library/react';
import ExecutionNode from './ExecutionNode';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import { ReactFlowProvider } from '@xyflow/react';

describe('ExecutionNode', () => {
  const renderWithProvider = (component: React.ReactElement) => {
    return render(<ReactFlowProvider>{component}</ReactFlowProvider>);
  };

  it('renders the execution label', () => {
    renderWithProvider(
      <ExecutionNode id='exec-1' data={{ label: 'train-step', state: undefined }} />,
    );
    expect(screen.getByText('train-step')).toBeInTheDocument();
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
      <ReactFlowProvider>
        <ExecutionNode id='task' data={{ label: 'Train', state }} />
      </ReactFlowProvider>,
    );
    expect(screen.getByRole('button', { name: 'Train' })).toHaveAccessibleDescription(label);
    expect(screen.getByRole('button', { name: 'Train' })).toHaveStyle({
      width: '200px',
      height: '56px',
    });
  });

  it('activates task inspection using the keyboard', async () => {
    const inspect = vi.fn();
    render(
      <ReactFlowProvider>
        <div onClick={inspect}>
          <ExecutionNode id='task' data={{ label: 'Train' }} />
        </div>
      </ReactFlowProvider>,
    );
    await userEvent.tab();
    await userEvent.keyboard('{Enter}');
    expect(inspect).toHaveBeenCalledTimes(1);
  });
});
