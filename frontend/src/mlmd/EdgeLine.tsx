/*
 * Copyright 2021 The Kubeflow Authors
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

import * as React from 'react';

// How many pixels away from the point the curve should start at.
const HORIZONTAL_CONTROL_POINT_OFFSET = 30;

interface EdgeLineProps {
  height: number;
  width: number;
  y1: number;
  y4: number;
}

export const EdgeLine: React.FC<EdgeLineProps> = props => {
  const { height, width, y1, y4 } = props;
  // Legacy react-svg-line-chart rendered with an inverted Y axis.
  // Preserve that orientation so edge direction matches existing lineage visuals.
  const startY = height - y1;
  const endY = height - y4;
  const controlPointStartX = HORIZONTAL_CONTROL_POINT_OFFSET;
  const controlPointEndX = width - HORIZONTAL_CONTROL_POINT_OFFSET;
  const path = [
    `M0,${startY}`,
    `C${controlPointStartX},${startY} ${controlPointEndX},${endY} ${width},${endY}`,
  ].join(' ');

  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden='true'>
      <path d={path} fill='none' stroke='#BDC1C6' strokeWidth={1} opacity={1} />
    </svg>
  );
};
