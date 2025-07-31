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

import React from 'react';
import { LineChart, Line, ResponsiveContainer } from 'recharts';

// How many pixels away from the point the curve should start at.
const HORIZONTAL_CONTROL_POINT_OFFSET = 30;

type EdgeLineProps = {
  height: number;
  width: number;
  y1: number;
  y4: number;
};

export const EdgeLine: React.FC<EdgeLineProps> = ({ height, width, y1, y4 }) => {
  const data = [
    { x: 0, y: y1 },
    { x: HORIZONTAL_CONTROL_POINT_OFFSET, y: y1 },
    { x: width - HORIZONTAL_CONTROL_POINT_OFFSET, y: y4 },
    { x: width, y: y4 },
  ];

  return (
    <ResponsiveContainer width={width} height={height}>
      <LineChart data={data} margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
        <Line
          type='linear'
          dataKey='y'
          stroke='#BDC1C6'
          strokeWidth={1}
          dot={false}
          isAnimationActive={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
};
