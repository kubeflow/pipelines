// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { V2beta1Run } from 'src/apisv2beta1/run';
import { hasFinishedV2 } from 'src/lib/StatusUtils';
import { TimelineTask } from './MlmdTaskTiming';

export interface TaskTiming {
  task: TimelineTask;
  id: string;
  label: string;
  created?: number;
  finished?: number;
  end?: number;
  elapsed?: number;
  iteration?: number;
  active: boolean;
}

export function taskTimestamp(value?: Date): number | undefined {
  const ms = value?.getTime();
  return ms !== undefined && Number.isFinite(ms) && ms > 0 ? ms : undefined;
}

function elapsedBetween(start?: number, end?: number): number | undefined {
  return start !== undefined && end !== undefined && end >= start ? end - start : undefined;
}

/** MLMD's last update is an approximate end, not a recorded completion transition. */
export function getRunTaskTiming(run: V2beta1Run, tasks: TimelineTask[], now: number) {
  const active = !hasFinishedV2(run.state);
  const rows: TaskTiming[] = tasks.map((task) => {
    const created = taskTimestamp(task.createdAt);
    const running = active && task.state === 'RUNNING';
    // A terminal run with stale RUNNING metadata must not keep accumulating elapsed time.
    // Skipped tasks never executed, so their metadata-write interval is not a task duration.
    const finished =
      task.state === 'CACHED' || task.state === 'SUCCEEDED' || task.state === 'FAILED'
        ? taskTimestamp(task.updatedAt)
        : undefined;
    const end = running ? now : finished;
    return {
      task,
      id: task.id,
      label: task.name,
      created,
      finished,
      end,
      elapsed: elapsedBetween(created, end),
      iteration: task.iteration,
      active: running,
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
