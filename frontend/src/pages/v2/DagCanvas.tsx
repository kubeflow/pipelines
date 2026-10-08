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
  ControlButton,
  Panel,
  Edge,
  MiniMap,
  Node,
  OnNodeDrag,
  OnNodesChange,
  ReactFlowInstance,
} from '@xyflow/react';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
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
const positionKey = (node: PipelineNode) => JSON.stringify([node.id, node.parentId ?? null]);

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
  // Node cards use rem-based Tailwind dimensions. Match their actual size in
  // both the application's 13px root font and standalone Storybook's 16px root.
  const [nodeScale] = useState(
    () => (parseFloat(getComputedStyle(document.documentElement).fontSize) || 16) / 16,
  );
  const reactFlowInstance = useRef<ReactFlowInstance<PipelineNode, Edge> | null>(null);
  const lastFocusedNodeId = useRef<string | null>(null);
  const [expansion, setExpansion] = useState({
    collapsed: new Set<string>(),
    expanded: new Set<string>(),
    allCollapsed: false,
  });
  const [positions, setPositions] = useState<Record<string, { x: number; y: number }>>({});
  const grouped = useMemo(
    () =>
      getSubDagElements
        ? buildGroupedFlow(elements, layers, getSubDagElements, expansion.collapsed, {
            expanded: expansion.expanded,
            collapseAll: expansion.allCollapsed,
            nodeScale,
          })
        : undefined,
    [elements, layers, getSubDagElements, expansion, nodeScale],
  );
  const toggleGroup = useCallback((id: string, isCollapsed: boolean) => {
    setExpansion((previous) => {
      const collapsed = new Set(previous.collapsed);
      const expanded = new Set(previous.expanded);
      if (isCollapsed) {
        collapsed.delete(id);
        expanded.add(id);
      } else {
        collapsed.add(id);
        expanded.delete(id);
      }
      return { ...previous, collapsed, expanded };
    });
    // A changed group size requires a fresh layout, not stale drag offsets.
    setPositions({});
  }, []);
  const expandAll = useCallback(() => {
    // Keep explicit opt-ins, but do not bypass the automatic budget for new groups.
    setExpansion((previous) => ({ ...previous, collapsed: new Set(), allCollapsed: false }));
    setPositions({});
    requestAnimationFrame(() => {
      void reactFlowInstance.current?.fitView();
    });
  }, []);
  const collapseAll = useCallback(() => {
    setExpansion({ collapsed: new Set(), expanded: new Set(), allCollapsed: true });
    setPositions({});
    requestAnimationFrame(() => {
      void reactFlowInstance.current?.fitView();
    });
  }, []);
  const selectedId =
    grouped && selectedNodeId ? scopedNodeId(selectedNodeLayers, selectedNodeId) : selectedNodeId;
  const focusedId =
    grouped && focusNodeId ? scopedNodeId(selectedNodeLayers, focusNodeId) : focusNodeId;
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
          position: positions[positionKey(node)] ?? node.position,
          selected: node.id === selectedId,
          draggable: nodesDraggable,
        };
        if (node.type === GROUP_NODE_TYPE) {
          return {
            ...selectedNode,
            data: { ...node.data, expand: (id: string) => toggleGroup(id, node.data.collapsed) },
          };
        }
        return selectedNode.type === NodeTypeNames.SUB_DAG && selectedNode.data
          ? { ...selectedNode, data: { ...selectedNode.data, expand: subDagExpand } }
          : selectedNode;
      }),
    [elements, grouped, positions, selectedId, subDagExpand, toggleGroup, nodesDraggable],
  );
  const edges = useMemo(
    () => grouped?.edges ?? elements.filter((el): el is Edge => !isNode(el)),
    [elements, grouped],
  );

  const nodeById = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  const onNodesChange = useCallback<OnNodesChange<PipelineNode>>(
    (changes) => {
      // Only accept position updates; selection belongs to the details panel and
      // this read-only graph does not support deleting or resizing pipeline nodes.
      const updates: Record<string, { x: number; y: number }> = {};
      for (const change of changes) {
        const node = 'id' in change ? nodeById.get(change.id) : undefined;
        if (change.type === 'position' && change.position && node?.draggable) {
          updates[positionKey(node)] = change.position;
        }
      }
      if (Object.keys(updates).length) setPositions((previous) => ({ ...previous, ...updates }));
    },
    [nodeById],
  );

  const onNodeDragStop = useCallback<OnNodeDrag<PipelineNode>>(
    (_event, draggedNode) => {
      if (!nodesDraggable) return;
      setPositions((previous) => ({
        ...previous,
        [positionKey(draggedNode)]: draggedNode.position,
      }));
      if (grouped) return;
      const updatedElements = elements.map((el) =>
        isNode(el) && el.id === draggedNode.id ? { ...el, position: draggedNode.position } : el,
      );
      setFlowElements(updatedElements);
    },
    [elements, grouped, setFlowElements, nodesDraggable],
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
      <SubDagLayer layers={layers} onLayersUpdate={onLayersUpdate} />
      <div data-testid='DagCanvas' style={{ width: '100%', height: '100%' }}>
        <ReactFlowProvider>
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
            onNodesChange={onNodesChange}
            onNodeDragStop={nodesDraggable ? onNodeDragStop : undefined}
            deleteKeyCode={null}
          >
            <MiniMap />
            <Panel
              position='bottom-left'
              className='react-flow__controls'
              role='group'
              aria-label='Graph controls'
            >
              {getSubDagElements && (
                <>
                  <ControlButton
                    aria-label='Expand all'
                    title='Expand all sub-DAGs (large groups may require individual expansion)'
                    disabled={!nodes.some((node) => node.type === GROUP_NODE_TYPE)}
                    onClick={expandAll}
                  >
                    <ExpandMoreIcon fontSize='small' />
                  </ControlButton>
                  <ControlButton
                    aria-label='Collapse all'
                    title='Collapse all sub-DAGs'
                    disabled={!nodes.some((node) => node.type === GROUP_NODE_TYPE)}
                    onClick={collapseAll}
                  >
                    <ExpandLessIcon fontSize='small' />
                  </ControlButton>
                </>
              )}
              {/* Keep native controls in normal flow so visual and keyboard order agree. */}
              <Controls style={{ position: 'static', margin: 0, boxShadow: 'none' }} />
            </Panel>
            <Background />
          </ReactFlow>
        </ReactFlowProvider>
      </div>
    </>
  );
}
