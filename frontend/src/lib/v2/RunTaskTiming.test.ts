// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { V2PipelineTask, V2Run } from 'src/apisv2/run';
import {
  formatTaskElapsed,
  getRunTaskTiming,
  sortTasksByCreationTime,
  taskTimestamp,
} from './RunTaskTiming';

const base = Date.parse('2026-01-01T00:00:00Z');
const at = (seconds: number) => new Date(base + seconds * 1000);
const run: V2Run = { state: 'SUCCEEDED', created_at: at(0), finished_at: at(300) };
const task = (overrides: Partial<V2PipelineTask> = {}): V2PipelineTask => ({
  task_id: 'task',
  name: 'train',
  type: 'RUNTIME',
  state: 'SUCCEEDED',
  create_time: at(10),
  end_time: at(130),
  ...overrides,
});

it('keeps distinct component iterations and importers, excluding all wrapper task types', () => {
  const groups = ['ROOT', 'DAG', 'LOOP', 'CONDITION', 'CONDITION_BRANCH', 'EXIT_HANDLER'] as const;
  const tasks = [
    ...groups.map((type) => task({ type })),
    task({ task_id: 'iteration-1', type_attributes: { iteration_index: '1' } }),
    task({ task_id: 'iteration-2', type_attributes: { iteration_index: '2' } }),
    task({ task_id: 'importer', type: 'IMPORTER' }),
  ];
  const result = getRunTaskTiming(run, tasks, base);
  expect(result.rows.map((row) => row.id)).toEqual(['iteration-1', 'iteration-2', 'importer']);
  expect(result.rows.map((row) => row.iteration)).toEqual([1, 2, undefined]);
  expect(result.runElapsed).toBe(300_000);
});

it('uses wall time only for active tasks and ignores a stale previous-attempt end', () => {
  const result = getRunTaskTiming(
    { ...run, state: 'RUNNING' },
    [
      task({ state: 'RUNNING', end_time: at(50) }),
      task({ task_id: 'cached', state: 'CACHED', end_time: at(11) }),
    ],
    base + 200_000,
  );
  expect(result.rows[0]).toMatchObject({ elapsed: 190_000, active: true, finished: undefined });
  expect(result.rows[1]).toMatchObject({ elapsed: 1000, active: false });
});

it.each(['SUCCEEDED', 'FAILED', 'CANCELED', 'SKIPPED'] as const)(
  'never extends unreconciled RUNNING tasks after a %s run',
  (state) => {
    const { rows } = getRunTaskTiming(
      { ...run, state },
      [task({ state: 'RUNNING', end_time: at(50) })],
      base + 500_000,
    );
    expect(rows[0]).toMatchObject({ elapsed: undefined, finished: undefined, active: false });
  },
);

it.each([
  { create_time: undefined },
  { end_time: undefined },
  { create_time: new Date(NaN) },
  { end_time: new Date(NaN) },
  { create_time: at(200), end_time: at(10) },
])('does not infer elapsed time from invalid timing: %j', (overrides) => {
  expect(getRunTaskTiming(run, [task(overrides)], base).rows[0].elapsed).toBeUndefined();
});

it('retains legitimate zero-duration tasks and marks retries without splitting attempts', () => {
  const retried = task({
    end_time: at(10),
    state_history: [
      { state: 'RUNNING' },
      { state: 'FAILED' },
      { state: 'RUNNING' },
      { state: 'SUCCEEDED' },
    ],
  });
  expect(getRunTaskTiming(run, [retried], base).rows[0]).toMatchObject({
    elapsed: 0,
    retried: true,
  });
});

it.each([
  ['automatic executor retry', ['RUNNING', 'FAILED', 'SUCCEEDED']],
  ['manual retry after driver failure', ['FAILED', 'RUNNING', 'SUCCEEDED']],
] as const)('recognizes %s without two RUNNING entries', (_name, states) => {
  const retried = task({ state_history: states.map((state) => ({ state })) });
  expect(getRunTaskTiming(run, [retried], base).rows[0]).toMatchObject({
    elapsed: 120_000,
    retried: true,
  });
});

it.each([
  ['FAILED', 'RUNNING'],
  ['RUNNING', 'FAILED', 'CACHED'],
] as const)('recognizes retry progress or cached recovery in %j', (...states) => {
  const retried = task({ state_history: states.map((state) => ({ state })) });
  expect(getRunTaskTiming(run, [retried], base).rows[0].retried).toBe(true);
});

