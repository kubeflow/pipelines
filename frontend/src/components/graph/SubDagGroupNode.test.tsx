// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import SubDagGroupNode from './SubDagGroupNode';

it.each(['Sub-DAG', 'Loop', 'Iteration', 'Condition'])(
  'uses a blue %s icon instead of a category line',
  (groupKind) => {
    render(
      <ReactFlowProvider>
        <SubDagGroupNode
          id='group'
          selected={false}
          data={{ label: 'Training', groupKind, expand: vi.fn() }}
        />
      </ReactFlowProvider>,
    );
    expect(screen.getByRole('img', { name: groupKind })).toHaveClass('text-mui-blue-600');
    expect(screen.getByTestId('subdag-header')).toHaveClass('bg-white');
    expect(screen.getByTestId('subdag-header')).toHaveStyle({ height: '48px' });
    expect(screen.getByRole('button', { name: 'Training' })).toHaveClass('text-sm');
    expect(
      [...screen.getByTestId('subdag-header').querySelectorAll('span')].some(
        (span) => span.textContent === groupKind,
      ),
    ).toBe(false);
    expect(getComputedStyle(screen.getByTestId('subdag-box')).backgroundColor).toBe(
      'rgba(0, 0, 0, 0)',
    );
  },
);

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
