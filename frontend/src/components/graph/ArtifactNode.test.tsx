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
import ArtifactNode from './ArtifactNode';
import { Artifact } from 'src/third_party/mlmd';
import { ReactFlowProvider } from '@xyflow/react';

describe('ArtifactNode', () => {
  const renderWithProvider = (component: React.ReactElement) => {
    return render(<ReactFlowProvider>{component}</ReactFlowProvider>);
  };

  it('renders the artifact label', () => {
    renderWithProvider(
      <ArtifactNode id='artifact-1' data={{ label: 'my-artifact', state: undefined }} />,
    );
    expect(screen.getByText('my-artifact')).toBeInTheDocument();
  });

  it('renders with LIVE state and correct icon', () => {
    renderWithProvider(
      <ArtifactNode
        id='artifact-1'
        data={{ label: 'live-artifact', state: Artifact.State.LIVE }}
      />,
    );
    expect(screen.getByText('live-artifact')).toBeInTheDocument();
    const liveIcon = screen.getByTestId('artifact-icon-live');
    expect(liveIcon).toBeInTheDocument();
    expect(liveIcon).toHaveClass('text-mui-yellow-800');
  });

  it('renders with undefined state and default icon', () => {
    renderWithProvider(
      <ArtifactNode id='artifact-1' data={{ label: 'unknown-artifact', state: undefined }} />,
    );
    expect(screen.getByText('unknown-artifact')).toBeInTheDocument();
    const defaultIcon = screen.getByTestId('artifact-icon-default');
    expect(defaultIcon).toBeInTheDocument();
    expect(defaultIcon).toHaveClass('text-mui-grey-300-dark');
  });

  it('sets the title attribute on the button', () => {
    renderWithProvider(
      <ArtifactNode id='artifact-1' data={{ label: 'titled-artifact', state: undefined }} />,
    );
    expect(screen.getByTitle('titled-artifact')).toBeInTheDocument();
  });

  it('renders with the correct id on the label span', () => {
    renderWithProvider(
      <ArtifactNode id='artifact-42' data={{ label: 'test', state: undefined }} />,
    );
    expect(screen.getByTestId('artifact-42')).toBeInTheDocument();
  });
});
