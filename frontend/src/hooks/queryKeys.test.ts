/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { queryKeys } from './queryKeys';

describe('queryKeys', () => {
  it('pipelineVersionTemplate includes both IDs', () => {
    expect(queryKeys.pipelineVersionTemplate('p1', 'v1')).toEqual([
      'PipelineVersionTemplate',
      { pipelineId: 'p1', pipelineVersionId: 'v1' },
    ]);
  });

  it('pipelineVersionTemplate handles undefined IDs', () => {
    expect(queryKeys.pipelineVersionTemplate(undefined, undefined)).toEqual([
      'PipelineVersionTemplate',
      { pipelineId: undefined, pipelineVersionId: undefined },
    ]);
  });

  it('v2RunDetail includes the run ID', () => {
    expect(queryKeys.v2RunDetail('run-123')).toEqual(['v2_run_detail', { id: 'run-123' }]);
  });

  it('v2RunDetail handles null', () => {
    expect(queryKeys.v2RunDetail(null)).toEqual(['v2_run_detail', { id: null }]);
  });

  it('v2RunDetails includes an array of run IDs', () => {
    expect(queryKeys.v2RunDetails(['r1', 'r2'])).toEqual(['v2_run_details', { ids: ['r1', 'r2'] }]);
  });

  it('artifact metadata query keys include their page or visualization identity', () => {
    expect(queryKeys.artifactTasksPage('artifact-1', 'page-2', 20)).toEqual([
      'artifact_tasks',
      { artifactId: 'artifact-1', pageSize: 20, pageToken: 'page-2' },
    ]);
    expect(queryKeys.artifactVisualizationKey('artifact-1')).toEqual([
      'artifact_visualization_key',
      { id: 'artifact-1' },
    ]);
  });

  it('v2RecurringRunDetail includes the recurring run ID', () => {
    expect(queryKeys.v2RecurringRunDetail('rr-1')).toEqual([
      'v2_recurring_run_detail',
      { id: 'rr-1' },
    ]);
  });

  it('pipeline includes the pipeline ID', () => {
    expect(queryKeys.pipeline('pipe-1')).toEqual(['pipeline', 'pipe-1']);
  });

  it('pipeline handles null', () => {
    expect(queryKeys.pipeline(null)).toEqual(['pipeline', null]);
  });

  it('pipelineVersion includes both pipeline and version IDs', () => {
    expect(queryKeys.pipelineVersion('p1', 'v2')).toEqual(['pipelineVersion', 'p1', 'v2']);
  });

  it('experiment includes the experiment ID', () => {
    expect(queryKeys.experiment('exp-1')).toEqual(['experiment', 'exp-1']);
  });

  it('returns distinct keys for different factories', () => {
    const runTasksKey = queryKeys.runTasks('run-1');
    const pipelineKey = queryKeys.pipeline('p1');
    expect(runTasksKey[0]).not.toEqual(pipelineKey[0]);
  });

  it('scopes retried task snapshots without changing the initial cache key', () => {
    expect(queryKeys.runTasks('run-1')).toEqual(['run_tasks', { id: 'run-1' }]);
    expect(queryKeys.runTasks('run-1', 2)).toEqual([
      'run_tasks',
      { id: 'run-1', retryRefreshVersion: 2 },
    ]);
  });

  it('pipelineVersions defaults to empty string when pipelineId is null', () => {
    expect(queryKeys.pipelineVersions(null)).toEqual(['pipeline_versions', '']);
  });
});
