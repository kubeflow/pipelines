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
import { ResourceTable } from './ResourceTable';

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
