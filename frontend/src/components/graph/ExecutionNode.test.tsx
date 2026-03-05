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
import { ReactFlowProvider } from 'react-flow-renderer';

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

  it('renders with COMPLETE state', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'completed-step', state: Execution.State.COMPLETE }}
      />,
    );
    expect(screen.getByText('completed-step')).toBeInTheDocument();
  });

  it('renders with RUNNING state', () => {
    renderWithProvider(
      <ExecutionNode
        id='exec-1'
        data={{ label: 'running-step', state: Execution.State.RUNNING }}
      />,
    );
    expect(screen.getByText('running-step')).toBeInTheDocument();
  });

  it('renders with FAILED state', () => {
    renderWithProvider(
      <ExecutionNode id='exec-1' data={{ label: 'failed-step', state: Execution.State.FAILED }} />,
    );
    expect(screen.getByText('failed-step')).toBeInTheDocument();
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

  it('returns an icon for COMPLETE state', () => {
    const icon = getIcon(Execution.State.COMPLETE);
    expect(icon).not.toBeNull();
  });

  it('returns an icon for RUNNING state', () => {
    const icon = getIcon(Execution.State.RUNNING);
    expect(icon).not.toBeNull();
  });

  it('returns an icon for FAILED state', () => {
    const icon = getIcon(Execution.State.FAILED);
    expect(icon).not.toBeNull();
  });

  it('returns an icon for NEW state', () => {
    const icon = getIcon(Execution.State.NEW);
    expect(icon).not.toBeNull();
  });

  it('returns an icon for CANCELED state', () => {
    const icon = getIcon(Execution.State.CANCELED);
    expect(icon).not.toBeNull();
  });

  it('returns an icon for CACHED state', () => {
    const icon = getIcon(Execution.State.CACHED);
    expect(icon).not.toBeNull();
  });

  it('returns an icon for UNKNOWN state', () => {
    const icon = getIcon(Execution.State.UNKNOWN);
    expect(icon).not.toBeNull();
  });
});

describe('getExecutionIcon', () => {
  it('returns an icon for undefined state', () => {
    const icon = getExecutionIcon(undefined);
    expect(icon).not.toBeNull();
  });

  it('returns an icon for defined state', () => {
    const icon = getExecutionIcon(Execution.State.RUNNING);
    expect(icon).not.toBeNull();
  });
});
