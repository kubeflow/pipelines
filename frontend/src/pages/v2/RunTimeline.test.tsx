// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { act, fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { getContrastRatio } from '@mui/material/styles';
import { V2beta1PipelineTask, V2beta1Run } from 'src/apisv2beta1/run';
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
  overrides: Partial<V2beta1PipelineTask> = {},
): V2beta1PipelineTask => ({
  task_id: name,
  name,
  display_name: name,
  type: 'RUNTIME',
  state: 'SUCCEEDED',
  create_time: at(start),
  end_time: finish === undefined ? undefined : at(finish),
  ...overrides,
});
const tasks = [
  task('Root', 0, 300, { type: 'ROOT' }),
  task('Alpha', 10, 19),
  task('Bravo', 20, 140, { type_attributes: { iteration_index: '2' } }),
  task('Charlie', 30, 80),
  task('Unknown', 0, undefined, { state: 'SKIPPED' }),
  task('Cached', 5, 5, { state: 'CACHED' }),
];
function setup(overrides: Partial<RunTimelineProps> = {}) {
  const props: RunTimelineProps = {
    run,
    tasks,
    loading: false,
    onOpenTask: vi.fn(),
    ...overrides,
  };
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
] as const)('renders %s status text with at least 4.5:1 contrast on white', (state) => {
  setup({ tasks: [task('Contrast check', 0, 10, { state })] });
  const inspector = screen.getByRole('complementary', { name: 'Selected task' });
  const status = inspector.querySelector('dl .rt-status') as HTMLElement;
  expect(getContrastRatio(getComputedStyle(status).color, '#ffffff')).toBeGreaterThanOrEqual(4.5);
});

it('renders components chronologically and preserves selection across refreshes', async () => {
  const { update } = setup();
  const chart = screen.getByRole('table', { name: 'Component timeline timings' });
  expect(
    within(chart)
      .getAllByRole('columnheader')
      .map((header) => header.textContent),
  ).toEqual(['Component', '', 'Elapsed']);
  expect(within(chart).queryByRole('columnheader', { name: 'Status' })).not.toBeInTheDocument();
  for (const row of within(chart).getAllByRole('row').slice(1)) {
    expect(within(row).getAllByRole('cell')).toHaveLength(3);
  }
  const names = within(chart)
    .getAllByRole('row')
    .slice(1)
    .map((row) => within(row).getAllByRole('button')[0].textContent);
  expect(names).toEqual(['Unknown', 'Cached', 'Alpha', 'Bravo', 'Charlie']);
  expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Root' })).not.toBeInTheDocument();
  await userEvent.click(within(chart).getByRole('button', { name: 'Alpha', exact: true }));
  update({ tasks: [...tasks, task('Longest new task', 1, 299)] });
  expect(
    within(screen.getByRole('complementary', { name: 'Selected task' })).getByRole('heading', {
      name: 'Alpha',
    }),
  ).toBeInTheDocument();
});

it.each(['name', 'bar', 'track', 'elapsed', 'row'] as const)(
  'opens component details when clicking the %s area',
  async (area) => {
    setup();
    const chart = screen.getByRole('table', { name: 'Component timeline timings' });
    const name = within(chart).getByRole('button', { name: 'Alpha', exact: true });
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
      within(screen.getByRole('complementary', { name: 'Selected task' })).getByRole('heading', {
        name: 'Alpha',
      }),
    ).toBeInTheDocument();
    expect(name).toHaveAttribute('aria-pressed', 'true');
  },
);

it('selects untimed components by clicking their placeholder', async () => {
  setup();
  await userEvent.click(screen.getByText('Timing unavailable'));
  expect(
    within(screen.getByRole('complementary', { name: 'Selected task' })).getByRole('heading', {
      name: 'Unknown',
    }),
  ).toBeInTheDocument();
});

it.each(['name', 'bar'] as const)(
  'keeps keyboard selection through the %s button',
  async (target) => {
    setup();
    const button = screen.getByRole('button', {
      name: target === 'name' ? 'Alpha' : 'Select Alpha, 9s',
      exact: true,
    });
    button.focus();
    await userEvent.keyboard(target === 'name' ? '{Enter}' : ' ');
    expect(
      within(screen.getByRole('complementary', { name: 'Selected task' })).getByRole('heading', {
        name: 'Alpha',
      }),
    ).toBeInTheDocument();
  },
);

