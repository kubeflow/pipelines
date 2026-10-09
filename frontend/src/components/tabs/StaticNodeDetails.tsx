/*
 * Copyright 2021 The Kubeflow Authors
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

import { Button } from '../ui/button';
import {
  ComponentSpec,
  ParameterType_ParameterTypeEnum,
  PipelineSpec,
} from 'src/generated/pipeline_spec';
import { KeyValue } from 'src/lib/DetailsTableTypes';
import { getStringEnumKey } from 'src/lib/Utils';
import { getComponentSpec } from 'src/lib/v2/NodeUtils';
import {
  getKeysFromArtifactNodeKey,
  getTaskKeyFromNodeKey,
  isArtifactNode,
  isTaskNode,
  PipelineFlowElement,
} from 'src/lib/v2/StaticFlow';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { InspectionFields } from '../inspection/InspectionFields';

const NODE_INFO_UNKNOWN = <p className='kfp-pipeline-loading'>Unable to retrieve node info</p>;

interface StaticNodeDetailsProps {
  templateString: string;
  layers: string[];
  onLayerChange: (layers: string[]) => void;
  element: PipelineFlowElement | null;
}

export function StaticNodeDetails({
  templateString,
  layers,
  onLayerChange,
  element,
}: StaticNodeDetailsProps) {
  if (!element) {
    return NODE_INFO_UNKNOWN;
  }
  if (!templateString.trim()) {
    return NODE_INFO_UNKNOWN;
  }
  try {
    const pipelineSpec = WorkflowUtils.convertYamlToV2PipelineSpec(templateString);
    return (() => {
      if (isTaskNode(element.id)) {
        // Execution and Sub-DAG nodes
        return (
          <TaskNodeDetail
            templateString={templateString}
            pipelineSpec={pipelineSpec}
            element={element}
            layers={layers}
            onLayerChange={onLayerChange}
          />
        );
      } else if (isArtifactNode(element.id)) {
        return <ArtifactNodeDetail pipelineSpec={pipelineSpec} element={element} layers={layers} />;
      }
      return NODE_INFO_UNKNOWN;
    })();
  } catch (e) {
    console.error(e);
    return NODE_INFO_UNKNOWN;
  }
}

interface TaskNodeDetailProps {
  templateString: string;
  pipelineSpec: PipelineSpec;
  element: PipelineFlowElement;
  layers: string[];
  onLayerChange: (layers: string[]) => void;
}

function TaskNodeDetail({
  templateString,
  pipelineSpec,
  element,
  layers,
  onLayerChange,
}: TaskNodeDetailProps) {
  const taskKey = getTaskKeyFromNodeKey(element.id);
  const componentSpec = getComponentSpec(pipelineSpec, layers, taskKey);
  if (!componentSpec) {
    return NODE_INFO_UNKNOWN;
  }

  const onSubDagOpenClick = () => {
    onLayerChange([...layers, taskKey]);
  };

  const inputArtifacts = getInputArtifacts(componentSpec);
  const inputParameters = getInputParameters(componentSpec);
  const outputArtifacts = getOutputArtifacts(componentSpec);
  const outputParameters = getOutputParameters(componentSpec);

  const componentDag = componentSpec.dag;

  const container = WorkflowUtils.getContainer(componentSpec, templateString);
  const args = container?.args;
  const command = container?.command;
  const image = container?.image;

  return (
    <div>
      {componentDag && (
        <div>
          <Button onClick={onSubDagOpenClick}>Open Sub-DAG</Button>
        </div>
      )}
      {inputArtifacts && (
        <div>
          <InspectionFields title='Input Artifacts' fields={inputArtifacts} />
        </div>
      )}
      {inputParameters && (
        <div>
          <InspectionFields title='Input Parameters' fields={inputParameters} />
        </div>
      )}
      {outputArtifacts && (
        <div>
          <InspectionFields title='Output Artifacts' fields={outputArtifacts} />
        </div>
      )}
      {outputParameters && (
        <div>
          <InspectionFields title='Output Parameters' fields={outputParameters} />
        </div>
      )}
      {image && (
        <div>
          <h3>Image</h3>
          <pre>{image}</pre>
        </div>
      )}
      {command && (
        <div>
          <h3>Command</h3>
          <div className='font-mono'>
            {command.map((cmd, index) => {
              return (
                <div key={index} style={{ whiteSpace: 'pre-wrap' }}>
                  {cmd}
                </div>
              );
            })}
          </div>
        </div>
      )}
      {args && (
        <div>
          <h3>Arguments</h3>
          <div className='font-mono'>
            {args.map((arg, index) => {
              return (
                <div key={index} style={{ whiteSpace: 'pre-wrap' }}>
                  {arg}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

interface ArtifactNodeDetailProps {
  pipelineSpec: PipelineSpec;
  element: PipelineFlowElement;
  layers: string[];
}

function ArtifactNodeDetail({ pipelineSpec, element, layers }: ArtifactNodeDetailProps) {
  const [taskKey, artifactKey] = getKeysFromArtifactNodeKey(element.id);
  const componentSpec = getComponentSpec(pipelineSpec, layers, taskKey);

  if (!componentSpec) {
    return NODE_INFO_UNKNOWN;
  }

  const artifactType = getOutputArtifacts(componentSpec).filter((a) => a[0] === artifactKey);
  const artifactInfo = [
    ['Upstream Task', taskKey],
    ['Artifact Name', artifactKey],
    ['Artifact Type', artifactType[0][1]],
  ];
  return (
    <div>
      {artifactInfo && (
        <div>
          <InspectionFields title='Artifact Info' fields={artifactInfo} />
        </div>
      )}
    </div>
  );
}

function getInputArtifacts(componentSpec: ComponentSpec) {
  const inputDefinitions = componentSpec.inputDefinitions;
  const artifacts = inputDefinitions?.artifacts;
  if (!artifacts) {
    return Array<KeyValue<string>>();
  }
  const inputArtifacts: Array<KeyValue<string>> = Object.keys(artifacts).map((key) => {
    const artifactSpec = artifacts[key];
    const type = artifactSpec.artifactType;
    let value = type?.schemaTitle || type?.instanceSchema;
    if (type && type.schemaVersion) {
      value += ' (version: ' + type?.schemaVersion + ')';
    }
    return [key, value];
  });
  return inputArtifacts;
}

function getOutputArtifacts(componentSpec: ComponentSpec) {
  const outputDefinitions = componentSpec.outputDefinitions;
  const artifacts = outputDefinitions?.artifacts;
  if (!artifacts) {
    return Array<KeyValue<string>>();
  }
  const outputArtifacts: Array<KeyValue<string>> = Object.keys(artifacts).map((key) => {
    const artifactSpec = artifacts[key];
    const type = artifactSpec.artifactType;
    let value = type?.schemaTitle || type?.instanceSchema;
    if (type && type.schemaVersion) {
      value += ' (version: ' + type?.schemaVersion + ')';
    }
    return [key, value];
  });
  return outputArtifacts;
}

function getInputParameters(componentSpec: ComponentSpec) {
  const inputDefinitions = componentSpec.inputDefinitions;
  const parameters = inputDefinitions?.parameters;
  if (!parameters) {
    return Array<KeyValue<string>>();
  }
  const inputParameters: Array<KeyValue<string>> = Object.keys(parameters).map((key) => {
    const parameterSpec = parameters[key];
    const type = parameterSpec?.parameterType;
    return [key, getStringEnumKey(ParameterType_ParameterTypeEnum, type)];
  });
  return inputParameters;
}

function getOutputParameters(componentSpec: ComponentSpec) {
  const outputDefinitions = componentSpec.outputDefinitions;
  const parameters = outputDefinitions?.parameters;
  if (!parameters) {
    return Array<KeyValue<string>>();
  }
  const outputParameters: Array<KeyValue<string>> = Object.keys(parameters).map((key) => {
    const parameterSpec = parameters[key];
    const type = parameterSpec?.parameterType;
    return [key, getStringEnumKey(ParameterType_ParameterTypeEnum, type)];
  });
  return outputParameters;
}
