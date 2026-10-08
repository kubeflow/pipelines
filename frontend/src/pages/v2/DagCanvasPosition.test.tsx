// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import type { ReactFlowProps, Node, Edge } from '@xyflow/react';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import { nestedArtifactSpec } from 'src/data/test/groupedFlow';
import { convertSubDagToFlowElements } from 'src/lib/v2/StaticFlow';
import { scopedNodeId } from 'src/lib/v2/GroupedFlow';
import DagCanvas from './DagCanvas';

// Drive live position changes and drag-stop without depending on jsdom SVG geometry.
vi.mock('@xyflow/react', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@xyflow/react')>()),
  ReactFlow: ({
    nodes,
    onNodesChange,
    onNodeDragStop,
  }: ReactFlowProps<Node<FlowElementDataBase>, Edge>) => (
    <div>
      {nodes?.map((node) => (
        <button
          key={node.id}
          data-testid={node.id}
          data-draggable={String(node.draggable)}
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
    </div>
  ),
}));

it('updates leaves during dragging but keeps groups immovable', () => {
  const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
  render(
    <DagCanvas
      layers={['root']}
      elements={resolve(['root'])}
      getSubDagElements={resolve}
      onLayersUpdate={vi.fn()}
      onElementClick={vi.fn()}
      setFlowElements={vi.fn()}
    />,
  );
  const leaf = screen.getByTestId('task.deploy');
  expect(leaf).toHaveAttribute('data-draggable', 'true');
  fireEvent.mouseMove(leaf);
  expect(leaf).toHaveTextContent('{"x":45,"y":90}');
  const group = screen.getByTestId('task.workflow');
  const position = group.textContent;
  expect(group).toHaveAttribute('data-draggable', 'false');
  fireEvent.mouseMove(group);
  fireEvent.click(group);
  expect(group.textContent).toBe(position);
});

it('keeps drag positions separate for focused and parent-relative coordinate frames', () => {
  const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
  const focusedLayers = ['root', 'workflow', 'fit'];
  const id = scopedNodeId(focusedLayers, 'task.train');
  const canvas = (layers: string[]) => (
    <DagCanvas
      layers={layers}
      elements={resolve(layers)}
      getSubDagElements={resolve}
      onLayersUpdate={vi.fn()}
      onElementClick={vi.fn()}
      setFlowElements={vi.fn()}
    />
  );
  const { rerender } = render(canvas(focusedLayers));
  fireEvent.click(screen.getByTestId(id));
  expect(screen.getByTestId(id)).toHaveTextContent('{"x":17,"y":23}');
  rerender(canvas(['root']));
  expect(screen.getByTestId(id)).not.toHaveTextContent('{"x":17,"y":23}');
  fireEvent.click(screen.getByTestId(id));
  expect(screen.getByTestId(id)).toHaveTextContent('{"x":77,"y":88}');
  rerender(canvas(focusedLayers));
  expect(screen.getByTestId(id)).toHaveTextContent('{"x":17,"y":23}');
  rerender(canvas(['root']));
  expect(screen.getByTestId(id)).toHaveTextContent('{"x":77,"y":88}');
});
