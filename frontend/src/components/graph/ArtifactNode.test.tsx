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
import { ThemeProvider } from 'src/components/shell/ThemeProvider';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import ArtifactNode from './ArtifactNode';
import { ReactFlowProvider } from '@xyflow/react';

beforeEach(() => {
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() })),
  );
});
afterEach(() => vi.unstubAllGlobals());

describe('ArtifactNode', () => {
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
          <ArtifactNode id='artifact-1' data={{ label: 'model' }} selected={selected} />
        </ReactFlowProvider>
      </ThemeProvider>
    );
    const { rerender } = render(node(false));
    const button = screen.getByRole('button', { name: 'model' });
    expect(button).not.toHaveAttribute('data-selected');
    rerender(node(true));
    expect(button).not.toHaveFocus();
    expect(button).toHaveAttribute('data-selected', 'true');
    rerender(node(false));
    expect(button).not.toHaveAttribute('data-selected');
  });

  it('renders the artifact label', () => {
    renderWithProvider(
      <ArtifactNode id='artifact-1' data={{ label: 'my-artifact', hasArtifact: false }} />,
    );
    expect(screen.getByText('my-artifact')).toBeInTheDocument();
  });

  it('announces available artifact data without an execution status icon', () => {
    renderWithProvider(
      <ArtifactNode id='artifact-1' data={{ label: 'live-artifact', hasArtifact: true }} />,
    );
    expect(screen.getByText('live-artifact')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'live-artifact' })).toHaveAccessibleDescription(
      'Artifact available',
    );
  });

  it('renders with undefined state and default icon', () => {
    renderWithProvider(
      <ArtifactNode id='artifact-1' data={{ label: 'unknown-artifact', hasArtifact: false }} />,
    );
    expect(screen.getByText('unknown-artifact')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'unknown-artifact' })).toHaveAccessibleDescription(
      'Artifact',
    );
  });

  it('keeps type and availability accessible without a visible subtitle', async () => {
    const user = userEvent.setup();
    const { container } = renderWithProvider(
      <ArtifactNode id='artifact-1' data={{ label: 'dataset', hasArtifact: true }} />,
    );
    expect(screen.getByRole('img', { name: 'Artifact' })).toBeInTheDocument();
    expect(container.querySelector('.kfp-graph-node-meta')).not.toBeInTheDocument();
    expect(container.querySelector('.kfp-graph-node-status')).toBeEmptyDOMElement();
    expect(container.querySelector('.kfp-graph-node-status')).toHaveAttribute(
      'aria-hidden',
      'true',
    );
    await user.tab();
    expect(await screen.findByRole('tooltip')).toHaveTextContent('dataset · Artifact available');
  });

  it('renders with the correct id on the label span', () => {
    renderWithProvider(
      <ArtifactNode id='artifact-42' data={{ label: 'test', hasArtifact: false }} />,
    );
    expect(screen.getByTestId('artifact-42')).toBeInTheDocument();
  });

  it('renders hidden, non-connectable edge anchors', () => {
    const { container } = renderWithProvider(
      <ArtifactNode id='artifact-1' data={{ label: 'anchored-artifact', hasArtifact: false }} />,
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
