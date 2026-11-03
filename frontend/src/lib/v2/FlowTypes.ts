// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import type { Edge, Node } from '@xyflow/react';
import type { FlowElementDataBase } from 'src/components/graph/Constants';

export type PipelineFlowElement = Node<FlowElementDataBase> | Edge;

/** A local graph element and the DAG instance that owns its task/artifact IDs. */
export interface LayerFlowElement {
  element: PipelineFlowElement;
  layers: string[];
}

/** Resolves the task and artifact nodes in one DAG layer. */
export type LayerElementsResolver = (layers: string[]) => PipelineFlowElement[];
