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
import { LineChart, Line, ResponsiveContainer } from 'recharts';
import { color } from '../Css';

interface EdgeLineProps {
  height?: number;
  width?: number;
  points: Array<{ x: number; y: number }>;
}

export class EdgeLine extends React.Component<EdgeLineProps> {
  public render(): JSX.Element {
    const { points, height = 100, width = 100 } = this.props;
    const data = points.map(point => ({ x: point.x, y: point.y }));

    return (
      <div style={{ width, height }}>
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data}>
            <Line
              type="monotone"
              dataKey="y"
              stroke={color.theme}
              dot={false}
              isAnimationActive={false}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    );
  }
}
