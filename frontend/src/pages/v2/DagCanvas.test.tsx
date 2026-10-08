// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CommonTestWrapper } from 'src/TestWrapper';
import { mockResizeObserver } from 'src/TestUtils';
import { nestedArtifactSpec } from 'src/data/test/groupedFlow';
import { convertSubDagToFlowElements, GraphExpansionLimitError } from 'src/lib/v2/StaticFlow';
import DagCanvas, { DagCanvasProps } from './DagCanvas';

const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);

function props(): DagCanvasProps {
  return {
    elements: resolve(['root']),
    layers: ['root'],
    getSubDagElements: resolve,
    onLayersUpdate: vi.fn(),
    onElementClick: vi.fn(),
    setFlowElements: vi.fn(),
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
  expect(within(collapse).getByTestId('ExpandLessIcon')).toBeInTheDocument();
  fireEvent.click(collapse);
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  expect(screen.getByText('Deploy model')).toBeInTheDocument();
  expect(options.onLayersUpdate).not.toHaveBeenCalled();
  expect(options.onElementClick).not.toHaveBeenCalled();
  const expand = screen.getByRole('button', { name: 'Expand Training pipeline' });
  expect(expand).toHaveAttribute('aria-expanded', 'false');
  expect(within(expand).getByTestId('ExpandMoreIcon')).toBeInTheDocument();
  fireEvent.click(expand);
  expect(within(collapse).getByTestId('ExpandLessIcon')).toBeInTheDocument();
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
  expect(options.onElementClick).toHaveBeenCalledWith(
    expect.anything(),
    expect.objectContaining({ id: 'task.train' }),
    ['root', 'workflow', 'fit'],
  );
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

it('explicitly expands a deferred group and preserves that choice through refresh', () => {
  const getSubDagElements = vi.fn((layers: string[], maxNodes?: number) => {
    if (maxNodes !== Infinity) throw new GraphExpansionLimitError(600);
    return resolve(layers);
  });
  const options = { ...props(), getSubDagElements };
  const { rerender } = render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  expect(
    screen.getByRole('button', { name: 'Expand Training pipeline' }),
  ).toHaveAccessibleDescription('600 nodes · expand to load');
  fireEvent.click(screen.getByRole('button', { name: 'Expand all' }));
  expect(screen.queryByText('Prepare data')).not.toBeInTheDocument();
  expect(getSubDagElements).not.toHaveBeenCalledWith(['root', 'workflow'], Infinity);
  fireEvent.click(screen.getByRole('button', { name: 'Expand Training pipeline' }));
  expect(screen.getByText('Prepare data')).toBeInTheDocument();
  expect(getSubDagElements).toHaveBeenCalledWith(['root', 'workflow'], Infinity);
  expect(
    screen.getByRole('button', { name: 'Expand Training and evaluation' }),
  ).toBeInTheDocument();
  rerender(
    <CommonTestWrapper>
      <DagCanvas {...options} elements={resolve(['root'])} />
    </CommonTestWrapper>,
  );
  expect(screen.getByText('Prepare data')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Collapse Training pipeline' }));
  expect(screen.queryByText('Prepare data')).not.toBeInTheDocument();
});

it('places bulk controls before Layers and expands/collapses all descendants', () => {
  const options = props();
  const { rerender } = render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  const expandAll = screen.getByRole('button', { name: 'Expand all' });
  const collapseAll = screen.getByRole('button', { name: 'Collapse all' });
  expect(within(expandAll).getByTestId('ExpandMoreIcon')).toBeInTheDocument();
  expect(within(collapseAll).getByTestId('ExpandLessIcon')).toBeInTheDocument();
  expect(
    expandAll.compareDocumentPosition(screen.getByText('Layers')) &
      Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
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
  expect(options.onLayersUpdate).not.toHaveBeenCalled();
});

it('sizes group headers like regular nodes under the application root font', () => {
  const previous = document.documentElement.style.fontSize;
  document.documentElement.style.fontSize = '13px';
  try {
    render(
      <CommonTestWrapper>
        <DagCanvas {...props()} />
      </CommonTestWrapper>,
    );
    expect(screen.getAllByTestId('subdag-header')[0]).toHaveStyle({ height: '39px' });
    fireEvent.click(screen.getByRole('button', { name: 'Collapse Training pipeline' }));
    expect(document.querySelector('[data-id="task.workflow"]')).toHaveStyle({ height: '39px' });
    expect(document.querySelector('[data-id="task.deploy"]')).toHaveStyle({ height: '39px' });
  } finally {
    document.documentElement.style.fontSize = previous;
  }
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
