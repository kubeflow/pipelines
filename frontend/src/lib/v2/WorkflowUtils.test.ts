// Copyright 2021 The Kubeflow Authors
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

import { testBestPractices } from 'src/TestUtils';
import { Workflow, WorkflowSpec, WorkflowStatus } from 'third_party/argo-ui/argo_template';
import { isV2Pipeline } from './WorkflowUtils';

testBestPractices();
describe('WorkflowUtils', () => {
  const WORKFLOW_EMPTY: Workflow = {
    metadata: {
      name: 'workflow',
    },
    // there are many unrelated fields here, omit them
    spec: {} as WorkflowSpec,
    status: {} as WorkflowStatus,
  };

  it('detects v2/v2 compatible pipeline', () => {
    const workflow = {
      ...WORKFLOW_EMPTY,
      metadata: {
        ...WORKFLOW_EMPTY.metadata,
        annotations: { 'pipelines.kubeflow.org/v2_pipeline': 'true' },
      },
    };
    expect(isV2Pipeline(workflow)).toBeTruthy();
  });

  it('detects v1 pipeline', () => {
    expect(isV2Pipeline(WORKFLOW_EMPTY)).toBeFalsy();
  });
});
