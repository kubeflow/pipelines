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

import { V2beta1IOType } from 'src/apisv2beta1/artifact';
import {
  INPUT_ARTIFACT_TASK_TYPES,
  isInputArtifactTaskType,
  isOutputArtifactTaskType,
  OUTPUT_ARTIFACT_TASK_TYPES,
} from './ArtifactTaskUtils';

describe('ArtifactTaskUtils', () => {
  it.each(OUTPUT_ARTIFACT_TASK_TYPES)('classifies %s as an output relationship', (type) => {
    expect(isOutputArtifactTaskType(type)).toBe(true);
    expect(isInputArtifactTaskType(type)).toBe(false);
  });

  it.each(INPUT_ARTIFACT_TASK_TYPES)('classifies %s as an input relationship', (type) => {
    expect(isInputArtifactTaskType(type)).toBe(true);
    expect(isOutputArtifactTaskType(type)).toBe(false);
  });

  it.each([undefined, V2beta1IOType.UNSPECIFIED])('leaves %s unclassified', (type) => {
    expect(isInputArtifactTaskType(type)).toBe(false);
    expect(isOutputArtifactTaskType(type)).toBe(false);
  });
});
