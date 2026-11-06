// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { act, cleanup, render, screen } from '@testing-library/react';
import { V2Run, V2RuntimeState } from 'src/apisv2/run';
import { RunElapsed } from './RunElapsed';

const created = new Date('2026-10-19T12:00:00Z');
const started = new Date('2026-10-19T12:01:00Z');
const running: V2Run = {
  state: V2RuntimeState.RUNNING,
  created_at: created,
  state_history: [{ state: V2RuntimeState.RUNNING, update_time: started }],
};
beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date('2026-10-19T12:02:00Z'));
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});
it('ticks from the actual running transition and stops when finished or unmounted', () => {
  const { rerender, unmount } = render(<RunElapsed run={running} />);
  expect(screen.getByText('Elapsed 0:01:00')).toBeVisible();
  act(() => vi.advanceTimersByTime(2000));
  expect(screen.getByText('Elapsed 0:01:02')).toBeVisible();
  rerender(
    <RunElapsed
      run={{
        ...running,
        state: V2RuntimeState.SUCCEEDED,
        finished_at: new Date('2026-10-19T12:01:30Z'),
      }}
    />,
  );
  expect(screen.getByText('Elapsed 0:00:30')).toBeVisible();
  expect(vi.getTimerCount()).toBe(0);
  rerender(<RunElapsed run={running} />);
  unmount();
  expect(vi.getTimerCount()).toBe(0);
});
it('keeps canceling runs ticking and preserves the start across refreshed run objects', () => {
  const { rerender } = render(<RunElapsed run={running} />);
  act(() => vi.advanceTimersByTime(1000));
  rerender(<RunElapsed run={{ ...running, state: V2RuntimeState.CANCELING }} />);
  act(() => vi.advanceTimersByTime(1000));
  expect(screen.getByText('Elapsed 0:01:02')).toBeVisible();
  expect(vi.getTimerCount()).toBe(1);
});
it('distinguishes pending runs, creation-time fallback, and unavailable timestamps', () => {
  const { rerender } = render(<RunElapsed run={{ ...running, state: V2RuntimeState.PENDING }} />);
  expect(screen.getByText('Not started')).toBeVisible();
  expect(vi.getTimerCount()).toBe(0);
  rerender(<RunElapsed run={{ state: V2RuntimeState.RUNNING, created_at: created }} />);
  expect(screen.getByText('Since created 0:02:00')).toBeVisible();
  rerender(<RunElapsed run={{ state: V2RuntimeState.RUNNING, created_at: new Date('invalid') }} />);
  expect(screen.getByText('Elapsed unavailable')).toBeVisible();
});
