/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { ComponentProps, ReactNode } from 'react';
import { act, render } from '@testing-library/react';
import { vi } from 'vitest';
import type { ReactFlow } from '@xyflow/react';
import DagCanvas, { DagCanvasProps } from './DagCanvas';
import { NodeTypeNames, PipelineFlowElement } from 'src/lib/v2/StaticFlow';

const { canvas } = vi.hoisted(() => ({ canvas: vi.fn() }));
vi.mock('@xyflow/react', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@xyflow/react')>();
  return {
    ...actual,
    ReactFlow: (props: { children: ReactNode }) => {
      canvas(props);
      return <div>{props.children}</div>;
    },
    ReactFlowProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
    Background: () => null,
    MiniMap: () => null,
    Controls: () => null,
    Panel: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  };
});

type FlowProps = ComponentProps<typeof ReactFlow>;
const current = () => canvas.mock.lastCall![0] as FlowProps;
function props(): DagCanvasProps {
  const elements: PipelineFlowElement[] = [
    {
      id: 'task.first',
      type: NodeTypeNames.EXECUTION,
      data: { label: 'First', state: 'SUCCEEDED', taskId: 'runtime-first' },
      position: { x: 0, y: 0 },
    },
    {
      id: 'task.loop.1',
      type: NodeTypeNames.SUB_DAG,
      data: { label: 'Iteration one', state: 'RUNNING', taskId: 'runtime-loop' },
      position: { x: 0, y: 106 },
    },
    { id: 'edge', source: 'task.first', target: 'task.loop.1' },
  ];
  return {
    elements,
    layers: ['root', 'loop'],
    setFlowElements: vi.fn(),
    onLayersUpdate: vi.fn(),
    onElementClick: vi.fn(),
  };
}
beforeEach(() => canvas.mockClear());

it('keeps source graph identities and positions while deriving selected and styled presentation', () => {
  const input = props(),
    original = structuredClone(input.elements);
  render(<DagCanvas {...input} selectedNodeId='task.loop.1' />);
  expect(input.elements).toEqual(original);
  expect(
    current().nodes?.map(({ id, position, selected }) => ({ id, position, selected })),
  ).toEqual([
    { id: 'task.first', position: { x: 0, y: 0 }, selected: false },
    { id: 'task.loop.1', position: { x: 0, y: 106 }, selected: true },
  ]);
  expect(current().edges?.[0]).toMatchObject({
    id: 'edge',
    source: 'task.first',
    target: 'task.loop.1',
    className: expect.stringContaining('kfp-graph-edge-running'),
  });
  expect(current().nodesFocusable).toBe(false);
});

it.each([
  ['SKIPPED', 'neutral'],
  ['FAILED', 'failed'],
  ['SUCCEEDED', 'succeeded'],
  ['CACHED', 'succeeded'],
  ['RUNNING', 'running'],
])('renders incoming edges for %s tasks with the %s tone', (state, tone) => {
  const input = props();
  input.elements = input.elements.map((element) =>
    element.id === 'task.loop.1' && 'position' in element
      ? { ...element, data: { ...element.data, state } }
      : element,
  );
  render(<DagCanvas {...input} />);
  expect(current().edges?.[0].className).toContain(`kfp-graph-edge-${tone}`);
});

it('expands the actual iteration node into the existing runtime layer identity', () => {
  const input = props();
  render(<DagCanvas {...input} />);
  const expand = current().nodes![1].data.expand as (id: string) => void;
  expand('task.loop.1');
  expect(input.onLayersUpdate).toHaveBeenCalledWith(['root', 'loop', 'loop.1']);
});

it('persists only the dragged position and respects read-only dragging', () => {
  const input = props();
  const { rerender } = render(<DagCanvas {...input} />);
  current().onNodeDragStop!(
    {} as never,
    { ...current().nodes![0], position: { x: 32, y: 48 } },
    [],
  );
  expect(input.setFlowElements).toHaveBeenCalledWith([
    { ...input.elements[0], position: { x: 32, y: 48 } },
    input.elements[1],
    input.elements[2],
  ]);
  rerender(<DagCanvas {...input} nodesDraggable={false} />);
  expect(current().onNodeDragStop).toBeUndefined();
});

it('fits a linked node once and preserves the viewport across state refreshes', () => {
  const input = props();
  const fitView = vi.fn();
  const { rerender } = render(<DagCanvas {...input} focusNodeId='task.first' />);
  act(() => current().onInit!({ fitView } as never));
  expect(fitView).toHaveBeenCalledTimes(1);
  expect(fitView.mock.calls[0][0].nodes.map(({ id }: { id: string }) => id)).toEqual([
    'task.first',
  ]);
  rerender(
    <DagCanvas {...input} elements={structuredClone(input.elements)} focusNodeId='task.first' />,
  );
  expect(fitView).toHaveBeenCalledTimes(1);
  rerender(<DagCanvas {...input} focusNodeId='task.loop.1' />);
  expect(fitView).toHaveBeenCalledTimes(2);
  expect(fitView.mock.calls[1][0].nodes.map(({ id }: { id: string }) => id)).toEqual([
    'task.loop.1',
  ]);
});

