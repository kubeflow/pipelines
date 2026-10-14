// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import {
  conditionSpec,
  deeplyNestedSpec,
  exitHandlerSpec,
  loopSpec,
  manySubDagsSpec,
  nestedArtifactSpec,
} from 'src/data/test/groupedFlow';
import { PipelineTaskSpec } from 'src/generated/pipeline_spec';
import { buildGroupedFlow, GROUP_HEADER_HEIGHT, GROUP_NODE_TYPE, layerNodeId } from './GroupedFlow';
import { convertSubDagToFlowElements, NodeTypeNames } from './StaticFlow';
import realNestedLoops from 'src/data/test/pipeline_with_loops_and_conditions.yaml?raw';
import { convertYamlToV2PipelineSpec } from './WorkflowUtils';

function staticGraph(spec = nestedArtifactSpec, collapsed = new Set<string>()) {
  const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
  return buildGroupedFlow(resolve(['root']), ['root'], resolve, collapsed);
}

describe('buildGroupedFlow', () => {
  it('expands nested groups by default, with parents before enclosed children', () => {
    const graph = staticGraph();
    expect(graph.nodes.filter((node) => node.type === GROUP_NODE_TYPE)).toHaveLength(2);
    expect(graph.nodes.some((node) => node.data.label === 'Train model')).toBe(true);
    for (const [index, node] of graph.nodes.entries()) {
      if (!node.parentId) continue;
      const parentIndex = graph.nodes.findIndex((parent) => parent.id === node.parentId);
      const parent = graph.nodes[parentIndex];
      expect(parentIndex).toBeLessThan(index);
      expect(node.extent).toEqual([
        [24, parent.data.headerHeight + 24],
        [parent.width! - 24, parent.height! - 24],
      ]);
      expect(node.position.x).toBeGreaterThanOrEqual(24);
      expect(node.position.y).toBeGreaterThanOrEqual(GROUP_HEADER_HEIGHT + 24);
    }
  });

  it('matches rem-sized cards and group headers with the application root font', () => {
    const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
    const graph = buildGroupedFlow(
      resolve(['root']),
      ['root'],
      resolve,
      new Set(['task.workflow']),
      { nodeScale: 13 / 16 },
    );
    const group = graph.nodes.find((node) => node.id === 'task.workflow')!;
    const leaf = graph.nodes.find((node) => node.id === 'task.deploy')!;
    expect(group.height).toBe(39);
    expect(group.data.headerHeight).toBe(leaf.height);
    expect(leaf.width).toBe(182);
  });

  it('keeps dependency edges on group boundaries and artifacts in their owning scope', () => {
    const graph = staticGraph();
    expect(graph.edges).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ source: 'task.workflow', target: 'artifact.workflow.model' }),
        expect.objectContaining({ source: 'artifact.workflow.model', target: 'task.deploy' }),
      ]),
    );
    const model = graph.nodes.find(
      (node) => node.id === layerNodeId(['root', 'workflow', 'fit'], 'artifact.train.model'),
    )!;
    expect(model.parentId).toBe(layerNodeId(['root', 'workflow'], 'task.fit'));
    expect(graph.sources.get(model.id)?.layers).toEqual(['root', 'workflow', 'fit']);
    expect(graph.sources.get(model.id)?.element.id).toBe('artifact.train.model');
  });

  it('collapses only the requested instance and reflows siblings', () => {
    const expanded = staticGraph(conditionSpec);
    const collapsed = staticGraph(conditionSpec, new Set(['task.accurate']));
    expect(collapsed.nodes.find((node) => node.id === 'task.accurate')?.height).toBe(48);
    expect(collapsed.nodes.filter((node) => node.data.label === 'Train model')).toHaveLength(1);
    expect(collapsed.nodes.find((node) => node.id === 'task.accurate')?.height).toBeLessThan(
      expanded.nodes.find((node) => node.id === 'task.accurate')!.height!,
    );
    expect(collapsed.nodes.filter((node) => node.data.groupKind === 'Condition')).toHaveLength(2);
  });

  it('retains descendant collapse state when a parent is reopened', () => {
    const child = layerNodeId(['root', 'workflow'], 'task.fit');
    const collapsed = new Set(['task.workflow', child]);
    expect(staticGraph(nestedArtifactSpec, collapsed).nodes).toHaveLength(3);
    collapsed.delete('task.workflow');
    const graph = staticGraph(nestedArtifactSpec, collapsed);
    expect(graph.nodes.find((node) => node.id === child)?.data.collapsed).toBe(true);
    expect(graph.nodes.some((node) => node.data.label === 'Train model')).toBe(false);
  });

  it('shows an empty group and contains errors in invalid nested scopes', () => {
    const root = convertSubDagToFlowElements(loopSpec, ['root']);
    const empty = buildGroupedFlow(root, ['root'], () => [], new Set());
    expect(empty.nodes.find((node) => node.id === 'task.sweep')?.data.empty).toBe(true);
    const invalid = buildGroupedFlow(
      root,
      ['root'],
      () => {
        throw new Error('Missing component');
      },
      new Set(),
    );
    expect(invalid.nodes.find((node) => node.id === 'task.sweep')?.data.expansionError).toBe(
      'Missing component',
    );
    expect(invalid.nodes.some((node) => node.id === 'task.summarize')).toBe(true);
  });

  it.each([9, 10, 11, 25])('expands at most 10 sub-DAGs in upstream order (%s groups)', (count) => {
    const spec = manySubDagsSpec(count);
    const resolve = vi.fn((layers: string[]) => convertSubDagToFlowElements(spec, layers));
    const graph = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    const groups = graph.nodes.filter((node) => node.type === GROUP_NODE_TYPE);
    expect(groups.filter((node) => !node.data.collapsed)).toHaveLength(Math.min(count, 10));
    for (let index = 0; index < count; index++) {
      expect(groups.find((node) => node.id === `task.group-${index}`)?.data.collapsed).toBe(
        index >= 10,
      );
    }
    expect(resolve).toHaveBeenCalledTimes(1 + Math.min(count, 10));
  });

  it('defaults to 10 of 25 nested sub-DAGs, each enclosing two executable components', () => {
    const spec = deeplyNestedSpec();
    const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
    const graph = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    expect(graph.nodes.filter((node) => node.type === GROUP_NODE_TYPE)).toHaveLength(11);
    expect(graph.nodes.filter((node) => node.data.label.startsWith('Component '))).toHaveLength(20);
    expect(
      graph.nodes.find(
        (node) =>
          node.id ===
          layerNodeId(
            Array.from({ length: 11 }, (_, i) => (i ? `level-${i}` : 'root')),
            'task.level-11',
          ),
      )?.data.collapsed,
    ).toBe(true);
  });

  it('keeps exit-handler ordering and its nested notification condition in the compact fixture', () => {
    const graph = staticGraph(exitHandlerSpec);
    expect(graph.nodes.filter((node) => node.type === GROUP_NODE_TYPE)).toHaveLength(3);
    expect(graph.nodes.map((node) => node.data.label)).toEqual(
      expect.arrayContaining(['my-pipeline', 'conditional-notification', 'condition-1']),
    );
  });

  it('rejects recursive components before expanding their descendants', () => {
    const spec = structuredClone(loopSpec);
    spec.components.body = {
      dag: {
        tasks: {
          recurse: PipelineTaskSpec.fromPartial({
            taskInfo: { name: 'Recursive child' },
            componentRef: { name: 'body' },
          }),
        },
      },
    };
    const resolve = vi.fn((layers: string[]) => convertSubDagToFlowElements(spec, layers));
    const graph = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    expect(graph.nodes.find((node) => node.data.expansionError)?.data.expansionError).toContain(
      'Circular sub-DAG component reference',
    );
    expect(resolve).toHaveBeenCalledTimes(2);
  });

  it('reports excessive nesting instead of overflowing the stack', () => {
    const root = convertSubDagToFlowElements(loopSpec, ['root']);
    const graph = buildGroupedFlow(root, Array(64).fill('scope'), () => [], new Set());
    expect(
      graph.nodes.find((node) => node.id === layerNodeId(Array(64).fill('scope'), 'task.sweep'))
        ?.data.expansionError,
    ).toContain('Sub-DAG nesting exceeds 64 layers');
  });

  it('shows the real nested-loops fixture inline', () => {
    const spec = convertYamlToV2PipelineSpec(realNestedLoops);
    const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
    const graph = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    expect(graph.nodes.some((node) => node.type === GROUP_NODE_TYPE)).toBe(true);
    expect(graph.nodes.some((node) => node.type === NodeTypeNames.EXECUTION)).toBe(true);
  });

  it('reports missing component references through the real static resolver', () => {
    const spec = structuredClone(nestedArtifactSpec);
    spec.components.workflow.dag!.tasks = {
      broken: PipelineTaskSpec.fromPartial({
        taskInfo: { name: 'broken' },
        componentRef: { name: 'missing' },
      }),
    };
    const graph = staticGraph(spec);
    const group = graph.nodes.find((node) => node.id === 'task.workflow')!;
    expect(group.data.expansionError).toContain('missing');
    expect(group.data.empty).toBe(false);
  });
});
