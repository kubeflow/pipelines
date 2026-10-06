// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { V2beta1PipelineTask, V2beta1Run } from 'src/apisv2beta1/run';
import { hasFinishedV2 } from 'src/lib/StatusUtils';
import { isTaskFinished } from './RuntimeArtifactUtils';
import { getTaskDisplayName } from './RunTaskUtils';

const GROUP_TYPES = new Set([
  'ROOT',
  'DAG',
  'LOOP',
  'CONDITION',
  'CONDITION_BRANCH',
  'EXIT_HANDLER',
]);

export interface TaskTiming {
  task: V2beta1PipelineTask;
  id: string;
  label: string;
  created?: number;
  finished?: number;
  end?: number;
  elapsed?: number;
  iteration?: number;
  active: boolean;
  retried: boolean;
}

export function taskTimestamp(value?: Date): number | undefined {
  const ms = value?.getTime();
  return ms !== undefined && Number.isFinite(ms) && ms > 0 ? ms : undefined;
}

function elapsedBetween(start?: number, end?: number): number | undefined {
  return start !== undefined && end !== undefined && end >= start ? end - start : undefined;
}

/** Build component spans without double-counting their enclosing DAG/loop groups. */
export function getRunTaskTiming(run: V2beta1Run, tasks: V2beta1PipelineTask[], now: number) {
  const active = !hasFinishedV2(run.state);
  const rows: TaskTiming[] = tasks
    .filter((task) => !GROUP_TYPES.has(task.type || ''))
    .map((task, index) => {
      const created = taskTimestamp(task.create_time);
      const running = active && task.state === 'RUNNING';
      // Ignore a stale previous-attempt end time while a task is running. A terminal
      // run with an unreconciled RUNNING task must not keep accumulating elapsed time.
      const finished = isTaskFinished(task.state) ? taskTimestamp(task.end_time) : undefined;
      const end = running ? now : finished;
      const rawIteration = task.type_attributes?.iteration_index;
      const iteration = rawIteration?.trim() ? Number(rawIteration) : NaN;
      return {
        task,
        id: task.task_id || `unknown-task-${index}`,
        label: getTaskDisplayName(task),
        created,
        finished,
        end,
        elapsed: elapsedBetween(created, end),
        iteration: Number.isSafeInteger(iteration) && iteration >= 0 ? iteration : undefined,
        active: running,
        retried:
          (task.state_history || []).filter((status) => status.state === 'RUNNING').length > 1,
      };
    });
  const runStart = taskTimestamp(run.created_at);
  const runEnd = active ? now : taskTimestamp(run.finished_at);
  const firstTask = rows.reduce<number | undefined>(
    (earliest, row) =>
      row.created === undefined ? earliest : Math.min(earliest ?? row.created, row.created),
    undefined,
  );
  const origin = Math.min(runStart ?? firstTask ?? now, firstTask ?? runStart ?? now);
  const latest = rows.reduce((end, row) => Math.max(end, row.end ?? origin), runEnd ?? origin);
  const span = Math.max(60_000, Math.ceil((latest - origin) / 60_000) * 60_000);
  return { rows, origin, span, active, runElapsed: elapsedBetween(runStart, runEnd) };
}

/** Chronological component order, with missing creation times last and stable ties. */
export function sortTasksByCreationTime(rows: TaskTiming[]): TaskTiming[] {
  return [...rows].sort((a, b) => {
    if (a.created === undefined && b.created !== undefined) return 1;
    if (b.created === undefined && a.created !== undefined) return -1;
    return (a.created ?? 0) - (b.created ?? 0) || a.id.localeCompare(b.id);
  });
}

export function formatTaskElapsed(ms?: number): string {
  if (ms === undefined || !Number.isFinite(ms) || ms < 0) return '—';
  const seconds = Math.floor(ms / 1000);
  if (seconds >= 3600)
    return `${Math.floor(seconds / 3600)}h ${String(Math.floor(seconds / 60) % 60).padStart(2, '0')}m ${String(seconds % 60).padStart(2, '0')}s`;
  return seconds >= 60
    ? `${Math.floor(seconds / 60)}m ${String(seconds % 60).padStart(2, '0')}s`
    : `${seconds}s`;
}
