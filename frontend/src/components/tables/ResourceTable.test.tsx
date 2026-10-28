/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useState } from 'react';
import { vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CustomTable, { ExpandState } from '../CustomTable';
import { LocalStorage } from '../../lib/LocalStorage';
import { ResourceTable } from './ResourceTable';

beforeEach(() => localStorage.clear());

function Table({
  single = false,
  limit = false,
  onOpenRow,
}: {
  single?: boolean;
  limit?: boolean;
  onOpenRow?: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <>
      <output aria-label='Selected resources'>{selected.join(',')}</output>
      <CustomTable
        columns={[{ label: 'Pipeline', sortKey: 'name' }]}
        rows={[
          {
            id: 'one',
            otherFields: ['Training'],
            expandState: expanded ? ExpandState.EXPANDED : ExpandState.COLLAPSED,
          },
          { id: 'two', otherFields: ['Evaluation'], expandState: ExpandState.NONE },
        ]}
        filterLabel='Filter pipelines by name'
        selectedIds={selected}
        updateSelection={setSelected}
        useRadioButtons={single}
        disableAdditionalSelection={limit && selected.length === 1}
        toggleExpansion={() => setExpanded(!expanded)}
        getExpandComponent={(index) => <p>Versions for pipeline {index + 1}</p>}
        reload={async () => ''}
        renderTable={(table) => (
          <ResourceTable
            table={table}
            onOpenRow={onOpenRow}
            label='Pipelines'
            singular='pipeline'
            plural='pipelines'
          />
        )}
      />
    </>
  );
}

it('expands a resource by keyboard without selecting it and preserves selection when closing', async () => {
  render(<Table />);
  const user = userEvent.setup();
  await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
  const expand = screen.getByRole('button', { name: 'Expand pipeline Training' });
  expand.focus();
  await user.keyboard('{Enter}');
  const collapse = screen.getByRole('button', { name: 'Collapse pipeline Training' });
  expect(collapse).toHaveAttribute('aria-expanded', 'true');
  expect(screen.getByText('Versions for pipeline 1')).toBeVisible();
  expect(document.getElementById(collapse.getAttribute('aria-controls')!)).toContainElement(
    screen.getByText('Versions for pipeline 1'),
  );
  expect(screen.getByLabelText('Selected resources')).toBeEmptyDOMElement();
  expect(
    screen.queryByRole('button', { name: 'Expand pipeline Evaluation' }),
  ).not.toBeInTheDocument();
  await user.click(screen.getByRole('checkbox', { name: 'Select pipeline Training' }));
  await user.click(collapse);
  expect(screen.queryByText('Versions for pipeline 1')).not.toBeInTheDocument();
  expect(screen.getByLabelText('Selected resources')).toHaveTextContent('one');
});

it('uses a single selection for resource-picker radio controls', async () => {
  render(<Table single />);
  await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole('radio', { name: 'Select pipeline Training' }));
  await userEvent.click(screen.getByRole('radio', { name: 'Select pipeline Evaluation' }));
  expect(screen.getByRole('radio', { name: 'Select pipeline Evaluation' })).toBeChecked();
  expect(screen.getByRole('radio', { name: 'Select pipeline Training' })).not.toBeChecked();
  expect(screen.getByLabelText('Selected resources')).toHaveTextContent('two');
});

it('honors selection limits for both checkboxes and row clicks while allowing deselection', async () => {
  render(<Table limit />);
  await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
  await userEvent.click(screen.getByRole('checkbox', { name: 'Select pipeline Training' }));
  expect(screen.getByRole('checkbox', { name: 'Select pipeline Evaluation' })).toHaveAttribute(
    'aria-disabled',
    'true',
  );
  await userEvent.click(screen.getByRole('checkbox', { name: 'Select pipeline Evaluation' }));
  expect(screen.getByLabelText('Selected resources')).toHaveTextContent('one');
  expect(
    screen.getByRole('checkbox', { name: 'Select all pipelines on this page' }),
  ).toHaveAttribute('aria-disabled', 'true');
  fireEvent.click(screen.getAllByTestId('table-row')[1]);
  expect(screen.getByLabelText('Selected resources')).toHaveTextContent('one');
  await userEvent.click(screen.getByRole('checkbox', { name: 'Select pipeline Training' }));
  expect(screen.getByRole('checkbox', { name: 'Select pipeline Evaluation' })).not.toHaveAttribute(
    'aria-disabled',
    'true',
  );
  fireEvent.click(screen.getAllByTestId('table-row')[1]);
  expect(screen.getByLabelText('Selected resources')).toHaveTextContent('two');
});

