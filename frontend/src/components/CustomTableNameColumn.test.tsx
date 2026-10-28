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

import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Link } from 'react-router';
import { NameWithTooltip, NameTooltip } from './CustomTableNameColumn';

describe('NameWithTooltip', () => {
  it('reveals the resource name on keyboard focus while retaining its display name', async () => {
    render(
      <NameWithTooltip
        value={{ display_name: 'My Pipeline', name: 'pipeline-123' }}
        id='test-id'
      />,
    );
    await userEvent.tab();
    expect(screen.getByText('My Pipeline')).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Name: pipeline-123');
  });

  it('falls back to name when display_name is not available', () => {
    render(<NameWithTooltip value={{ name: 'pipeline-123' }} id='test-id' />);
    expect(screen.getByText('pipeline-123')).toBeInTheDocument();
  });

  it('renders empty string when both display_name and name are missing', () => {
    render(<NameWithTooltip value={{}} id='test-id' />);
    expect(screen.queryByText(/\S/)).toBeNull();
  });

  it('renders empty string when value is undefined', () => {
    render(<NameWithTooltip value={undefined} id='test-id' />);
    expect(screen.queryByText(/\S/)).toBeNull();
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

it('shows the full linked experiment name on focus and preserves navigation', async () => {
  render(
    <MemoryRouter>
      <NameTooltip name='Full experiment name'>
        <Link to='/experiments/id'>Experiment</Link>
      </NameTooltip>
    </MemoryRouter>,
  );
  await userEvent.tab();
  expect(await screen.findByRole('tooltip')).toHaveTextContent('Full experiment name');
  expect(screen.getByRole('link', { name: 'Experiment' })).toHaveAttribute(
    'href',
    '/experiments/id',
  );
});

it('reveals the full resource name for touch input', async () => {
  render(
    <NameWithTooltip value={{ display_name: 'Short', name: 'full-resource-name' }} id='touch' />,
  );
  fireEvent.pointerDown(screen.getByText('Short'), { pointerType: 'touch' });
  expect(await screen.findByRole('tooltip')).toHaveTextContent('Name: full-resource-name');
});
