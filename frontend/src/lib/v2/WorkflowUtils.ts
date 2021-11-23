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

import jsyaml from 'js-yaml';
import { FeatureKey, isFeatureEnabled } from 'src/features';
import { ComponentSpec, PipelineSpec } from 'src/generated/pipeline_spec';
import { ml_pipelines } from 'src/generated/pipeline_spec/pbjs_ml_pipelines';
import * as StaticGraphParser from 'src/lib/StaticGraphParser';
import { convertFlowElements } from 'src/lib/v2/StaticFlow';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
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

// Assuming template is the JSON format of PipelineSpec in api/v2alpha1/pipeline_spec.proto
export function convertJsonToV2PipelineSpec(template: string): PipelineSpec {
  const pipelineSpecJSON = JSON.parse(template);

  // const message = ml_pipelines.PipelineSpec.fromObject(pipelineJob['pipelineSpec']);
  const message = ml_pipelines.PipelineSpec.fromObject(pipelineSpecJSON);
  const buffer = ml_pipelines.PipelineSpec.encode(message).finish();
  const pipelineSpec = PipelineSpec.deserializeBinary(buffer);
  return pipelineSpec;
}

// This needs to be changed to use pipeline_manifest vs workflow_manifest to distinguish V1 and V2.
export function isPipelineSpec(templateString: string) {
  if (!templateString) {
    return false;
  }
  try {
    const template = jsyaml.safeLoad(templateString);
    if (WorkflowUtils.isArgoWorkflowTemplate(template)) {
      StaticGraphParser.createGraph(template!);
      return false;
    } else if (isFeatureEnabled(FeatureKey.V2)) {
      const pipelineSpec = WorkflowUtils.convertJsonToV2PipelineSpec(templateString);
      convertFlowElements(pipelineSpec);
      return true;
    } else {
      return false;
    }
  } catch (err) {
    return false;
  }
}

// Given the PipelineSpec payload and targeted componentSpec, returns
// the `container` object for its image, command, arguments, etc.
export function getContainer(componentSpec: ComponentSpec, templateString: string) {
  const executionLabel = componentSpec?.getExecutorLabel();

  const jsonTemplate = JSON.parse(templateString);
  const deploymentSpec = jsonTemplate['deploymentSpec'];

  const executorsMap = deploymentSpec['executors'];
  if (!executorsMap || !executionLabel) {
    return null;
  }
  return executorsMap?.[executionLabel]?.['container'];
}
