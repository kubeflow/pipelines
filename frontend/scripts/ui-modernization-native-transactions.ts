/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */
import assert from 'node:assert/strict';

type Body = Record<string, unknown>;
interface Mutation {
  method: string;
  path: string;
  body: Body | null;
  status: number;
}
const createdAt = '2026-09-26T12:00:00.000Z';
export const nativePipeline = {
  pipeline_id: 'native-pipeline',
  display_name: 'Native typed pipeline',
  created_at: createdAt,
};
export const nativeSpec = {
  pipelineInfo: { name: 'native-transaction-fixture' },
  root: {
    inputDefinitions: {
      parameters: {
        count: { parameterType: 'NUMBER_INTEGER', defaultValue: 0 },
        enabled: { parameterType: 'BOOLEAN', defaultValue: false },
        message: { parameterType: 'STRING', defaultValue: '' },
        config: { parameterType: 'STRUCT', defaultValue: { nested: false } },
      },
    },
    dag: { tasks: {} },
  },
  deploymentSpec: { executors: {} },
  schemaVersion: '2.1.0',
  sdkVersion: 'kfp-2.0.0',
};
const version = {
  ...nativePipeline,
  pipeline_version_id: 'native-version',
  display_name: 'Native version',
  pipeline_spec: nativeSpec,
};
export class NativeTransactions {
  active = false;
  scenario = '';
  mutations: Mutation[] = [];
  unexpected: { method: string; path: string }[] = [];
  experiments: Body[] = [];
  runs: Body[] = [];
  schedules: Body[] = [];
  private failNext = new Set<string>();
  start(scenario: string) {
    assert.ok(
      ['experiment', 'run', 'recurring'].includes(scenario),
      'Unknown transaction scenario',
    );
    this.active = true;
    this.scenario = scenario;
    this.experiments = [
      {
        experiment_id: 'native-experiment',
        display_name: 'Native experiment',
        namespace: 'team-a',
        created_at: createdAt,
        storage_state: 'AVAILABLE',
      },
    ];
    this.runs = [];
    this.schedules = [];
    this.failNext = new Set(
      scenario === 'experiment'
        ? ['/apis/v2beta1/experiments']
        : scenario === 'recurring'
          ? ['/apis/v2beta1/recurringruns/native-schedule:disable']
          : [],
    );
  }
  snapshot() {
    return { scenario: this.scenario, mutations: this.mutations, unexpected: this.unexpected };
  }
  handle(method: string, url: URL, body: Body | null) {
    if (!this.active) return null;
    const path = url.pathname;
    const json = (value: unknown, status = 200) => ({ status, body: value });
    if (method === 'POST') {
      const allowed =
        this.scenario === 'experiment'
          ? ['/apis/v2beta1/experiments']
          : this.scenario === 'run'
            ? ['/apis/v2beta1/runs']
            : [
                '/apis/v2beta1/recurringruns',
                '/apis/v2beta1/recurringruns/native-schedule:disable',
              ];
      if (!allowed.includes(path)) {
        this.unexpected.push({ method, path });
        return json({ message: 'Unexpected fixture mutation' }, 405);
      }
      try {
        if (path.endsWith(':disable')) {
          assert.equal(body, null, 'Schedule action has no body');
          assert.equal(this.schedules.length, 1);
        } else {
          assert.ok(body && typeof body === 'object' && !Array.isArray(body));
          if (path.endsWith('/experiments')) assert.equal(body.namespace, 'team-a');
          else assert.equal(body.experiment_id, 'native-experiment');
        }
      } catch (error) {
        // The HTTP adapter returns 400 for rejected fixture payloads.
        this.mutations.push({ method, path, body, status: 400 });
        throw error;
      }
      const failure = this.failNext.delete(path);
      this.mutations.push({ method, path, body, status: failure ? 503 : 200 });
      if (failure)
        return json({ code: 14, message: 'Fixture mutation temporarily unavailable' }, 503);
      if (path.endsWith(':disable')) {
        this.schedules[0].status = 'DISABLED';
        return json({});
      }
      assert.ok(body && typeof body === 'object' && !Array.isArray(body));
      if (path.endsWith('/experiments')) {
        const item = {
          ...body,
          experiment_id: 'native-created-experiment',
          storage_state: 'AVAILABLE',
          created_at: createdAt,
        };
        this.experiments.push(item);
        return json(item);
      }
      if (path.endsWith('/runs')) {
        const item = {
          ...body,
          run_id: 'native-created-run',
          state: 'SUCCEEDED',
          storage_state: 'AVAILABLE',
          created_at: createdAt,
          pipeline_spec: nativeSpec,
          run_details: { task_details: [] },
        };
        this.runs.push(item);
        return json(item);
      }
      const item = {
        ...body,
        recurring_run_id: 'native-schedule',
        namespace: 'team-a',
        status: 'ENABLED',
        created_at: createdAt,
        pipeline_spec: nativeSpec,
      };
      this.schedules.push(item);
      return json(item);
    }
    if (method !== 'GET' && method !== 'HEAD') {
      this.unexpected.push({ method, path });
      return json({ message: 'Unexpected fixture method' }, 405);
    }
    if (path === '/apis/v2beta1/pipelines') return json({ pipelines: [nativePipeline] });
    if (path === '/apis/v2beta1/pipelines/native-pipeline') return json(nativePipeline);
    if (path === '/apis/v2beta1/pipelines/native-pipeline/versions')
      return json({ pipeline_versions: [version] });
    if (path === '/apis/v2beta1/pipelines/native-pipeline/versions/native-version')
      return json(version);
    for (const [kind, records, key] of [
      ['experiments', this.experiments, 'experiment_id'],
      ['runs', this.runs, 'run_id'],
      ['recurringruns', this.schedules, 'recurring_run_id'],
    ] as const) {
      if (path === `/apis/v2beta1/${kind}`)
        return json({
          [kind === 'recurringruns' ? 'recurringRuns' : kind]: records,
          total_size: records.length,
        });
      const prefix = `/apis/v2beta1/${kind}/`;
      if (path.startsWith(prefix)) {
        const id = path.slice(prefix.length);
        if (kind === 'runs' && id.endsWith('/tasks')) return json({ tasks: [] });
        const item = records.find((record) => record[key] === id);
        return item ? json(item) : json({ message: 'Fixture not found' }, 404);
      }
    }
    return null;
  }
}
