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
  useNodesInitialized,
  useReactFlow,
  useStore,
  getViewportForBounds,
} from '@xyflow/react';
import 'src/components/graph/Graph.css';
import { PipelineTaskTaskState } from 'src/apisv2/run';
import { getTaskStatus } from 'src/lib/StatusUtils';
import { TaskStateIndicator } from 'src/components/graph/ExecutionNode';
import GraphControls from 'src/components/graph/GraphControls';
import { FlowElementDataBase, GRAPH_FIT_OPTIONS } from 'src/components/graph/Constants';
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
  const initialFitComplete = useRef(false);
  const manuallyMoved = useRef(false);
  const canvasRef = useRef<HTMLDivElement>(null);
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
  // Runtime task pages can replace the initial static graph with loop iterations.
  // Track geometry, not task status, so background polling does not reset the view.
  const layoutKey = useMemo(
    () =>
      JSON.stringify(
        grouped.nodes.map(({ id, parentId, position, width, height }) => [
          id,
          parentId,
          position.x,
          position.y,
          width,
          height,
        ]),
      ),
    [grouped.nodes],
  );
  const toggleGroup = useCallback((id: string, isCollapsed: boolean) => {
    manuallyMoved.current = true;
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
    manuallyMoved.current = true;
    setPositions({});
    requestAnimationFrame(() => {
      void reactFlowInstance.current?.fitView(GRAPH_FIT_OPTIONS);
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
  const edges = useMemo(
    () =>
      grouped.edges.map((edge) => ({
        ...edge,
        className: `${edge.className || ''} kfp-graph-edge`,
        style: { ...edge.style, strokeWidth: 1.5 },
      })),
    [grouped.edges],
  );
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
      manuallyMoved.current = true;
      event.preventDefault();
      event.stopPropagation();
      void instance.fitView(GRAPH_FIT_OPTIONS);
      return;
    }
    const direction = panDirections.get(event.key);
    if (!locked || !direction) return;
    event.preventDefault();
    event.stopPropagation();
    manuallyMoved.current = true;
    const viewport = instance.getViewport();
    void instance.setViewport({
      ...viewport,
      x: viewport.x + direction.x,
      y: viewport.y + direction.y,
    });
  };

  const fitCurrentView = useCallback(
    (instance: ReactFlowInstance<PipelineNode, Edge>) => {
      const canvas = canvasRef.current;
      if (!canvas || !nodes.length) return;
      const focusedNodes = focusedId ? nodes.filter((node) => node.id === focusedId) : [];
      // Nodes are already measured here. fitView queues work until a later controlled
      // node update, which can overwrite a manual zoom when an inspector selects a node.
      // Apply the initial viewport immediately instead; only explicit Fit uses fitView.
      void instance.setViewport(
        getViewportForBounds(
          instance.getNodesBounds(focusedNodes.length ? focusedNodes : nodes),
          canvas.clientWidth,
          canvas.clientHeight,
          0.05,
          GRAPH_FIT_OPTIONS.maxZoom,
          0.1,
        ),
      );
    },
    [focusedId, nodes],
  );

  // External synchronization: focus a node requested by URL navigation in React Flow.
  useEffect(() => {
    if (!focusedId) {
      lastFocusedNodeId.current = null;
      return;
    }
    if (
      initialFitComplete.current &&
      reactFlowInstance.current &&
      lastFocusedNodeId.current !== focusedId
    ) {
      lastFocusedNodeId.current = focusedId;
      const instance = reactFlowInstance.current;
      const focusedNode = instance.getNode(focusedId);
      if (focusedNode) {
        const bounds = instance.getNodesBounds([focusedNode]);
        // Selecting a task opens the inspector; keep the user's scale while
        // centering that task. Fit is reserved for initial load and explicit Fit.
        void instance.setCenter(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2, {
          zoom: instance.getZoom(),
        });
      }
    }
  }, [focusedId, nodes]);

  // External synchronization: React Flow keeps its translation when its DOM viewport
  // shrinks. Move the existing view to the new center without changing user zoom.
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    let previous = { width: canvas.clientWidth, height: canvas.clientHeight };
    const observer = new ResizeObserver(() => {
      const next = { width: canvas.clientWidth, height: canvas.clientHeight };
      const instance = reactFlowInstance.current;
      if (
        instance &&
        previous.width > 0 &&
        next.width > 0 &&
        next.height > 0 &&
        (next.width !== previous.width || next.height !== previous.height)
      ) {
        const viewport = instance.getViewport();
        let x = viewport.x + (next.width - previous.width) / 2;
        let y = viewport.y + (next.height - previous.height) / 2;
        const selected = instance.getNodes().find((node) => node.selected);
        if (selected) {
          const bounds = instance.getNodesBounds([selected]);
          const keepVisible = (offset: number, start: number, size: number, available: number) => {
            const extent = size * viewport.zoom;
            if (extent > available - 32) return (available - extent) / 2 - start * viewport.zoom;
            return (
              Math.min(available - 16 - extent, Math.max(16, start * viewport.zoom + offset)) -
              start * viewport.zoom
            );
          };
          x = keepVisible(x, bounds.x, bounds.width, next.width);
          y = keepVisible(y, bounds.y, bounds.height, next.height);
        }
        void instance.setViewport({ ...viewport, x, y });
      }
      previous = next;
    });
    observer.observe(canvas);
    return () => observer.disconnect();
  }, []);

  return (
    <div
      ref={canvasRef}
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
          proOptions={{ hideAttribution: true }}
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
          }}
          onMoveStart={(event) => {
            if (event) manuallyMoved.current = true;
          }}
          minZoom={0.05}
          nodeTypes={NODE_TYPES}
          edgeTypes={{}}
          onNodeClick={handleElementClick}
          onEdgeClick={handleElementClick}
          onNodesChange={onNodesChange}
          onNodeDragStop={canDrag ? onNodeDragStop : undefined}
          deleteKeyCode={null}
        >
          <InitialGraphFit
            layoutKey={layoutKey}
            onReady={(instance) => {
              reactFlowInstance.current = instance;
              lastFocusedNodeId.current = focusedId || null;
              initialFitComplete.current = true;
              if (!manuallyMoved.current) fitCurrentView(instance);
            }}
          />
          <MiniMap position='top-right' />
          <GraphControls
            showSubDagControls={!!getSubDagElements}
            hasSubDags={nodes.some((node) => node.type === GROUP_NODE_TYPE)}
            canExpand={nodes.some((node) => node.type === GROUP_NODE_TYPE && node.data.collapsed)}
            canCollapse={nodes.some(
              (node) => node.type === GROUP_NODE_TYPE && !node.data.collapsed,
            )}
            locked={locked}
            onLockChange={setLocked}
            onViewportInteraction={() => {
              manuallyMoved.current = true;
            }}
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
                      <TaskStateIndicator state={state} />
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

// External synchronization: wait for React Flow's measured nodes and viewport,
// rather than fitting during onInit while its container is still being laid out.
function InitialGraphFit({
  layoutKey,
  onReady,
}: {
  layoutKey: string;
  onReady: (instance: ReactFlowInstance<PipelineNode, Edge>) => void;
}) {
  const instance = useReactFlow<PipelineNode, Edge>();
  // Controlled nodes intentionally ignore dimension-change events. Check measured
  // internals directly: the default cached readiness flag only updates on setNodes.
  // Collapsed descendants are omitted from this canvas, not hidden nodes.
  const nodesReady = useNodesInitialized({ includeHiddenNodes: true });
  const viewportReady = useStore((store) => store.width > 0 && store.height > 0);
  const fittedLayout = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (
      !nodesReady ||
      !viewportReady ||
      !instance.viewportInitialized ||
      fittedLayout.current === layoutKey
    )
      return;
    const frame = requestAnimationFrame(() => {
      fittedLayout.current = layoutKey;
      onReady(instance);
    });
    return () => cancelAnimationFrame(frame);
  }, [nodesReady, viewportReady, instance, onReady, layoutKey]);
  return null;
}
