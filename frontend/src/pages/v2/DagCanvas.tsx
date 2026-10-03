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
  Panel,
  Node,
  OnNodeDrag,
  OnNodesChange,
  ReactFlowInstance,
} from '@xyflow/react';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import SubDagLayer from 'src/components/graph/SubDagLayer';
import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import { getTaskStatus } from 'src/components/graph/ExecutionNode';
import 'src/components/graph/Graph.css';
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
  const reactFlowInstance = useRef<ReactFlowInstance<PipelineNode, Edge> | null>(null);
  const lastFocusedNodeId = useRef<string | null>(null);
  const sourceNodes = useMemo(() => elements.filter(isNode), [elements]);
  const measurementScope = JSON.stringify(layers);
  const [measurements, setMeasurements] = useState<{
    scope: string;
    nodes: Map<string, { type?: string; width: number; height: number }>;
  }>(() => ({ scope: measurementScope, nodes: new Map() }));
  // External widget synchronization: retain actual ResizeObserver measurements across
  // controlled node updates. ReactFlow also needs these to preserve its handle bounds.
  const onNodesChange = useCallback<OnNodesChange<PipelineNode>>(
    (changes) => {
      const nodeTypes = new Map(sourceNodes.map(({ id, type }) => [id, type]));
      const dimensions = changes.filter(
        (change) => change.type === 'dimensions' && change.dimensions && nodeTypes.has(change.id),
      );
      if (!dimensions.length) return;
      setMeasurements((previous) => {
        const next = previous.scope === measurementScope ? new Map(previous.nodes) : new Map();
        let changed = previous.scope !== measurementScope;
        for (const [id, measurement] of next) {
          if (!nodeTypes.has(id) || nodeTypes.get(id) !== measurement.type) {
            next.delete(id);
            changed = true;
          }
        }
        for (const change of dimensions) {
          if (change.type !== 'dimensions' || !change.dimensions) continue;
          const current = next.get(change.id);
          if (
            current?.width !== change.dimensions.width ||
            current?.height !== change.dimensions.height
          ) {
            next.set(change.id, { type: nodeTypes.get(change.id), ...change.dimensions });
            changed = true;
          }
        }
        return changed ? { scope: measurementScope, nodes: next } : previous;
      });
    },
    [measurementScope, sourceNodes],
  );
  const subDagExpand = useCallback(
    (nodeKey: string) => {
      const newLayers = [...layers, getTaskKeyFromNodeKey(nodeKey)];
      onLayersUpdate(newLayers);
    },
    [layers, onLayersUpdate],
  );

  const nodes = useMemo(
    () =>
      sourceNodes.map((node) => {
        const measured =
          measurements.scope === measurementScope ? measurements.nodes.get(node.id) : undefined;
        const selectedNode = {
          ...node,
          ...(measured?.type === node.type && measured
            ? { measured: { width: measured.width, height: measured.height } }
            : {}),
          selected: node.id === selectedNodeId,
        };
        return selectedNode.type === NodeTypeNames.SUB_DAG && selectedNode.data
          ? { ...selectedNode, data: { ...selectedNode.data, expand: subDagExpand } }
          : selectedNode;
      }),
    [measurementScope, measurements, sourceNodes, selectedNodeId, subDagExpand],
  );
  const edges = useMemo(() => {
    const states = new Map(elements.filter(isNode).map((node) => [node.id, node.data.state]));
    return elements
      .filter((element): element is Edge => !isNode(element))
      .map((edge) => {
        const state = states.get(edge.target);
        const tone =
          state === 'FAILED' || state === 'SKIPPED'
            ? 'failed'
            : state === 'SUCCEEDED' || state === 'CACHED'
              ? 'succeeded'
              : state === 'RUNNING'
                ? 'running'
                : 'neutral';
        return {
          ...edge,
          className: `${edge.className || ''} kfp-graph-edge-${tone}`,
          style: { ...edge.style, strokeWidth: 1.5 },
        };
      });
  }, [elements]);
  const visibleStates = [
    ...new Set(
      nodes
        .map((node) => node.data.state as PipelineTaskTaskState | undefined)
        .filter((state): state is PipelineTaskTaskState => state !== undefined),
    ),
  ];

  const onNodeDragStop = useCallback<OnNodeDrag<PipelineNode>>(
    (_event, draggedNode) => {
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

  const fitCurrentView = useCallback(
    (instance: ReactFlowInstance<PipelineNode, Edge>) => {
      const focusedNodes = focusNodeId
        ? nodes.filter((node) => node.id === focusNodeId)
        : undefined;
      void instance.fitView(focusedNodes?.length ? { nodes: focusedNodes } : undefined);
    },
    [focusNodeId, nodes],
  );

  useEffect(() => {
    if (!focusNodeId) {
      lastFocusedNodeId.current = null;
      return;
    }
    if (reactFlowInstance.current && lastFocusedNodeId.current !== focusNodeId) {
      lastFocusedNodeId.current = focusNodeId;
      fitCurrentView(reactFlowInstance.current);
    }
  }, [fitCurrentView, focusNodeId]);

  return (
    <div className='kfp-graph-workspace'>
      <SubDagLayer layers={layers} onLayersUpdate={onLayersUpdate} />
      <div data-testid='DagCanvas' className='kfp-graph-canvas'>
        <ReactFlowProvider>
          {/* Only dimension changes are synchronized. Native buttons own keyboard
              activation; selection comes from the page, and drag persistence uses
              onNodeDragStop. Removal and internal selection changes are ignored. */}
          <ReactFlow<PipelineNode, Edge>
            aria-label='Pipeline graph'
            ariaLabelConfig={{
              'node.a11yDescription.default':
                'Press Enter or Space to inspect a node. Use the expand button to open a nested pipeline.',
              'edge.a11yDescription.default': 'Connection between pipeline nodes.',
            }}
            nodes={nodes}
            edges={edges}
            snapToGrid={true}
            nodesDraggable={nodesDraggable}
            nodesFocusable={false}
            onInit={(instance) => {
              reactFlowInstance.current = instance;
              lastFocusedNodeId.current = focusNodeId || null;
              fitCurrentView(instance);
            }}
            nodeTypes={NODE_TYPES}
            edgeTypes={{}}
            onNodesChange={onNodesChange}
            onNodeClick={handleNodeClick}
            onEdgeClick={handleEdgeClick}
            onNodeDragStop={nodesDraggable ? onNodeDragStop : undefined}
          >
            <MiniMap position='top-right' />
            <Controls position='bottom-right' />
            <Background gap={22} size={1} />
            {visibleStates.length > 0 && (
              <Panel position='bottom-left' className='kfp-graph-legend'>
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
    </div>
  );
}
