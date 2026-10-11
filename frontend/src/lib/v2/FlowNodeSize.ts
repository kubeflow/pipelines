// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import type { Node } from '@xyflow/react';
import { FlowElementDataBase, GRAPH_NODE_WIDTH, GRAPH_NODE_HEIGHT } from 'src/components/graph/Constants';
export const NODE_CARD_HEIGHT = GRAPH_NODE_HEIGHT;
export const SUB_DAG_STATUS_CARD_WIDTH = GRAPH_NODE_WIDTH;

/** Modernized cards use fixed CSS pixels, independently of the root font size. */
export function getFlowNodeSize(_node: Pick<Node<FlowElementDataBase>, 'type' | 'data'>, _nodeScale = 1): { width: number; height: number } {
  return { width: GRAPH_NODE_WIDTH, height: GRAPH_NODE_HEIGHT };
}
