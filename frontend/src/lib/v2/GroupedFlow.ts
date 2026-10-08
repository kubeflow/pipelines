// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { Edge, Node } from '@xyflow/react';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import {
  buildGraphLayout,
  getTaskKeyFromNodeKey,
  GraphExpansionLimitError,
  isNode,
  NodeTypeNames,
  PipelineFlowElement,
} from './StaticFlow';

export type LayerElements = (layers: string[], maxNodes?: number) => PipelineFlowElement[];
export const AUTOMATIC_EXPANSION_NODE_LIMIT = 500;
export const GROUP_NODE_TYPE = 'subDagGroup';
export const GROUP_HEADER_HEIGHT = 48;
const GROUP_PADDING = 24;

export interface GroupedFlow {
  nodes: Node<FlowElementDataBase>[];
  edges: Edge[];
  /** Keep local IDs and scope for the existing task/artifact detail APIs. */
  sources: Map<string, { element: PipelineFlowElement; layers: string[] }>;
  width: number;
  height: number;
}

export function scopedNodeId(layers: string[], nodeId: string): string {
  // Preserve existing root IDs for links and integrations; nested IDs include
  // the full instance path, not just a reused component or task name.
  return layers.length === 1 ? nodeId : JSON.stringify([...layers, nodeId]);
}

/**
 * Lay out scopes bottom-up, treating each expanded scope as a sized node in its
 * parent. Cross-scope dependencies terminate at the group boundary, not at an
 * arbitrary internal task. Runtime iteration scopes use the same layer resolver
 * as focused navigation, so repeated component instances remain distinct.
 */
export function buildGroupedFlow(
  elements: PipelineFlowElement[],
  layers: string[],
  getLayerElements: LayerElements,
  collapsed: ReadonlySet<string>,
  expanded: ReadonlySet<string> = new Set(),
  collapseAll = false,
): GroupedFlow {
  let remaining = AUTOMATIC_EXPANSION_NODE_LIMIT;
  return buildLayer(elements, layers, new Set());

  function buildLayer(
    elements: PipelineFlowElement[],
    layers: string[],
    ancestorComponents: ReadonlySet<string>,
  ): GroupedFlow {
    const layerNodes = elements.filter(isNode);
    remaining = Math.max(0, remaining - layerNodes.length);
    const sources: GroupedFlow['sources'] = new Map();
    const children = new Map<string, GroupedFlow>();
    const idFor = (id: string) => scopedNodeId(layers, id);
    const nodes = layerNodes.map((element) => {
      const id = idFor(element.id);
      sources.set(id, { element, layers });
      const group = element.type === NodeTypeNames.SUB_DAG;
      let isCollapsed = collapsed.has(id) || (collapseAll && !expanded.has(id));
      // Card widths include status icons (256px) and artifact cards (240px).
      let width = element.type === NodeTypeNames.ARTIFACT ? 240 : element.data.state ? 256 : 224;
      let height = 48;
      let expansionError: string | undefined;
      let expansionDeferred: string | undefined;
      let empty = false;
      if (group) {
        width = element.data.state ? 288 : 256;
        height = GROUP_HEADER_HEIGHT;
        if (!isCollapsed) {
          try {
            const componentRef = element.data.componentRefName as string | undefined;
            if (componentRef && ancestorComponents.has(componentRef)) {
              throw new Error(`Circular sub-DAG component reference: ${componentRef}.`);
            }
            if (layers.length >= 64) throw new Error('Sub-DAG nesting exceeds 64 layers.');
            const explicitlyExpanded = expanded.has(id);
            if (!explicitlyExpanded && remaining === 0) throw new GraphExpansionLimitError();
            const childLayers = [...layers, getTaskKeyFromNodeKey(element.id)];
            // Resolvers check the budget before constructing nodes or running Dagre.
            // Explicit expansion admits this layer; descendants still share the budget.
            const childElements = getLayerElements(
              childLayers,
              explicitlyExpanded ? Infinity : remaining,
            );
            const childCount = childElements.filter(isNode).length;
            if (!explicitlyExpanded && childCount > remaining)
              throw new GraphExpansionLimitError(childCount);
            const child = buildLayer(
              childElements,
              childLayers,
              componentRef ? new Set([...ancestorComponents, componentRef]) : ancestorComponents,
            );
            children.set(id, child);
            empty = child.nodes.length === 0;
            width = Math.max(width, child.width + 2 * GROUP_PADDING);
            height += Math.max(child.height, 48) + 2 * GROUP_PADDING;
          } catch (error) {
            if (error instanceof GraphExpansionLimitError) {
              isCollapsed = true;
              expansionDeferred =
                error.nodeCount === undefined
                  ? 'Automatic limit · expand to load'
                  : `${error.nodeCount.toLocaleString('en-US')} nodes · expand to load`;
              width = Math.max(width, 320);
            } else {
              expansionError = error instanceof Error ? error.message : String(error);
              height += 72;
            }
          }
        }
      }
      return {
        ...element,
        id,
        type: group ? GROUP_NODE_TYPE : element.type,
        data: { ...element.data, collapsed: isCollapsed, expansionError, expansionDeferred, empty },
        position: { ...element.position },
        width,
        height,
        style: { ...element.style, width, height },
        zIndex: group ? 0 : 1,
        draggable: !group,
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
                extent: 'parent' as const,
                position: {
                  x: inner.position.x + GROUP_PADDING,
                  y: inner.position.y + GROUP_HEADER_HEIGHT + GROUP_PADDING,
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
