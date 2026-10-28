// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { ThemeProvider } from 'src/components/shell/ThemeProvider';
import userEvent from '@testing-library/user-event';
import { fireEvent, render, screen } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { PipelineTaskTaskState } from 'src/apisv2/run';
import type { SubDagKind } from './Constants';
import SubDagGroupNode from './SubDagGroupNode';

beforeEach(() => {
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() })),
  );
});
afterEach(() => vi.unstubAllGlobals());

it.each<[SubDagKind, string]>([
  ['Sub-DAG', 'Sub-DAG'],
  ['Loop', 'Loop'],
  ['Iteration', 'Iteration'],
  ['Condition', 'Conditional'],
])('identifies %s accessibly and displays its task name', (groupKind, label) => {
  render(
    <ThemeProvider>
      <ReactFlowProvider>
        <SubDagGroupNode
          id='group'
          selected={false}
          data={{ label: 'Training', groupKind, expand: vi.fn() }}
        />
      </ReactFlowProvider>
    </ThemeProvider>,
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
    <ThemeProvider>
      <ReactFlowProvider>
        <SubDagGroupNode
          id='group'
          selected={false}
          data={{ label: 'Training', collapsed, state: PipelineTaskTaskState.SUCCEEDED, expand }}
        />
      </ReactFlowProvider>
    </ThemeProvider>,
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
    <ThemeProvider>
      <ReactFlowProvider>
        <div onClick={select}>
          <SubDagGroupNode
            id='group'
            selected={true}
            data={{ label: 'Training', state: PipelineTaskTaskState.SUCCEEDED, expand }}
          />
        </div>
      </ReactFlowProvider>
    </ThemeProvider>,
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
    <ThemeProvider>
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
      </ReactFlowProvider>
    </ThemeProvider>,
  );
  expect(screen.getByRole('button', { name: 'Expand Training' })).toHaveAccessibleDescription(
    'Automatic expansion limit reached. Expand to load this group.',
  );
  expect(screen.queryByText('No tasks in this scope')).not.toBeInTheDocument();
});

it('reserves the status slot after the collapse control even without runtime status', () => {
  const group = (state?: PipelineTaskTaskState) => (
    <ThemeProvider>
      <ReactFlowProvider>
        <SubDagGroupNode
          id='group'
          selected={false}
          data={{ label: 'Training', state, expand: vi.fn() }}
        />
      </ReactFlowProvider>
    </ThemeProvider>
  );
  const { rerender } = render(group());
  const header = screen.getByTestId('subdag-header');
  const slots = () => Array.from(header.children);
  expect(slots()[3]).toBe(screen.getByTestId('subdag-status'));
  expect(slots()[2]).toBe(screen.getByRole('button', { name: 'Collapse Training' }));
  rerender(group(PipelineTaskTaskState.FAILED));
  expect(slots()[3]).toHaveAttribute('aria-label', 'Failed');
  expect(slots()[2]).toBe(screen.getByRole('button', { name: 'Collapse Training' }));
});

it('shows a warning icon for a running sub-DAG with a lifecycle message', () => {
  render(
    <ThemeProvider>
      <ReactFlowProvider>
        <SubDagGroupNode
          id='group'
          selected={false}
          data={{
            label: 'Training',
            state: PipelineTaskTaskState.RUNNING,
            lifecycleMessage: 'OOMKilled',
            expand: vi.fn(),
          }}
        />
      </ReactFlowProvider>
    </ThemeProvider>,
  );
  expect(document.querySelector('[data-state="WARNING"]')).toBeInTheDocument();
  expect(document.querySelector('[data-state="RUNNING"]')).not.toBeInTheDocument();
});

it('announces the warning in the sub-DAG node name', () => {
  render(
    <ThemeProvider>
      <ReactFlowProvider>
        <SubDagGroupNode
          id='group'
          selected={false}
          data={{
            label: 'Training',
            state: PipelineTaskTaskState.RUNNING,
            lifecycleMessage: 'OOMKilled',
            expand: vi.fn(),
          }}
        />
      </ReactFlowProvider>
    </ThemeProvider>,
  );
  expect(
    screen.getByRole('button', { name: 'Training, pod lifecycle warning' }),
  ).toBeInTheDocument();
});

it('shows the failed icon, not the warning, once the sub-DAG has failed', () => {
  render(
    <ThemeProvider>
      <ReactFlowProvider>
        <SubDagGroupNode
          id='group'
          selected={false}
          data={{
            label: 'Training',
            state: PipelineTaskTaskState.FAILED,
            lifecycleMessage: 'OOMKilled',
            expand: vi.fn(),
          }}
        />
      </ReactFlowProvider>
    </ThemeProvider>,
  );
  expect(document.querySelector('[data-state="FAILED"]')).toBeInTheDocument();
  expect(document.querySelector('[data-state="WARNING"]')).not.toBeInTheDocument();
});

it('exposes group status on keyboard focus without changing selection behavior', async () => {
  render(
    <ThemeProvider>
      <ReactFlowProvider>
        <SubDagGroupNode
          id='group'
          selected={false}
          data={{ label: 'Training', state: PipelineTaskTaskState.SUCCEEDED, expand: vi.fn() }}
        />
      </ReactFlowProvider>
    </ThemeProvider>,
  );
  const label = screen.getByRole('button', { name: 'Training' });
  expect(label).toHaveAccessibleDescription('Succeeded');
  await userEvent.tab();
  expect(await screen.findByRole('tooltip')).toHaveTextContent('Training · Succeeded');
});
