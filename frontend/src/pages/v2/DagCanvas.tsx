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
} from '@xyflow/react';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import SubDagLayer from 'src/components/graph/SubDagLayer';
import { color } from 'src/Css';
import {
  getTaskKeyFromNodeKey,
  isNode,
  NodeTypeNames,
  NODE_TYPES,
  PipelineFlowElement,
} from 'src/lib/v2/StaticFlow';

type PipelineNode = Node<FlowElementDataBase>;

export interface DagCanvasProps {
  elements: PipelineFlowElement[];
  setFlowElements: (elements: PipelineFlowElement[]) => void;
  layers: string[];
  onLayersUpdate: (layers: string[]) => void;
  onElementClick: (event: ReactMouseEvent, element: PipelineFlowElement) => void;
  nodesDraggable?: boolean;
}

export default function DagCanvas({
  elements,
  layers,
  onLayersUpdate,
  setFlowElements,
  onElementClick,
  nodesDraggable = true,
}: DagCanvasProps) {
  const subDagExpand = useCallback(
    (nodeKey: string) => {
      const newLayers = [...layers, getTaskKeyFromNodeKey(nodeKey)];
      onLayersUpdate(newLayers);
    },
    [layers, onLayersUpdate],
  );

  const nodes = useMemo(
    () =>
      elements
        .filter(isNode)
        .map((node) =>
          node.type === NodeTypeNames.SUB_DAG && node.data
            ? { ...node, data: { ...node.data, expand: subDagExpand } }
            : node,
        ),
    [elements, subDagExpand],
  );
  const edges = useMemo(() => elements.filter((el): el is Edge => !isNode(el)), [elements]);

  const onNodeDragStop = useCallback(
    (_event: ReactMouseEvent, draggedNode: PipelineNode) => {
      const updatedElements = elements.map((el) =>
        isNode(el) && el.id === draggedNode.id ? { ...el, position: draggedNode.position } : el,
      );
      setFlowElements(updatedElements);
    },
    [elements, setFlowElements],
  );

  const handleNodeClick = useCallback(
    (event: ReactMouseEvent, node: PipelineNode) => onElementClick(event, node),
    [onElementClick],
  );

  const handleEdgeClick = useCallback(
    (event: ReactMouseEvent, edge: Edge) => onElementClick(event, edge),
    [onElementClick],
  );

  return (
    <>
      <SubDagLayer layers={layers} onLayersUpdate={onLayersUpdate}></SubDagLayer>
      <div data-testid='DagCanvas' style={{ width: '100%', height: '100%' }}>
        <ReactFlowProvider>
          {/* onNodesChange/onEdgesChange are intentionally omitted: this DAG viewer
              does not need keyboard deletion, multi-select, or internal selection
              tracking. Drag persistence is handled via onNodeDragStop only. */}
          <ReactFlow<PipelineNode, Edge>
            style={{ background: color.lightGrey }}
            nodes={nodes}
            edges={edges}
            snapToGrid={true}
            nodesDraggable={nodesDraggable}
            onInit={(instance) => instance.fitView()}
            nodeTypes={NODE_TYPES}
            edgeTypes={{}}
            onNodeClick={handleNodeClick}
            onEdgeClick={handleEdgeClick}
            onNodeDragStop={nodesDraggable ? onNodeDragStop : undefined}
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
