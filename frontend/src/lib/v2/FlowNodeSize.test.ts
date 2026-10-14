// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { Execution } from 'src/third_party/mlmd';
import { getFlowNodeSize } from './FlowNodeSize';
import { NodeTypeNames } from './StaticFlow';

it.each([1, 13 / 16])('sizes ordinary cards at scale %s', (scale) => {
  for (const [type, state, width] of [
    [NodeTypeNames.EXECUTION, undefined, 224],
    [NodeTypeNames.EXECUTION, Execution.State.COMPLETE, 256],
    [NodeTypeNames.ARTIFACT, undefined, 240],
  ] as const) {
    const node = { type, data: { label: 'Task', state } };
    const size = { width: width * scale, height: 48 * scale };
    expect(getFlowNodeSize(node, scale)).toEqual(size);
  }
});

it.each([1, 13 / 16])('sizes inline group cards with and without status at scale %s', (scale) => {
  const group = { type: NodeTypeNames.SUB_DAG, data: { label: 'Pipeline' } };
  expect(getFlowNodeSize(group, scale)).toEqual({ width: 288 * scale, height: 48 * scale });
  expect(
    getFlowNodeSize({ ...group, data: { ...group.data, state: Execution.State.RUNNING } }, scale),
  ).toEqual({ width: 320 * scale, height: 48 * scale });
});

describe.each([1, 13 / 16])('zero-valued MLMD state at scale %s', (scale) => {
  it.each([
    [NodeTypeNames.EXECUTION, 256],
    [NodeTypeNames.SUB_DAG, 320],
  ] as const)('reserves status width for UNKNOWN %s nodes', (type, width) => {
    expect(Execution.State.UNKNOWN).toBe(0);
    expect(
      getFlowNodeSize({ type, data: { label: 'Task', state: Execution.State.UNKNOWN } }, scale),
    ).toEqual({ width: width * scale, height: 48 * scale });
  });
});
