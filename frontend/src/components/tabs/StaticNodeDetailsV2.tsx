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
import { KeyValue } from 'src/lib/StaticGraphParser';
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
    const pipelineSpec = WorkflowUtils.convertJsonToV2PipelineSpec(templateString);
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

  const componentDag = componentSpec.getDag();

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
  let currentDag = pipelineSpec.getRoot()?.getDag();
  const taskLayers = [...layers.slice(1), taskKey];
  let componentSpec;
  for (let i = 0; i < taskLayers.length; i++) {
    const pipelineTaskSpec = currentDag?.getTasksMap().get(taskLayers[i]);
    const componentName = pipelineTaskSpec?.getComponentRef()?.getName();
    if (!componentName) {
      return null;
    }
    componentSpec = pipelineSpec.getComponentsMap().get(componentName);
    if (!componentSpec) {
      return null;
    }
    currentDag = componentSpec.getDag();
  }
  return componentSpec;
}

function getInputArtifacts(componentSpec: ComponentSpec) {
  const inputDefinitions = componentSpec.getInputDefinitions();
  const artifacts = inputDefinitions?.getArtifactsMap();
  const inputArtifacts: Array<KeyValue<string>> = (artifacts?.toArray() || []).map(entry => {
    const artifactSpec = artifacts?.get(entry[0]);
    const type = artifactSpec?.getArtifactType();
    let value = type?.getSchemaTitle() || type?.getInstanceSchema();
    if (type && type.getSchemaVersion()) {
      value += ' (version: ' + type?.getSchemaVersion() + ')';
    }
    return [entry[0], value];
  });
  return inputArtifacts;
}

function getOutputArtifacts(componentSpec: ComponentSpec) {
  const outputDefinitions = componentSpec.getOutputDefinitions();
  const artifacts = outputDefinitions?.getArtifactsMap();
  const inputArtifacts: Array<KeyValue<string>> = (artifacts?.toArray() || []).map(entry => {
    const artifactSpec = artifacts?.get(entry[0]);
    const type = artifactSpec?.getArtifactType();
    let value = type?.getSchemaTitle() || type?.getInstanceSchema();
    if (type && type.getSchemaVersion()) {
      value += ' (version: ' + type?.getSchemaVersion() + ')';
    }
    return [entry[0], value];
  });
  return inputArtifacts;
}

// TODO: Directly calling PrimitiveType.PrimitiveTypeEnum cannot get the string key of enum.
// Temporarily duplicate the enum definition here, until we figure out the solution.
enum PrimitiveTypeEnum {
  PRIMITIVE_TYPE_UNSPECIFIED = 0,
  INT = 1,
  DOUBLE = 2,
  STRING = 3,
}

function getInputParameters(componentSpec: ComponentSpec) {
  const inputDefinitions = componentSpec.getInputDefinitions();
  const parameters = inputDefinitions?.getParametersMap();
  const inputParameters: Array<KeyValue<string>> = (parameters?.toArray() || []).map(entry => {
    const parameterSpec = parameters?.get(entry[0]);
    // TODO: type is deprecated in favor of parameter_type.
    const type = parameterSpec?.getType();
    return [entry[0], PrimitiveTypeEnum[type || 0]];
  });
  return inputParameters;
}

function getOutputParameters(componentSpec: ComponentSpec) {
  const outputDefinitions = componentSpec.getOutputDefinitions();
  const parameters = outputDefinitions?.getParametersMap();
  const outputParameters: Array<KeyValue<string>> = (parameters?.toArray() || []).map(entry => {
    const parameterSpec = parameters?.get(entry[0]);
    // TODO: type is deprecated in favor of parameter_type.
    const type = parameterSpec?.getType();
    return [entry[0], PrimitiveTypeEnum[type || 0]];
  });
  return outputParameters;
}
