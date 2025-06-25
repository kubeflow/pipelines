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

import React, { MouseEvent as ReactMouseEvent } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  ReactFlowProvider,
  ReactFlowInstance,
  Node,
  Edge,
} from 'reactflow';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import SubDagLayer from 'src/components/graph/SubDagLayer';
import { color } from 'src/Css';
import { getTaskKeyFromNodeKey, NodeTypeNames, NODE_TYPES } from 'src/lib/v2/StaticFlow';

// Type aliases for backward compatibility
type FlowElement<T = any> = Node<T> | Edge<T>;
type Elements<T = any> = FlowElement<T>[];

export interface DagCanvasProps {
  elements: Elements<FlowElementDataBase>;
  setFlowElements: (elements: Elements<any>) => void;
  layers: string[];
  onLayersUpdate: (layers: string[]) => void;
  onElementClick: (event: ReactMouseEvent, element: Node | Edge) => void;
}

export default function DagCanvas({
  elements,
  layers,
  onLayersUpdate,
  setFlowElements,
  onElementClick,
}: DagCanvasProps) {
  const onLoad = (reactFlowInstance: ReactFlowInstance) => {
    reactFlowInstance.fitView();
  };

  const subDagExpand = (nodeKey: string) => {
    const newLayers = [...layers, getTaskKeyFromNodeKey(nodeKey)];
    onLayersUpdate(newLayers);
  };

  elements.forEach(elem => {
    // For each SubDag node, provide a callback function if expand button is clicked.
    if (elem && elem.type === NodeTypeNames.SUB_DAG && elem.data) {
      elem.data.expand = subDagExpand;
    }
  });

  return (
    <>
      <SubDagLayer layers={layers} onLayersUpdate={onLayersUpdate}></SubDagLayer>
      <div data-testid='DagCanvas' style={{ width: '100%', height: '100%' }}>
        <ReactFlowProvider>
          <ReactFlow
            style={{ background: color.lightGrey }}
            nodes={elements.filter(el => !('source' in el)) as Node[]}
            edges={elements.filter(el => 'source' in el) as Edge[]}
            snapToGrid={true}
            onInit={onLoad}
            nodeTypes={NODE_TYPES}
            edgeTypes={{}}
            onNodeClick={(event, node) => onElementClick && onElementClick(event, node)}
            onEdgeClick={(event, edge) => onElementClick && onElementClick(event, edge)}
            onNodeDragStop={(event, node) => {
              setFlowElements(
                elements.map(value => {
                  if (value.id === node.id) {
                    return node;
                  }
                  return value;
                }),
              );
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
}
