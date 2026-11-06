// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { describe, expect, it } from 'vitest';
import {
  complexPipelineSpec,
  complexRunSamples,
  complexRunTasks,
} from '../mock-backend/complex-run-samples';
import { V2Run } from '../src/apisv2/run';
const runs: V2Run[] = complexRunSamples.map((sample) => ({
  run_id: `sample-run-${sample.id}`,
  pipeline_version_reference: { pipeline_id: sample.id, pipeline_version_id: `${sample.id}-v1` },
  pipeline_spec: complexPipelineSpec(sample),
  created_at: new Date('2026-01-01T00:00:00Z'),
  finished_at: new Date('2026-01-01T00:12:00Z'),
  state: 'SUCCEEDED',
}));
import { sampleRunTasks, sampleTaskLogs } from '../mock-backend/sample-run-tasks';
import {
  convertSubDagToRuntimeFlowElements,
  getTaskRuntimeLayers,
  getNodeRuntimeInfo,
} from '../src/lib/v2/DynamicFlow';
import { convertFlowElements, NodeTypeNames } from '../src/lib/v2/StaticFlow';
import { PipelineSpec } from '../src/generated/pipeline_spec';
import { getRunTaskTiming } from '../src/lib/v2/RunTaskTiming';

describe('complex review runs', () => {
  it.each(complexRunSamples)('$name has resolvable graph and timeline tasks', (sample) => {
    const run = runs.find((entry) => entry.run_id === `sample-run-${sample.id}`)!;
    expect(run.pipeline_version_reference).toBeDefined();
    const tasks = sampleRunTasks(run);
    expect(new Set(tasks.map((task) => task.task_id)).size).toBe(tasks.length);
    expect(
      convertFlowElements(PipelineSpec.fromJSON(complexPipelineSpec(sample))).length,
    ).toBeGreaterThan(2);
    for (const task of tasks) {
      expect(task.create_time!.getTime()).toBeGreaterThanOrEqual(run.created_at!.getTime());
      expect(task.end_time!.getTime()).toBeLessThanOrEqual(run.finished_at!.getTime());
      expect(task.end_time!.getTime()).toBeGreaterThanOrEqual(task.create_time!.getTime());
      if (task.parent_task_id) {
        const parent = tasks.find((candidate) => candidate.task_id === task.parent_task_id)!;
        expect(parent).toBeDefined();
        expect(task.create_time!.getTime()).toBeGreaterThanOrEqual(parent.create_time!.getTime());
        expect(task.end_time!.getTime()).toBeLessThanOrEqual(parent.end_time!.getTime());
      }
      if (task.type !== 'RUNTIME') continue;
      expect(task.inputs?.parameters).toEqual([
        { parameter_key: 'dataset_uri', value: expect.stringContaining(task.name!) },
      ]);
      expect(task.outputs?.parameters).toEqual([{ parameter_key: 'row_count', value: 12480 }]);
      const logLines = sampleTaskLogs(task, run).trim().split('\n');
      const timestamps = logLines.map((line) => Date.parse(line.split(' ')[0]));
      expect(timestamps.every(Number.isFinite)).toBe(true);
      expect(timestamps).toEqual([...timestamps].sort((a, b) => a - b));
      expect(logLines.at(-2)).toContain('Processed 12480 rows');
      expect(logLines.at(-1)).toContain('Component completed successfully');
      const layers = getTaskRuntimeLayers(task, tasks)!;
      expect(layers).toBeDefined();
      expect(
        convertSubDagToRuntimeFlowElements(
          PipelineSpec.fromJSON(complexPipelineSpec(sample)),
          layers,
          tasks,
        ).some((node) => node.id === `task.${task.name}`),
      ).toBe(true);
      expect(
        getNodeRuntimeInfo(
          {
            id: `task.${task.name}`,
            type: NodeTypeNames.EXECUTION,
            position: { x: 0, y: 0 },
            data: { label: task.name! },
          },
          tasks,
          layers,
        ).task?.task_id,
      ).toBe(task.task_id);
    }
    const timing = getRunTaskTiming(run, tasks, Date.now());
    expect(timing.rows).toHaveLength(tasks.filter((task) => task.type === 'RUNTIME').length);
    expect(timing.rows.every((row) => row.elapsed! > 0)).toBe(true);
  });
  it('shows a skipped branch without inventing executed descendants', () => {
    const run = runs.find((entry) => entry.run_id === 'sample-run-nested-conditions')!;
    const tasks = sampleRunTasks(run);
    expect(tasks.find((task) => task.name === 'reject-invalid-data')?.state).toBe('SKIPPED');
    expect(tasks.some((task) => task.name === 'report-validation-failure')).toBe(false);
    expect(
      getTaskRuntimeLayers(
        tasks.find((task) => task.name === 'encode')!,
        tasks,
      ),
    ).toEqual(['root', 'prepare-features', 'transform-features']);
  });
  it('materializes three distinct parallel iterations with the same component names', () => {
    const run = runs.find((entry) => entry.run_id === 'sample-run-parallel-folds')!;
    const tasks = sampleRunTasks(run);
    expect(tasks.filter((task) => task.name === 'train-fold')).toHaveLength(3);
    expect(
      tasks
        .filter((task) => task.type_attributes?.iteration_index !== undefined)
        .map((task) => task.type_attributes!.iteration_index),
    ).toEqual(['0', '1', '2']);
  });
  it('provides a 32-component overview and leaves queued runs empty', () => {
    const run = runs.find((entry) => entry.run_id === 'sample-run-regional-training')!;
    expect(sampleRunTasks(run).filter((task) => task.type === 'RUNTIME')).toHaveLength(32);
    const queued: V2Run = { run_id: 'sample-run-validation-pending', state: 'PENDING' };
    expect(sampleRunTasks(queued)).toEqual([]);
    expect(complexRunTasks({ run_id: 'unknown' })).toBeUndefined();
  });
});
