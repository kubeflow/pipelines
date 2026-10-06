// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { useEffect, useState } from 'react';
import { Alert, Button, CircularProgress, Tooltip } from '@mui/material';
import { V2beta1PipelineTask, V2beta1Run } from 'src/apisv2beta1/run';
import { hasFinishedV2 } from 'src/lib/StatusUtils';
import { formatDateString } from 'src/lib/Utils';
import {
  formatTaskElapsed,
  getRunTaskTiming,
  sortTasksByCreationTime,
  taskTimestamp,
  TaskTiming,
} from 'src/lib/v2/RunTaskTiming';
import './RunWaterfall.css';

const STATE_COLORS: Record<string, string> = {
  SUCCEEDED: '#24864b',
  RUNNING: '#1a73e8',
  FAILED: '#c43a35',
  CACHED: '#8152b8',
  SKIPPED: '#7b8492',
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
    <span className='rt-status' style={{ color: STATE_COLORS[state || ''] || '#7b8492' }}>
      {label}
    </span>
  );
}

export interface RunWaterfallProps {
  run: V2beta1Run;
  tasks: V2beta1PipelineTask[];
  loading: boolean;
  error?: Error | null;
  onOpenTask: (taskId: string) => void;
}

export default function RunWaterfall({
  run,
  tasks,
  loading,
  error,
  onOpenTask,
}: RunWaterfallProps) {
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
    <section className='run-waterfall' aria-label='Run waterfall'>
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
            <Waterfall
              rows={ordered}
              origin={timing.origin}
              span={timing.span}
              now={active ? now : undefined}
              selectedId={selected?.id}
              onSelect={setSelectedId}
            />
          </div>
          <TaskInspector row={selected} onOpenTask={onOpenTask} />
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
        <small>{row.retried ? 'Elapsed across retries' : 'Task elapsed'}</small>
      </div>
      <dl>
        <dt>Status</dt>
        <dd>
          <Status state={row.task.state} />
        </dd>
        <dt>Created</dt>
        <dd>{timeLabel(row.created)}</dd>
        <dt>Finished</dt>
        <dd>{row.active ? 'In progress' : timeLabel(row.finished)}</dd>
        {row.iteration !== undefined && (
          <>
            <dt>Iteration</dt>
            <dd>{row.iteration}</dd>
          </>
        )}
      </dl>
      {row.retried && (
        <Alert severity='warning'>
          This task span includes retries and waiting between attempts.
        </Alert>
      )}
      {row.task.state === 'CACHED' && (
        <Alert severity='info'>
          Cache hit. This span is cache-resolution overhead, not component computation.
        </Alert>
      )}
      {row.elapsed === undefined && (
        <Alert severity='info'>Insufficient timestamps. No duration has been inferred.</Alert>
      )}
      <h3>State history</h3>
      <ol className='rt-history'>
        {(row.task.state_history || []).map((status, index) => (
          <li key={index}>
            <Status state={status.state} />
            <time>{timeLabel(taskTimestamp(status.update_time))}</time>
            {status.error?.message && <small>{status.error.message}</small>}
          </li>
        ))}
      </ol>
      {!row.task.state_history?.length && <p>No state transitions recorded.</p>}
      <Button
        size='small'
        variant='outlined'
        disabled={!row.task.task_id}
        onClick={() => onOpenTask(row.task.task_id!)}
      >
        Open task in graph
      </Button>
    </aside>
  );
}

function Waterfall({
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
      <div className='rt-chart' role='table' aria-label='Component waterfall timings'>
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
                {row.retried && (
                  <Tooltip title='Span includes retries'>
                    <span aria-label='Retried task'>↻</span>
                  </Tooltip>
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
                      backgroundColor: STATE_COLORS[row.task.state || ''] || '#7b8492',
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
