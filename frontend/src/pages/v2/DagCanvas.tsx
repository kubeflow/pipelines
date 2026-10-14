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
  KeyboardEvent as ReactKeyboardEvent,
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
  Edge,
  MiniMap,
  Panel,
  Node,
  OnNodeDrag,
  OnNodesChange,
  ReactFlowInstance,
} from '@xyflow/react';
import 'src/components/graph/Graph.css';
import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import { getTaskStatus } from 'src/lib/StatusUtils';
import GraphControls from 'src/components/graph/GraphControls';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import { NODE_TYPES } from 'src/components/graph/NodeTypes';
import { SUB_DAG_HEADER_CLASS_NAME } from 'src/components/graph/SubDagGroupNode';
import { buildGroupedFlow, GROUP_NODE_TYPE, layerNodeId } from 'src/lib/v2/GroupedFlow';
import type {
  LayerElementsResolver,
  LayerFlowElement,
  PipelineFlowElement,
} from 'src/lib/v2/FlowTypes';
import { getTaskNodeKey } from 'src/lib/v2/StaticFlow';
import { loadGraphExpansion, saveGraphExpansion } from 'src/lib/v2/GraphExpansionState';

type PipelineNode = Node<FlowElementDataBase>;
const emptyLayer: LayerElementsResolver = () => [];
const positionKey = (node: PipelineNode) => JSON.stringify([node.id, node.parentId ?? null]);
// Scrolling the view moves graph content in the opposite direction.
const panDirections = new Map([
  ['ArrowLeft', { x: 40, y: 0 }],
  ['ArrowRight', { x: -40, y: 0 }],
  ['ArrowUp', { x: 0, y: 40 }],
  ['ArrowDown', { x: 0, y: -40 }],
]);
const editableSelector =
  'input, textarea, select, [contenteditable]:not([contenteditable="false"]), [role="textbox"], [role="combobox"], [role="slider"], [role="spinbutton"]';

export interface DagCanvasProps {
  elements: PipelineFlowElement[];
  layers: string[];
  onElementClick: (event: ReactMouseEvent, selection: LayerFlowElement) => void;
  getSubDagElements?: LayerElementsResolver;
  selectedElement?: LayerFlowElement | null;
  nodesDraggable?: boolean;
  focusNodeId?: string;
  /** Stable run or pipeline-version identity; omitted for ephemeral graphs. */
  graphId?: string;
}

export default function DagCanvas(props: DagCanvasProps) {
  return <DagCanvasView key={props.graphId} {...props} />;
}

