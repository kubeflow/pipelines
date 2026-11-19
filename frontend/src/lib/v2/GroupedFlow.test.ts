// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
import {
  deeplyNestedSpec,
  manySubDagsSpec,
  nestedArtifactSpec,
  resolveGraph,
} from 'src/data/test/groupedFlow';
import { buildGroupedFlow, GROUP_HEADER_HEIGHT, GROUP_NODE_TYPE, layerNodeId } from './GroupedFlow';

function graph(spec = nestedArtifactSpec, collapsed = new Set<string>(), options = {}) {
  const resolve = (layers: string[]) => resolveGraph(spec, layers);
  return buildGroupedFlow(resolve(['root']), ['root'], resolve, collapsed, options);
}

describe('native graph grouping', () => {
  it('lays parents before children and keeps cards within their parent bounds', () => {
    const result = graph();
    expect(result.nodes.filter((node) => node.type === GROUP_NODE_TYPE)).toHaveLength(2);
    expect(result.nodes.some((node) => node.data.label === 'Train model')).toBe(true);
    for (const [index, node] of result.nodes.entries()) {
      if (!node.parentId) continue;
      const parentIndex = result.nodes.findIndex((parent) => parent.id === node.parentId);
      const parent = result.nodes[parentIndex];
      expect(parentIndex).toBeLessThan(index);
      expect(node.extent).toEqual([
        [24, parent.data.headerHeight + 24],
        [parent.width! - 24, parent.height! - 24],
      ]);
      expect(node.position.y).toBeGreaterThanOrEqual(GROUP_HEADER_HEIGHT + 24);
      expect(node.position.x + node.width!).toBeLessThanOrEqual(parent.width! - 24);
      expect(node.position.y + node.height!).toBeLessThanOrEqual(parent.height! - 24);
    }
  });
  it('scales collapsed cards with the root font', () => {
    const result = graph(nestedArtifactSpec, new Set(['task.workflow']), { nodeScale: 13 / 16 });
    expect(result.nodes.find((node) => node.id === 'task.workflow')!.height).toBe(39);
    expect(result.nodes.find((node) => node.id === 'task.deploy')!.width).toBe(182);
  });
  it.each([1, 13 / 16])('keeps nested groups below headers at scale %s', (nodeScale) => {
    const child = layerNodeId(['root', 'workflow'], 'task.fit');
    const result = graph(nestedArtifactSpec, new Set([child]), { nodeScale });
    const node = result.nodes.find((node) => node.id === child)!;
    const parent = result.nodes.find((parent) => parent.id === node.parentId)!;
    expect(node.data.collapsed).toBe(true);
    expect(node.extent).toEqual([
      [24, GROUP_HEADER_HEIGHT * nodeScale + 24],
      [parent.width! - 24, parent.height! - 24],
    ]);
  });
  it('preserves artifact edges and records the producing scope', () => {
    const result = graph();
    expect(result.edges).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ source: 'task.workflow', target: 'artifact.workflow.model' }),
        expect.objectContaining({ source: 'artifact.workflow.model', target: 'task.deploy' }),
      ]),
    );
    const id = layerNodeId(['root', 'workflow', 'fit'], 'artifact.train.model');
    expect(result.sources.get(id)?.layers).toEqual(['root', 'workflow', 'fit']);
    expect(result.nodes.find((node) => node.id === id)?.parentId).toBe(
      layerNodeId(['root', 'workflow'], 'task.fit'),
    );
  });
  it('retains descendant collapse state when a parent is reopened', () => {
    const child = layerNodeId(['root', 'workflow'], 'task.fit');
    const collapsed = new Set(['task.workflow', child]);
    expect(graph(nestedArtifactSpec, collapsed).nodes).toHaveLength(3);
    collapsed.delete('task.workflow');
    const result = graph(nestedArtifactSpec, collapsed);
    expect(result.nodes.find((node) => node.id === child)?.data.collapsed).toBe(true);
    expect(result.nodes.some((node) => node.data.label === 'Train model')).toBe(false);
  });
  it('contains nested resolver errors without losing siblings', () => {
    const root = resolveGraph(nestedArtifactSpec, ['root']);
    const result = buildGroupedFlow(
      root,
      ['root'],
      () => {
        throw new Error('Missing template');
      },
      new Set(),
    );
    expect(result.nodes.find((node) => node.id === 'task.workflow')?.data.expansionError).toBe(
      'Missing template',
    );
    expect(result.nodes.some((node) => node.id === 'task.deploy')).toBe(true);
  });
  it('bounds default expansion for large and deep native DAGs', () => {
    for (const spec of [manySubDagsSpec(25, 5), deeplyNestedSpec()]) {
      const result = graph(spec);
      expect(
        result.nodes.filter((node) => node.type === GROUP_NODE_TYPE && !node.data.collapsed),
      ).toHaveLength(10);
      expect(new Set(result.nodes.map((node) => node.id)).size).toBe(result.nodes.length);
      const ids = new Set(result.nodes.map((node) => node.id));
      expect(result.edges.every((edge) => ids.has(edge.source) && ids.has(edge.target))).toBe(true);
    }
  });
  it('does not mutate source elements or shared templates', () => {
    const spec = structuredClone(nestedArtifactSpec);
    const before = structuredClone(spec);
    expect(graph(spec)).toEqual(graph(spec));
    expect(spec).toEqual(before);
  });
});
