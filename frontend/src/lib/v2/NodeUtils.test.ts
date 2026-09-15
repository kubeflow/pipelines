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

import { PipelineSpec } from 'src/generated/pipeline_spec';
import { getComponentSpec } from './NodeUtils';

describe('getComponentSpec', () => {
  it('ignores runtime-only ParallelFor iteration layers', () => {
    const pipelineSpec = {
      root: {
        dag: { tasks: { loop: { componentRef: { name: 'loop-component' } } } },
      },
      components: {
        'loop-component': {
          dag: { tasks: { body: { componentRef: { name: 'body-component' } } } },
        },
        'body-component': {
          dag: { tasks: { train: { componentRef: { name: 'train-component' } } } },
        },
        'train-component': { executorLabel: 'exec-train' },
      },
    } as PipelineSpec;

    expect(
      getComponentSpec(pipelineSpec, ['root', 'loop', 'loop.1', 'body'], 'train')?.executorLabel,
    ).toBe('exec-train');
  });
});
