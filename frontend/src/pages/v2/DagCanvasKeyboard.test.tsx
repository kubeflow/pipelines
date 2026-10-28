// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { useEffect, useRef } from 'react';
import type { ReactFlowInstance, ReactFlowProps } from '@xyflow/react';
import DagCanvas from './DagCanvas';
import { mockResizeObserver } from 'src/TestUtils';

const readiness = vi.hoisted(() => ({ nodes: false, viewport: false, instance: false }));
const viewport = vi.hoisted(() => ({
  getViewport: vi.fn(() => ({ x: 10, y: 20, zoom: 2 })),
  setViewport: vi.fn(),
  fitView: vi.fn(),
  getNodesBounds: vi.fn(() => ({ x: 0, y: 0, width: 200, height: 56 })),
  zoomIn: vi.fn(),
  zoomOut: vi.fn(),
}));

vi.mock('@xyflow/react', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@xyflow/react')>()),
  useReactFlow: () => ({ ...viewport, viewportInitialized: readiness.instance }),
  useNodesInitialized: () => readiness.nodes,
  useStore: () => readiness.viewport,
  MiniMap: () => null,
  Background: () => null,
  ReactFlow: ({ onInit, children, nodesFocusable, fitViewOptions, maxZoom }: ReactFlowProps) => {
    const initialized = useRef(false);
    useEffect(() => {
      if (!initialized.current) {
        initialized.current = true;
        onInit?.(viewport as unknown as ReactFlowInstance);
      }
    }, [onInit]);
    return (
      <div data-testid='flow' data-fit-max={fitViewOptions?.maxZoom} data-max-zoom={maxZoom ?? 2}>
        <button data-focusable={String(nodesFocusable)}>Node</button>
        <input aria-label='Input' />
        <textarea aria-label='Textarea' />
        <select aria-label='Select'>
          <option>Option</option>
        </select>
        <div role='textbox' contentEditable suppressContentEditableWarning aria-label='Editable'>
          Text
        </div>
        {children}
      </div>
    );
  },
}));

beforeEach(() => {
  vi.clearAllMocks();
  readiness.nodes = false;
  readiness.viewport = false;
  readiness.instance = false;
  mockResizeObserver();
});
const canvas = () => <DagCanvas elements={[]} layers={['root']} onElementClick={vi.fn()} />;

it.each([
  ['ArrowLeft', 50, 20],
  ['ArrowRight', -30, 20],
  ['ArrowUp', 10, 60],
  ['ArrowDown', 10, -20],
])(
  'scrolls the locked view with %s by translating content oppositely, without changing zoom',
  (key, x, y) => {
    render(canvas());
    fireEvent.keyDown(screen.getByRole('region', { name: 'Pipeline graph' }), { key });
    expect(viewport.setViewport).toHaveBeenCalledExactlyOnceWith({ x, y, zoom: 2 });
  },
);

it.each([
  ['f', true],
  ['F', true],
  ['f', false],
  ['F', false],
])('fits the whole graph with %s when locked=%s', (key, locked) => {
  render(canvas());
  if (!locked) fireEvent.click(screen.getByLabelText('Unlock node positions'));
  viewport.fitView.mockClear();
  fireEvent.keyDown(screen.getByRole('button', { name: 'Node' }), { key });
  expect(viewport.fitView).toHaveBeenCalledExactlyOnceWith({ maxZoom: 1 });
  expect(viewport.setViewport).not.toHaveBeenCalled();
});

it('pans from focused nodes only while locked and resumes panning when relocked', () => {
  render(canvas());
  const node = screen.getByRole('button', { name: 'Node' });
  fireEvent.keyDown(node, { key: 'ArrowRight' });
  expect(viewport.setViewport).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByLabelText('Unlock node positions'));
  fireEvent.keyDown(node, { key: 'ArrowRight' });
  expect(viewport.setViewport).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByLabelText('Lock node positions'));
  fireEvent.keyDown(node, { key: 'ArrowRight' });
  expect(viewport.setViewport).toHaveBeenCalledTimes(2);
});

it.each(['Input', 'Textarea', 'Select', 'Editable'])(
  'leaves navigation shortcuts in %s controls alone',
  (label) => {
    render(canvas());
    viewport.fitView.mockClear();
    for (const key of ['ArrowLeft', 'f', 'F'])
      fireEvent.keyDown(screen.getByLabelText(label), { key });
    expect(viewport.setViewport).not.toHaveBeenCalled();
    expect(viewport.fitView).not.toHaveBeenCalled();
  },
);

