/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type * as dagre from 'dagre';
import type * as React from 'react';
import type { SelectedNodeInfo } from './StaticGraphParser';

/**
 * Presentation data the parsers attach to each dagre node, on top of the
 * layout fields dagre computes itself.
 *
 * `StaticGraphParser` supplies `bgColor` and `info`; `WorkflowParser`
 * supplies `icon`, `statusColoring`, and `isPlaceholder`.
 *
 * Producers should write their `setNode` literals with `satisfies
 * GraphNodeInput`. dagre types the label parameter as `Label`, an index
 * signature accepting anything, so the graph's own type parameter checks
 * readers but cannot check writers.
 */
export interface GraphNodeData {
  bgColor?: string;
  icon?: React.ReactNode;
  info?: SelectedNodeInfo;
  isPlaceholder?: boolean;
  statusColoring?: string;
}

/**
 * What a producer passes to `setNode`: presentation data plus the layout
 * hints dagre reads. `x` and `y` are computed by `dagre.layout`, so they are
 * deliberately absent here.
 */
export type GraphNodeInput = GraphNodeData & {
  height?: number;
  label?: string;
  width?: number;
};

/** A dagre graph whose nodes carry {@link GraphNodeData}. */
export type DagreGraph = dagre.graphlib.Graph<GraphNodeData>;
