// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { V2beta1Run } from 'src/apisv2beta1/run';
import { TimelineTask } from './MlmdTaskTiming';
import {
  formatTaskElapsed,
  getRunTaskTiming,
  sortTasksByCreationTime,
  taskTimestamp,
} from './RunTaskTiming';

const base = Date.parse('2026-01-01T00:00:00Z');
const at = (seconds: number) => new Date(base + seconds * 1000);
const run: V2beta1Run = { state: 'SUCCEEDED', created_at: at(0), finished_at: at(300) };
const task = (overrides: Partial<TimelineTask> = {}): TimelineTask => ({
  id: 'task',
  name: 'train',
  state: 'SUCCEEDED',
  createdAt: at(10),
  updatedAt: at(130),
  ...overrides,
});

it('uses wall time only for active tasks, not a running execution’s last metadata update', () => {
  const { rows } = getRunTaskTiming(
    { ...run, state: 'RUNNING' },
    [
      task({ state: 'RUNNING', updatedAt: at(50) }),
      task({ id: 'cached', state: 'CACHED', updatedAt: at(11) }),
    ],
    base + 200_000,
  );
  expect(rows[0]).toMatchObject({ elapsed: 190_000, active: true, finished: undefined });
  expect(rows[1]).toMatchObject({ elapsed: 1000, active: false });
});

it.each(['SUCCEEDED', 'FAILED', 'CANCELED', 'SKIPPED'] as const)(
  'never extends stale RUNNING metadata after a %s run',
  (state) => {
    const { rows } = getRunTaskTiming(
      { ...run, state },
      [task({ state: 'RUNNING' })],
      base + 500_000,
    );
    expect(rows[0]).toMatchObject({ elapsed: undefined, finished: undefined, active: false });
  },
);

it.each(['PENDING', 'SKIPPED', undefined] as const)('does not time a %s execution', (state) => {
  expect(getRunTaskTiming(run, [task({ state })], base).rows[0].elapsed).toBeUndefined();
});

it.each([
  { createdAt: undefined },
  { updatedAt: undefined },
  { createdAt: new Date(NaN) },
  { updatedAt: new Date(NaN) },
  { createdAt: at(200), updatedAt: at(10) },
])('does not infer elapsed time from invalid timing: %j', (overrides) => {
  expect(getRunTaskTiming(run, [task(overrides)], base).rows[0].elapsed).toBeUndefined();
});

it('retains legitimate zero-duration execution metadata intervals', () => {
  expect(getRunTaskTiming(run, [task({ updatedAt: at(10) })], base).rows[0].elapsed).toBe(0);
});

it('uses earliest component time as chart fallback, without inventing run elapsed', () => {
  const result = getRunTaskTiming(
    { state: 'SUCCEEDED' },
    [task({ createdAt: at(100) }), task({ id: 'first', createdAt: at(5) })],
    base,
  );
  expect(result.origin).toBe(base + 5000);
  expect(result.runElapsed).toBeUndefined();
  expect(result.span).toBe(180_000);
});

it('orders components with missing timestamps last and stable identity ties', () => {
  const rows = getRunTaskTiming(
    run,
    [
      task({ id: 'later', createdAt: at(100) }),
      task({ id: 'unknown-b', createdAt: undefined }),
      task({ id: 'same-b' }),
      task({ id: 'earliest', createdAt: at(2) }),
      task({ id: 'same-a' }),
      task({ id: 'unknown-a', createdAt: undefined }),
    ],
    base,
  ).rows;
  expect(sortTasksByCreationTime(rows).map((row) => row.id)).toEqual([
    'earliest',
    'same-a',
    'same-b',
    'later',
    'unknown-a',
    'unknown-b',
  ]);
  expect(rows[0].id).toBe('later');
});

it.each([
  [undefined, '—'],
  [0, '0s'],
  [59_999, '59s'],
  [60_000, '1m 00s'],
  [3_661_000, '1h 01m 01s'],
  [-1, '—'],
  [NaN, '—'],
] as const)('formats %s as %s', (ms, expected) => expect(formatTaskElapsed(ms)).toBe(expected));

it('rejects epoch-zero and invalid sentinels', () => {
  expect(taskTimestamp(new Date(0))).toBeUndefined();
  expect(taskTimestamp(new Date(NaN))).toBeUndefined();
});
