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
import { loadYaml } from 'src/lib/YamlLoad';
import {
  ComponentSpec,
  PipelineDeploymentConfig,
  PipelineDeploymentConfig_ExecutorSpec,
  PipelineSpec,
  PlatformSpec,
} from 'src/generated/pipeline_spec';

// This key is used to retrieve the platform-agnostic pipeline definition
export const PIPELINE_SPEC_TEMPLATE_KEY = 'pipeline_spec';
export const PLATFORM_SPEC_TEMPLATE_KEY = 'platform_spec';

function getPipelineDefFromYaml(template: string) {
  // If pipeline_spec exists in the parsed YAML,
  // which means the original yaml contains platform_spec,
  // then the PipelineSpec(IR) is stored in 'pipeline_spec' field.
  const parsedTemplate = loadYaml(template) as Record<string, unknown>;
  return parsedTemplate[PIPELINE_SPEC_TEMPLATE_KEY] ?? parsedTemplate;
}

function getPlatformDefFromYaml(template: string) {
  return (loadYaml(template) as Record<string, unknown>)[PLATFORM_SPEC_TEMPLATE_KEY];
}

// Assuming template is the JSON format of PipelineSpec in api/v2alpha1/pipeline_spec.proto
export function convertYamlToV2PipelineSpec(template: string): PipelineSpec {
  const pipelineSpecDef = getPipelineDefFromYaml(template);
  return convertPipelineSpecDef(pipelineSpecDef);
}

function convertPipelineSpecDef(pipelineSpecDef: unknown): PipelineSpec {
  const pipelineSpec = PipelineSpec.fromJSON(pipelineSpecDef);
  if (!pipelineSpec.root?.dag || !pipelineSpec.pipelineInfo || !pipelineSpec.deploymentSpec) {
    throw new Error('Important infomation is missing. Pipeline Spec is invalid.');
  }
  return pipelineSpec;
}

export function tryConvertYamlToV2PipelineSpec(templateString: string): PipelineSpec | undefined {
  if (!templateString) {
    return undefined;
  }
  try {
    const template = getPipelineDefFromYaml(templateString);
    const pipelineSpec = convertPipelineSpecDef(template);
    return pipelineSpec;
  } catch {
    return undefined;
  }
}

export function convertYamlToPlatformSpec(template: string) {
  const platformSpecDef = getPlatformDefFromYaml(template);
  const platformSpec = PlatformSpec.fromJSON(platformSpecDef || '');
  return Object.keys(platformSpec.platforms).length !== 0 ? platformSpec : undefined;
}

// Given the PipelineSpec payload and targeted componentSpec, returns
// the `container` object for its image, command, arguments, etc.
export function getContainer(componentSpec: ComponentSpec, templateString: string) {
  const executionLabel = componentSpec?.executorLabel;
  if (!executionLabel) {
    return null;
  }

  const pipelineSpecDef = getPipelineDefFromYaml(templateString);
  const pipelineSpec = PipelineSpec.fromJSON(pipelineSpecDef);
  const deploymentSpecObj = pipelineSpec?.deploymentSpec;
  if (!deploymentSpecObj) {
    return null;
  }

  const deploymentSpec = PipelineDeploymentConfig.fromJSON(deploymentSpecObj);
  const executorsObj = deploymentSpec?.executors;
  if (!executorsObj) {
    return null;
  }

  const executorSpec = PipelineDeploymentConfig_ExecutorSpec.fromJSON(executorsObj[executionLabel]);
  return executorSpec ? executorSpec.container : null;
}