it('ignores modified keys, unrelated keys, and keyboard input outside the graph', () => {
  render(
    <>
      <input aria-label='Outside graph' />
      {canvas()}
    </>,
  );
  const graph = screen.getByRole('region', { name: 'Pipeline graph' });
  viewport.fitView.mockClear();
  for (const modifier of ['altKey', 'ctrlKey', 'metaKey']) {
    for (const key of ['ArrowRight', 'f']) fireEvent.keyDown(graph, { key, [modifier]: true });
  }
  for (const key of ['Enter', 'Tab', 'constructor']) fireEvent.keyDown(graph, { key });
  for (const key of ['ArrowRight', 'f'])
    fireEvent.keyDown(screen.getByLabelText('Outside graph'), { key });
  expect(viewport.setViewport).not.toHaveBeenCalled();
  expect(viewport.fitView).not.toHaveBeenCalled();
});

it('enables keyboard node movement only while unlocked', () => {
  render(canvas());
  const node = screen.getByRole('button', { name: 'Node' });
  expect(node).toHaveAttribute('data-focusable', 'false');
  fireEvent.click(screen.getByRole('button', { name: 'Unlock node positions' }));
  expect(node).toHaveAttribute('data-focusable', 'true');
  fireEvent.click(screen.getByRole('button', { name: 'Lock node positions' }));
  expect(node).toHaveAttribute('data-focusable', 'false');
});

it('applies the measured initial viewport immediately without queuing a later fit', async () => {
  const graph = () => (
    <DagCanvas
      elements={[{ id: 'task', position: { x: 0, y: 0 }, data: { label: 'Task' } }]}
      layers={['root']}
      onElementClick={vi.fn()}
    />
  );
  const { rerender } = render(graph());
  const region = screen.getByRole('region', { name: 'Pipeline graph' });
  Object.defineProperty(region, 'clientWidth', { value: 1000 });
  Object.defineProperty(region, 'clientHeight', { value: 600 });
  expect(screen.getByTestId('flow')).toHaveAttribute('data-max-zoom', '2');
  readiness.nodes = true;
  rerender(graph());
  expect(viewport.setViewport).not.toHaveBeenCalled();
  readiness.viewport = true;
  rerender(graph());
  await new Promise((resolve) => requestAnimationFrame(resolve));
  expect(viewport.setViewport).not.toHaveBeenCalled();
  readiness.instance = true;
  rerender(graph());
  await waitFor(() =>
    expect(viewport.setViewport).toHaveBeenCalledExactlyOnceWith({
      x: 400,
      y: 272,
      zoom: 1,
    }),
  );
  // A queued fitView would run on the next controlled-node update, after a user zoom.
  expect(viewport.fitView).not.toHaveBeenCalled();
  viewport.setViewport.mockClear();
  fireEvent.click(screen.getByRole('button', { name: 'Zoom out' }));
  rerender(graph());
  await new Promise((resolve) => requestAnimationFrame(resolve));
  expect(viewport.fitView).not.toHaveBeenCalled();
  expect(viewport.setViewport).not.toHaveBeenCalled();
});

it('preserves a manual pan made while the graph is still initializing', async () => {
  const { rerender } = render(canvas());
  fireEvent.keyDown(screen.getByRole('region', { name: 'Pipeline graph' }), { key: 'ArrowLeft' });
  readiness.nodes = true;
  readiness.viewport = true;
  readiness.instance = true;
  rerender(canvas());
  await new Promise((resolve) => requestAnimationFrame(resolve));
  expect(viewport.fitView).not.toHaveBeenCalled();
  expect(viewport.setViewport).toHaveBeenCalledExactlyOnceWith({ x: 50, y: 20, zoom: 2 });
});

it.each(['Zoom in', 'Zoom out', 'Fit view'])(
  'preserves an early %s control action',
  async (label) => {
    const { rerender } = render(canvas());
    fireEvent.click(screen.getByRole('button', { name: label }));
    const calls = viewport.fitView.mock.calls.length;
    readiness.nodes = true;
    readiness.viewport = true;
    readiness.instance = true;
    rerender(canvas());
    await new Promise((resolve) => requestAnimationFrame(resolve));
    expect(viewport.fitView).toHaveBeenCalledTimes(calls);
    if (label === 'Zoom in') expect(viewport.zoomIn).toHaveBeenCalledOnce();
    if (label === 'Zoom out') expect(viewport.zoomOut).toHaveBeenCalledOnce();
  },
);
