// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { conditionSpec, loopSpec, loopTasks, nestedArtifactSpec } from 'src/data/test/groupedFlow';
import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import { buildGroupedFlow, GROUP_NODE_TYPE, scopedNodeId } from './GroupedFlow';
import { convertSubDagToFlowElements, NodeTypeNames } from './StaticFlow';
import {
  createRuntimeLayerResolver,
  convertSubDagToRuntimeFlowElements,
  getNodeRuntimeInfo,
} from './DynamicFlow';
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
      expect(node.position.x).toBeGreaterThanOrEqual(24);
      expect(node.position.y).toBeGreaterThanOrEqual(64);
      expect(node.position.x + node.width!).toBeLessThanOrEqual(parent.width! - 24);
      expect(node.position.y + node.height!).toBeLessThanOrEqual(parent.height! - 24);
    }
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
      (node) => node.id === scopedNodeId(['root', 'workflow', 'fit'], 'artifact.train.model'),
    )!;
    expect(model.parentId).toBe(scopedNodeId(['root', 'workflow'], 'task.fit'));
    expect(graph.sources.get(model.id)?.layers).toEqual(['root', 'workflow', 'fit']);
    expect(graph.sources.get(model.id)?.element.id).toBe('artifact.train.model');
  });

  it('collapses only the requested instance, removes its internal edges, and reflows siblings', () => {
    const expanded = staticGraph(conditionSpec);
    const collapsed = staticGraph(conditionSpec, new Set(['task.accurate']));
    expect(collapsed.nodes.find((node) => node.id === 'task.accurate')?.height).toBe(64);
    expect(collapsed.nodes.filter((node) => node.data.label === 'Train model')).toHaveLength(1);
    expect(collapsed.nodes.find((node) => node.id === 'task.accurate')?.height).toBeLessThan(
      expanded.nodes.find((node) => node.id === 'task.accurate')!.height!,
    );
    const ids = new Set(collapsed.nodes.map((node) => node.id));
    expect(collapsed.edges.every((edge) => ids.has(edge.source) && ids.has(edge.target))).toBe(
      true,
    );
    expect(collapsed.nodes.filter((node) => node.data.groupKind === 'Condition')).toHaveLength(2);
  });

  it('gives repeated task/artifact names unique IDs and leaves inputs unchanged', () => {
    const resolve = (layers: string[]) => convertSubDagToFlowElements(conditionSpec, layers);
    const input = resolve(['root']);
    const original = structuredClone(input);
    const graph = buildGroupedFlow(input, ['root'], resolve, new Set());
    expect(input).toEqual(original);
    expect(new Set(graph.nodes.map((node) => node.id)).size).toBe(graph.nodes.length);
    expect(new Set(graph.edges.map((edge) => edge.id)).size).toBe(graph.edges.length);
    expect(staticGraph(conditionSpec)).toEqual(staticGraph(conditionSpec));
  });

  it('retains descendant collapse state when a parent is reopened', () => {
    const child = scopedNodeId(['root', 'workflow'], 'task.fit');
    const collapsed = new Set(['task.workflow', child]);
    expect(staticGraph(nestedArtifactSpec, collapsed).nodes).toHaveLength(3);
    expect(staticGraph(nestedArtifactSpec, collapsed).height).toBeLessThan(staticGraph().height);
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

  it('rejects recursive components before expanding their descendants', () => {
    const root = convertSubDagToFlowElements(loopSpec, ['root']);
    const resolve = vi.fn(() => root);
    const graph = buildGroupedFlow(root, ['root'], resolve, new Set());
    expect(resolve).toHaveBeenCalledTimes(1);
    expect(graph.nodes.some((node) => node.data.expansionError?.includes('Circular'))).toBe(true);
  });

  it('reports excessive nesting instead of overflowing the stack', () => {
    const root = convertSubDagToFlowElements(loopSpec, ['root']);
    const graph = buildGroupedFlow(root, Array(64).fill('scope'), () => [], new Set());
    expect(graph.nodes.some((node) => node.data.expansionError?.includes('64 layers'))).toBe(true);
  });

  it('expands a real compiler fixture with nested loops without duplicate IDs', () => {
    const graph = staticGraph(convertYamlToV2PipelineSpec(realNestedLoops));
    expect(graph.nodes.filter((node) => node.type === GROUP_NODE_TYPE).length).toBeGreaterThan(2);
    expect(new Set(graph.nodes.map((node) => node.id)).size).toBe(graph.nodes.length);
    expect(
      graph.nodes.every(
        (node) => Number.isFinite(node.position.x) && Number.isFinite(node.position.y),
      ),
    ).toBe(true);
  });

  it('renders runtime iterations with distinct states, task identity, and artifact details', () => {
    const resolve = createRuntimeLayerResolver(loopSpec, loopTasks);
    const graph = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    expect(graph.nodes.filter((node) => node.data.groupKind === 'Iteration')).toHaveLength(2);
    const trainNodes = graph.nodes.filter(
      (node) => node.type === NodeTypeNames.EXECUTION && node.data.taskId?.startsWith('train-'),
    );
    expect(trainNodes.map((node) => node.data.state)).toEqual([
      PipelineTaskTaskState.CACHED,
      PipelineTaskTaskState.SUCCEEDED,
    ]);
    for (const node of trainNodes) {
      const source = graph.sources.get(node.id)!;
      expect(getNodeRuntimeInfo(source.element, loopTasks, source.layers).task?.task_id).toBe(
        node.data.taskId,
      );
    }
    const artifact = graph.sources.get(
      scopedNodeId(['root', 'sweep', 'sweep.1'], 'artifact.train.model'),
    )!;
    expect(
      getNodeRuntimeInfo(artifact.element, loopTasks, artifact.layers).artifactGroup?.artifacts?.[0]
        .artifact_id,
    ).toBe('2');
  });

  it('uses the declarative loop body before iteration metadata arrives', () => {
    const resolve = (layers: string[]) => convertSubDagToRuntimeFlowElements(loopSpec, layers, []);
    const graph = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    expect(graph.nodes.some((node) => node.data.label === 'Train model')).toBe(true);
    expect(graph.nodes.some((node) => node.data.groupKind === 'Iteration')).toBe(false);
  });
});
