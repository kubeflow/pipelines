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
  it('artifactTypes returns a stable key', () => {
    expect(queryKeys.artifactTypes()).toEqual(['artifact_types']);
  });

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

  it('v2RecurringRunDetail includes the recurring run ID', () => {
    expect(queryKeys.v2RecurringRunDetail('rr-1')).toEqual([
      'v2_recurring_run_detail',
      { id: 'rr-1' },
    ]);
  });

  it('mlmdPackage includes the run ID', () => {
    expect(queryKeys.mlmdPackage('run-456')).toEqual(['mlmd_package', { id: 'run-456' }]);
  });

  it('contextByExecution includes execution ID and state', () => {
    expect(queryKeys.contextByExecution(42, 3)).toEqual([
      'context_by_execution',
      { id: 42, state: 3 },
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
    const artifactTypesKey = queryKeys.artifactTypes();
    const pipelineKey = queryKeys.pipeline('p1');
    expect(artifactTypesKey[0]).not.toEqual(pipelineKey[0]);
  });

  it('pipelineVersions defaults to empty string when pipelineId is null', () => {
    expect(queryKeys.pipelineVersions(null)).toEqual(['pipeline_versions', '']);
  });

  it('executionLogs includes executionId and namespace', () => {
    expect(queryKeys.executionLogs(5, 'kubeflow')).toEqual([
      'execution_logs',
      { executionId: 5, namespace: 'kubeflow' },
    ]);
  });
});
