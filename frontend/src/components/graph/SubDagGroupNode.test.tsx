// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { forceRenderStyles } from 'typestyle';
import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import SubDagGroupNode from './SubDagGroupNode';

it.each([
  ['Sub-DAG', 'Sub-DAG', 'LayersIcon'],
  ['Loop', 'Loop', 'RepeatIcon'],
  ['Iteration', 'Iteration', 'RepeatOneIcon'],
  ['Condition', 'Conditional', 'ConditionIcon'],
])('shows %s as a grey category icon without type text', (groupKind, label, iconId) => {
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
  const icon = screen.getByRole('img', { name: label });
  expect(icon).toBe(screen.getByTestId(iconId));
  expect(icon).toHaveClass('text-mui-grey-600');
  expect(screen.queryByTestId('subdag-kind')).not.toBeInTheDocument();
  expect(screen.queryByTestId('subdag-share-icon')).not.toBeInTheDocument();
  expect(screen.getByTestId('subdag-header').firstElementChild).toContainElement(icon);
  expect(screen.getByTestId('subdag-header')).toHaveClass('bg-white');
  expect(screen.getByTestId('subdag-header')).toHaveStyle({ height: '48px' });
  expect(screen.getByRole('button', { name: 'Training' })).toHaveClass('text-sm');
  expect(getComputedStyle(screen.getByTestId('subdag-box')).backgroundColor).toBe(
    'rgba(219, 234, 254, 0.28)',
  );
});

it.each([false, true])(
  'lets the outer group clip status corners when collapsed=%s',
  (collapsed) => {
    render(
      <ReactFlowProvider>
        <SubDagGroupNode
          id='group'
          selected={false}
          data={{
            label: 'Training',
            collapsed,
            state: PipelineTaskTaskState.SUCCEEDED,
            expand: vi.fn(),
          }}
        />
      </ReactFlowProvider>,
    );
    forceRenderStyles();
    expect(screen.getByTestId('subdag-box')).toHaveClass('shadow-lg');
    const cell = screen.getByTestId('subdag-status').firstElementChild!;
    expect(parseFloat(getComputedStyle(cell).borderBottomRightRadius)).toBe(0);
    expect(parseFloat(getComputedStyle(cell).borderTopRightRadius)).toBe(0);
    expect(screen.getByTestId('subdag-header')).toHaveClass('bg-white');
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
  expect(title).not.toHaveClass('nodrag');
  expect(toggle).toHaveClass('nodrag');
  expect(title).toHaveStyle({ fontWeight: '400', background: 'transparent' });
  fireEvent.click(title);
  expect(select).toHaveBeenCalledTimes(1);
  fireEvent.click(toggle);
  expect(expand).toHaveBeenCalledWith('group');
  expect(select).toHaveBeenCalledTimes(1);
});