it('selects and deselects a selectable row even when a navigation callback is supplied', async () => {
  const onOpenRow = vi.fn();
  render(<Table onOpenRow={onOpenRow} />);
  await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
  const row = screen.getAllByTestId('table-row')[0];
  fireEvent.click(row);
  expect(screen.getByLabelText('Selected resources')).toHaveTextContent('one');
  fireEvent.click(row);
  expect(screen.getByLabelText('Selected resources')).toBeEmptyDOMElement();
  expect(onOpenRow).not.toHaveBeenCalled();
});

it('uses column sizing without adding selection controls to read-only collections', async () => {
  render(
    <CustomTable
      columns={[{ label: 'Artifact' }, { label: 'Created', width: 190 }]}
      rows={[{ id: 'dataset', otherFields: ['Training data', 'Today'] }]}
      selectedIds={[]}
      disableSelection={true}
      reload={async () => ''}
      renderTable={(table) => <ResourceTable table={table} label='Artifacts' />}
    />,
  );
  const table = screen.getByRole('table', { name: 'Artifacts' });
  await waitFor(() => expect(table).toHaveAttribute('aria-busy', 'false'));
  expect(table).toHaveStyle({ tableLayout: 'fixed' });
  expect(table.querySelectorAll('col')).toHaveLength(2);
  expect(table.querySelectorAll('col')[1]).toHaveStyle({ width: '190px' });
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
  expect(screen.getByTestId('table-row')).toHaveAttribute('data-interactive', 'false');
});

it('resizes columns by keyboard, remembers widths, and resets to the responsive layout', async () => {
  const bounds = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    width: 240,
    height: 40,
    top: 0,
    left: 0,
    right: 240,
    bottom: 40,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  });
  try {
    const first = render(<Table />);
    await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
    const resize = screen.getByRole('separator', { name: 'Resize Pipeline column' });
    fireEvent.keyDown(resize, { key: 'ArrowRight' });
    expect(resize).toHaveAttribute('aria-valuenow', '256');
    expect(screen.getByRole('table').querySelectorAll('col')[2]).toHaveStyle({ width: '256px' });
    expect(screen.getByRole('table').querySelectorAll('col')[0]).toHaveStyle({ width: '40px' });
    expect(screen.getByRole('table').querySelectorAll('col')[1]).toHaveStyle({ width: '36px' });
    expect(screen.getByLabelText('Selected resources')).toBeEmptyDOMElement();
    first.unmount();
    render(<Table />);
    expect(screen.getByRole('separator', { name: 'Resize Pipeline column' })).toHaveAttribute(
      'aria-valuenow',
      '256',
    );
    fireEvent.click(screen.getByRole('button', { name: 'Reset column widths' }));
    expect(screen.queryByRole('button', { name: 'Reset column widths' })).not.toBeInTheDocument();
    expect(
      LocalStorage.getTableColumnWidths(JSON.stringify(['Pipelines', ['name']])),
    ).toBeUndefined();
  } finally {
    bounds.mockRestore();
  }
});

it('keeps the resized target exact and fills surplus with a different column, including after reload', async () => {
  const bounds = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    width: 240,
    height: 40,
    top: 0,
    left: 0,
    right: 240,
    bottom: 40,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  });
  const content = () => (
    <CustomTable
      columns={[{ label: 'Name' }, { label: 'URI' }, { label: 'Created', width: 190 }]}
      rows={[{ id: 'dataset', otherFields: ['Training', 's3://data', 'Today'] }]}
      selectedIds={[]}
      disableSelection
      reload={async () => ''}
      renderTable={(table) => <ResourceTable table={table} label='Resize examples' />}
    />
  );
  try {
    const view = render(content());
    await waitFor(() => expect(screen.getByRole('table')).toHaveAttribute('aria-busy', 'false'));
    fireEvent.keyDown(screen.getByRole('separator', { name: 'Resize Name column' }), {
      key: 'ArrowLeft',
    });
    const table = screen.getByRole('table');
    expect(table).toHaveStyle({ width: '100%', minWidth: '544px' });
    expect(table.querySelectorAll('col')[0]).toHaveStyle({ width: '224px' });
    expect(table.querySelectorAll('col')[1].style.width).toBe('');
    view.unmount();
    render(content());
    expect(screen.getByRole('table').querySelectorAll('col')[0]).toHaveStyle({ width: '224px' });
    expect(screen.getByRole('table').querySelectorAll('col')[1].style.width).toBe('');
    fireEvent.keyDown(screen.getByRole('separator', { name: 'Resize URI column' }), {
      key: 'ArrowLeft',
    });
    expect(screen.getByRole('table').querySelectorAll('col')[1]).toHaveStyle({ width: '224px' });
    expect(screen.getByRole('table').querySelectorAll('col')[0].style.width).toBe('');
  } finally {
    bounds.mockRestore();
  }
});
