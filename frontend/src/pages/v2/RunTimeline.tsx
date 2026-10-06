// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { useEffect, useState } from 'react';
import { Alert, Button, CircularProgress } from '@mui/material';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { TimelineTask } from 'src/lib/v2/MlmdTaskTiming';
import { hasFinishedV2 } from 'src/lib/StatusUtils';
import { formatDateString } from 'src/lib/Utils';
import {
  formatTaskElapsed,
  getRunTaskTiming,
  sortTasksByCreationTime,
  taskTimestamp,
  TaskTiming,
} from 'src/lib/v2/RunTaskTiming';
import './RunTimeline.css';

const NEUTRAL_STATE_COLOR = '#637087';
const STATE_COLORS: Record<string, string> = {
  SUCCEEDED: '#24864b',
  RUNNING: '#1a73e8',
  FAILED: '#c43a35',
  CACHED: '#8152b8',
  SKIPPED: NEUTRAL_STATE_COLOR,
};
function timeLabel(ms?: number): string {
  return ms === undefined ? 'Not recorded' : formatDateString(new Date(ms));
}
function Status({ state }: { state?: string }) {
  const label =
    state && state !== 'RUNTIME_STATE_UNSPECIFIED'
      ? state[0] + state.slice(1).toLowerCase()
      : 'Unknown';
  return (
    <span className='rt-status' style={{ color: STATE_COLORS[state || ''] || NEUTRAL_STATE_COLOR }}>
      {label}
    </span>
  );
}

export interface RunTimelineProps {
  run: V2beta1Run;
  tasks: TimelineTask[];
  loading: boolean;
  error?: Error | null;
  onOpenTask: (taskId: string) => void;
}

