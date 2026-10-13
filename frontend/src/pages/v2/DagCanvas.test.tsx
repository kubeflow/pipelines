// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CommonTestWrapper } from 'src/TestWrapper';
import { mockResizeObserver } from 'src/TestUtils';
import { manySubDagsSpec, nestedArtifactSpec } from 'src/data/test/groupedFlow';
import { convertSubDagToFlowElements } from 'src/lib/v2/StaticFlow';
import DagCanvas, { DagCanvasProps } from './DagCanvas';

const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);

function props(): DagCanvasProps {
  return {
    elements: resolve(['root']),
    layers: ['root'],
    getSubDagElements: resolve,
    onElementClick: vi.fn(),
  };
}

beforeEach(() => mockResizeObserver());

it('shows nested tasks immediately and collapses in place without navigating or selecting', () => {
  const options = props();
  render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  expect(screen.getByText('Train model')).toBeInTheDocument();
  const collapse = screen.getByRole('button', { name: 'Collapse Training pipeline' });
  expect(collapse).toHaveAttribute('aria-expanded', 'true');
  expect(within(collapse).getByTestId('UnfoldLessIcon')).toBeInTheDocument();
  fireEvent.click(collapse);
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  expect(screen.getByText('Deploy model')).toBeInTheDocument();
  expect(options.onElementClick).not.toHaveBeenCalled();
  const expand = screen.getByRole('button', { name: 'Expand Training pipeline' });
  expect(expand).toHaveAttribute('aria-expanded', 'false');
  expect(within(expand).getByTestId('UnfoldMoreIcon')).toBeInTheDocument();
  fireEvent.click(expand);
  expect(within(collapse).getByTestId('UnfoldLessIcon')).toBeInTheDocument();
  expect(screen.getByText('Train model')).toBeInTheDocument();
});

it('passes the local task ID and full scope to the details panel', () => {
  const options = props();
  render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  fireEvent.click(screen.getByText('Train model'));
  expect(options.onElementClick).toHaveBeenCalledWith(expect.anything(), {
    element: expect.objectContaining({ id: 'task.train' }),
    layers: ['root', 'workflow', 'fit'],
  });
});

it('preserves nested collapse choices across refreshes and parent toggles', () => {
  const options = props();
  const { rerender } = render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Collapse Training and evaluation' }));
  fireEvent.click(screen.getByRole('button', { name: 'Collapse Training pipeline' }));
  rerender(
    <CommonTestWrapper>
      <DagCanvas {...options} elements={resolve(['root'])} />
    </CommonTestWrapper>,
  );
  expect(screen.queryByText('Prepare data')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Expand Training pipeline' }));
  expect(screen.getByText('Prepare data')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Expand Training and evaluation' })).toHaveAttribute(
    'aria-expanded',
    'false',
  );
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
});

it('defaults to 20 upstream groups and preserves manual and bulk overrides through refresh', () => {
  let spec = manySubDagsSpec();
  const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
  const options = { ...props(), elements: resolve(['root']), getSubDagElements: resolve };
  const canvas = () => (
    <CommonTestWrapper>
      <DagCanvas {...options} elements={resolve(['root'])} />
    </CommonTestWrapper>
  );
  const { rerender } = render(canvas());
  expect(screen.getAllByRole('button', { name: /^Collapse Group / })).toHaveLength(20);
  expect(screen.getByRole('button', { name: 'Expand Group 20' })).toHaveAccessibleDescription(
    'Only the first 20 sub-DAGs are expanded by default. Expand this sub-DAG or use Expand all to show more.',
  );
  fireEvent.click(screen.getByRole('button', { name: 'Expand Group 24' }));
  rerender(canvas());
  expect(screen.getByRole('button', { name: 'Collapse Group 24' })).toBeInTheDocument();
  expect(screen.getAllByRole('button', { name: /^Collapse Group / })).toHaveLength(21);

  fireEvent.click(screen.getByRole('button', { name: 'Expand all' }));
  expect(screen.getAllByRole('button', { name: /^Collapse Group / })).toHaveLength(25);
  spec = manySubDagsSpec(26);
  rerender(canvas());
  expect(screen.getAllByRole('button', { name: /^Collapse Group / })).toHaveLength(26);
  fireEvent.click(screen.getByRole('button', { name: 'Collapse Group 24' }));
  rerender(canvas());
  expect(screen.getByRole('button', { name: 'Expand Group 24' })).toBeInTheDocument();

  fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));
  spec = manySubDagsSpec(27);
  rerender(canvas());
  expect(screen.queryAllByRole('button', { name: /^Collapse Group / })).toHaveLength(0);
  expect(screen.getAllByRole('button', { name: /^Expand Group / })).toHaveLength(27);
  fireEvent.click(screen.getByRole('button', { name: 'Expand Group 24' }));
  expect(screen.getAllByRole('button', { name: /^Collapse Group / })).toHaveLength(1);
});

