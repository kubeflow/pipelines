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

<<<<<<< HEAD
import { MouseEvent as ReactMouseEvent, useCallback, useEffect, useMemo, useRef } from 'react';
=======
>>>>>>> 943648036 (fix(frontend): render DAG minimap and node selection)
import {
  MouseEvent as ReactMouseEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from 'react';
import {
  applyNodeChanges,
  Background,
  Controls,
  Edge,
  MiniMap,
  Node,
<<<<<<< HEAD
  OnNodeDrag,
  ReactFlowInstance,
=======
  NodeChange,
  OnNodeDrag,
  ReactFlow,
  ReactFlowInstance,
  ReactFlowProvider,
  useNodesState,
>>>>>>> 943648036 (fix(frontend): render DAG minimap and node selection)
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
  onElementClick: (
    event: ReactMouseEvent,
    element: PipelineFlowElement,
  ) => void;
  nodesDraggable?: boolean;
  selectedNodeId?: string;
  focusNodeId?: string;
}

export default function DagCanvas({
  elements,
  layers,
  onLayersUpdate,
  setFlowElements,
  onElementClick,
  nodesDraggable = true,
  selectedNodeId,
  focusNodeId,
}: DagCanvasProps) {
<<<<<<< HEAD
  const reactFlowInstance = useRef<ReactFlowInstance<PipelineNode, Edge> | null>(null);
  const lastFocusedNodeId = useRef<string | null>(null);
=======
  const reactFlowInstance = useRef<
    ReactFlowInstance<PipelineNode, Edge> | null
  >(null);
  const lastFocusedNodeId = useRef<string | null>(null);

>>>>>>> 943648036 (fix(frontend): render DAG minimap and node selection)
  const subDagExpand = useCallback(
    (nodeKey: string) => {
      const newLayers = [...layers, getTaskKeyFromNodeKey(nodeKey)];
      onLayersUpdate(newLayers);
    },
    [layers, onLayersUpdate],
  );

  const nodes = useMemo(
    () =>
      elements.filter(isNode).map((node) => {
        const selectedNode = { ...node, selected: node.id === selectedNodeId };
        return selectedNode.type === NodeTypeNames.SUB_DAG && selectedNode.data
          ? { ...selectedNode, data: { ...selectedNode.data, expand: subDagExpand } }
          : selectedNode;
      }),
    [elements, selectedNodeId, subDagExpand],
  );

  const onNodeDragStop = useCallback<OnNodeDrag<PipelineNode>>(
    (_event, draggedNode) => {
      const updatedElements = elements.map((el) =>
        isNode(el) && el.id === draggedNode.id
          ? { ...el, position: draggedNode.position }
          : el,
      );

      setFlowElements(updatedElements);
    },
    [elements, setFlowElements],
  );

  const handleNodeClick = useCallback(
    (event: ReactMouseEvent, node: PipelineNode) =>
      onElementClick(event, node),
    [onElementClick],
  );

  const handleEdgeClick = useCallback(
    (event: ReactMouseEvent, edge: Edge) => onElementClick(event, edge),
    [onElementClick],
  );

  const fitCurrentView = useCallback(
    (instance: ReactFlowInstance<PipelineNode, Edge>) => {
      const focusedNodes = focusNodeId
        ? nodes.filter((node) => node.id === focusNodeId)
        : undefined;
<<<<<<< HEAD
      void instance.fitView(focusedNodes?.length ? { nodes: focusedNodes } : undefined);
=======

      void instance.fitView(
        focusedNodes?.length ? { nodes: focusedNodes } : undefined,
      );
>>>>>>> 943648036 (fix(frontend): render DAG minimap and node selection)
    },
    [focusNodeId, nodes],
  );

  useEffect(() => {
    if (!focusNodeId) {
      lastFocusedNodeId.current = null;
      return;
    }
<<<<<<< HEAD
    if (reactFlowInstance.current && lastFocusedNodeId.current !== focusNodeId) {
=======

    if (
      reactFlowInstance.current &&
      lastFocusedNodeId.current !== focusNodeId
    ) {
>>>>>>> 943648036 (fix(frontend): render DAG minimap and node selection)
      lastFocusedNodeId.current = focusNodeId;
      fitCurrentView(reactFlowInstance.current);
    }
  }, [fitCurrentView, focusNodeId]);

  return (
    <>
      <SubDagLayer
        layers={layers}
        onLayersUpdate={onLayersUpdate}
      ></SubDagLayer>

      <div
        data-testid='DagCanvas'
        style={{ width: '100%', height: '100%' }}
      >
        <ReactFlowProvider>
<<<<<<< HEAD
          {/* onNodesChange/onEdgesChange are intentionally omitted: this DAG viewer
              does not need keyboard deletion, multi-select, or internal selection
              tracking. Drag persistence is handled via onNodeDragStop only. */}
=======
          {/* React Flow change handling is intentionally limited to internal
              dimension changes. Selection is controlled by selectedNodeId.
              Drag persistence is handled via onNodeDragStop only. */}
>>>>>>> 943648036 (fix(frontend): render DAG minimap and node selection)
          <ReactFlow<PipelineNode, Edge>
            style={{ background: color.lightGrey }}
            nodes={nodes}
            edges={edges}
            snapToGrid={true}
            nodesDraggable={nodesDraggable}
<<<<<<< HEAD
=======
            onNodesChange={onNodesChange}
>>>>>>> 943648036 (fix(frontend): render DAG minimap and node selection)
            onInit={(instance) => {
              reactFlowInstance.current = instance;
              lastFocusedNodeId.current = focusNodeId || null;
              fitCurrentView(instance);
            }}
            nodeTypes={NODE_TYPES}
            edgeTypes={{}}
            onNodeClick={handleNodeClick}
            onEdgeClick={handleEdgeClick}
<<<<<<< HEAD
            onNodeDragStop={nodesDraggable ? onNodeDragStop : undefined}
=======
            onNodeDragStop={
              nodesDraggable ? onNodeDragStop : undefined
            }
>>>>>>> 943648036 (fix(frontend): render DAG minimap and node selection)
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