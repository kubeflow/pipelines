// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { describe, expect, it } from 'vitest';
import { V2Run, V2RuntimeState } from '../src/apisv2/run';
import {
  sampleDriverLogs,
  sampleRetryTaskLogs,
  sampleRunPipelineSpec,
  sampleRunTasks,
} from '../mock-backend/sample-run-tasks';
import { PipelineSpec } from '../src/generated/pipeline_spec';
import { convertFlowElements, NodeTypeNames } from '../src/lib/v2/StaticFlow';
import { updateFlowElementsState } from '../src/lib/v2/DynamicFlow';
import { getRunTaskTiming } from '../src/lib/v2/RunTaskTiming';

const samples: V2Run[] = Object.values(V2RuntimeState)
  .filter((state) =>
    ['RUNNING', 'SUCCEEDED', 'FAILED', 'CANCELED', 'CANCELING', 'PENDING'].includes(state),
  )
  .map((state) => ({
    run_id: `sample-run-${state.toLowerCase()}`,
    display_name: state,
    state,
    created_at: new Date('2026-01-01T00:00:00Z'),
    finished_at: ['SUCCEEDED', 'FAILED', 'CANCELED'].includes(state)
      ? new Date('2026-01-01T00:12:00Z')
      : undefined,
  }));
describe('review sample timelines', () => {
  it.each(samples)('$display_name has consistent graph nodes, states and timestamps', (run) => {
    const tasks = sampleRunTasks(run);
    if (run.state === 'PENDING') {
      expect(tasks).toEqual([]);
      return;
    }
    expect(tasks.length).toBeGreaterThan(3);
    expect(new Set(tasks.map((task) => task.task_id)).size).toBe(tasks.length);
    const root = tasks.find((task) => task.type === 'ROOT')!;
    const children = tasks.filter((task) => task.type === 'RUNTIME');
    for (const task of children) {
      expect(task.run_id).toBe(run.run_id);
      if (task.state === 'SKIPPED') {
        expect(task.status_metadata?.custom_properties?.driver_logs_uri).toBeUndefined();
      } else {
        expect(task.status_metadata?.custom_properties?.driver_logs_uri).toBe(
          `s3://mlpipeline/review-driver-logs/${task.task_id}`,
        );
        expect(sampleDriverLogs(task, 'kubeflow')).toContain(
          `Creating task ${task.name} in pod kubeflow`,
        );
        expect(sampleDriverLogs(task, 'kubeflow')).toContain(task.create_time!.toISOString());
      }
      expect(task.parent_task_id).toBe(root.task_id);
      expect(sampleRunPipelineSpec.root.dag.tasks).toHaveProperty(task.name!);
      expect(task.scope_path).toBe(`root.${task.name}`);
      expect(task.create_time!.getTime()).toBeGreaterThanOrEqual(run.created_at!.getTime());
      if (task.end_time) {
        expect(task.end_time.getTime()).toBeGreaterThanOrEqual(task.create_time!.getTime());
        if (run.finished_at)
          expect(task.end_time.getTime()).toBeLessThanOrEqual(run.finished_at.getTime());
      }
    }
    const training = children.find((task) => task.name === 'train-model')!;
    const validation = children.find((task) => task.name === 'validate-data')!;
    expect(training.create_time).toEqual(validation.create_time);
    if (run.state === 'RUNNING' || run.state === 'CANCELING') {
      expect(training.state).toBe('RUNNING');
      expect(training.end_time).toBeUndefined();
      expect(children.some((task) => task.name === 'evaluate-model')).toBe(false);
    } else if (run.state === 'SUCCEEDED') {
      expect(children.every((task) => task.state === 'SUCCEEDED')).toBe(true);
      const evaluation = children.find((task) => task.name === 'evaluate-model')!;
      expect(evaluation.create_time!.getTime()).toBeGreaterThan(training.end_time!.getTime());
    } else {
      expect(training.state).toBe('FAILED');
      expect(training.state_history?.at(-1)?.error?.code).toBe(run.state === 'CANCELED' ? 1 : 3);
      expect(children.find((task) => task.name === 'evaluate-model')?.state).toBe('SKIPPED');
    }
    const timing = getRunTaskTiming(run, tasks, Date.now());
    expect(timing.rows).toHaveLength(children.length);
    expect(timing.rows.every((row) => row.elapsed !== undefined && row.elapsed >= 0)).toBe(true);
  });
  it('does not invent tasks for other runs', () => {
    expect(sampleRunTasks({ run_id: 'unknown' })).toEqual([]);
  });
});

describe('review sample artifacts', () => {
  it.each(samples)(
    '$display_name only exposes produced outputs, with matching consumer IDs',
    (run) => {
      const tasks = sampleRunTasks(run);
      if (run.state === 'PENDING') {
        expect(tasks).toEqual([]);
        return;
      }
      const outputArtifacts = tasks.flatMap(
        (task) => task.outputs?.artifacts?.flatMap((group) => group.artifacts || []) || [],
      );
      expect(outputArtifacts).toHaveLength(run.state === 'SUCCEEDED' ? 3 : 1);
      for (const task of tasks) {
        for (const group of task.inputs?.artifacts || []) {
          for (const artifact of group.artifacts || []) {
            expect(outputArtifacts).toContainEqual(artifact);
            expect(artifact.created_at!.getTime()).toBeLessThanOrEqual(task.create_time!.getTime());
          }
        }
        if (task.state !== 'SUCCEEDED') expect(task.outputs?.artifacts || []).toEqual([]);
      }
      const graph = updateFlowElementsState(
        ['root'],
        convertFlowElements(PipelineSpec.fromJSON(sampleRunPipelineSpec)),
        tasks,
      );
      const artifactNodes = graph.filter(
        (node) => 'type' in node && node.type === NodeTypeNames.ARTIFACT,
      );
      expect(artifactNodes).toHaveLength(3);
      expect(
        artifactNodes.filter((node) => (node.data as { hasArtifact?: boolean }).hasArtifact),
      ).toHaveLength(outputArtifacts.length);
    },
  );
});

describe('review sample retry logs', () => {
  it('exposes distinct archived output for both automatic attempts', () => {
    const run: V2Run = {
      ...samples.find((sample) => sample.state === 'FAILED')!,
      run_id: 'sample-run-import-failed',
    };
    const task = sampleRunTasks(run).find((task) => task.name === 'train-model')!;
    expect(task.pods?.filter((pod) => pod.type === 'EXECUTOR')).toHaveLength(2);
    const logs = task.outputs?.artifacts?.find((group) => group.artifact_key === 'executor-logs');
    expect(logs?.artifacts?.map((artifact) => artifact.uri)).toEqual([
      `s3://mlpipeline/review-executor-logs/${task.task_id}/executor-logs-0`,
      `s3://mlpipeline/review-executor-logs/${task.task_id}/executor-logs-1`,
    ]);
    expect(sampleRetryTaskLogs(task, run, 0)).toContain('TimeoutError:');
    expect(sampleRetryTaskLogs(task, run, 1)).toContain('invalid feature');
    expect(sampleRetryTaskLogs(task, run, 1)).not.toContain('TimeoutError:');
  });
});
