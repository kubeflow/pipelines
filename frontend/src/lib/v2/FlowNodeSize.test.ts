// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { PipelineTaskTaskState } from 'src/apisv2beta1/run';
import { getFlowNodeSize } from './FlowNodeSize';
import { NodeTypeNames } from './StaticFlow';

it.each([1, 13 / 16])(
  'sizes ordinary cards consistently across both renderers at scale %s',
  (scale) => {
    for (const [type, state, width] of [
      [NodeTypeNames.EXECUTION, undefined, 224],
      [NodeTypeNames.EXECUTION, PipelineTaskTaskState.SUCCEEDED, 256],
      [NodeTypeNames.ARTIFACT, undefined, 240],
    ] as const) {
      const node = { type, data: { label: 'Task', state } };
      const size = { width: width * scale, height: 48 * scale };
      expect(getFlowNodeSize(node, scale)).toEqual(size);
      expect(getFlowNodeSize(node, scale, false)).toEqual(size);
    }
  },
);

it.each([1, 13 / 16])(
  'distinguishes inline and bordered click-through group cards at scale %s',
  (scale) => {
    const group = { type: NodeTypeNames.SUB_DAG, data: { label: 'Pipeline' } };
    expect(getFlowNodeSize(group, scale)).toEqual({ width: 288 * scale, height: 48 * scale });
    expect(
      getFlowNodeSize(
        { ...group, data: { ...group.data, state: PipelineTaskTaskState.RUNNING } },
        scale,
      ),
    ).toEqual({ width: 320 * scale, height: 48 * scale });
    expect(getFlowNodeSize(group, scale, false)).toEqual({
      width: 288 * scale + 4,
      height: 96 * scale + 4,
    });
  },
);
