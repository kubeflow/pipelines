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

  it('returns a green CheckCircle icon for COMPLETE state', () => {
    const { container } = render(getIcon(Execution.State.COMPLETE)!);
    expect(container.querySelector('.bg-mui-green-50')).toBeInTheDocument();
    expect(container.querySelector('[data-testid="CheckCircleIcon"]')).toBeInTheDocument();
  });

  it('returns a green Refresh icon for RUNNING state', () => {
    const { container } = render(getIcon(Execution.State.RUNNING)!);
    expect(container.querySelector('.bg-mui-green-50')).toBeInTheDocument();
    expect(container.querySelector('[data-testid="RefreshIcon"]')).toBeInTheDocument();
  });

  it('returns a red Error icon for FAILED state', () => {
    const { container } = render(getIcon(Execution.State.FAILED)!);
    expect(container.querySelector('.bg-mui-red-50')).toBeInTheDocument();
    expect(container.querySelector('[data-testid="ErrorIcon"]')).toBeInTheDocument();
  });

  it('returns a blue PowerSettingsNew icon for NEW state', () => {
    const { container } = render(getIcon(Execution.State.NEW)!);
    expect(container.querySelector('.bg-mui-blue-50')).toBeInTheDocument();
    expect(container.querySelector('[data-testid="PowerSettingsNewIcon"]')).toBeInTheDocument();
  });

  it('returns a grey icon for CANCELED state', () => {
    const { container } = render(getIcon(Execution.State.CANCELED)!);
    expect(container.querySelector('.bg-mui-grey-200')).toBeInTheDocument();
  });

  it('returns a green CloudDownload icon for CACHED state', () => {
    const { container } = render(getIcon(Execution.State.CACHED)!);
    expect(container.querySelector('.bg-mui-green-50')).toBeInTheDocument();
    expect(container.querySelector('[data-testid="CloudDownloadIcon"]')).toBeInTheDocument();
  });

  it('returns a grey MoreHoriz icon for UNKNOWN state', () => {
    const { container } = render(getIcon(Execution.State.UNKNOWN)!);
    expect(container.querySelector('.bg-mui-grey-200')).toBeInTheDocument();
    expect(container.querySelector('[data-testid="MoreHorizIcon"]')).toBeInTheDocument();
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