it('shows 20 expanded and five collapsed groups in the 25-sub-DAG visual example', () => {
  const spec = manySubDagsSpec(25, 5);
  const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
  render(
    <CommonTestWrapper>
      <DagCanvas {...props()} elements={resolve(['root'])} getSubDagElements={resolve} />
    </CommonTestWrapper>,
  );
  for (let index = 0; index < 25; index++) {
    expect(
      screen.getByRole('button', {
        name: `${index < 20 ? 'Collapse' : 'Expand'} Group ${index}`,
      }),
    ).toHaveAttribute('aria-expanded', String(index < 20));
  }
  expect(screen.getAllByRole('button', { name: /^Collapse Group / })).toHaveLength(20);
  expect(screen.getAllByRole('button', { name: /^Expand Group / })).toHaveLength(5);
  expect(screen.getAllByText('Step', { exact: true })).toHaveLength(20);

  fireEvent.click(screen.getByRole('button', { name: 'Expand all', exact: true }));
  expect(screen.getAllByRole('button', { name: /^Collapse Group / })).toHaveLength(25);
  expect(screen.queryAllByRole('button', { name: /^Expand Group / })).toHaveLength(0);
  expect(screen.getAllByText('Step', { exact: true })).toHaveLength(25);

  fireEvent.click(screen.getByRole('button', { name: 'Collapse all', exact: true }));
  expect(screen.getAllByRole('button', { name: /^Expand Group / })).toHaveLength(25);
  expect(screen.queryAllByText('Step', { exact: true })).toHaveLength(0);
});

it('opens deep-linked ancestors inline once, without undoing later collapse choices', () => {
  const options = props();
  const { rerender } = render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));
  const selectedElement = {
    element: resolve(['root', 'workflow', 'fit']).find((node) => node.id === 'task.train')!,
    layers: ['root', 'workflow', 'fit'],
  };
  const canvas = () => (
    <CommonTestWrapper>
      <DagCanvas {...options} selectedElement={selectedElement} focusNodeId='task.train' />
    </CommonTestWrapper>
  );
  rerender(canvas());
  expect(screen.getByText('Train model')).toBeInTheDocument();
  expect(screen.getByText('Deploy model')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));
  rerender(canvas());
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
});

it('places expand then collapse above native zoom controls and toggles all descendants', () => {
  const options = props();
  const { rerender } = render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  const expandAll = screen.getByRole('button', { name: 'Expand all' });
  const collapseAll = screen.getByRole('button', { name: 'Collapse all' });
  expect(within(expandAll).getByTestId('UnfoldMoreIcon')).toBeInTheDocument();
  expect(within(collapseAll).getByTestId('UnfoldLessIcon')).toBeInTheDocument();
  const toolbar = screen.getByRole('group', { name: 'Graph controls' });
  const buttons = within(toolbar).getAllByRole('button');
  expect(buttons[0]).toBe(expandAll);
  expect(buttons[1]).toBe(collapseAll);
  expect(buttons[2]).toHaveAccessibleName('Zoom in');
  expect(buttons[3]).toHaveAccessibleName('Zoom out');
  expect(buttons[4]).toHaveAccessibleName('Fit view');
  expect(buttons[5]).toHaveAccessibleName('Lock graph');
  expect(buttons).toHaveLength(6);
  expect(screen.queryByRole('button', { name: 'Render subdags' })).not.toBeInTheDocument();
  expect(screen.queryByText('Layers')).not.toBeInTheDocument();
  fireEvent.click(collapseAll);
  expect(screen.queryByText('Prepare data')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Expand Training pipeline' }));
  expect(
    screen.getByRole('button', { name: 'Expand Training and evaluation' }),
  ).toBeInTheDocument();
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  fireEvent.click(expandAll);
  expect(screen.getByText('Train model')).toBeInTheDocument();
  expect(
    screen.getByRole('button', { name: 'Collapse Training and evaluation' }),
  ).toBeInTheDocument();
  fireEvent.click(collapseAll);
  rerender(
    <CommonTestWrapper>
      <DagCanvas {...options} elements={resolve(['root'])} />
    </CommonTestWrapper>,
  );
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  expect(options.onElementClick).not.toHaveBeenCalled();
});

it.each([
  [13, 39],
  [16, 48],
])('renders node-height group headers with a %spx root font', (rootFontSize, expectedHeight) => {
  const previous = document.documentElement.style.fontSize;
  document.documentElement.style.fontSize = `${rootFontSize}px`;
  try {
    render(
      <CommonTestWrapper>
        <DagCanvas {...props()} />
      </CommonTestWrapper>,
    );
    expect(screen.getByText('Train model')).toBeInTheDocument();
    for (const header of screen.getAllByTestId('subdag-header')) {
      expect(header).toHaveStyle({ height: `${expectedHeight}px` });
    }
    fireEvent.click(screen.getByRole('button', { name: 'Collapse Training pipeline' }));
    expect(screen.queryByText('Train model')).not.toBeInTheDocument();
    expect(screen.getByTestId('rf__node-task.workflow')).toHaveStyle({
      height: `${expectedHeight}px`,
    });
    expect(screen.getByText('Deploy model')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Expand Training pipeline' }));
    expect(screen.getByText('Train model')).toBeInTheDocument();
  } finally {
    document.documentElement.style.fontSize = previous;
  }
});

it('locks dragging and node selection without disabling viewport controls', () => {
  const options = props();
  render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Lock graph' }));

  fireEvent.click(screen.getByText('Train model'));
  expect(options.onElementClick).not.toHaveBeenCalled();
  expect(screen.getByRole('button', { name: 'Zoom in' })).not.toBeDisabled();
  fireEvent.click(screen.getByRole('button', { name: 'Unlock graph' }));
  fireEvent.click(screen.getByText('Train model'));
  expect(options.onElementClick).toHaveBeenCalledTimes(1);
});

it('supports keyboard collapse and expansion', async () => {
  render(
    <CommonTestWrapper>
      <DagCanvas {...props()} />
    </CommonTestWrapper>,
  );
  screen.getByRole('button', { name: 'Collapse Training pipeline' }).focus();
  await userEvent.keyboard('{Enter}');
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  await userEvent.keyboard(' ');
  expect(screen.getByText('Train model')).toBeInTheDocument();
});
