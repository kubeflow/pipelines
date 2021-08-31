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

import React from 'react';
import ReactFlow, {
  Background,
  Controls,
  Elements,
  MiniMap,
  OnLoadParams,
  Node,
  ReactFlowProvider,
} from 'react-flow-renderer';
import SubDagLayer from 'src/components/graph/SubDagLayer';
import { color } from 'src/Css';
import { getTaskKeyFromNodeKey, NODE_TYPES, TaskType } from 'src/lib/v2/StaticFlow';

export interface StaticCanvasProps {
  elements: Elements;
  layers: string[];
  onLayersUpdate: (layers: string[]) => void;
}

const StaticCanvas = ({ elements, layers, onLayersUpdate }: StaticCanvasProps) => {
  const onLoad = (reactFlowInstance: OnLoadParams) => {
    reactFlowInstance.fitView();
  };

  const doubleClickNode = (node: Node) => {
    if (node.data['taskType'] !== TaskType.DAG) {
      return;
    }
    const newLayers = [...layers, getTaskKeyFromNodeKey(node.id)]; //remove `task.`
    onLayersUpdate(newLayers);
  };

  return (
    <>
      <SubDagLayer layers={layers} onLayersUpdate={onLayersUpdate}></SubDagLayer>
      <div data-testid='StaticCanvas' style={{ width: '100%', height: '100%' }}>
        <ReactFlowProvider>
          <ReactFlow
            style={{ background: color.lightGrey }}
            elements={elements}
            snapToGrid={true}
            onLoad={onLoad}
            nodeTypes={NODE_TYPES}
            edgeTypes={{}}
            onNodeDoubleClick={(event, element) => {
              doubleClickNode(element);
            }}
          >
            <MiniMap />
            <Controls />
            <Background />
          </ReactFlow>
        </ReactFlowProvider>
      </div>
    </>
  );
};
export default StaticCanvas;
