// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
import {
  buildGraphLayout,
  getArtifactNodeKey,
  getKeysFromArtifactNodeKey,
  getTaskNodeKey,
  getTaskKeyFromNodeKey,
  isNode,
} from './StaticFlow';

it('round-trips native view keys and rejects invalid keys', () => {
  expect(getTaskKeyFromNodeKey(getTaskNodeKey('train'))).toBe('train');
  expect(getKeysFromArtifactNodeKey(getArtifactNodeKey('train', 'model'))).toEqual([
    'train',
    'model',
  ]);
  expect(() => getTaskKeyFromNodeKey('wrong')).toThrow();
  expect(() => getKeysFromArtifactNodeKey('artifact.invalid')).toThrow();
});

it('lays dependencies above consumers using card dimensions', () => {
  const result = buildGraphLayout([
    { id: 'a', data: { label: 'A' }, position: { x: 0, y: 0 }, width: 300, height: 60 },
    { id: 'b', data: { label: 'B' }, position: { x: 0, y: 0 } },
    { id: 'edge', source: 'a', target: 'b' },
  ]);
  const nodes = result.filter(isNode);
  expect(nodes[0].position.y + 60).toBeLessThan(nodes[1].position.y);
  expect(result[2]).toMatchObject({ source: 'a', target: 'b' });
});
