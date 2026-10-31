// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CommonTestWrapper } from 'src/TestWrapper';
import { mockResizeObserver } from 'src/TestUtils';
import { deeplyNestedSpec, manySubDagsSpec, nestedArtifactSpec } from 'src/data/test/groupedFlow';
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

function renderManyGroups(count = 11) {
  let spec = manySubDagsSpec(count);
  const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
  const canvas = () => (
    <CommonTestWrapper>
      <DagCanvas {...props()} elements={resolve(['root'])} getSubDagElements={resolve} />
    </CommonTestWrapper>
  );
  const { rerender } = render(canvas());
  return (count: number) => {
    spec = manySubDagsSpec(count);
    rerender(canvas());
  };
}

beforeEach(() => {
  mockResizeObserver();
  localStorage.clear();
});

it.each(['Expand all', 'Collapse all'])(
  'restores %s and individual choices after remount',
  (action) => {
    const canvas = () => (
      <CommonTestWrapper>
        <DagCanvas {...props()} graphId='run:persisted' />
      </CommonTestWrapper>
    );
    const first = render(canvas());
    fireEvent.click(screen.getByRole('button', { name: action, exact: true }));
    if (action === 'Collapse all')
      fireEvent.click(screen.getByRole('button', { name: 'Expand Training pipeline' }));
    else fireEvent.click(screen.getByRole('button', { name: 'Collapse Training and evaluation' }));
    first.unmount();
    render(canvas());
    expect(screen.getByText('Prepare data')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Expand Training and evaluation' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  },
);

it('isolates saved graph identities without overwriting the previous graph', () => {
  const canvas = (graphId: string) => (
    <CommonTestWrapper>
      <DagCanvas {...props()} graphId={graphId} />
    </CommonTestWrapper>
  );
  const { rerender } = render(canvas('pipeline-version:1'));
  fireEvent.click(screen.getByRole('button', { name: 'Collapse Training pipeline' }));
  rerender(canvas('pipeline-version:2'));
  expect(screen.getByText('Train model')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Collapse Training and evaluation' }));
  rerender(canvas('pipeline-version:1'));
  expect(screen.getByRole('button', { name: 'Expand Training pipeline' })).toBeInTheDocument();
  rerender(canvas('pipeline-version:2'));
  expect(screen.getByText('Prepare data')).toBeInTheDocument();
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
});

it('keeps interaction working if saving preferences fails', () => {
  const save = vi.spyOn(localStorage, 'setItem').mockImplementation(() => {
    throw new Error('Blocked');
  });
  try {
    render(
      <CommonTestWrapper>
        <DagCanvas {...props()} graphId='run:blocked' />
      </CommonTestWrapper>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Collapse all', exact: true }));
    expect(screen.queryByText('Train model')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Expand all', exact: true }));
    expect(screen.getByText('Train model')).toBeInTheDocument();
  } finally {
    save.mockRestore();
  }
});

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

it('defaults to 10 upstream groups and preserves manual expansion through refresh', () => {
  const refresh = renderManyGroups();
  // Direct accessible-label queries avoid jsdom style walks over every node under coverage.
  expect(screen.getAllByLabelText(/^Collapse Group /, { selector: 'button' })).toHaveLength(10);
  expect(
    screen.getByLabelText('Expand Group 10', { selector: 'button' }),
  ).toHaveAccessibleDescription(
    'Only the first 10 sub-DAGs are expanded by default. Expand this sub-DAG or use Expand all to show more.',
  );
  fireEvent.click(screen.getByLabelText('Expand Group 10', { selector: 'button' }));
  refresh(11);
  expect(screen.getByLabelText('Collapse Group 10', { selector: 'button' })).toBeInTheDocument();
  expect(screen.getAllByLabelText(/^Collapse Group /, { selector: 'button' })).toHaveLength(11);
});

it('preserves bulk expansion and manual exceptions when new groups arrive', () => {
  const refresh = renderManyGroups();
  fireEvent.click(screen.getByLabelText('Expand all', { selector: 'button' }));
  expect(screen.getAllByLabelText(/^Collapse Group /, { selector: 'button' })).toHaveLength(11);
  refresh(12);
  expect(screen.getAllByLabelText(/^Collapse Group /, { selector: 'button' })).toHaveLength(12);
  fireEvent.click(screen.getByLabelText('Collapse Group 10', { selector: 'button' }));
  refresh(12);
  expect(screen.getByLabelText('Expand Group 10', { selector: 'button' })).toBeInTheDocument();
});

it('preserves bulk collapse when new groups arrive and allows an individual expansion', () => {
  const refresh = renderManyGroups();
  fireEvent.click(screen.getByLabelText('Collapse all', { selector: 'button' }));
  refresh(12);
  expect(screen.queryAllByLabelText(/^Collapse Group /, { selector: 'button' })).toHaveLength(0);
  expect(screen.getAllByLabelText(/^Expand Group /, { selector: 'button' })).toHaveLength(12);
  fireEvent.click(screen.getByLabelText('Expand Group 10', { selector: 'button' }));
  expect(screen.getAllByLabelText(/^Collapse Group /, { selector: 'button' })).toHaveLength(1);
});

it('shows 10 expanded and fifteen collapsed groups in the 25-sub-DAG visual example', () => {
  const spec = manySubDagsSpec(25, 5);
  const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
  render(
    <CommonTestWrapper>
      <DagCanvas {...props()} elements={resolve(['root'])} getSubDagElements={resolve} />
    </CommonTestWrapper>,
  );
  for (let index = 0; index < 25; index++) {
    expect(
      screen.getByLabelText(`${index < 10 ? 'Collapse' : 'Expand'} Group ${index}`, {
        selector: 'button',
      }),
    ).toHaveAttribute('aria-expanded', String(index < 10));
  }
  expect(screen.getAllByLabelText(/^Collapse Group /, { selector: 'button' })).toHaveLength(10);
  expect(screen.getAllByLabelText(/^Expand Group /, { selector: 'button' })).toHaveLength(15);
  expect(screen.getAllByText('Step', { exact: true })).toHaveLength(10);

  fireEvent.click(screen.getByLabelText('Expand all', { selector: 'button' }));
  expect(screen.getAllByLabelText(/^Collapse Group /, { selector: 'button' })).toHaveLength(25);
  expect(screen.queryAllByLabelText(/^Expand Group /, { selector: 'button' })).toHaveLength(0);
  expect(screen.getAllByText('Step', { exact: true })).toHaveLength(25);

  fireEvent.click(screen.getByLabelText('Collapse all', { selector: 'button' }));
  expect(screen.getAllByLabelText(/^Expand Group /, { selector: 'button' })).toHaveLength(25);
  expect(screen.queryAllByText('Step', { exact: true })).toHaveLength(0);
});

it('reveals all 25 nested levels on demand and selects the deepest component', () => {
  const spec = deeplyNestedSpec();
  const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
  const options = { ...props(), getSubDagElements: resolve };
  render(
    <CommonTestWrapper>
      <DagCanvas {...options} elements={resolve(['root'])} />
    </CommonTestWrapper>,
  );
  expect(screen.getAllByText(/^Component \d{3}$/)).toHaveLength(20);
  expect(screen.getAllByLabelText(/^Collapse Level /, { selector: 'button' })).toHaveLength(10);
  expect(screen.getByLabelText('Expand Level 11', { selector: 'button' })).toBeInTheDocument();
  expect(screen.queryByText('Component 050', { exact: true })).not.toBeInTheDocument();
  fireEvent.click(screen.getByLabelText('Expand all', { selector: 'button' }));
  expect(screen.getAllByText(/^Component \d{3}$/)).toHaveLength(50);
  expect(screen.getAllByLabelText(/^Collapse Level /, { selector: 'button' })).toHaveLength(25);
  fireEvent.click(screen.getByText('Component 050', { exact: true }));
  expect(options.onElementClick).toHaveBeenCalledWith(expect.anything(), {
    element: expect.objectContaining({ id: 'task.component-50' }),
    layers: ['root', ...Array.from({ length: 25 }, (_, index) => `level-${index + 1}`)],
  });
});

it('retains individual collapse choices in deeply nested graphs through refresh', () => {
  const spec = deeplyNestedSpec();
  const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
  const canvas = () => (
    <CommonTestWrapper>
      <DagCanvas {...props()} elements={resolve(['root'])} getSubDagElements={resolve} />
    </CommonTestWrapper>
  );
  const { rerender } = render(canvas());
  fireEvent.click(screen.getByLabelText('Collapse Level 10', { selector: 'button' }));
  rerender(canvas());
  expect(screen.getAllByText(/^Component \d{3}$/)).toHaveLength(18);
  expect(screen.getByLabelText('Expand Level 10', { selector: 'button' })).toBeInTheDocument();
});

it('collapses all 25 nested levels and keeps descendants collapsed when reopening a parent', () => {
  const spec = deeplyNestedSpec();
  const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
  render(
    <CommonTestWrapper>
      <DagCanvas {...props()} elements={resolve(['root'])} getSubDagElements={resolve} />
    </CommonTestWrapper>,
  );
  fireEvent.click(screen.getByLabelText('Expand all', { selector: 'button' }));
  expect(screen.getAllByText(/^Component \d{3}$/)).toHaveLength(50);
  fireEvent.click(screen.getByLabelText('Collapse all', { selector: 'button' }));
  expect(screen.queryAllByText(/^Component \d{3}$/)).toHaveLength(0);
  fireEvent.click(screen.getByLabelText('Expand Level 01', { selector: 'button' }));
  expect(screen.getAllByText(/^Component \d{3}$/)).toHaveLength(2);
  expect(screen.getByLabelText('Expand Level 02', { selector: 'button' })).toBeInTheDocument();
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
  expect(buttons[5]).toHaveAccessibleName('Unlock graph');
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

it('starts locked while keeping node details and viewport controls available', () => {
  const options = props();
  render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  expect(screen.getByRole('button', { name: 'Unlock graph' })).toHaveAttribute(
    'aria-pressed',
    'true',
  );
  fireEvent.click(screen.getByText('Train model'));
  expect(options.onElementClick).toHaveBeenCalledTimes(1);
  expect(screen.getByRole('button', { name: 'Zoom in' })).not.toBeDisabled();
  fireEvent.click(screen.getByRole('button', { name: 'Unlock graph' }));
  expect(screen.getByRole('button', { name: 'Lock graph' })).toHaveAttribute(
    'aria-pressed',
    'false',
  );
  fireEvent.click(screen.getByText('Train model'));
  expect(options.onElementClick).toHaveBeenCalledTimes(2);
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
