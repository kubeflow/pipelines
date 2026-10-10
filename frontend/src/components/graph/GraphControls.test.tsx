// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { act, cleanup, fireEvent, render, screen, within } from '@testing-library/react';
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
    renderSubDags: true,
    locked: false,
    onRenderSubDagsChange: vi.fn(),
    onLockChange: vi.fn(),
    onExpandAll: vi.fn(),
    onCollapseAll: vi.fn(),
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

it('uses fullscreen controls at the bottom and dispatches each action', () => {
  const options = props();
  const { rerender } = render(<GraphControls {...options} />);
  const buttons = screen.getAllByRole('button');
  expect(buttons.map((button) => button.getAttribute('aria-label'))).toEqual([
    'Expand all',
    'Collapse all',
    'Zoom in',
    'Zoom out',
    'Fit view',
    'Lock graph',
    'Render subdags',
  ]);
  expect(within(buttons[6]).getByTestId('FullscreenExitIcon')).toBeInTheDocument();
  for (const button of buttons) fireEvent.click(button);
  expect(options.onExpandAll).toHaveBeenCalledOnce();
  expect(options.onCollapseAll).toHaveBeenCalledOnce();
  expect(mocked.viewport.zoomIn).toHaveBeenCalledOnce();
  expect(mocked.viewport.zoomOut).toHaveBeenCalledOnce();
  expect(mocked.viewport.fitView).toHaveBeenCalledOnce();
  expect(options.onLockChange).toHaveBeenCalledWith(true);
  expect(options.onRenderSubDagsChange).toHaveBeenCalledWith(false);
  rerender(<GraphControls {...options} locked renderSubDags={false} />);
  fireEvent.click(screen.getByRole('button', { name: 'Unlock graph' }));
  expect(options.onLockChange).toHaveBeenLastCalledWith(false);
  expect(screen.getByTestId('FullscreenIcon')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Render subdags' })).toHaveAttribute(
    'aria-pressed',
    'false',
  );
});

it.each([
  ['Expand all', 'Expand all sub-DAGs'],
  ['Collapse all', 'Collapse all sub-DAGs'],
  ['Zoom in', 'Zoom in'],
  ['Zoom out', 'Zoom out'],
  ['Fit view', 'Fit graph to view'],
  ['Lock graph', 'Lock node dragging and selection'],
  ['Render subdags', 'Disable inline sub-DAG rendering'],
])('shows the %s tooltip after 500ms, not sooner', (label, tooltip) => {
  render(<GraphControls {...props()} />);
  const button = screen.getByRole('button', { name: label });
  expect(button).not.toHaveAttribute('title');
  fireEvent.mouseOver(button.parentElement!);
  act(() => {
    vi.advanceTimersByTime(499);
  });
  expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  act(() => {
    vi.advanceTimersByTime(1);
  });
  expect(screen.getByRole('tooltip')).toHaveTextContent(tooltip);
});

it('retains the 500ms delay when moving between icons and explains disabled actions', () => {
  render(<GraphControls {...props()} renderSubDags={false} />);
  const zoom = screen.getByRole('button', { name: 'Zoom in' });
  fireEvent.mouseOver(zoom.parentElement!);
  act(() => {
    vi.advanceTimersByTime(500);
  });
  expect(screen.getByRole('tooltip')).toHaveTextContent('Zoom in');
  fireEvent.mouseLeave(zoom.parentElement!);
  act(() => {
    vi.advanceTimersByTime(250);
  });
  const expand = screen.getByRole('button', { name: 'Expand all' });
  expect(expand).toBeDisabled();
  fireEvent.mouseOver(expand.parentElement!);
  act(() => {
    vi.advanceTimersByTime(499);
  });
  expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  act(() => {
    vi.advanceTimersByTime(1);
  });
  expect(screen.getByRole('tooltip')).toHaveTextContent('Enable sub-DAG rendering to expand all');
});

it('respects zoom limits without disabling fit or mode controls', () => {
  mocked.store.transform[2] = mocked.store.maxZoom;
  const { rerender } = render(<GraphControls {...props()} />);
  expect(screen.getByRole('button', { name: 'Zoom in' })).toBeDisabled();
  mocked.store.transform[2] = mocked.store.minZoom;
  rerender(<GraphControls {...props()} />);
  expect(screen.getByRole('button', { name: 'Zoom out' })).toBeDisabled();
  expect(screen.getByRole('button', { name: 'Fit view' })).not.toBeDisabled();
  expect(screen.getByRole('button', { name: 'Render subdags' })).not.toBeDisabled();
});
