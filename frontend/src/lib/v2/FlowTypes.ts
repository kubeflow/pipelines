// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import type { Edge, Node } from '@xyflow/react';
import type { FlowElementDataBase } from 'src/components/graph/Constants';

export type PipelineFlowElement = Node<FlowElementDataBase> | Edge;

/** A local graph element and the DAG instance that owns its task/artifact IDs. */
export interface ScopedFlowElement {
  element: PipelineFlowElement;
  layers: string[];
}

/** Resolves one DAG layer, rejecting oversized layers before constructing nodes. */
export type LayerElementsResolver = (layers: string[], maxNodes?: number) => PipelineFlowElement[];
