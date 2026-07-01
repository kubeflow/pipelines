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
import { render, screen, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
import SubDagNode from './SubDagNode';
import { Execution } from 'src/third_party/mlmd';
import { ReactFlowProvider } from '@xyflow/react';

describe('SubDagNode', () => {
  const renderWithProvider = (component: React.ReactElement) => {
    return render(<ReactFlowProvider>{component}</ReactFlowProvider>);
  };

  const defaultData = {
    label: 'sub-pipeline',
    expand: vi.fn(),
    state: undefined as Execution.State | undefined,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the sub-dag label', () => {
    renderWithProvider(<SubDagNode id='subdag-1' data={defaultData} />);
    expect(screen.getByText('sub-pipeline')).toBeInTheDocument();
  });

  it('sets accessible label on the button', () => {
    renderWithProvider(<SubDagNode id='subdag-1' data={defaultData} />);
    expect(screen.getByRole('button', { name: 'sub-pipeline' })).toBeInTheDocument();
  });

  it('renders COMPLETE state icon', () => {
    renderWithProvider(
      <SubDagNode id='subdag-1' data={{ ...defaultData, state: Execution.State.COMPLETE }} />,
    );
    expect(screen.getByTestId('CheckCircleIcon')).toBeInTheDocument();
  });

  it('renders RUNNING state icon', () => {
    renderWithProvider(
      <SubDagNode id='subdag-1' data={{ ...defaultData, state: Execution.State.RUNNING }} />,
    );
    expect(screen.getByTestId('RefreshIcon')).toBeInTheDocument();
  });

  it('calls expand callback when expand button is clicked', () => {
    const expandFn = vi.fn();
    renderWithProvider(<SubDagNode id='subdag-1' data={{ ...defaultData, expand: expandFn }} />);
    const expandButton = screen.getByTestId('expand-button');
    fireEvent.click(expandButton);
    expect(expandFn).toHaveBeenCalledWith('subdag-1');
  });

  it('renders with the correct id on the label span', () => {
    renderWithProvider(<SubDagNode id='subdag-42' data={defaultData} />);
    expect(screen.getByTestId('subdag-42')).toBeInTheDocument();
  });

  it('renders hidden, non-connectable edge anchors', () => {
    const { container } = renderWithProvider(<SubDagNode id='subdag-1' data={defaultData} />);
    const handles = container.querySelectorAll('.react-flow__handle');

    expect(handles).toHaveLength(2);
    handles.forEach((handle) => {
      expect(handle).toHaveStyle({
        height: '1px',
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
