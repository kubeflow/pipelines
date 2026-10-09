// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import type { Node } from '@xyflow/react';
import type { FlowElementDataBase } from 'src/components/graph/Constants';
import { NodeTypeNames } from './StaticFlow';

export const NODE_CARD_HEIGHT = 48;
export const SUB_DAG_STATUS_CARD_WIDTH = 320;

/** Rendered card size in graph coordinates; rem dimensions scale, borders do not. */
export function getFlowNodeSize(
  node: Pick<Node<FlowElementDataBase>, 'type' | 'data'>,
  nodeScale = 1,
  renderSubDags = true,
): { width: number; height: number } {
  if (node.type === NodeTypeNames.SUB_DAG) {
    return renderSubDags
      ? {
          width: (node.data.state ? SUB_DAG_STATUS_CARD_WIDTH : 288) * nodeScale,
          height: NODE_CARD_HEIGHT * nodeScale,
        }
      : { width: 288 * nodeScale + 4, height: 96 * nodeScale + 4 };
  }
  return {
    width: (node.type === NodeTypeNames.ARTIFACT ? 240 : node.data.state ? 256 : 224) * nodeScale,
    height: NODE_CARD_HEIGHT * nodeScale,
  };
}
