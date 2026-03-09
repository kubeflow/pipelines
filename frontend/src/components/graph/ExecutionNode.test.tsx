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

  it('renders with COMPLETE state and green background', () => {
    const { container } = renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'completed-step', state: Execution.State.COMPLETE }}
      />,
    );
    expect(screen.getByText('completed-step')).toBeInTheDocument();
    expect(container.querySelector('.bg-mui-green-50')).toBeInTheDocument();
  });

  it('renders with RUNNING state and green background', () => {
    const { container } = renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'running-step', state: Execution.State.RUNNING }}
      />,
    );
    expect(screen.getByText('running-step')).toBeInTheDocument();
    expect(container.querySelector('.bg-mui-green-50')).toBeInTheDocument();
  });

  it('renders with FAILED state and red background', () => {
    const { container } = renderWithProvider(
      <ExecutionNode id='exec-1' data={{ label: 'failed-step', state: Execution.State.FAILED }} />,
    );
    expect(screen.getByText('failed-step')).toBeInTheDocument();
    expect(container.querySelector('.bg-mui-red-50')).toBeInTheDocument();
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
    ['COMPLETE', Execution.State.COMPLETE, '.bg-mui-green-50', 'CheckCircleIcon'],
    ['RUNNING', Execution.State.RUNNING, '.bg-mui-green-50', 'RefreshIcon'],
    ['FAILED', Execution.State.FAILED, '.bg-mui-red-50', 'ErrorIcon'],
    ['NEW', Execution.State.NEW, '.bg-mui-blue-50', 'PowerSettingsNewIcon'],
    ['CANCELED', Execution.State.CANCELED, '.bg-mui-grey-200', undefined],
    ['CACHED', Execution.State.CACHED, '.bg-mui-green-50', 'CloudDownloadIcon'],
    ['UNKNOWN', Execution.State.UNKNOWN, '.bg-mui-grey-200', 'MoreHorizIcon'],
  ] as const)('returns the correct icon for %s state', (_label, state, bgClass, iconTestId) => {
    const { container } = render(getIcon(state)!);
    expect(container.querySelector(bgClass)).toBeInTheDocument();
    if (iconTestId) {
      expect(container.querySelector(`[data-testid="${iconTestId}"]`)).toBeInTheDocument();
    }
  });
});

describe('getExecutionIcon', () => {
  it('returns a grey ListAlt icon for undefined state', () => {
    const { container } = render(getExecutionIcon(undefined));
    expect(container.querySelector('.text-mui-grey-500')).toBeInTheDocument();
    expect(container.querySelector('[data-testid="ListAltIcon"]')).toBeInTheDocument();
  });

  it('returns a blue ListAlt icon for defined state', () => {
    const { container } = render(getExecutionIcon(Execution.State.RUNNING));
    expect(container.querySelector('.text-mui-blue-600')).toBeInTheDocument();
    expect(container.querySelector('[data-testid="ListAltIcon"]')).toBeInTheDocument();
  });
});
