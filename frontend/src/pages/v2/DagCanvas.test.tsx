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
  expect(within(collapse).getByTestId('UnfoldLessIcon')).toBeInTheDocument();
  fireEvent.click(collapse);
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  expect(screen.getByText('Deploy model')).toBeInTheDocument();
  expect(options.onLayersUpdate).not.toHaveBeenCalled();
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
  ).toHaveAccessibleDescription(new GraphExpansionLimitError(600).message);
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
  expect(buttons[6]).toHaveAccessibleName('Render subdags');
  expect(buttons[6]).toHaveAttribute('aria-pressed', 'true');
  expect(within(buttons[6]).getByTestId('FullscreenExitIcon')).toBeInTheDocument();
  expect(toolbar).not.toContainElement(screen.getByText('Layers'));
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

it('can expand and collapse with the application root font', () => {
  const previous = document.documentElement.style.fontSize;
  document.documentElement.style.fontSize = '13px';
  try {
    render(
      <CommonTestWrapper>
        <DagCanvas {...props()} />
      </CommonTestWrapper>,
    );
    expect(screen.getByText('Train model')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse Training pipeline' }));
    expect(screen.queryByText('Train model')).not.toBeInTheDocument();
    expect(screen.getByText('Deploy model')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Expand Training pipeline' }));
    expect(screen.getByText('Train model')).toBeInTheDocument();
  } finally {
    document.documentElement.style.fontSize = previous;
  }
});

it('switches to click-through mode without recursively resolving children and preserves the mode on refresh', () => {
  const getSubDagElements = vi.fn(resolve);
  const options = { ...props(), getSubDagElements };
  const { rerender } = render(
    <CommonTestWrapper>
      <DagCanvas {...options} />
    </CommonTestWrapper>,
  );
  const toggle = screen.getByRole('button', { name: 'Render subdags' });
  expect(toggle).toHaveAttribute('aria-pressed', 'true');
  fireEvent.click(toggle);
  expect(toggle).toHaveAttribute('aria-pressed', 'false');
  expect(within(toggle).getByTestId('FullscreenIcon')).toBeInTheDocument();
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  expect(screen.getByText('Training pipeline')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Expand all' })).toBeDisabled();
  expect(screen.getByRole('button', { name: 'Collapse all' })).toBeDisabled();
  getSubDagElements.mockClear();
  rerender(
    <CommonTestWrapper>
      <DagCanvas {...options} elements={resolve(['root'])} />
    </CommonTestWrapper>,
  );
  expect(toggle).toHaveAttribute('aria-pressed', 'false');
  expect(getSubDagElements).not.toHaveBeenCalled();
  fireEvent.click(screen.getByTestId('expand-button'));
  expect(options.onLayersUpdate).toHaveBeenCalledWith(['root', 'workflow']);
  const layers = ['root', 'workflow'];
  rerender(
    <CommonTestWrapper>
      <DagCanvas {...options} layers={layers} elements={resolve(layers)} />
    </CommonTestWrapper>,
  );
  expect(toggle).toHaveAttribute('aria-pressed', 'false');
  expect(screen.getByText('Prepare data')).toBeInTheDocument();
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  fireEvent.click(toggle);
  expect(screen.getByText('Train model')).toBeInTheDocument();
  expect(options.onLayersUpdate).toHaveBeenCalledTimes(1);
  expect(options.onElementClick).not.toHaveBeenCalled();
});

it('retains inline collapse choices across rendering-mode changes', () => {
  render(
    <CommonTestWrapper>
      <DagCanvas {...props()} />
    </CommonTestWrapper>,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Collapse Training and evaluation' }));
  const toggle = screen.getByRole('button', { name: 'Render subdags' });
  fireEvent.click(toggle);
  fireEvent.click(toggle);
  expect(
    screen.getByRole('button', { name: 'Expand Training and evaluation' }),
  ).toBeInTheDocument();
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
});

it('selects the visible namesake in its own layer after disabling inline rendering', () => {
  const options = props();
  options.elements = [
    ...options.elements,
    {
      id: 'task.train',
      type: 'EXECUTION',
      data: { label: 'Root train' },
      position: { x: 0, y: 0 },
    },
  ];
  render(
    <CommonTestWrapper>
      <DagCanvas
        {...options}
        selectedElement={{
          element: resolve(['root', 'workflow', 'fit']).find(
            (element) => element.id === 'task.train',
          )!,
          layers: ['root', 'workflow', 'fit'],
        }}
      />
    </CommonTestWrapper>,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Render subdags' }));
  expect(screen.queryByText('Train model')).not.toBeInTheDocument();
  fireEvent.click(screen.getByText('Root train'));
  expect(options.onElementClick).toHaveBeenCalledWith(expect.anything(), {
    element: expect.objectContaining({ id: 'task.train', data: { label: 'Root train' } }),
    layers: ['root'],
  });
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
  const mode = screen.getByRole('button', { name: 'Render subdags' });
  fireEvent.click(mode);
  fireEvent.click(screen.getByText('Training pipeline'));
  expect(options.onElementClick).not.toHaveBeenCalled();
  fireEvent.click(mode);
  fireEvent.click(screen.getByText('Train model'));
  expect(options.onElementClick).not.toHaveBeenCalled();
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
