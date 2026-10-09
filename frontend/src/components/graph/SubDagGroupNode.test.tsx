// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import type { SubDagKind } from './Constants';
import SubDagGroupNode from './SubDagGroupNode';

it.each<[SubDagKind, string]>([
  ['Sub-DAG', 'Sub-DAG'],
  ['Loop', 'Loop'],
  ['Iteration', 'Iteration'],
  ['Condition', 'Conditional'],
])('identifies %s accessibly and displays its task name', (groupKind, label) => {
  render(
    <ReactFlowProvider>
      <SubDagGroupNode
        id='group'
        selected={false}
        data={{ label: 'Training', groupKind, expand: vi.fn() }}
      />
    </ReactFlowProvider>,
  );
  expect(screen.getByRole('img', { name: label })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Training' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Collapse Training' })).toHaveAttribute(
    'aria-expanded',
    'true',
  );
});

it.each([false, true])('offers the correct toggle action when collapsed=%s', (collapsed) => {
  const expand = vi.fn();
  render(
    <ReactFlowProvider>
      <SubDagGroupNode
        id='group'
        selected={false}
        data={{ label: 'Training', collapsed, state: PipelineTaskTaskState.SUCCEEDED, expand }}
      />
    </ReactFlowProvider>,
  );
  const toggle = screen.getByRole('button', {
    name: `${collapsed ? 'Expand' : 'Collapse'} Training`,
  });
  expect(toggle).toHaveAttribute('aria-expanded', String(!collapsed));
  fireEvent.click(toggle);
  expect(expand).toHaveBeenCalledExactlyOnceWith('group');
});

it('selects via the title without expanding, and toggles without selecting', () => {
  const expand = vi.fn();
  const select = vi.fn();
  render(
    <ReactFlowProvider>
      <div onClick={select}>
        <SubDagGroupNode
          id='group'
          selected={true}
          data={{ label: 'Training', state: PipelineTaskTaskState.SUCCEEDED, expand }}
        />
      </div>
    </ReactFlowProvider>,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Training' }));
  expect(select).toHaveBeenCalledTimes(1);
  expect(expand).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Collapse Training' }));
  expect(expand).toHaveBeenCalledWith('group');
  expect(select).toHaveBeenCalledTimes(1);
});

it('explains how to open a deferred group without presenting it as empty', () => {
  render(
    <ReactFlowProvider>
      <SubDagGroupNode
        id='group'
        selected={false}
        data={{
          label: 'Training',
          collapsed: true,
          expansionDeferred: 'Automatic expansion limit reached. Expand to load this group.',
          expand: vi.fn(),
        }}
      />
    </ReactFlowProvider>,
  );
  expect(screen.getByRole('button', { name: 'Expand Training' })).toHaveAccessibleDescription(
    'Automatic expansion limit reached. Expand to load this group.',
  );
  expect(screen.queryByText('No tasks in this scope')).not.toBeInTheDocument();
});
