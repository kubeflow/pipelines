// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import type { ReactFlowProps, Node, Edge } from '@xyflow/react';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import { nestedArtifactSpec } from 'src/data/test/groupedFlow';
import { convertSubDagToFlowElements } from 'src/lib/v2/StaticFlow';
import { layerNodeId } from 'src/lib/v2/GroupedFlow';
import DagCanvas from './DagCanvas';
import { mockResizeObserver } from 'src/TestUtils';

beforeEach(() => mockResizeObserver());

// Drive live position changes and drag-stop without depending on jsdom SVG geometry.
vi.mock('@xyflow/react', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@xyflow/react')>()),
  MiniMap: () => null,
  Background: () => null,
  ReactFlow: ({
    nodes,
    edges,
    proOptions,
    onNodeMouseEnter,
    onNodeMouseLeave,
    children,
    onNodesChange,
    onNodeDragStop,
  }: ReactFlowProps<Node<FlowElementDataBase>, Edge>) => (
    <div data-testid='flow' data-hide-attribution={String(proOptions?.hideAttribution)}>
      {edges?.map((edge) => (
        <span
          key={edge.id}
          data-testid='edge'
          className={edge.className}
          data-source={edge.source}
          data-target={edge.target}
        />
      ))}
      {nodes?.map((node) => (
        <button
          key={node.id}
          data-testid={node.id}
          data-draggable={String(node.draggable)}
          data-selected={String(node.selected ?? false)}
          onMouseEnter={(event) => onNodeMouseEnter?.(event, node)}
          onMouseLeave={(event) => onNodeMouseLeave?.(event, node)}
          onMouseMove={() =>
            onNodesChange?.([
              { type: 'position', id: node.id, position: { x: 45, y: 90 }, dragging: true },
            ])
          }
          onClick={(event) =>
            onNodeDragStop?.(
              event.nativeEvent,
              {
                ...node,
                position: node.parentId ? { x: 77, y: 88 } : { x: 17, y: 23 },
              },
              [],
            )
          }
        >
          {JSON.stringify(node.position)}
        </button>
      ))}
      {children}
    </div>
  ),
}));

