// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { CoordinateExtent, Edge, Node } from '@xyflow/react';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import { buildGraphLayout, getTaskKeyFromNodeKey, isNode, NodeTypeNames } from './StaticFlow';

import type { LayerElementsResolver, LayerFlowElement, PipelineFlowElement } from './FlowTypes';
import { getFlowNodeSize, NODE_CARD_HEIGHT, SUB_DAG_STATUS_CARD_WIDTH } from './FlowNodeSize';
export const DEFAULT_EXPANDED_SUB_DAG_LIMIT = 10;
export const GROUP_NODE_TYPE = 'subDagGroup';
export const GROUP_HEADER_HEIGHT = NODE_CARD_HEIGHT;
const GROUP_PADDING = 24;

export interface GroupedFlow {
  nodes: Node<FlowElementDataBase>[];
  edges: Edge[];
  /** Keep local IDs and their layer for the existing task/artifact detail APIs. */
  sources: Map<string, LayerFlowElement>;
  width: number;
  height: number;
}

export function layerNodeId(layers: string[], nodeId: string): string {
  // Preserve existing root IDs for links and integrations; nested IDs include
  // the full instance path, not just a reused component or task name.
  return layers.length === 1 ? nodeId : JSON.stringify([...layers, nodeId]);
}

/**
 * Lay out layers bottom-up, treating each expanded layer as a sized node in its
 * parent. Cross-layer dependencies terminate at the group boundary, not at an
 * arbitrary internal task. Runtime iteration layers use the same resolver
 * as task details, so repeated component instances remain distinct.
 */
