// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import {
  ArtifactArtifactType,
  PipelineTaskTaskState,
  V2beta1PipelineTask,
} from 'src/apisv2beta1/run';
import {
  flattenArtifactGroups,
  EXECUTOR_LOGS_ARTIFACT_KEY,
  formatParameters,
  getArtifactDisplayName,
  getArtifactIdentity,
  getArtifactTypeName,
  getOutputArtifactByName,
  getScalarMetricEntries,
  isClassificationMetricArtifact,
  isLegacyUiMetadataArtifact,
  isVisualizableArtifact,
  isTaskFinished,
} from './RuntimeArtifactUtils';

describe('RuntimeArtifactUtils', () => {
  const task: V2beta1PipelineTask = {
    outputs: {
      artifacts: [
        {
          artifact_key: 'models',
          artifacts: [
            {
              artifact_id: 'model-1',
              name: 'model',
            },
            { artifact_id: 'model-2', name: 'model' },
          ],
        },
      ],
    },
  };

  it('flattens artifact groups while retaining the key, group, and index', () => {
    const entries = flattenArtifactGroups(task.outputs?.artifacts);

    expect(entries).toHaveLength(2);
    expect(entries[0]).toMatchObject({ artifactKey: 'models', index: 0 });
    expect(entries[1]).toMatchObject({ artifactKey: 'models', index: 1 });
    expect(entries[0].group).toBe(task.outputs?.artifacts?.[0]);
    expect(
      getArtifactDisplayName(
        entries[1].artifact,
        entries[1].artifactKey,
        1,
        entries[1].group.artifacts,
      ),
    ).toBe('model (2)');
  });

  it('only adds artifact suffixes when labels would otherwise be ambiguous', () => {
    const distinctArtifacts = [{ name: 'model-a' }, { name: 'model-b' }];
    expect(getArtifactDisplayName(distinctArtifacts[1], 'models', 1, distinctArtifacts)).toBe(
      'model-b',
    );

    const unnamedArtifacts = [{ artifact_id: 'artifact-1' }, { artifact_id: 'artifact-2' }];
    expect(getArtifactDisplayName(unnamedArtifacts[1], 'models', 1, unnamedArtifacts)).toBe(
      'models (2)',
    );
  });

  it('defines the native executor logs output key', () => {
    expect(EXECUTOR_LOGS_ARTIFACT_KEY).toBe('executor-logs');
  });

  it('uses the stable artifact identity fields in priority order', () => {
    expect(getArtifactIdentity({ artifact_id: 'id', uri: 's3://bucket/key', name: 'name' })).toBe(
      'id',
    );
    expect(getArtifactIdentity({ uri: 's3://bucket/key', name: 'name' })).toBe('s3://bucket/key');
    expect(getArtifactIdentity({ name: 'name' })).toBe('name');
    expect(getArtifactIdentity({})).toBeUndefined();
  });

  it('finds the latest output artifact by either output key or artifact name', () => {
    expect(getOutputArtifactByName(task, 'models')?.artifact_id).toBe('model-2');
    expect(getOutputArtifactByName(task, 'model')?.artifact_id).toBe('model-2');
    expect(getOutputArtifactByName(task, 'missing')).toBeUndefined();
  });

  it('selects executor logs by retry index rather than Artifact API order', () => {
    const retryLogsTask: V2beta1PipelineTask = {
      outputs: {
        artifacts: [
          {
            artifact_key: EXECUTOR_LOGS_ARTIFACT_KEY,
            artifacts: [
              { artifact_id: 'artifact-a', uri: 's3://logs/executor-logs-2' },
              { artifact_id: 'artifact-b', uri: 's3://logs/executor-logs-0' },
              { artifact_id: 'artifact-c', uri: 's3://logs/executor-logs-1' },
            ],
          },
        ],
      },
    };

    expect(getOutputArtifactByName(retryLogsTask, EXECUTOR_LOGS_ARTIFACT_KEY)?.artifact_id).toBe(
      'artifact-a',
    );
  });

  it('formats native parameter values and preserves falsey values', () => {
    expect(
      formatParameters([
        { parameter_key: 'text', value: 'hello' as any },
        { parameter_key: 'count', value: 0 as any },
        { parameter_key: 'enabled', value: false as any },
        { parameter_key: 'config', value: { nested: true } },
      ]),
    ).toEqual([
      ['text', 'hello'],
      ['count', '0'],
      ['enabled', 'false'],
      ['config', '{"nested":true}'],
    ]);
  });

  it('uses canonical schema titles for artifact types', () => {
    expect(getArtifactTypeName({ type: ArtifactArtifactType.Metric })).toBe('system.Metrics');
    expect(getArtifactTypeName({ type: ArtifactArtifactType.ClassificationMetric })).toBe(
      'system.ClassificationMetrics',
    );
    expect(getArtifactTypeName({ type: ArtifactArtifactType.SlicedClassificationMetric })).toBe(
      'system.SlicedClassificationMetrics',
    );
  });

  it('expands every scalar metric value while preferring number_value for its name', () => {
    const metric = {
      name: 'accuracy',
      type: ArtifactArtifactType.Metric,
      number_value: 0.91,
      metadata: { accuracy: 0.8, loss: 0.09 },
    };
    expect(getScalarMetricEntries(metric)).toEqual([
      { name: 'accuracy', value: '0.91' },
      { name: 'loss', value: '0.09' },
    ]);
    expect(
      getScalarMetricEntries({
        name: 'metrics',
        type: ArtifactArtifactType.Metric,
        metadata: { accuracy: 0.91, loss: 0.09 },
      }),
    ).toEqual([
      { name: 'accuracy', value: '0.91' },
      { name: 'loss', value: '0.09' },
    ]);
    expect(
      getScalarMetricEntries({ ...metric, number_value: undefined, metadata: { accuracy: null } }),
    ).toEqual([{ name: 'accuracy', value: '-' }]);
    expect(
      getScalarMetricEntries({
        ...metric,
        number_value: undefined,
        metadata: { accuracy: { source: 'legacy-client' } },
      }),
    ).toEqual([{ name: 'accuracy', value: '{"source":"legacy-client"}' }]);
    expect(isVisualizableArtifact(metric)).toBe(true);
    expect(
      isClassificationMetricArtifact({ type: ArtifactArtifactType.ClassificationMetric }),
    ).toBe(true);
    expect(isVisualizableArtifact({ type: ArtifactArtifactType.Dataset })).toBe(false);
  });

  it('limits multi-key expansion to numeric metrics and preserves a named legacy fallback', () => {
    expect(
      getScalarMetricEntries({
        name: 'metrics',
        type: ArtifactArtifactType.Metric,
        metadata: { accuracy: 0.91, model_type: 'resnet', reported_accuracy: '0.95' },
      }),
    ).toEqual([{ name: 'accuracy', value: '0.91' }]);
    expect(
      getScalarMetricEntries({
        name: 'reported_accuracy',
        type: ArtifactArtifactType.Metric,
        metadata: { reported_accuracy: '0.95' },
      }),
    ).toEqual([{ name: 'reported_accuracy', value: '0.95' }]);
    expect(
      getScalarMetricEntries({
        name: 'constructor',
        type: ArtifactArtifactType.Metric,
        metadata: {},
      }),
    ).toEqual([{ name: 'constructor', value: '-' }]);
  });

  it('recognizes legacy UI metadata by artifact name or output key', () => {
    expect(isLegacyUiMetadataArtifact({ name: 'mlpipeline-ui-metadata' })).toBe(true);
    expect(isLegacyUiMetadataArtifact({}, 'mlpipeline_ui_metadata')).toBe(true);
    expect(isLegacyUiMetadataArtifact({ name: 'dataset' }, 'model')).toBe(false);
    expect(isVisualizableArtifact({ name: 'mlpipeline-ui-metadata' })).toBe(true);
  });

  it('recognizes every terminal task state', () => {
    expect(isTaskFinished(PipelineTaskTaskState.SUCCEEDED)).toBe(true);
    expect(isTaskFinished(PipelineTaskTaskState.FAILED)).toBe(true);
    expect(isTaskFinished(PipelineTaskTaskState.SKIPPED)).toBe(true);
    expect(isTaskFinished(PipelineTaskTaskState.CACHED)).toBe(true);
    expect(isTaskFinished(PipelineTaskTaskState.RUNNING)).toBe(false);
  });
});
