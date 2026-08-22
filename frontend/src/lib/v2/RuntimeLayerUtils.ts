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

const RUNTIME_ITERATION_LAYER_PATTERN = /^(.*)\.(\d+)$/;

export interface RuntimeIterationLayer {
  iterationIndex: number;
  taskName: string;
}

export function formatRuntimeIterationLayer(taskName: string, iterationIndex: number): string {
  return `${taskName}.${iterationIndex}`;
}

export function parseRuntimeIterationLayer(
  layer: string,
  expectedTaskName?: string,
): RuntimeIterationLayer | undefined {
  const match = layer.match(RUNTIME_ITERATION_LAYER_PATTERN);
  if (!match || (expectedTaskName !== undefined && match[1] !== expectedTaskName)) {
    return undefined;
  }
  return { taskName: match[1], iterationIndex: Number(match[2]) };
}

export function isRuntimeIterationLayer(layer: string, expectedTaskName?: string): boolean {
  return parseRuntimeIterationLayer(layer, expectedTaskName) !== undefined;
}