it('does not show hover text over a bar', () => {
  vi.useFakeTimers();
  setup();
  const bar = screen.getByRole('button', { name: 'Select Alpha, 9s' });
  fireEvent.mouseOver(bar);
  act(() => vi.advanceTimersByTime(1000));
  expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  expect(bar).not.toHaveAttribute('title');
});

it('keeps status in metadata as selection and task state change, without a relative offset', async () => {
  const { update } = setup();
  const inspector = screen.getByRole('complementary', { name: 'Selected task' });
  const metadata = inspector.querySelector('dl') as HTMLElement;
  expect(within(inspector).getByText('SELECTED TASK')).toBeInTheDocument();
  expect(Array.from(metadata.querySelectorAll('dt')).map((field) => field.textContent)).toEqual([
    'Status',
    'Created',
    'Finished',
    'Iteration',
  ]);
  const statusValue = within(metadata).getByText('Status').nextElementSibling;
  expect(statusValue).toHaveTextContent('Succeeded');
  expect(statusValue?.textContent).not.toContain('●');
  expect(inspector.querySelector('.rt-status-badge')).toBeNull();
  expect(inspector.querySelector(':scope > .rt-status')).toBeNull();
  expect(within(inspector).queryByText(/into run/)).not.toBeInTheDocument();
  const chart = screen.getByRole('table', { name: 'Component timeline timings' });
  await userEvent.click(within(chart).getByRole('button', { name: 'Cached', exact: true }));
  const cachedMetadata = screen
    .getByRole('complementary', { name: 'Selected task' })
    .querySelector('dl') as HTMLElement;
  const cachedStatus = within(cachedMetadata).getByText('Status').nextElementSibling;
  expect(cachedStatus).toHaveTextContent('Cached');
  update({
    tasks: tasks.map((item) => (item.task_id === 'Cached' ? { ...item, state: 'FAILED' } : item)),
  });
  expect(cachedStatus).toHaveTextContent('Failed');
  expect(cachedMetadata.querySelectorAll('.rt-status')).toHaveLength(1);
});

it('preserves details scrolling on refresh but resets it when selecting another task', async () => {
  const { update } = setup();
  const chart = screen.getByRole('table', { name: 'Component timeline timings' });
  await userEvent.click(within(chart).getByRole('button', { name: 'Alpha', exact: true }));
  const inspector = screen.getByRole('complementary', { name: 'Selected task' });
  inspector.scrollTop = 120;
  update({ tasks: tasks.map((item) => ({ ...item })) });
  expect(screen.getByRole('complementary', { name: 'Selected task' })).toBe(inspector);
  expect(inspector.scrollTop).toBe(120);
  await userEvent.click(within(chart).getByRole('button', { name: 'Charlie', exact: true }));
  const nextInspector = screen.getByRole('complementary', { name: 'Selected task' });
  expect(nextInspector).not.toBe(inspector);
  expect(nextInspector.scrollTop).toBe(0);
});

it('renders all 25 components and selects the last row without pagination', async () => {
  setup({
    tasks: Array.from({ length: 25 }, (_, index) =>
      task(`Component ${index + 1}`, index * 10, index * 10 + 50),
    ),
  });
  const chart = screen.getByRole('table', { name: 'Component timeline timings' });
  expect(within(chart).getAllByRole('row')).toHaveLength(26);
  expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
  const lastRow = within(chart)
    .getByRole('button', { name: 'Component 25', exact: true })
    .closest('[role="row"]') as HTMLElement;
  await userEvent.click(within(lastRow).getAllByRole('cell')[2]);
  expect(
    within(screen.getByRole('complementary', { name: 'Selected task' })).getByRole('heading', {
      name: 'Component 25',
      exact: true,
    }),
  ).toBeInTheDocument();
});

it('opens the selected component in the graph', async () => {
  const { props } = setup();
  const chart = screen.getByRole('table', { name: 'Component timeline timings' });
  await userEvent.click(within(chart).getByRole('button', { name: 'Alpha', exact: true }));
  await userEvent.click(screen.getByRole('button', { name: 'Open task in graph' }));
  expect(props.onOpenTask).toHaveBeenCalledWith('Alpha');
});

