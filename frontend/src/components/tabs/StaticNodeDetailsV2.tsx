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

import { Button } from '@material-ui/core';
import * as React from 'react';
import { FlowElement } from 'react-flow-renderer';
import { ComponentSpec, PipelineSpec } from 'src/generated/pipeline_spec';
import { ParameterType } from 'src/generated/pipeline_spec/pipeline_spec_pb';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { getStringEnumKey } from 'src/lib/Utils';
import {
  getKeysFromArtifactNodeKey,
  getTaskKeyFromNodeKey,
  isArtifactNode,
  isTaskNode,
} from 'src/lib/v2/StaticFlow';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import DetailsTable from '../DetailsTable';
import { FlowElementDataBase } from '../graph/Constants';

const NODE_INFO_UNKNOWN = (
  <div className='relative flex flex-col h-screen'>
    <div className='absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2'>
      Unable to retrieve node info
    </div>
  </div>
);

interface StaticNodeDetailsV2Props {
  templateString: string;
  layers: string[];
  onLayerChange: (layers: string[]) => void;
  element: FlowElement<FlowElementDataBase> | null;
}

export function StaticNodeDetailsV2({
  templateString,
  layers,
  onLayerChange,
  element,
}: StaticNodeDetailsV2Props) {
  if (!element) {
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
  element: FlowElement<FlowElementDataBase>;
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
  const args = container?.['args'];
  const command = container?.['command'];
  const image = container?.['image'];

  return (
    <div>
      {componentDag && (
        <div>
          <Button variant='contained' onClick={onSubDagOpenClick}>
            Open Workflow
          </Button>
        </div>
      )}
      {inputArtifacts && (
        <div>
          <DetailsTable title='Input Artifacts' fields={inputArtifacts} />
        </div>
      )}
      {inputParameters && (
        <div>
          <DetailsTable title='Input Parameters' fields={inputParameters} />
        </div>
      )}
      {outputArtifacts && (
        <div>
          <DetailsTable title='Output Artifacts' fields={outputArtifacts} />
        </div>
      )}
      {outputParameters && (
        <div>
          <DetailsTable title='Output Parameters' fields={outputParameters} />
        </div>
      )}
      {image && (
        <div>
          <div className='text-xl font-bold pt-6'>Image</div>
          <div className='font-mono '>{image}</div>
        </div>
      )}
      {command && (
        <div>
          <div className='text-xl font-bold pt-6'>Command</div>
          <div className='font-mono '>{command}</div>
        </div>
      )}
      {args && (
        <div>
          <div className='text-xl font-bold pt-6'>Arguments</div>
          <div className='font-mono '>{args}</div>
        </div>
      )}
    </div>
  );
}

interface ArtifactNodeDetailProps {
  pipelineSpec: PipelineSpec;
  element: FlowElement<FlowElementDataBase>;
  layers: string[];
}

function ArtifactNodeDetail({ pipelineSpec, element, layers }: ArtifactNodeDetailProps) {
  const [taskKey, artifactKey] = getKeysFromArtifactNodeKey(element.id);
  const componentSpec = getComponentSpec(pipelineSpec, layers, taskKey);

  if (!componentSpec) {
    return NODE_INFO_UNKNOWN;
  }

  const artifactType = getOutputArtifacts(componentSpec).filter(a => a[0] === artifactKey);
  const artifactInfo = [
    ['Upstream Task', taskKey],
    ['Artifact Name', artifactKey],
    ['Artifact Type', artifactType[0][1]],
  ];
  return (
    <div>
      {artifactInfo && (
        <div>
          <DetailsTable title='Artifact Info' fields={artifactInfo} />
        </div>
      )}
    </div>
  );
}

function getComponentSpec(pipelineSpec: PipelineSpec, layers: string[], taskKey: string) {
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

function getInputArtifacts(componentSpec: ComponentSpec) {
  const inputDefinitions = componentSpec.inputDefinitions;
  const artifacts = inputDefinitions?.artifacts;
  if (!artifacts) {
    return Array<KeyValue<string>>();
  }
  const inputArtifacts: Array<KeyValue<string>> = Object.keys(artifacts).map(key => {
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
  const outputArtifacts: Array<KeyValue<string>> = Object.keys(artifacts).map(key => {
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
  const inputParameters: Array<KeyValue<string>> = Object.keys(parameters).map(key => {
    const parameterSpec = parameters[key];
    const type = parameterSpec?.parameterType;
    return [key, getStringEnumKey(ParameterType.ParameterTypeEnum, type)];
  });
  return inputParameters;
}

function getOutputParameters(componentSpec: ComponentSpec) {
  const outputDefinitions = componentSpec.outputDefinitions;
  const parameters = outputDefinitions?.parameters;
  if (!parameters) {
    return Array<KeyValue<string>>();
  }
  const outputParameters: Array<KeyValue<string>> = Object.keys(parameters).map(key => {
    const parameterSpec = parameters[key];
    const type = parameterSpec?.parameterType;
    return [key, getStringEnumKey(ParameterType.ParameterTypeEnum, type)];
  });
  return outputParameters;
}
