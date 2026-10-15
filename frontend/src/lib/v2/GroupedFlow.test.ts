// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import {
  conditionSpec,
  deeplyNestedSpec,
  exitHandlerSpec,
  loopSpec,
  loopTasks,
  manySubDagsSpec,
  nestedArtifactSpec,
} from 'src/data/test/groupedFlow';
import { PipelineTaskSpec } from 'src/generated/pipeline_spec';
import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import { buildGroupedFlow, GROUP_HEADER_HEIGHT, GROUP_NODE_TYPE, layerNodeId } from './GroupedFlow';
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
      expect(node.extent).toEqual([
        [24, parent.data.headerHeight + 24],
        [parent.width! - 24, parent.height! - 24],
      ]);
      expect(node.position.x).toBeGreaterThanOrEqual(24);
      expect(node.position.y).toBeGreaterThanOrEqual(GROUP_HEADER_HEIGHT + 24);
      expect(node.position.x + node.width!).toBeLessThanOrEqual(parent.width! - 24);
      expect(node.position.y + node.height!).toBeLessThanOrEqual(parent.height! - 24);
    }
  });

  it('keeps fixed pixel cards and group headers aligned at different root font sizes', () => {
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
    expect(group.height).toBe(56);
    expect(group.data.headerHeight).toBe(leaf.height);
    expect(leaf.width).toBe(200);
  });

  it.each([1, 13 / 16])(
    'keeps collapsed nested groups below their parent header at scale %s',
    (nodeScale) => {
      const resolve = (layers: string[]) => convertSubDagToFlowElements(nestedArtifactSpec, layers);
      const graph = buildGroupedFlow(
        resolve(['root']),
        ['root'],
        resolve,
        new Set([layerNodeId(['root', 'workflow'], 'task.fit')]),
        { nodeScale },
      );
      const group = graph.nodes.find(
        (node) => node.id === layerNodeId(['root', 'workflow'], 'task.fit'),
      )!;
      const parent = graph.nodes.find((node) => node.id === group.parentId)!;
      expect(group.data.collapsed).toBe(true);
      expect(group.extent).toEqual([
        [24, GROUP_HEADER_HEIGHT + 24],
        [parent.width! - 24, parent.height! - 24],
      ]);
      expect(parent.extent).toBeUndefined();
    },
  );

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

  it('collapses only the requested instance, removes its internal edges, and reflows siblings', () => {
    const expanded = staticGraph(conditionSpec);
    const collapsed = staticGraph(conditionSpec, new Set(['task.accurate']));
    expect(collapsed.nodes.find((node) => node.id === 'task.accurate')?.height).toBe(56);
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
    const child = layerNodeId(['root', 'workflow'], 'task.fit');
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

  it('reports missing component references through the real static and runtime resolvers', () => {
    const spec = structuredClone(nestedArtifactSpec);
    spec.components.workflow.dag!.tasks = {
      broken: PipelineTaskSpec.fromPartial({
        taskInfo: { name: 'broken' },
        componentRef: { name: 'missing' },
      }),
    };
    const staticResult = staticGraph(spec);
    const resolve = createRuntimeLayerResolver(spec, []);
    const runtimeResult = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    for (const graph of [staticResult, runtimeResult]) {
      const group = graph.nodes.find((node) => node.id === 'task.workflow')!;
      expect(group.data.expansionError).toContain('missing');
      expect(group.data.empty).toBe(false);
    }
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

  it('prioritizes shallow groups before nested groups and preserves upstream parent order', () => {
    const spec = manySubDagsSpec(6);
    spec.components.inner = spec.components.body;
    spec.components.body = {
      dag: {
        tasks: {
          nested: PipelineTaskSpec.fromPartial({
            taskInfo: { name: 'Nested' },
            componentRef: { name: 'inner' },
          }),
        },
      },
    };
    const graph = staticGraph(spec);
    const expanded = graph.nodes.filter(
      (node) => node.type === GROUP_NODE_TYPE && !node.data.collapsed,
    );
    expect(expanded).toHaveLength(10);
    expect(expanded.filter((node) => !node.parentId)).toHaveLength(6);
    for (let index = 0; index < 6; index++) {
      const nested = graph.nodes.find(
        (node) => node.id === layerNodeId(['root', `group-${index}`], 'task.nested'),
      )!;
      expect(nested.data.collapsed).toBe(index >= 4);
    }
  });

  it('stops a deeply nested chain at the eleventh group without resolving its children', () => {
    const spec = manySubDagsSpec(1);
    for (let depth = 0; depth < 25; depth++) {
      spec.components[`layer-${depth}`] = {
        dag: {
          tasks: {
            child: PipelineTaskSpec.fromPartial({
              taskInfo: { name: `Depth ${depth + 1}` },
              componentRef: { name: depth < 24 ? `layer-${depth + 1}` : 'step' },
            }),
          },
        },
      };
    }
    spec.root!.dag!.tasks['group-0'].componentRef = { name: 'layer-0' };
    const resolve = vi.fn((layers: string[]) => convertSubDagToFlowElements(spec, layers));
    const graph = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    const groups = graph.nodes.filter((node) => node.type === GROUP_NODE_TYPE);
    expect(groups.filter((node) => !node.data.collapsed)).toHaveLength(10);
    expect(groups).toHaveLength(11);
    expect(groups[10].data.collapsed).toBe(true);
    expect(resolve).toHaveBeenCalledTimes(11);
    const all = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set(), {
      expansionMode: 'all',
    });
    expect(
      all.nodes.filter((node) => node.type === GROUP_NODE_TYPE && !node.data.collapsed),
    ).toHaveLength(25);
    expect(all.nodes.some((node) => node.type === NodeTypeNames.EXECUTION)).toBe(true);
  });

  it('defaults to 10 of 25 nested sub-DAGs, each enclosing two executable components', () => {
    const spec = deeplyNestedSpec();
    expect(
      Object.values(spec.components).filter((component) => component.executorLabel),
    ).toHaveLength(50);
    const graph = staticGraph(spec);
    expect(graph.nodes.filter((node) => node.type === NodeTypeNames.EXECUTION)).toHaveLength(20);
    const groups = graph.nodes.filter((node) => node.type === GROUP_NODE_TYPE);
    expect(groups).toHaveLength(11);
    expect(groups.filter((node) => !node.data.collapsed)).toHaveLength(10);
    expect(groups.find((node) => node.data.collapsed)?.data.label).toBe('Level 11');
    expect(graph.nodes.some((node) => node.data.label === 'Component 021')).toBe(false);

    const resolve = (layers: string[]) => convertSubDagToFlowElements(spec, layers);
    const full = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set(), {
      expansionMode: 'all',
    });
    expect(full.nodes.filter((node) => node.type === NodeTypeNames.EXECUTION)).toHaveLength(50);
    expect(full.nodes.filter((node) => node.type === GROUP_NODE_TYPE)).toHaveLength(25);
    expect(full.nodes).toHaveLength(75);
    expect(full.edges).toHaveLength(48);
    expect(full.nodes.every((node) => !node.data.collapsed && !node.data.expansionError)).toBe(
      true,
    );
    const nodes = new Map(full.nodes.map((node) => [node.id, node]));
    expect(nodes.size).toBe(75);
    expect(full.edges.every((edge) => nodes.has(edge.source) && nodes.has(edge.target))).toBe(true);
    for (const node of full.nodes) {
      expect(Number.isFinite(node.position.x) && Number.isFinite(node.position.y)).toBe(true);
      if (!node.parentId) continue;
      const parent = nodes.get(node.parentId)!;
      expect(node.position.x).toBeGreaterThanOrEqual(24);
      expect(node.position.y).toBeGreaterThanOrEqual(GROUP_HEADER_HEIGHT + 24);
      expect(node.position.x + node.width!).toBeLessThanOrEqual(parent.width! - 24);
      expect(node.position.y + node.height!).toBeLessThanOrEqual(parent.height! - 24);
    }
    const deepestLayers = [
      'root',
      ...Array.from({ length: 25 }, (_, index) => `level-${index + 1}`),
    ];
    const deepestId = layerNodeId(deepestLayers, 'task.component-50');
    expect(full.sources.get(deepestId)).toMatchObject({
      element: { id: 'task.component-50', data: { label: 'Component 050' } },
      layers: deepestLayers,
    });
    let ancestor = nodes.get(deepestId);
    let depth = 0;
    while (ancestor?.parentId) {
      depth++;
      ancestor = nodes.get(ancestor.parentId);
    }
    expect(depth).toBe(25);
  });

  it('counts sub-DAGs, not the component or artifact nodes inside them', () => {
    const spec = manySubDagsSpec(1);
    spec.components.body.dag!.tasks = Object.fromEntries(
      Array.from({ length: 501 }, (_, index) => [
        `step-${index}`,
        PipelineTaskSpec.fromPartial({
          taskInfo: { name: `Step ${index}` },
          componentRef: { name: 'step' },
        }),
      ]),
    );
    const graph = staticGraph(spec);
    expect(graph.nodes).toHaveLength(502);
    expect(graph.nodes.find((node) => node.id === 'task.group-0')?.data.collapsed).toBe(false);
  });

  it('counts runtime loops and iterations toward the default and lets bulk actions override it', () => {
    const tasks = loopTasks.map((task) =>
      task.task_id === 'sweep' ? { ...task, type_attributes: { iteration_count: '30' } } : task,
    );
    const resolve = createRuntimeLayerResolver(loopSpec, tasks);
    const root = resolve(['root']);
    const graph = buildGroupedFlow(root, ['root'], resolve, new Set());
    const expandedGroups = (graph: ReturnType<typeof buildGroupedFlow>) =>
      graph.nodes.filter((node) => node.type === GROUP_NODE_TYPE && !node.data.collapsed);
    expect(expandedGroups(graph)).toHaveLength(10);
    const deferred = graph.nodes.find((node) => node.data.expansionDeferred)!;
    const manuallyExpanded = buildGroupedFlow(root, ['root'], resolve, new Set(), {
      expanded: new Set([deferred.id]),
    });
    expect(manuallyExpanded.nodes.find((node) => node.id === deferred.id)?.data.collapsed).toBe(
      false,
    );
    expect(expandedGroups(manuallyExpanded)).toHaveLength(11);
    const all = buildGroupedFlow(root, ['root'], resolve, new Set(), { expansionMode: 'all' });
    expect(expandedGroups(all)).toHaveLength(31);
    const none = buildGroupedFlow(root, ['root'], resolve, new Set(), { expansionMode: 'none' });
    expect(expandedGroups(none)).toHaveLength(0);
    const reopened = buildGroupedFlow(root, ['root'], resolve, new Set(), {
      expansionMode: 'none',
      expanded: new Set(['task.sweep']),
    });
    expect(expandedGroups(reopened)).toHaveLength(1);
    expect(reopened.nodes.filter((node) => node.data.groupKind === 'Iteration')).toHaveLength(30);
  });

  it('keeps exit-handler ordering and its nested notification condition in the compact fixture', () => {
    const graph = staticGraph(exitHandlerSpec);
    expect(graph.nodes.filter((node) => node.type === GROUP_NODE_TYPE)).toHaveLength(3);
    expect(graph.edges).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          source: 'task.exit-handler-1',
          target: 'task.conditional-notification',
        }),
      ]),
    );
    expect(graph.nodes.find((node) => node.data.label === 'condition-1')?.data.groupKind).toBe(
      'Condition',
    );
  });

  it('rejects recursive components before expanding their descendants', () => {
    const root = convertSubDagToFlowElements(loopSpec, ['root']);
    const resolve = vi.fn(() => root);
    const graph = buildGroupedFlow(root, ['root'], resolve, new Set());
    expect(resolve).toHaveBeenCalledTimes(1);
    expect(graph.nodes.find((node) => node.data.expansionError)?.data.expansionError).toMatch(
      /Circular sub-DAG component reference: .+\. Remove the recursive component reference from the pipeline spec\./,
    );
  });

  it('reports excessive nesting instead of overflowing the stack', () => {
    const root = convertSubDagToFlowElements(loopSpec, ['root']);
    const graph = buildGroupedFlow(root, Array(64).fill('scope'), () => [], new Set());
    expect(graph.nodes.find((node) => node.data.expansionError)?.data.expansionError).toBe(
      'Sub-DAG nesting exceeds 64 layers. Reduce the nesting depth in the pipeline spec.',
    );
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
      layerNodeId(['root', 'sweep', 'sweep.1'], 'artifact.train.model'),
    )!;
    expect(
      getNodeRuntimeInfo(artifact.element, loopTasks, artifact.layers).artifactGroup?.artifacts?.[0]
        .artifact_id,
    ).toBe('2');
  });

  it('uses terminal run state when resolving multiple layers through a shared task index', () => {
    const resolve = createRuntimeLayerResolver(loopSpec, loopTasks, true, true);
    const graph = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    const loop = graph.nodes.find((node) => node.id === 'task.sweep')!;
    const runningTask = graph.nodes.find((node) => node.data.taskId === 'evaluate-1')!;
    expect(loop.data.state).toBe(PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED);
    expect(runningTask.data.state).toBe(PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED);
    expect(graph.nodes.find((node) => node.data.taskId === 'train-0')?.data.state).toBe(
      PipelineTaskTaskState.CACHED,
    );
  });

  it('uses the declarative loop body before iteration metadata arrives', () => {
    const resolve = (layers: string[]) => convertSubDagToRuntimeFlowElements(loopSpec, layers, []);
    const graph = buildGroupedFlow(resolve(['root']), ['root'], resolve, new Set());
    expect(graph.nodes.some((node) => node.data.label === 'Train model')).toBe(true);
    expect(graph.nodes.some((node) => node.data.groupKind === 'Iteration')).toBe(false);
  });
});
