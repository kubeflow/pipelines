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

import { render, screen, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
import SubDagLayer from './SubDagLayer';

describe('SubDagLayer', () => {
  it('renders the Layers label', () => {
    render(<SubDagLayer layers={['root']} onLayersUpdate={vi.fn()} />);
    expect(screen.getByText('Layers')).toBeInTheDocument();
  });

  it('renders breadcrumb for a single layer', () => {
    render(<SubDagLayer layers={['root']} onLayersUpdate={vi.fn()} />);
    expect(screen.getByText('root')).toBeInTheDocument();
  });

  it('renders breadcrumbs for multiple layers', () => {
    render(<SubDagLayer layers={['root', 'child', 'grandchild']} onLayersUpdate={vi.fn()} />);
    expect(screen.getByText('root')).toBeInTheDocument();
    expect(screen.getByText('child')).toBeInTheDocument();
    expect(screen.getByText('grandchild')).toBeInTheDocument();
  });

  it('disables the active (last) breadcrumb', () => {
    render(<SubDagLayer layers={['root', 'child']} onLayersUpdate={vi.fn()} />);
    const activeButton = screen.getByText('child');
    expect(activeButton.closest('button')).toBeDisabled();
  });

  it('enables inactive breadcrumbs', () => {
    render(<SubDagLayer layers={['root', 'child']} onLayersUpdate={vi.fn()} />);
    const inactiveButton = screen.getByText('root');
    expect(inactiveButton.closest('button')).not.toBeDisabled();
  });

  it('calls onLayersUpdate when an inactive breadcrumb is clicked', () => {
    const onLayersUpdate = vi.fn();
    render(
      <SubDagLayer layers={['root', 'child', 'grandchild']} onLayersUpdate={onLayersUpdate} />,
    );
    fireEvent.click(screen.getByText('root'));
    expect(onLayersUpdate).toHaveBeenCalledWith(['root']);
  });

  it('calls onLayersUpdate with correct slice when middle breadcrumb is clicked', () => {
    const onLayersUpdate = vi.fn();
    render(
      <SubDagLayer layers={['root', 'child', 'grandchild']} onLayersUpdate={onLayersUpdate} />,
    );
    fireEvent.click(screen.getByText('child'));
    expect(onLayersUpdate).toHaveBeenCalledWith(['root', 'child']);
  });
});
