// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { forceRenderStyles } from 'typestyle';
import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import SubDagGroupNode from './SubDagGroupNode';

it.each([
  ['Sub-DAG', 'Sub-DAG'],
  ['Loop', 'Loop'],
  ['Iteration', 'Iteration'],
  ['Condition', 'Conditional'],
])('shows %s as a left-aligned type label without an abstraction icon', (groupKind, label) => {
  render(
    <ReactFlowProvider>
      <SubDagGroupNode
        id='group'
        selected={false}
        data={{ label: 'Training', groupKind, expand: vi.fn() }}
      />
    </ReactFlowProvider>,
  );
  forceRenderStyles();
  expect(screen.getByTestId('subdag-kind')).toHaveTextContent(label);
  expect(screen.getByTestId('subdag-header').firstElementChild).toBe(
    screen.getByTestId('subdag-kind'),
  );
  expect(screen.queryByRole('img', { name: groupKind })).not.toBeInTheDocument();
  expect(screen.getByTestId('subdag-header')).toHaveClass('bg-white');
  expect(screen.getByTestId('subdag-header')).toHaveStyle({ height: '48px' });
  expect(screen.getByRole('button', { name: 'Training' })).toHaveClass('text-sm');
  expect(getComputedStyle(screen.getByTestId('subdag-box')).backgroundColor).toBe(
    'rgba(232, 240, 250, 0.18)',
  );
});

it('places the toggle directly before the full-height regular-node status icon', () => {
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
  const toggle = screen.getByRole('button', { name: 'Collapse Training' });
  expect(toggle.nextElementSibling).toBe(screen.getByTestId('subdag-status'));
  expect(screen.getByTestId('subdag-status')).toHaveClass('h-full');
  expect(screen.getByTestId('CheckCircleIcon')).toBeInTheDocument();
  const title = screen.getByRole('button', { name: 'Training' });
  expect(title).not.toHaveClass('focus:ring');
  expect(title).toHaveStyle({ fontWeight: '400', background: 'transparent' });
  fireEvent.click(title);
  expect(select).toHaveBeenCalledTimes(1);
  fireEvent.click(toggle);
  expect(expand).toHaveBeenCalledWith('group');
  expect(select).toHaveBeenCalledTimes(1);
});
