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

import { PipelineSpec } from 'src/generated/pipeline_spec';
import { ml_pipelines } from 'src/generated/pipeline_spec/pbjs_ml_pipelines';
import { Workflow } from 'third_party/argo-ui/argo_template';

export function isV2Pipeline(workflow: Workflow): boolean {
  return workflow?.metadata?.annotations?.['pipelines.kubeflow.org/v2_pipeline'] === 'true';
}

export function isArgoWorkflowTemplate(template: Workflow): boolean {
  if (template?.kind === 'Workflow' && template?.apiVersion?.startsWith('argoproj.io/')) {
    return true;
  }
  return false;
}

// Assuming template is the JSON format of PipelineJob in api/v2alpha1/pipeline_spec.proto
// TODO(zijianjoy): We need to change `template` format to PipelineSpec once SDK support is in.
export function convertJsonToV2PipelineSpec(template: string): PipelineSpec {
  const pipelineJob = JSON.parse(template);

  const message = ml_pipelines.PipelineSpec.fromObject(pipelineJob['pipelineSpec']);
  const buffer = ml_pipelines.PipelineSpec.encode(message).finish();
  const pipelineSpec = PipelineSpec.deserializeBinary(buffer);
  return pipelineSpec;
}
