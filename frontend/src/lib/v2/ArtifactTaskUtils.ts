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

import { V2IOType } from 'src/apisv2/artifact';

export const OUTPUT_ARTIFACT_TASK_TYPES = [
  V2IOType.OUTPUT,
  V2IOType.ITERATOR_OUTPUT,
  V2IOType.ONE_OF_OUTPUT,
  V2IOType.TASK_FINAL_STATUS_OUTPUT,
] as const;

export const INPUT_ARTIFACT_TASK_TYPES = [
  V2IOType.COMPONENT_DEFAULT_INPUT,
  V2IOType.TASK_OUTPUT_INPUT,
  V2IOType.COMPONENT_INPUT,
  V2IOType.RUNTIME_VALUE_INPUT,
  V2IOType.COLLECTED_INPUTS,
  V2IOType.ITERATOR_INPUT,
  V2IOType.ITERATOR_INPUT_RAW,
] as const;

const OUTPUT_ARTIFACT_TASK_TYPE_SET: ReadonlySet<V2IOType> = new Set(OUTPUT_ARTIFACT_TASK_TYPES);
const INPUT_ARTIFACT_TASK_TYPE_SET: ReadonlySet<V2IOType> = new Set(INPUT_ARTIFACT_TASK_TYPES);

export function isOutputArtifactTaskType(type?: V2IOType): boolean {
  return type !== undefined && OUTPUT_ARTIFACT_TASK_TYPE_SET.has(type);
}

export function isInputArtifactTaskType(type?: V2IOType): boolean {
  return type !== undefined && INPUT_ARTIFACT_TASK_TYPE_SET.has(type);
}
