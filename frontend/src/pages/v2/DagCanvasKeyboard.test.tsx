// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import { useEffect } from 'react';
import type { ReactFlowInstance, ReactFlowProps } from '@xyflow/react';
import DagCanvas from './DagCanvas';

const viewport = vi.hoisted(() => ({
  getViewport: vi.fn(() => ({ x: 10, y: 20, zoom: 2 })),
  setViewport: vi.fn(),
  fitView: vi.fn(),
}));

vi.mock('@xyflow/react', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@xyflow/react')>()),
  MiniMap: () => null,
  Background: () => null,
  ReactFlow: ({ onInit, children }: ReactFlowProps) => {
    useEffect(() => {
      onInit?.(viewport as unknown as ReactFlowInstance);
    }, [onInit]);
    return (
      <div>
        <button>Node</button>
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

beforeEach(() => vi.clearAllMocks());
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
  if (!locked) fireEvent.click(screen.getByLabelText('Unlock graph'));
  viewport.fitView.mockClear();
  fireEvent.keyDown(screen.getByRole('button', { name: 'Node' }), { key });
  expect(viewport.fitView).toHaveBeenCalledExactlyOnceWith();
  expect(viewport.setViewport).not.toHaveBeenCalled();
});

it('pans from focused nodes only while locked and resumes panning when relocked', () => {
  render(canvas());
  const node = screen.getByRole('button', { name: 'Node' });
  fireEvent.keyDown(node, { key: 'ArrowRight' });
  expect(viewport.setViewport).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByLabelText('Unlock graph'));
  fireEvent.keyDown(node, { key: 'ArrowRight' });
  expect(viewport.setViewport).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByLabelText('Lock graph'));
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
