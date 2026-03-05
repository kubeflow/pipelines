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
import ReduceGraphSwitch from './ReduceGraphSwitch';

describe('ReduceGraphSwitch', () => {
  it('renders the switch with label', () => {
    render(<ReduceGraphSwitch />);
    expect(screen.getByText('Simplify Graph')).toBeInTheDocument();
  });

  it('renders the switch in unchecked state by default', () => {
    render(<ReduceGraphSwitch />);
    const switchInput = screen.getByRole('checkbox');
    expect(switchInput).not.toBeChecked();
  });

  it('renders the switch in checked state when checked prop is true', () => {
    render(<ReduceGraphSwitch checked={true} />);
    const switchInput = screen.getByRole('checkbox');
    expect(switchInput).toBeChecked();
  });

  it('calls onChange when the switch is toggled', () => {
    const handleChange = vi.fn();
    render(<ReduceGraphSwitch onChange={handleChange} />);
    const switchInput = screen.getByRole('checkbox');
    fireEvent.click(switchInput);
    expect(handleChange).toHaveBeenCalled();
  });

  it('renders tooltip with transitive reduction help text on hover', async () => {
    render(<ReduceGraphSwitch />);
    const label = screen.getByText('Simplify Graph');
    fireEvent.mouseOver(label);
    const tooltipText = await screen.findByText(/transitive reduction/i);
    expect(tooltipText).toBeInTheDocument();
  });
});