it.each([
  [],
  ['RUNNING', 'SUCCEEDED'],
  ['RUNNING', 'FAILED'],
  ['FAILED'],
  ['RUNNING', 'CACHED'],
  ['RUNNING', 'SKIPPED'],
  ['RUNNING', 'FAILED', 'FAILED'],
  ['RUNNING', 'FAILED', 'RUNTIME_STATE_UNSPECIFIED'],
] as const)('does not infer a retry from a single attempt history %j', (...states) => {
  const singleAttempt = task({ state_history: states.map((state) => ({ state })) });
  expect(getRunTaskTiming(run, [singleAttempt], base).rows[0].retried).toBe(false);
});

it.each<{ name: string; pods: NonNullable<V2PipelineTask['pods']>; retried: boolean }>([
  {
    name: 'driver and executor',
    pods: [
      { type: 'DRIVER', name: 'driver', uid: 'driver-uid' },
      { type: 'EXECUTOR', name: 'executor', uid: 'executor-uid' },
    ],
    retried: false,
  },
  {
    name: 'duplicate executor records',
    pods: [
      { type: 'EXECUTOR', name: 'executor', uid: 'uid' },
      { type: 'EXECUTOR', name: 'executor', uid: 'uid' },
    ],
    retried: false,
  },
  {
    name: 'executor enriched with UID',
    pods: [
      { type: 'EXECUTOR', name: 'executor' },
      { type: 'EXECUTOR', name: 'executor', uid: 'uid' },
    ],
    retried: false,
  },
  {
    name: 'distinct executor UIDs',
    pods: [
      { type: 'EXECUTOR', name: 'executor', uid: 'uid-1' },
      { type: 'EXECUTOR', name: 'executor', uid: 'uid-2' },
    ],
    retried: true,
  },
  {
    name: 'distinct executor names',
    pods: [
      { type: 'EXECUTOR', name: 'executor-1' },
      { type: 'EXECUTOR', name: 'executor-2' },
    ],
    retried: true,
  },
  {
    name: 'unidentified executor records',
    pods: [{ type: 'EXECUTOR' }, { type: 'EXECUTOR' }],
    retried: false,
  },
  {
    name: 'multiple drivers only',
    pods: [
      { type: 'DRIVER', name: 'driver-1' },
      { type: 'DRIVER', name: 'driver-2' },
    ],
    retried: false,
  },
])('uses executor attempt evidence: $name', ({ pods, retried }) => {
  const failed = task({
    state: 'FAILED',
    state_history: [{ state: 'RUNNING' }, { state: 'FAILED' }],
    pods,
  });
  expect(getRunTaskTiming({ ...run, state: 'FAILED' }, [failed], base).rows[0].retried).toBe(
    retried,
  );
});

it('uses the earliest component timestamp as a chart fallback, without inventing run elapsed', () => {
  const result = getRunTaskTiming(
    { state: 'SUCCEEDED' },
    [task({ create_time: at(100) }), task({ task_id: 'first', create_time: at(5) })],
    base,
  );
  expect(result.origin).toBe(base + 5000);
  expect(result.runElapsed).toBeUndefined();
  expect(result.span).toBe(180_000);
});

it('orders tasks chronologically with missing timestamps last and stable ties', () => {
  const rows = getRunTaskTiming(
    run,
    [
      task({ task_id: 'later', create_time: at(100) }),
      task({ task_id: 'unknown-b', create_time: undefined }),
      task({ task_id: 'same-time-b', create_time: at(10) }),
      task({ task_id: 'earliest', create_time: at(2) }),
      task({ task_id: 'same-time-a', create_time: at(10) }),
      task({ task_id: 'unknown-a', create_time: undefined }),
    ],
    base,
  ).rows;
  expect(sortTasksByCreationTime(rows).map((row) => row.id)).toEqual([
    'earliest',
    'same-time-a',
    'same-time-b',
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
] as const)('formats %s as %s', (ms, expected) => {
  expect(formatTaskElapsed(ms)).toBe(expected);
});

it.each(['', ' ', '-1', '1.5', 'invalid'])(
  'does not invent an iteration from %j',
  (iteration_index) => {
    const { rows } = getRunTaskTiming(run, [task({ type_attributes: { iteration_index } })], base);
    expect(rows[0].iteration).toBeUndefined();
  },
);

it('rejects epoch-zero and invalid timestamp sentinels', () => {
  expect(taskTimestamp(new Date(0))).toBeUndefined();
  expect(taskTimestamp(new Date(NaN))).toBeUndefined();
});
