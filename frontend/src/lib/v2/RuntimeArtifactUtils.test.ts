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

import { PipelineTaskTaskState, V2beta1PipelineTask } from 'src/apisv2beta1/run';
import {
  flattenArtifactGroups,
  formatParameters,
  getArtifactDisplayName,
  getArtifactSessionInfo,
  getOutputArtifactByName,
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
              metadata: { store_session_info: 'session-a' } as any,
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
    expect(getArtifactDisplayName(entries[1].artifact, entries[1].artifactKey, 1)).toBe(
      'model (2)',
    );
  });

  it('finds output artifacts by either output key or artifact name', () => {
    expect(getOutputArtifactByName(task, 'models')?.artifact_id).toBe('model-1');
    expect(getOutputArtifactByName(task, 'model')?.artifact_id).toBe('model-1');
    expect(getOutputArtifactByName(task, 'missing')).toBeUndefined();
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

  it('reads session metadata and recognizes every terminal task state', () => {
    expect(getArtifactSessionInfo(task.outputs?.artifacts?.[0].artifacts?.[0]!)).toBe('session-a');
    expect(isTaskFinished(PipelineTaskTaskState.SUCCEEDED)).toBe(true);
    expect(isTaskFinished(PipelineTaskTaskState.FAILED)).toBe(true);
    expect(isTaskFinished(PipelineTaskTaskState.SKIPPED)).toBe(true);
    expect(isTaskFinished(PipelineTaskTaskState.CACHED)).toBe(true);
    expect(isTaskFinished(PipelineTaskTaskState.RUNNING)).toBe(false);
  });
});