it('retains actual dimensions across batched notifications and selected/source rerenders', () => {
  const input = props();
  const original = structuredClone(input.elements);
  const { rerender } = render(<DagCanvas {...input} />);
  const onNodesChange = current().onNodesChange!;
  act(() => {
    onNodesChange([
      { type: 'dimensions', id: 'task.first', dimensions: { width: 200, height: 56 } },
    ]);
    onNodesChange([
      { type: 'dimensions', id: 'task.loop.1', dimensions: { width: 210, height: 64 } },
    ]);
  });
  expect(current().nodes?.map((node) => node.measured)).toEqual([
    { width: 200, height: 56 },
    { width: 210, height: 64 },
  ]);
  rerender(
    <DagCanvas {...input} elements={structuredClone(input.elements)} selectedNodeId='task.first' />,
  );
  expect(current().nodes?.[0]).toMatchObject({
    selected: true,
    measured: { width: 200, height: 56 },
  });
  expect(current().nodes?.[1].measured).toEqual({ width: 210, height: 64 });
  expect(input.elements).toEqual(original);
  expect(input.setFlowElements).not.toHaveBeenCalled();
});

it('keeps equal measurements stable and ignores nonmeasurement or absent-node changes', () => {
  const input = props();
  render(<DagCanvas {...input} />);
  const dimensions = {
    type: 'dimensions' as const,
    id: 'task.first',
    dimensions: { width: 200, height: 56 },
  };
  act(() => current().onNodesChange!([dimensions]));
  const nodes = current().nodes;
  act(() => current().onNodesChange!([dimensions]));
  expect(current().nodes).toBe(nodes);
  act(() =>
    current().onNodesChange!([
      { type: 'select', id: 'task.first', selected: true },
      { type: 'remove', id: 'task.loop.1' },
      { type: 'position', id: 'task.first', position: { x: 100, y: 100 } },
      { type: 'dimensions', id: 'task.first' },
      { type: 'dimensions', id: 'absent', dimensions: { width: 300, height: 90 } },
    ]),
  );
  expect(current().nodes).toBe(nodes);
  expect(input.setFlowElements).not.toHaveBeenCalled();
});

it('preserves surviving measurements across node additions, removal and reordering', () => {
  const input = props();
  const { rerender } = render(<DagCanvas {...input} />);
  act(() =>
    current().onNodesChange!([
      { type: 'dimensions', id: 'task.first', dimensions: { width: 200, height: 56 } },
      { type: 'dimensions', id: 'task.loop.1', dimensions: { width: 210, height: 64 } },
    ]),
  );
  const added: PipelineFlowElement = {
    id: 'task.added',
    type: NodeTypeNames.EXECUTION,
    data: { label: 'Added' },
    position: { x: 0, y: 212 },
  };
  rerender(
    <DagCanvas
      {...input}
      elements={[added, input.elements[1], input.elements[0], input.elements[2]]}
    />,
  );
  expect(current().nodes?.map(({ id, measured }) => ({ id, measured }))).toEqual([
    { id: 'task.added', measured: undefined },
    { id: 'task.loop.1', measured: { width: 210, height: 64 } },
    { id: 'task.first', measured: { width: 200, height: 56 } },
  ]);
  rerender(<DagCanvas {...input} elements={[input.elements[0], added]} />);
  act(() =>
    current().onNodesChange!([
      { type: 'dimensions', id: 'task.added', dimensions: { width: 200, height: 56 } },
    ]),
  );
  rerender(<DagCanvas {...input} elements={[...input.elements, added]} />);
  expect(current().nodes?.[0].measured).toEqual({ width: 200, height: 56 });
  expect(current().nodes?.[1].measured).toBeUndefined();
  expect(current().nodes?.[2].measured).toEqual({ width: 200, height: 56 });
});

it('separates layer and node-type measurements without requiring a writable parent graph', () => {
  const input = props();
  const noop = () => {};
  const { rerender } = render(<DagCanvas {...input} setFlowElements={noop} />);
  act(() =>
    current().onNodesChange!([
      { type: 'dimensions', id: 'task.first', dimensions: { width: 200, height: 56 } },
    ]),
  );
  rerender(<DagCanvas {...input} setFlowElements={noop} layers={['root', 'other']} />);
  expect(current().nodes?.[0].measured).toBeUndefined();
  act(() =>
    current().onNodesChange!([
      { type: 'dimensions', id: 'task.first', dimensions: { width: 220, height: 60 } },
    ]),
  );
  expect(current().nodes?.[0].measured).toEqual({ width: 220, height: 60 });
  rerender(<DagCanvas {...input} setFlowElements={noop} />);
  expect(current().nodes?.[0].measured).toBeUndefined();
  act(() =>
    current().onNodesChange!([
      { type: 'dimensions', id: 'task.first', dimensions: { width: 200, height: 56 } },
    ]),
  );
  const changedType = input.elements.map((element) =>
    element.id === 'task.first' ? { ...element, type: NodeTypeNames.ARTIFACT } : element,
  );
  rerender(<DagCanvas {...input} setFlowElements={noop} elements={changedType} />);
  expect(current().nodes?.[0].measured).toBeUndefined();
  act(() =>
    current().onNodesChange!([
      { type: 'dimensions', id: 'task.first', dimensions: { width: 180, height: 48 } },
    ]),
  );
  expect(current().nodes?.[0].measured).toEqual({ width: 180, height: 48 });
});