function DagCanvasView({
  elements,
  layers,
  onElementClick,
  nodesDraggable = true,
  selectedElement,
  focusNodeId,
  getSubDagElements,
  graphId,
}: DagCanvasProps) {
  const reactFlowInstance = useRef<ReactFlowInstance<PipelineNode, Edge> | null>(null);
  const lastFocusedNodeId = useRef<string | null>(null);
  const [locked, setLocked] = useState(true);
  const canDrag = nodesDraggable && !locked;
  const [expansion, setExpansion] = useState(() => loadGraphExpansion(graphId));
  // External synchronization: persist preferences after committed state changes.
  useEffect(() => saveGraphExpansion(graphId, expansion), [graphId, expansion]);
  const [positions, setPositions] = useState<Record<string, { x: number; y: number }>>({});
  const selectionLayers = selectedElement?.layers ?? layers;
  const selectedId = selectedElement
    ? layerNodeId(selectionLayers, selectedElement.element.id)
    : undefined;
  const focusedId = focusNodeId ? layerNodeId(selectionLayers, focusNodeId) : undefined;
  const [previousFocus, setPreviousFocus] = useState<string>();
  if (focusedId !== previousFocus) {
    setPreviousFocus(focusedId);
    if (focusedId) {
      // A new deep link opens its ancestors once; later user collapse choices win.
      const expanded = new Set(expansion.expanded);
      const collapsed = new Set(expansion.collapsed);
      for (let index = layers.length; index < selectionLayers.length; index++) {
        const id = layerNodeId(
          selectionLayers.slice(0, index),
          getTaskNodeKey(selectionLayers[index]),
        );
        expanded.add(id);
        collapsed.delete(id);
      }
      setExpansion({ ...expansion, expanded, collapsed });
      setPositions({});
    }
  }
  const grouped = useMemo(
    () =>
      buildGroupedFlow(elements, layers, getSubDagElements ?? emptyLayer, expansion.collapsed, {
        expanded: expansion.expanded,
        expansionMode: getSubDagElements ? expansion.mode : 'none',
      }),
    [elements, layers, getSubDagElements, expansion],
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
  const resetPositionsAndFitView = useCallback(() => {
    setPositions({});
    requestAnimationFrame(() => {
      void reactFlowInstance.current?.fitView();
    });
  }, []);
  const expandAll = useCallback(() => {
    setExpansion({ collapsed: new Set(), expanded: new Set(), mode: 'all' });
    resetPositionsAndFitView();
  }, [resetPositionsAndFitView]);
  const collapseAll = useCallback(() => {
    setExpansion({ collapsed: new Set(), expanded: new Set(), mode: 'none' });
    resetPositionsAndFitView();
  }, [resetPositionsAndFitView]);
  const nodes = useMemo(
    () =>
      grouped.nodes.map<PipelineNode>((node) => {
        const selectedNode = {
          ...node,
          position: positions[positionKey(node)] ?? node.position,
          selected: node.id === selectedId,
          draggable: canDrag,
          selectable: true,
        };
        if (node.type === GROUP_NODE_TYPE) {
          return {
            ...selectedNode,
            // Let empty group space hit the pane for panning; only the header accepts pointers.
            style: { ...node.style, pointerEvents: 'none' },
            dragHandle: `.${SUB_DAG_HEADER_CLASS_NAME}`,
            data: { ...node.data, expand: (id: string) => toggleGroup(id, node.data.collapsed) },
          };
        }
        return selectedNode;
      }),
    [grouped, positions, selectedId, toggleGroup, canDrag],
  );
  const edges = useMemo(() => {
    const states = new Map(nodes.map((node) => [node.id, node.data.state]));
    return grouped.edges.map((edge) => ({
      ...edge,
      className: `${edge.className || ''} kfp-graph-edge-${getTaskStatus(states.get(edge.target)).tone}`,
      style: { ...edge.style, strokeWidth: 1.5 },
    }));
  }, [grouped.edges, nodes]);
  const visibleStates = [
    ...new Set(
      nodes
        .map((node) => node.data.state as PipelineTaskTaskState | undefined)
        .filter((state): state is PipelineTaskTaskState => state !== undefined),
    ),
  ];

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
      if (!canDrag) return;
      setPositions((previous) => ({
        ...previous,
        [positionKey(draggedNode)]: draggedNode.position,
      }));
    },
    [canDrag],
  );

  const handleElementClick = useCallback(
    (event: ReactMouseEvent, element: PipelineFlowElement) => {
      onElementClick(event, grouped.sources.get(element.id) ?? { element, layers });
    },
    [grouped, layers, onElementClick],
  );

  const handleGraphKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    const instance = reactFlowInstance.current;
    if (
      !instance ||
      event.altKey ||
      event.ctrlKey ||
      event.metaKey ||
      (event.target instanceof Element && event.target.closest(editableSelector))
    )
      return;
    if (event.key === 'f' || event.key === 'F') {
      event.preventDefault();
      event.stopPropagation();
      void instance.fitView();
      return;
    }
    const direction = panDirections.get(event.key);
    if (!locked || !direction) return;
    event.preventDefault();
    event.stopPropagation();
    const viewport = instance.getViewport();
    void instance.setViewport({
      ...viewport,
      x: viewport.x + direction.x,
      y: viewport.y + direction.y,
    });
  };

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
    <div
      data-testid='DagCanvas'
      role='region'
      aria-label='Pipeline graph'
      aria-description={
        locked
          ? 'Drag or use arrow keys to pan. Press F to fit the graph. Click nodes to inspect details. Unlock to move nodes.'
          : 'Drag nodes or use arrow keys on a selected node to move it. Press F to fit the graph.'
      }
      tabIndex={0}
      className={`kfp-graph-canvas ${locked ? 'kfp-graph-locked' : ''}`}
      onKeyDownCapture={handleGraphKeyDown}
      onPointerDownCapture={(event) => {
        if (
          event.target instanceof Element &&
          !event.target.closest(editableSelector + ', button, .react-flow__panel')
        )
          event.currentTarget.focus({ preventScroll: true });
      }}
      style={{ width: '100%', height: '100%' }}
    >
      <ReactFlowProvider>
        <ReactFlow<PipelineNode, Edge>
          nodes={nodes}
          edges={edges}
          snapToGrid={true}
          nodesDraggable={canDrag}
          elementsSelectable
          selectionKeyCode={locked ? null : 'Shift'}
          multiSelectionKeyCode={locked ? null : undefined}
          nodesConnectable={false}
          nodesFocusable={canDrag}
          onInit={(instance) => {
            reactFlowInstance.current = instance;
            lastFocusedNodeId.current = focusedId || null;
            fitCurrentView(instance);
          }}
          fitView
          fitViewOptions={focusedId ? { nodes: [{ id: focusedId }] } : undefined}
          minZoom={0.05}
          nodeTypes={NODE_TYPES}
          edgeTypes={{}}
          onNodeClick={handleElementClick}
          onEdgeClick={handleElementClick}
          onNodesChange={onNodesChange}
          onNodeDragStop={canDrag ? onNodeDragStop : undefined}
          deleteKeyCode={null}
        >
          <MiniMap position='top-right' />
          <GraphControls
            showSubDagControls={!!getSubDagElements}
            hasSubDags={nodes.some((node) => node.type === GROUP_NODE_TYPE)}
            locked={locked}
            onLockChange={setLocked}
            onExpandAll={expandAll}
            onCollapseAll={collapseAll}
          />
          <Background gap={22} size={1} />
          {visibleStates.length > 0 && (
            <Panel position='bottom-right' className='kfp-graph-legend'>
              <ul aria-label='Task states in this graph'>
                {visibleStates.map((state) => {
                  const status = getTaskStatus(state);
                  return (
                    <li key={state} data-tone={status.tone}>
                      <span className='kfp-graph-state-dot' aria-hidden='true' />
                      {status.label}
                    </li>
                  );
                })}
              </ul>
            </Panel>
          )}
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}
