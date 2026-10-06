// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { act, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { getContrastRatio } from '@mui/material/styles';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { TimelineTask } from 'src/lib/v2/MlmdTaskTiming';
import RunTimeline, { RunTimelineProps } from './RunTimeline';

const base = Date.parse('2026-01-01T00:00:00Z');
const at = (seconds: number) => new Date(base + seconds * 1000);
const run: V2beta1Run = {
  run_id: 'run',
  state: 'SUCCEEDED',
  created_at: at(0),
  finished_at: at(300),
};
const task = (
  name: string,
  start: number,
  finish?: number,
  overrides: Partial<TimelineTask> = {},
): TimelineTask => ({
  id: name,
  name,
  state: 'SUCCEEDED',
  createdAt: at(start),
  updatedAt: finish === undefined ? undefined : at(finish),
  graphTarget: { layers: ['root'], nodeId: name, executionId: 1 },
  ...overrides,
});
const tasks = [
  task('Alpha', 10, 19),
  task('Bravo', 20, 140, { iteration: 2 }),
  task('Charlie', 30, 80),
  task('Unknown', 0, undefined, { state: 'SKIPPED' }),
  task('Cached', 5, 5, { state: 'CACHED' }),
];
function setup(overrides: Partial<RunTimelineProps> = {}) {
  const props: RunTimelineProps = { run, tasks, loading: false, onOpenTask: vi.fn(), ...overrides };
  const view = render(<RunTimeline {...props} />);
  return {
    ...view,
    props,
    update: (next: Partial<RunTimelineProps>) =>
      view.rerender(<RunTimeline {...props} {...next} />),
  };
}
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

it.each([
  'SUCCEEDED',
  'RUNNING',
  'FAILED',
  'CACHED',
  'SKIPPED',
  'RUNTIME_STATE_UNSPECIFIED',
  undefined,
] as const)('renders %s status with at least 4.5:1 contrast on white', (state) => {
  setup({ tasks: [task('Contrast', 0, 10, { state })] });
  const status = screen.getByRole('complementary').querySelector('dl .rt-status') as HTMLElement;
  expect(getContrastRatio(getComputedStyle(status).color, '#ffffff')).toBeGreaterThanOrEqual(4.5);
});

it('orders components chronologically and preserves selection across refreshes', async () => {
  const { update } = setup();
  const chart = screen.getByRole('table');
  expect(
    within(chart)
      .getAllByRole('columnheader')
      .map((header) => header.textContent),
  ).toEqual(['Component', '', 'Elapsed']);
  expect(
    within(chart)
      .getAllByRole('row')
      .slice(1)
      .map((row) => within(row).getAllByRole('button')[0].textContent),
  ).toEqual(['Unknown', 'Cached', 'Alpha', 'Bravo', 'Charlie']);
  await userEvent.click(within(chart).getByRole('button', { name: 'Alpha', exact: true }));
  update({ tasks: [...tasks, task('Longest new task', 1, 299)] });
  expect(
    within(screen.getByRole('complementary')).getByRole('heading', { name: 'Alpha' }),
  ).toBeInTheDocument();
});

it.each(['name', 'bar', 'track', 'elapsed', 'row'] as const)(
  'selects a task by its %s area',
  async (area) => {
    setup();
    const name = screen.getByRole('button', { name: 'Alpha', exact: true });
    const row = name.closest('[role="row"]') as HTMLElement;
    const cells = within(row).getAllByRole('cell');
    const targets = {
      name,
      bar: within(row).getByRole('button', { name: 'Select Alpha, 9s' }),
      track: cells[1],
      elapsed: cells[2],
      row,
    };
    await userEvent.click(targets[area]);
    expect(
      within(screen.getByRole('complementary')).getByRole('heading', { name: 'Alpha' }),
    ).toBeInTheDocument();
    expect(name).toHaveAttribute('aria-pressed', 'true');
  },
);

it.each(['Alpha', 'Select Alpha, 9s'])('supports keyboard selection: %s', async (name) => {
  setup();
  screen.getByRole('button', { name, exact: true }).focus();
  await userEvent.keyboard('{Enter}');
  expect(
    within(screen.getByRole('complementary')).getByRole('heading', { name: 'Alpha' }),
  ).toBeInTheDocument();
});

it('selects untimed tasks without inventing timing', async () => {
  setup();
  await userEvent.click(screen.getByText('Timing unavailable'));
  expect(
    screen.getByText('Insufficient timestamps. No duration has been inferred.'),
  ).toBeInTheDocument();
});