export function buildGroupedFlow(
  elements: PipelineFlowElement[],
  layers: string[],
  getLayerElements: LayerElementsResolver,
  collapsed: ReadonlySet<string>,
  options: {
    expanded?: ReadonlySet<string>;
    expansionMode?: 'default' | 'all' | 'none';
    nodeScale?: number;
  } = {},
): GroupedFlow {
  const { expanded = new Set<string>(), expansionMode = 'default', nodeScale = 1 } = options;
  const headerHeight = GROUP_HEADER_HEIGHT;
  type Layer = {
    elements: PipelineFlowElement[];
    layers: string[];
    ancestorComponents: ReadonlySet<string>;
  };
  const root: Layer = { elements, layers, ancestorComponents: new Set() };
  const pending = [root];
  const childLayers = new Map<string, Layer>();
  const errors = new Map<string, string>();
  const deferred = new Set<string>();
  let defaultExpansions = 0;

  // Resolve breadth-first before laying out bottom-up. This gives shallower
  // groups priority, then upstream groups in each layer's top-to-bottom layout.
  for (let index = 0; index < pending.length; index++) {
    const layer = pending[index];
    const groups = layer.elements
      .filter(isNode)
      .filter((node) => node.type === NodeTypeNames.SUB_DAG)
      .sort((a, b) => a.position.y - b.position.y || a.position.x - b.position.x);
    for (const element of groups) {
      const id = layerNodeId(layer.layers, element.id);
      const defaultExpanded = defaultExpansions++ < DEFAULT_EXPANDED_SUB_DAG_LIMIT;
      if (collapsed.has(id)) continue;
      if (!expanded.has(id) && expansionMode !== 'all') {
        if (expansionMode === 'none') continue;
        if (!defaultExpanded) {
          deferred.add(id);
          continue;
        }
      }
      try {
        const componentRef = element.data.componentRefName as string | undefined;
        if (componentRef && layer.ancestorComponents.has(componentRef)) {
          throw new Error(
            `Circular sub-DAG component reference: ${componentRef}. Remove the recursive component reference from the pipeline spec.`,
          );
        }
        if (layer.layers.length >= 64) {
          throw new Error(
            'Sub-DAG nesting exceeds 64 layers. Reduce the nesting depth in the pipeline spec.',
          );
        }
        const layers = [...layer.layers, getTaskKeyFromNodeKey(element.id)];
        const child: Layer = {
          elements: getLayerElements(layers),
          layers,
          ancestorComponents: componentRef
            ? new Set([...layer.ancestorComponents, componentRef])
            : layer.ancestorComponents,
        };
        childLayers.set(id, child);
        pending.push(child);
      } catch (error) {
        errors.set(id, error instanceof Error ? error.message : String(error));
      }
    }
  }
  return buildLayer(root);

  function buildLayer({ elements, layers }: Layer): GroupedFlow {
    const sources: GroupedFlow['sources'] = new Map();
    const children = new Map<string, GroupedFlow>();
    const idFor = (id: string) => layerNodeId(layers, id);
    const nodes = elements.filter(isNode).map((element) => {
      const id = idFor(element.id);
      sources.set(id, { element, layers });
      const group = element.type === NodeTypeNames.SUB_DAG;
      const childLayer = childLayers.get(id);
      const expansionError = errors.get(id);
      const isCollapsed = group && !childLayer && !expansionError;
      const expansionDeferred = deferred.has(id)
        ? `Only the first ${DEFAULT_EXPANDED_SUB_DAG_LIMIT} sub-DAGs are expanded by default. Expand this sub-DAG or use Expand all to show more.`
        : undefined;
      let { width, height } = getFlowNodeSize(element, nodeScale);
      let empty = false;
      if (childLayer) {
        const child = buildLayer(childLayer);
        children.set(id, child);
        empty = child.nodes.length === 0;
        width = Math.max(width, child.width + 2 * GROUP_PADDING);
        height += Math.max(child.height, NODE_CARD_HEIGHT) + 2 * GROUP_PADDING;
      } else if (expansionError) {
        height += 72;
      } else if (expansionDeferred) {
        width = Math.max(width, SUB_DAG_STATUS_CARD_WIDTH);
      }
      return {
        ...element,
        id,
        type: group ? GROUP_NODE_TYPE : element.type,
        data: {
          ...element.data,
          collapsed: isCollapsed,
          expansionError,
          expansionDeferred,
          empty,
          headerHeight,
        },
        position: { ...element.position },
        width,
        height,
        style: { ...element.style, width, height },
        zIndex: group ? 0 : 1,
      };
    });
    const edges = elements
      .filter((el): el is Edge => !isNode(el))
      .map((edge) => {
        // Legacy artifact edge IDs omit the producer; use both endpoints instead.
        const id = JSON.stringify([...layers, edge.source, edge.target]);
        sources.set(id, { element: edge, layers });
        return { ...edge, id, source: idFor(edge.source), target: idFor(edge.target), zIndex: 1 };
      });
    const uniqueEdges = [...new Map(edges.map((edge) => [edge.id, edge])).values()];
    buildGraphLayout([...nodes, ...uniqueEdges]);
    const result: GroupedFlow = { nodes: [], edges: uniqueEdges, sources, width: 0, height: 0 };
    for (const node of nodes) {
      result.width = Math.max(result.width, node.position.x + node.width);
      result.height = Math.max(result.height, node.position.y + node.height);
      result.nodes.push(node);
      const child = children.get(node.id);
      if (!child) continue;
      // React Flow requires parents before descendants, with relative positions.
      result.nodes.push(
        ...child.nodes.map((inner) =>
          inner.parentId
            ? inner
            : {
                ...inner,
                parentId: node.id,
                // Parent-relative bounds exclude the header and preserve body padding.
                // React Flow enforces these throughout dragging, even if the parent moves.
                extent: [
                  [GROUP_PADDING, headerHeight + GROUP_PADDING],
                  [node.width - GROUP_PADDING, node.height - GROUP_PADDING],
                ] as CoordinateExtent,
                position: {
                  x: inner.position.x + GROUP_PADDING,
                  y: inner.position.y + headerHeight + GROUP_PADDING,
                },
              },
        ),
      );
      result.edges.push(...child.edges);
      child.sources.forEach((source, id) => sources.set(id, source));
    }
    return result;
  }
}
