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

import React, { MouseEvent as ReactMouseEvent, useCallback, useMemo } from 'react';
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  Controls,
  Edge,
  MiniMap,
  Node,
  applyNodeChanges,
  NodeChange,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import SubDagLayer from 'src/components/graph/SubDagLayer';
import { color } from 'src/Css';
import {
  getTaskKeyFromNodeKey,
  NodeTypeNames,
  NODE_TYPES,
  PipelineFlowElement,
} from 'src/lib/v2/StaticFlow';

type PipelineNode = Node<FlowElementDataBase>;

function isNode(el: PipelineFlowElement): el is PipelineNode {
  return 'position' in el;
}

export interface DagCanvasProps {
  elements: PipelineFlowElement[];
  setFlowElements: (elements: PipelineFlowElement[]) => void;
  layers: string[];
  onLayersUpdate: (layers: string[]) => void;
  onElementClick: (event: ReactMouseEvent, element: PipelineFlowElement) => void;
}

export default function DagCanvas({
  elements,
  layers,
  onLayersUpdate,
  setFlowElements,
  onElementClick,
}: DagCanvasProps) {
  const nodes = useMemo(() => elements.filter(isNode), [elements]);
  const edges = useMemo(() => elements.filter((el): el is Edge => !isNode(el)), [elements]);

  const onNodesChange = useCallback(
    (changes: NodeChange[]) => {
      const updatedNodes = applyNodeChanges(changes, nodes) as PipelineNode[];
      setFlowElements([...updatedNodes, ...edges]);
    },
    [nodes, edges, setFlowElements],
  );

  const subDagExpand = (nodeKey: string) => {
    const newLayers = [...layers, getTaskKeyFromNodeKey(nodeKey)];
    onLayersUpdate(newLayers);
  };

  nodes.forEach(node => {
    if (node && node.type === NodeTypeNames.SUB_DAG && node.data) {
      node.data.expand = subDagExpand;
    }
  });

  return (
    <>
      <SubDagLayer layers={layers} onLayersUpdate={onLayersUpdate}></SubDagLayer>
      <div data-testid='DagCanvas' style={{ width: '100%', height: '100%' }}>
        <ReactFlowProvider>
          <ReactFlow<PipelineNode, Edge>
            style={{ background: color.lightGrey }}
            nodes={nodes}
            edges={edges}
            snapToGrid={true}
            onInit={instance => instance.fitView()}
            nodeTypes={NODE_TYPES}
            edgeTypes={{}}
            onNodeClick={(event, node) => onElementClick(event, node)}
            onEdgeClick={(event, edge) => onElementClick(event, edge)}
            onNodesChange={onNodesChange}
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