it('distinguishes cached, retried, and unknown timing in task details', async () => {
  setup({
    tasks: [
      ...tasks,
      task('Retry', 0, 200, {
        state_history: [{ state: 'RUNNING' }, { state: 'FAILED' }, { state: 'RUNNING' }],
      }),
    ],
  });
  expect(
    screen.getByText('This task span includes retries and waiting between attempts.'),
  ).toBeInTheDocument();
  const chart = screen.getByRole('table', { name: 'Component timeline timings' });
  await userEvent.click(within(chart).getByRole('button', { name: 'Cached', exact: true }));
  expect(screen.getByText(/This span is cache-resolution overhead/)).toBeInTheDocument();
  expect(within(chart).getByRole('button', { name: 'Select Cached, 0s' })).toBeInTheDocument();
  await userEvent.click(within(chart).getByRole('button', { name: 'Unknown', exact: true }));
  expect(
    screen.getByText('Insufficient timestamps. No duration has been inferred.'),
  ).toBeInTheDocument();
});

it('updates active elapsed times, freezes terminal tasks, and cleans up the clock', () => {
  vi.useFakeTimers();
  vi.setSystemTime(at(60));
  const activeTask = task('Active', 0, undefined, { state: 'RUNNING' });
  const runningRun: V2beta1Run = { ...run, state: 'RUNNING', finished_at: undefined };
  const { update, unmount } = setup({ run: runningRun, tasks: [activeTask] });
  const chart = screen.getByRole('table', { name: 'Component timeline timings' });
  expect(within(chart).getByRole('button', { name: 'Select Active, 1m 00s' })).toBeInTheDocument();
  act(() => vi.advanceTimersByTime(2000));
  expect(within(chart).getByRole('button', { name: 'Select Active, 1m 02s' })).toBeInTheDocument();
  update({ run: { ...run, finished_at: at(62) }, tasks: [task('Active', 0, 62)] });
  act(() => vi.advanceTimersByTime(5000));
  expect(within(chart).getByRole('button', { name: 'Select Active, 1m 02s' })).toBeInTheDocument();
  expect(document.querySelector('.rt-running')).toBeNull();
  unmount();
  expect(vi.getTimerCount()).toBe(0);
});

it.each(['RUNNING', 'SUCCEEDED'] as const)('keeps the header minimal for a %s run', (state) => {
  setup({ run: { ...run, state } });
  expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
  expect(document.querySelector('.rt-legend')).toBeNull();
  expect(screen.queryByText(/Elapsed time includes startup and waiting/)).not.toBeInTheDocument();
  expect(screen.queryByText(/Live · refreshes/)).not.toBeInTheDocument();
  expect(screen.queryByText('Completed snapshot')).not.toBeInTheDocument();
  expect(screen.queryByText(/Synthetic local data/)).not.toBeInTheDocument();
  expect(screen.queryByText('EXECUTION INSIGHTS')).not.toBeInTheDocument();
  expect(screen.queryByRole('heading', { name: /timeline/i })).not.toBeInTheDocument();
  expect(screen.queryByRole('group', { name: 'Run timing summary' })).not.toBeInTheDocument();
  expect(screen.queryByText('Run elapsed')).not.toBeInTheDocument();
  expect(screen.queryByText('Component tasks')).not.toBeInTheDocument();
  expect(screen.queryByText('Longest task span')).not.toBeInTheDocument();
  expect(screen.queryByText(/into run/)).not.toBeInTheDocument();
  const chart = screen.getByRole('table', { name: 'Component timeline timings' });
  expect(
    within(chart)
      .getAllByRole('columnheader')
      .map((header) => header.textContent),
  ).toEqual(['Component', '', 'Elapsed']);
});

it('shows loading, empty, and refresh-error states', () => {
  const { update } = setup({ loading: true });
  expect(screen.getByLabelText('Loading component tasks')).toBeInTheDocument();
  update({ loading: false, tasks: [] });
  expect(screen.getByText(/No component task data yet/)).toBeInTheDocument();
  update({ loading: false, error: new Error('offline') });
  expect(screen.getByText(/Showing the last available snapshot/)).toBeInTheDocument();
  expect(screen.getByRole('table', { name: 'Component timeline timings' })).toBeInTheDocument();
});
