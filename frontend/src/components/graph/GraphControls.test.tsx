// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import GraphControls, { GraphControlsProps } from './GraphControls';

const mocked = vi.hoisted(() => ({
  viewport: { zoomIn: vi.fn(), zoomOut: vi.fn(), fitView: vi.fn() },
  store: { transform: [0, 0, 1], minZoom: 0.05, maxZoom: 2 },
}));
vi.mock('@xyflow/react', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@xyflow/react')>()),
  useReactFlow: () => mocked.viewport,
  useStore: (selector: (state: typeof mocked.store) => unknown) => selector(mocked.store),
}));

function props(): GraphControlsProps {
  return {
    showSubDagControls: true,
    hasSubDags: true,
    locked: false,
    onLockChange: vi.fn(),
    onExpandAll: vi.fn(),
    onCollapseAll: vi.fn(),
    onViewportInteraction: vi.fn(),
  };
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.clearAllMocks();
  mocked.store.transform[2] = 1;
});
afterEach(() => {
  cleanup();
  vi.runOnlyPendingTimers();
  vi.useRealTimers();
});

it('orders the six graph controls and dispatches each action', () => {
  const options = props();
  const { rerender } = render(<GraphControls {...options} />);
  const buttons = screen.getAllByRole('button');
  expect(buttons.map((button) => button.getAttribute('aria-label'))).toEqual([
    'Expand all',
    'Collapse all',
    'Zoom in',
    'Zoom out',
    'Fit view',
    'Lock node positions',
  ]);
  expect(screen.getByRole('button', { name: 'Fit view' })).toHaveAttribute(
    'aria-keyshortcuts',
    'F',
  );
  for (const button of buttons) fireEvent.click(button);
  expect(options.onExpandAll).toHaveBeenCalledOnce();
  expect(options.onCollapseAll).toHaveBeenCalledOnce();
  expect(options.onViewportInteraction).toHaveBeenCalledTimes(3);
  expect(mocked.viewport.zoomIn).toHaveBeenCalledOnce();
  expect(mocked.viewport.zoomOut).toHaveBeenCalledOnce();
  expect(mocked.viewport.fitView).toHaveBeenCalledExactlyOnceWith({ maxZoom: 1 });
  expect(options.onLockChange).toHaveBeenCalledWith(true);
  rerender(<GraphControls {...options} locked />);
  fireEvent.click(screen.getByRole('button', { name: 'Unlock node positions' }));
  expect(options.onLockChange).toHaveBeenLastCalledWith(false);
});

it.each([
  ['Expand all', 'Expand all sub-DAGs'],
  ['Collapse all', 'Collapse all sub-DAGs'],
  ['Zoom in', 'Zoom in'],
  ['Zoom out', 'Zoom out'],
  ['Fit view', 'Fit graph to view (F)'],
  ['Lock node positions', 'Lock nodes to pan with dragging and arrow keys'],
])('shows the %s tooltip after 500ms, not sooner', async (label, tooltip) => {
  render(<GraphControls {...props()} />);
  const button = screen.getByRole('button', { name: label });
  expect(button).not.toHaveAttribute('title');
  fireEvent.mouseEnter(button.parentElement!);
  fireEvent.mouseMove(button.parentElement!);
  await act(async () => {
    await vi.advanceTimersByTimeAsync(499);
  });
  expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1);
  });
  // Base UI mounts the portal on the next animation frame after opening.
  await act(async () => {
    vi.advanceTimersToNextFrame();
  });
  expect(screen.getByRole('tooltip')).toHaveTextContent(tooltip);
});

it('retains the 500ms delay when moving between icons and explains disabled actions', async () => {
  render(<GraphControls {...props()} canExpand={false} />);
  const zoom = screen.getByRole('button', { name: 'Zoom in' });
  fireEvent.mouseEnter(zoom.parentElement!);
  fireEvent.mouseMove(zoom.parentElement!);
  await act(async () => {
    await vi.advanceTimersByTimeAsync(500);
  });
  // Base UI mounts the portal on the next animation frame after opening.
  await act(async () => {
    vi.advanceTimersToNextFrame();
  });
  expect(screen.getByRole('tooltip')).toHaveTextContent('Zoom in');
  fireEvent.mouseLeave(zoom.parentElement!);
  await act(async () => {
    await vi.advanceTimersByTimeAsync(250);
  });
  const expand = screen.getByRole('button', { name: 'Expand all' });
  expect(expand).toBeDisabled();
  fireEvent.mouseEnter(expand.parentElement!);
  fireEvent.mouseMove(expand.parentElement!);
  await act(async () => {
    await vi.advanceTimersByTimeAsync(499);
  });
  expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1);
  });
  // Base UI mounts the portal on the next animation frame after opening.
  await act(async () => {
    vi.advanceTimersToNextFrame();
  });
  expect(screen.getByRole('tooltip')).toHaveTextContent('All sub-DAGs are expanded');
});

it('respects zoom limits without disabling fit or expansion controls', () => {
  mocked.store.transform[2] = mocked.store.maxZoom;
  const { rerender } = render(<GraphControls {...props()} />);
  expect(screen.getByRole('button', { name: 'Zoom in' })).toBeDisabled();
  mocked.store.transform[2] = mocked.store.minZoom;
  rerender(<GraphControls {...props()} />);
  expect(screen.getByRole('button', { name: 'Zoom out' })).toBeDisabled();
  expect(screen.getByRole('button', { name: 'Fit view' })).not.toBeDisabled();
  expect(screen.getByRole('button', { name: 'Expand all' })).not.toBeDisabled();
});

it('hides group actions on flat graphs while keeping viewport controls', () => {
  render(<GraphControls {...props()} hasSubDags={false} />);
  expect(screen.queryByRole('button', { name: 'Expand all' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Collapse all' })).not.toBeInTheDocument();
  expect(screen.getAllByRole('button')).toHaveLength(4);
});

it('does not dispatch disabled group actions', () => {
  const options = props();
  render(<GraphControls {...options} canCollapse={false} />);
  const collapse = screen.getByRole('button', { name: 'Collapse all' });
  expect(collapse).toBeDisabled();
  fireEvent.click(collapse);
  expect(options.onCollapseAll).not.toHaveBeenCalled();
});
