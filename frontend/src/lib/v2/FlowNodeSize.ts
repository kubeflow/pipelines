// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import type { Node } from '@xyflow/react';
import type { FlowElementDataBase } from 'src/components/graph/Constants';
import { NodeTypeNames } from './StaticFlow';

export const NODE_CARD_HEIGHT = 48;

/** Rendered card size in graph coordinates; rem dimensions scale, borders do not. */
export function getFlowNodeSize(
  node: Pick<Node<FlowElementDataBase>, 'type' | 'data'>,
  nodeScale = 1,
  subDagMode: 'inline' | 'click-through' = 'inline',
): { width: number; height: number } {
  if (node.type === NodeTypeNames.SUB_DAG) {
    return subDagMode === 'click-through'
      ? { width: 288 * nodeScale + 4, height: 96 * nodeScale + 4 }
      : { width: (node.data.state ? 320 : 288) * nodeScale, height: NODE_CARD_HEIGHT * nodeScale };
  }
  return {
    width: (node.type === NodeTypeNames.ARTIFACT ? 240 : node.data.state ? 256 : 224) * nodeScale,
    height: NODE_CARD_HEIGHT * nodeScale,
  };
}