it('shows approximate MLMD semantics, latest state, and no inferred retry history', async () => {
  const { update } = setup();
  expect(screen.getByText('Approximate MLMD elapsed')).toBeInTheDocument();
  expect(screen.getByText(/not exact component start and finish times/)).toBeInTheDocument();
  expect(screen.getByText(/State history and retry attempts are unavailable/)).toBeInTheDocument();
  const metadata = screen.getByRole('complementary').querySelector('dl')!;
  expect(Array.from(metadata.querySelectorAll('dt')).map((field) => field.textContent)).toEqual([
    'Status',
    'Created',
    'Last updated',
    'Iteration',
  ]);
  expect(screen.queryByText('Finished')).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole('button', { name: 'Cached', exact: true }));
  expect(screen.getByText(/earlier retry attempts may be included/)).toBeInTheDocument();
  expect(screen.queryByLabelText('Retried task')).not.toBeInTheDocument();
  update({
    tasks: tasks.map((item) => (item.id === 'Cached' ? { ...item, state: 'FAILED' } : item)),
  });
  expect(within(screen.getByRole('complementary')).getByText('Failed')).toBeInTheDocument();
});

it('preserves detail scrolling on refresh and resets it for a different selection', async () => {
  const { update } = setup();
  await userEvent.click(screen.getByRole('button', { name: 'Alpha', exact: true }));
  const inspector = screen.getByRole('complementary');
  inspector.scrollTop = 120;
  update({ tasks: tasks.map((item) => ({ ...item })) });
  expect(screen.getByRole('complementary')).toBe(inspector);
  expect(inspector.scrollTop).toBe(120);
  await userEvent.click(screen.getByRole('button', { name: 'Charlie', exact: true }));
  expect(screen.getByRole('complementary')).not.toBe(inspector);
  expect(screen.getByRole('complementary').scrollTop).toBe(0);
});

it('renders and selects all 25 rows without pagination', async () => {
  setup({
    tasks: Array.from({ length: 25 }, (_, index) =>
      task(`Component ${index + 1}`, index * 10, index * 10 + 50),
    ),
  });
  expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(26);
  await userEvent.click(screen.getByRole('button', { name: 'Component 25', exact: true }));
  expect(
    within(screen.getByRole('complementary')).getByRole('heading', { name: 'Component 25' }),
  ).toBeInTheDocument();
});

it('opens the exact execution identity, not its display name', async () => {
  const { props } = setup({ tasks: [task('Same label', 0, 10, { id: 'execution-42' })] });
  await userEvent.click(screen.getByRole('button', { name: 'Open task in graph' }));
  expect(props.onOpenTask).toHaveBeenCalledWith('execution-42');
});

it('disables graph navigation when parent metadata is incomplete', () => {
  setup({ tasks: [task('Orphan', 0, 10, { graphTarget: undefined })] });
  expect(screen.getByRole('button', { name: 'Open task in graph' })).toBeDisabled();
  expect(screen.getByText(/Graph location unavailable/)).toBeInTheDocument();
});

it('updates live durations, freezes terminal executions, and cleans up the clock', () => {
  vi.useFakeTimers();
  vi.setSystemTime(at(60));
  const { update, unmount } = setup({
    run: { ...run, state: 'RUNNING' },
    tasks: [task('Active', 0, 10, { state: 'RUNNING' })],
  });
  expect(screen.getByRole('button', { name: 'Select Active, 1m 00s' })).toBeInTheDocument();
  act(() => vi.advanceTimersByTime(2000));
  expect(screen.getByRole('button', { name: 'Select Active, 1m 02s' })).toBeInTheDocument();
  update({ run: { ...run, finished_at: at(62) }, tasks: [task('Active', 0, 62)] });
  act(() => vi.advanceTimersByTime(5000));
  expect(screen.getByRole('button', { name: 'Select Active, 1m 02s' })).toBeInTheDocument();
  expect(document.querySelector('.rt-running')).toBeNull();
  unmount();
  expect(vi.getTimerCount()).toBe(0);
});

it('shows loading, empty, and refresh-error states while retaining previous data', () => {
  const { update } = setup({ loading: true });
  expect(screen.getByLabelText('Loading component tasks')).toBeInTheDocument();
  update({ loading: false, tasks: [] });
  expect(screen.getByText(/No component task data yet/)).toBeInTheDocument();
  update({ loading: false, error: new Error('offline') });
  expect(screen.getByText(/Showing the last available snapshot/)).toBeInTheDocument();
  expect(screen.getByRole('table')).toBeInTheDocument();
});
