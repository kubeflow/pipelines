// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { GROUP_NODE_TYPE } from 'src/lib/v2/GroupedFlow';
import { NodeTypeNames } from 'src/lib/v2/StaticFlow';
import ArtifactNode from './ArtifactNode';
import ExecutionNode from './ExecutionNode';
import SubDagGroupNode from './SubDagGroupNode';

export const NODE_TYPES = {
  [NodeTypeNames.EXECUTION]: ExecutionNode,
  [NodeTypeNames.ARTIFACT]: ArtifactNode,
  [GROUP_NODE_TYPE]: SubDagGroupNode,
};