it('updates leaves and groups during dragging without changing child-relative positions', () => {
  const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
  render(
    <DagCanvas
      layers={['root']}
      elements={resolve(['root'])}
      getSubDagElements={resolve}
      onElementClick={vi.fn()}
    />,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Unlock node positions' }));
  const leaf = screen.getByTestId('task.deploy');
  expect(leaf).toHaveAttribute('data-draggable', 'true');
  fireEvent.mouseMove(leaf);
  expect(leaf).toHaveTextContent('{"x":45,"y":90}');
  const group = screen.getByTestId('task.workflow');
  const child = screen.getByTestId(layerNodeId(['root', 'workflow'], 'task.prepare'));
  const childPosition = child.textContent;
  expect(group).toHaveAttribute('data-draggable', 'true');
  fireEvent.mouseMove(group);
  expect(group).toHaveTextContent('{"x":45,"y":90}');
  expect(child.textContent).toBe(childPosition);
  fireEvent.click(group);
  expect(group).toHaveTextContent('{"x":17,"y":23}');
  expect(child.textContent).toBe(childPosition);
});

it('still respects a read-only host for both leaves and groups', () => {
  const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
  render(
    <DagCanvas
      layers={['root']}
      elements={resolve(['root'])}
      getSubDagElements={resolve}
      nodesDraggable={false}
      onElementClick={vi.fn()}
    />,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Unlock node positions' }));
  for (const id of ['task.workflow', 'task.deploy']) {
    const node = screen.getByTestId(id);
    const position = node.textContent;
    expect(node).toHaveAttribute('data-draggable', 'false');
    fireEvent.mouseMove(node);
    fireEvent.click(node);
    expect(node.textContent).toBe(position);
  }
});

it('rejects live drag updates while locked, including after a refresh', () => {
  const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
  const options = {
    layers: ['root'],
    elements: resolve(['root']),
    getSubDagElements: resolve,
    onElementClick: vi.fn(),
  };
  const { rerender } = render(<DagCanvas {...options} />);
  expect(screen.getByRole('button', { name: 'Unlock node positions' })).toBeInTheDocument();
  rerender(<DagCanvas {...options} elements={resolve(['root'])} />);
  for (const id of ['task.workflow', 'task.deploy']) {
    const node = screen.getByTestId(id);
    const position = node.textContent;
    fireEvent.mouseMove(node);
    fireEvent.click(node);
    expect(node.textContent).toBe(position);
  }
  fireEvent.click(screen.getByRole('button', { name: 'Unlock node positions' }));
  fireEvent.mouseMove(screen.getByTestId('task.workflow'));
  expect(screen.getByTestId('task.workflow')).toHaveTextContent('{"x":45,"y":90}');
});

it('marks only the selected layer instance, not its flat namesake, as selected', () => {
  const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
  const layers = ['root', 'workflow', 'fit'];
  const nested = resolve(layers).find((element) => element.id === 'task.train')!;
  render(
    <DagCanvas
      layers={['root']}
      elements={[
        ...resolve(['root']),
        {
          id: 'task.train',
          type: 'EXECUTION',
          data: { label: 'Root train' },
          position: { x: 0, y: 0 },
        },
      ]}
      getSubDagElements={resolve}
      selectedElement={{ element: nested, layers }}
      onElementClick={vi.fn()}
    />,
  );
  expect(screen.getByTestId(layerNodeId(layers, 'task.train'))).toHaveAttribute(
    'data-selected',
    'true',
  );
  expect(screen.getByTestId('task.train')).toHaveAttribute('data-selected', 'false');
  fireEvent.click(screen.getByRole('button', { name: 'Collapse all' }));
  expect(screen.getByTestId('task.train')).toHaveAttribute('data-selected', 'false');
});

it('preserves nested task drag positions across data refreshes', () => {
  const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
  const id = layerNodeId(['root', 'workflow', 'fit'], 'task.train');
  const canvas = () => (
    <DagCanvas
      layers={['root']}
      elements={resolve(['root'])}
      getSubDagElements={resolve}
      onElementClick={vi.fn()}
    />
  );
  const { rerender } = render(canvas());
  fireEvent.click(screen.getByRole('button', { name: 'Unlock node positions' }));
  fireEvent.click(screen.getByTestId(id));
  expect(screen.getByTestId(id)).toHaveTextContent('{"x":77,"y":88}');
  rerender(canvas());
  expect(screen.getByTestId(id)).toHaveTextContent('{"x":77,"y":88}');
});

it('keeps dependencies neutral when hovering or inspecting tasks', () => {
  const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
  const elements = resolve(['root']);
  const options = {
    layers: ['root'],
    elements,
    getSubDagElements: resolve,
    onElementClick: vi.fn(),
  };
  const { rerender } = render(<DagCanvas {...options} />);
  const edges = screen.getAllByTestId('edge');
  expect(edges.length).toBeGreaterThan(0);
  edges.forEach((edge) => expect(edge.className.trim()).toBe('kfp-graph-edge'));
  const task = screen.getByTestId('task.deploy');
  fireEvent.mouseEnter(task);
  const connected = () =>
    screen
      .getAllByTestId('edge')
      .filter(
        (edge) => edge.dataset.source === 'task.deploy' || edge.dataset.target === 'task.deploy',
      );
  expect(connected().length).toBeGreaterThan(0);
  connected().forEach((edge) => expect(edge).not.toHaveClass('kfp-graph-edge-highlighted'));
  fireEvent.mouseLeave(task);
  connected().forEach((edge) => expect(edge).not.toHaveClass('kfp-graph-edge-highlighted'));
  const element = elements.find((element) => element.id === 'task.deploy')!;
  rerender(<DagCanvas {...options} selectedElement={{ element, layers: ['root'] }} />);
  connected().forEach((edge) => expect(edge).not.toHaveClass('kfp-graph-edge-highlighted'));
  expect(screen.getByTestId('flow')).toHaveAttribute('data-hide-attribution', 'true');
});
