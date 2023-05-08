// Copyright 2023 The Kubeflow Authors
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

import { PipelineSpec } from 'src/generated/pipeline_spec';

export function getComponentSpec(pipelineSpec: PipelineSpec, layers: string[], taskKey: string) {
  let currentDag = pipelineSpec.root?.dag;
  const taskLayers = [...layers.slice(1), taskKey];
  let componentSpec;
  for (let i = 0; i < taskLayers.length; i++) {
    const pipelineTaskSpec = currentDag?.tasks[taskLayers[i]];
    const componentName = pipelineTaskSpec?.componentRef?.name;
    if (!componentName) {
      return null;
    }
    componentSpec = pipelineSpec.components[componentName];
    if (!componentSpec) {
      return null;
    }
    currentDag = componentSpec.dag;
  }
  return componentSpec;
}
