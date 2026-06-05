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
import ExecutionNode, { getIcon, getExecutionIcon } from './ExecutionNode';
import { Execution } from 'src/third_party/mlmd';
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

  it('renders with COMPLETE state and correct icon', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'completed-step', state: Execution.State.COMPLETE }}
      />,
    );
    expect(screen.getByText('completed-step')).toBeInTheDocument();
    expect(screen.getByTestId('CheckCircleIcon')).toBeInTheDocument();
  });

  it('renders with RUNNING state and correct icon', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'running-step', state: Execution.State.RUNNING }}
      />,
    );
    expect(screen.getByText('running-step')).toBeInTheDocument();
    expect(screen.getByTestId('RefreshIcon')).toBeInTheDocument();
  });

  it('renders with FAILED state and correct icon', () => {
    renderWithProvider(
      <ExecutionNode id='exec-1' data={{ label: 'failed-step', state: Execution.State.FAILED }} />,
    );
    expect(screen.getByText('failed-step')).toBeInTheDocument();
    expect(screen.getByTestId('ErrorIcon')).toBeInTheDocument();
  });

  it('sets the title attribute', () => {
    renderWithProvider(
      <ExecutionNode id='exec-1' data={{ label: 'titled-step', state: undefined }} />,
    );
    expect(screen.getByTitle('titled-step')).toBeInTheDocument();
  });
});

describe('getIcon', () => {
  it('returns null for undefined state', () => {
    expect(getIcon(undefined)).toBeNull();
  });

  it.each([
    ['COMPLETE', Execution.State.COMPLETE, 'CheckCircleIcon', 'bg-mui-green-50'],
    ['RUNNING', Execution.State.RUNNING, 'RefreshIcon', 'bg-mui-green-50'],
    ['FAILED', Execution.State.FAILED, 'ErrorIcon', 'bg-mui-red-50'],
    ['NEW', Execution.State.NEW, 'PowerSettingsNewIcon', 'bg-mui-blue-50'],
    ['CANCELED', Execution.State.CANCELED, 'StopCircleIcon', 'bg-mui-grey-200'],
    ['CACHED', Execution.State.CACHED, 'CloudDownloadIcon', 'bg-mui-green-50'],
    ['UNKNOWN', Execution.State.UNKNOWN, 'MoreHorizIcon', 'bg-mui-grey-200'],
  ] as const)(
    'returns the correct icon for %s state',
    (_label, state, iconTestId, backgroundClass) => {
      render(getIcon(state)!);
      const stateIcon = screen.getByTestId(iconTestId);
      expect(stateIcon).toBeInTheDocument();
      expect(stateIcon.parentElement).toHaveClass(backgroundClass);
    },
  );
});

describe('getExecutionIcon', () => {
  it('returns a default ListAlt icon for undefined state', () => {
    render(getExecutionIcon(undefined));
    expect(screen.getByTestId('execution-icon-default')).toBeInTheDocument();
  });

  it('returns an active ListAlt icon for defined state', () => {
    render(getExecutionIcon(Execution.State.RUNNING));
    expect(screen.getByTestId('execution-icon-active')).toBeInTheDocument();
  });
});
