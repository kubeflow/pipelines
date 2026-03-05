/*
 * Copyright 2021 The Kubeflow Authors
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
import { NameWithTooltip } from './CustomTableNameColumn';

describe('NameWithTooltip', () => {
  it('renders display_name when available', () => {
    render(
      <NameWithTooltip
        value={{ display_name: 'My Pipeline', name: 'pipeline-123' }}
        id='test-id'
      />,
    );
    expect(screen.getByText('My Pipeline')).toBeInTheDocument();
  });

  it('falls back to name when display_name is not available', () => {
    render(<NameWithTooltip value={{ name: 'pipeline-123' }} id='test-id' />);
    expect(screen.getByText('pipeline-123')).toBeInTheDocument();
  });

  it('renders empty string when both display_name and name are missing', () => {
    const { container } = render(<NameWithTooltip value={{}} id='test-id' />);
    expect(container.querySelector('span')?.textContent).toBe('');
  });

  it('renders empty string when value is undefined', () => {
    const { container } = render(<NameWithTooltip value={undefined} id='test-id' />);
    expect(container.querySelector('span')?.textContent).toBe('');
  });

  it('prefers display_name over name', () => {
    render(
      <NameWithTooltip
        value={{ display_name: 'Display Name', name: 'internal-name' }}
        id='test-id'
      />,
    );
    expect(screen.getByText('Display Name')).toBeInTheDocument();
    expect(screen.queryByText('internal-name')).toBeNull();
  });
});