export default function RunTimeline({ run, tasks, loading, error, onOpenTask }: RunTimelineProps) {
  const [now, setNow] = useState(Date.now);
  const [selectedId, setSelectedId] = useState<string>();
  const active = !hasFinishedV2(run.state);
  useEffect(() => {
    if (!active) return;
    // External synchronization with wall time, independent of the existing task poller.
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [active]);

  const timing = getRunTaskTiming(run, tasks, now);
  const ordered = sortTasksByCreationTime(timing.rows);
  const longest = timing.rows.reduce<TaskTiming | undefined>(
    (current, row) =>
      row.elapsed !== undefined && (current?.elapsed === undefined || row.elapsed > current.elapsed)
        ? row
        : current,
    undefined,
  );
  const selected = timing.rows.find((row) => row.id === selectedId) || longest || timing.rows[0];

  return (
    <section className='run-timeline' aria-label='Run timeline'>
      {error && (
        <Alert severity='warning' className='rt-refresh-warning'>
          Unable to refresh component tasks.{' '}
          {tasks.length ? 'Showing the last available snapshot. ' : ''}Refresh the page to try
          again.
        </Alert>
      )}
      {loading ? (
        <CircularProgress aria-label='Loading component tasks' />
      ) : !timing.rows.length ? (
        <Alert severity='info'>
          No component task data yet. Tasks appear as the pipeline progresses.
        </Alert>
      ) : (
        <div className='rt-split'>
          <div className='rt-panel'>
            <TimelineChart
              rows={ordered}
              origin={timing.origin}
              span={timing.span}
              now={active ? now : undefined}
              selectedId={selected?.id}
              onSelect={setSelectedId}
            />
          </div>
          <TaskInspector key={selected?.id} row={selected} onOpenTask={onOpenTask} />
        </div>
      )}
    </section>
  );
}

function TaskInspector({
  row,
  onOpenTask,
}: {
  row?: TaskTiming;
  onOpenTask: (id: string) => void;
}) {
  if (!row) return null;
  return (
    <aside className='rt-panel rt-inspector' aria-label='Selected task'>
      <div className='rt-eyebrow'>SELECTED TASK</div>
      <h2>{row.label}</h2>
      <div className='rt-big-duration'>
        {formatTaskElapsed(row.elapsed)}
        <small>Approximate MLMD elapsed</small>
      </div>
      <dl>
        <dt>Status</dt>
        <dd>
          <Status state={row.task.state} />
        </dd>
        <dt>Created</dt>
        <dd>{timeLabel(row.created)}</dd>
        <dt>Last updated</dt>
        <dd>{timeLabel(taskTimestamp(row.task.updatedAt))}</dd>
        {row.iteration !== undefined && (
          <>
            <dt>Iteration</dt>
            <dd>{row.iteration}</dd>
          </>
        )}
      </dl>
      <p>
        Elapsed time uses MLMD creation and last-update timestamps, not exact component start and
        finish times. It can include waiting, metadata updates, and retries.
      </p>
      {row.task.state === 'CACHED' && (
        <Alert severity='info'>
          Cache hit. This is not component computation time; earlier retry attempts may be included.
        </Alert>
      )}
      {row.elapsed === undefined && (
        <Alert severity='info'>Insufficient timestamps. No duration has been inferred.</Alert>
      )}
      <p>MLMD stores only the latest state. State history and retry attempts are unavailable.</p>
      {!row.task.graphTarget && (
        <Alert severity='info'>
          Graph location unavailable. Refresh after the parent execution metadata arrives.
        </Alert>
      )}
      <Button
        size='small'
        variant='outlined'
        disabled={!row.task.graphTarget}
        onClick={() => onOpenTask(row.id)}
      >
        Open task in graph
      </Button>
    </aside>
  );
}

function TimelineChart({
  rows,
  origin,
  span,
  now,
  selectedId,
  onSelect,
}: {
  rows: TaskTiming[];
  origin: number;
  span: number;
  now?: number;
  selectedId?: string;
  onSelect: (id: string) => void;
}) {
  const nowPosition =
    now === undefined ? undefined : Math.max(0, Math.min(100, ((now - origin) / span) * 100));
  return (
    <div className='rt-chart-scroll'>
      <div className='rt-chart' role='table' aria-label='Component timeline timings'>
        <div className='rt-chart-row rt-axis' role='row'>
          <span role='columnheader'>Component</span>
          <span role='columnheader' aria-label='Timeline' />
          <span role='columnheader' className='rt-row-duration'>
            Elapsed
          </span>
        </div>
        {rows.map((row) => {
          const left = Math.max(
            0,
            Math.min(100, (((row.created ?? origin) - origin) / span) * 100),
          );
          const width = Math.max(0, Math.min(100 - left, ((row.elapsed || 0) / span) * 100));
          return (
            <div
              key={row.id}
              role='row'
              className={`rt-chart-row ${selectedId === row.id ? 'rt-selected' : ''}`}
              onClick={() => onSelect(row.id)}
            >
              <div className='rt-task-label' role='cell'>
                <button aria-pressed={selectedId === row.id} title={row.label}>
                  {row.label}
                </button>
                {row.iteration !== undefined && (
                  <small title='Loop iteration'>[{row.iteration}]</small>
                )}
              </div>
              <div className='rt-track' role='cell'>
                {row.elapsed !== undefined ? (
                  <button
                    aria-label={`Select ${row.label}, ${formatTaskElapsed(row.elapsed)}`}
                    aria-pressed={selectedId === row.id}
                    className={`rt-bar ${row.active ? 'rt-running' : ''} ${row.task.state === 'CACHED' ? 'rt-cached' : ''}`}
                    style={{
                      left: `${left}%`,
                      width: `${width}%`,
                      backgroundColor: STATE_COLORS[row.task.state || ''] || NEUTRAL_STATE_COLOR,
                    }}
                  />
                ) : (
                  <span className='rt-untimed'>Timing unavailable</span>
                )}
                {nowPosition !== undefined && (
                  <span className='rt-now-line' style={{ left: `${nowPosition}%` }} />
                )}
              </div>
              <span role='cell' className='rt-row-duration'>
                {formatTaskElapsed(row.elapsed)}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
