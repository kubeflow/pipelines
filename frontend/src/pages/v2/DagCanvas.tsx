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

import {
  MouseEvent as ReactMouseEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  Controls,
  Edge,
  MiniMap,
  Node,
  OnNodeDrag,
  ReactFlowInstance,
} from '@xyflow/react';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import SubDagLayer from 'src/components/graph/SubDagLayer';
import SubDagGroupNode from 'src/components/graph/SubDagGroupNode';
import {
  buildGroupedFlow,
  GROUP_NODE_TYPE,
  LayerElements,
  scopedNodeId,
} from 'src/lib/v2/GroupedFlow';
import { color } from 'src/Css';
import {
  getTaskKeyFromNodeKey,
  isNode,
  NodeTypeNames,
  NODE_TYPES,
  PipelineFlowElement,
} from 'src/lib/v2/StaticFlow';

type PipelineNode = Node<FlowElementDataBase>;
const nodeTypes = { ...NODE_TYPES, [GROUP_NODE_TYPE]: SubDagGroupNode };

export interface DagCanvasProps {
  elements: PipelineFlowElement[];
  setFlowElements: (elements: PipelineFlowElement[]) => void;
  layers: string[];
  onLayersUpdate: (layers: string[]) => void;
  onElementClick: (event: ReactMouseEvent, element: PipelineFlowElement, layers: string[]) => void;
  getSubDagElements?: LayerElements;
  selectedNodeLayers?: string[];
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
  getSubDagElements,
  selectedNodeLayers = layers,
}: DagCanvasProps) {
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(new Set());
  const [positions, setPositions] = useState<Record<string, { x: number; y: number }>>({});
  const grouped = useMemo(
    () =>
      getSubDagElements
        ? buildGroupedFlow(elements, layers, getSubDagElements, collapsed)
        : undefined,
    [elements, layers, getSubDagElements, collapsed],
  );
  const toggleGroup = useCallback((id: string) => {
    setCollapsed((previous) => {
      const next = new Set(previous);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
    // A changed group size requires a fresh layout, not stale drag offsets.
    setPositions({});
  }, []);
  const selectedId =
    grouped && selectedNodeId ? scopedNodeId(selectedNodeLayers, selectedNodeId) : selectedNodeId;
  const focusedId =
    grouped && focusNodeId ? scopedNodeId(selectedNodeLayers, focusNodeId) : focusNodeId;
  const reactFlowInstance = useRef<ReactFlowInstance<PipelineNode, Edge> | null>(null);
  const lastFocusedNodeId = useRef<string | null>(null);
  const subDagExpand = useCallback(
    (nodeKey: string) => {
      const newLayers = [...layers, getTaskKeyFromNodeKey(nodeKey)];
      onLayersUpdate(newLayers);
    },
    [layers, onLayersUpdate],
  );

  const nodes = useMemo(
    () =>
      (grouped?.nodes ?? elements.filter(isNode)).map((node) => {
        const selectedNode = {
          ...node,
          position: positions[node.id] ?? node.position,
          selected: node.id === selectedId,
        };
        if (node.type === GROUP_NODE_TYPE) {
          return { ...selectedNode, data: { ...node.data, expand: toggleGroup } };
        }
        return selectedNode.type === NodeTypeNames.SUB_DAG && selectedNode.data
          ? { ...selectedNode, data: { ...selectedNode.data, expand: subDagExpand } }
          : selectedNode;
      }),
    [elements, grouped, positions, selectedId, subDagExpand, toggleGroup],
  );
  const edges = useMemo(
    () => grouped?.edges ?? elements.filter((el): el is Edge => !isNode(el)),
    [elements, grouped],
  );

  const onNodeDragStop = useCallback<OnNodeDrag<PipelineNode>>(
    (_event, draggedNode) => {
      if (grouped) {
        setPositions((previous) => ({ ...previous, [draggedNode.id]: draggedNode.position }));
        return;
      }
      const updatedElements = elements.map((el) =>
        isNode(el) && el.id === draggedNode.id ? { ...el, position: draggedNode.position } : el,
      );
      setFlowElements(updatedElements);
    },
    [elements, grouped, setFlowElements],
  );

  const handleNodeClick = useCallback(
    (event: ReactMouseEvent, node: PipelineNode) => {
      const source = grouped?.sources.get(node.id);
      onElementClick(event, source?.element ?? node, source?.layers ?? layers);
    },
    [grouped, layers, onElementClick],
  );

  const handleEdgeClick = useCallback(
    (event: ReactMouseEvent, edge: Edge) => {
      const source = grouped?.sources.get(edge.id);
      onElementClick(event, source?.element ?? edge, source?.layers ?? layers);
    },
    [grouped, layers, onElementClick],
  );

  const fitCurrentView = useCallback(
    (instance: ReactFlowInstance<PipelineNode, Edge>) => {
      const focusedNodes = focusedId ? nodes.filter((node) => node.id === focusedId) : undefined;
      void instance.fitView(focusedNodes?.length ? { nodes: focusedNodes } : undefined);
    },
    [focusedId, nodes],
  );

  // External synchronization: focus a node requested by URL navigation in React Flow.
  useEffect(() => {
    if (!focusedId) {
      lastFocusedNodeId.current = null;
      return;
    }
    if (reactFlowInstance.current && lastFocusedNodeId.current !== focusedId) {
      lastFocusedNodeId.current = focusedId;
      fitCurrentView(reactFlowInstance.current);
    }
  }, [fitCurrentView, focusedId]);

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
            onInit={(instance) => {
              reactFlowInstance.current = instance;
              lastFocusedNodeId.current = focusedId || null;
              fitCurrentView(instance);
            }}
            fitView
            fitViewOptions={focusedId ? { nodes: [{ id: focusedId }] } : undefined}
            minZoom={0.05}
            nodeTypes={nodeTypes}
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
