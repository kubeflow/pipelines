/*
 * Copyright 2018 The Kubeflow Authors
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
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  Brush,
} from 'recharts';
import Viewer, { ViewerConfig } from './Viewer';
import { color, fontsize, commonCss } from '../../Css';
import { stylesheet } from 'typestyle';

const css = stylesheet({
  axis: {
    fontSize: fontsize.medium,
    fontWeight: 'bolder',
  },
  tooltip: {
    backgroundColor: '#1d2744',
    borderRadius: 5,
    boxShadow: '1px 1px 5px #aaa',
    padding: 10,
    color: 'white',
  },
  tooltipLabel: {
    fontWeight: 'bold',
    whiteSpace: 'nowrap',
  },
  root: {
    margin: 'auto',
  },
});

// Used the following color palette, with some slight brightness modifications, as reference:
// https://alumni.media.mit.edu/~wad/color/palette.html
export const lineColors = [
  '#4285f4',
  '#2b9c1e',
  '#e00000',
  '#8026c0',
  '#9dafff',
  '#82c57a',
  '#814a19',
  '#ff9233',
  '#29d0d0',
  '#ffee33',
  '#f540c9',
];

export interface DisplayPoint {
  label: string;
  x: number;
  y: number;
}

export interface ROCCurveConfig extends ViewerConfig {
  data: DisplayPoint[];
}

interface ROCCurveProps {
  configs: ROCCurveConfig[];
  maxDimension?: number;
  colors?: string[];
  forceLegend?: boolean; // Forces the legend to display even with just one ROC Curve
  disableAnimation?: boolean;
}

interface ROCCurveState {
  hoveredValues: DisplayPoint[];
  activeSeriesIndex: number;
}

class ROCCurve extends Viewer<ROCCurveProps, ROCCurveState> {
  constructor(props: any) {
    super(props);

    this.state = {
      hoveredValues: new Array(this.props.configs.length).fill(''),
      activeSeriesIndex: -1,
    };
  }

  public getDisplayName(): string {
    return 'ROC Curve';
  }

  public isAggregatable(): boolean {
    return true;
  }

  public render(): JSX.Element {
    const width = this.props.maxDimension || 800;
    const height = width * 0.65;
    const isSmall = width < 600;
    const datasets = this.props.configs.map(d => d.data);
    const numLines = datasets.length;
    const baseLineData = Array.from(Array(100).keys()).map(x => ({ x: x / 100, y: x / 100 }));

    const { activeSeriesIndex } = this.state;

    const CustomTooltip = ({ active, payload, label }: any) => {
      if (active && payload && payload.length) {
        return (
          <div className={css.tooltip}>
            {payload.map((entry: any, index: number) => (
              <div key={index} className={css.tooltipLabel}>
                {`Series #${index + 1}: (${entry.payload.x.toFixed(3)}, ${entry.payload.y.toFixed(3)})`}
              </div>
            ))}
          </div>
        );
      }
      return null;
    };

    return (
      <div>
        <ResponsiveContainer width="100%" height={height}>
          <LineChart margin={{ top: 20, right: 20, bottom: 20, left: 20 }}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="x"
              type="number"
              domain={[0, 1]}
              label={{ value: 'fpr', position: 'bottom' }}
              fontSize={fontsize.medium}
            />
            <YAxis
              dataKey="y"
              type="number"
              domain={[0, 1]}
              label={{ value: 'tpr', angle: -90, position: 'left' }}
              fontSize={fontsize.medium}
            />
            <Tooltip content={<CustomTooltip />} />
            {(this.props.forceLegend || datasets.length > 1) && (
              <Legend
                onMouseEnter={(e: any) => this.setState({ activeSeriesIndex: e.dataKey })}
                onMouseLeave={() => this.setState({ activeSeriesIndex: -1 })}
              />
            )}
            {!isSmall && <Brush dataKey="x" height={30} stroke="#8884d8" />}

            {/* Reference line */}
            <Line
              data={baseLineData}
              type="monotone"
              dataKey="y"
              stroke={color.disabledBg}
              strokeWidth={1}
              strokeDasharray="5 5"
              dot={false}
              isAnimationActive={!this.props.disableAnimation && !isSmall}
            />

            {/* Data lines */}
            {datasets.map((data, i) => (
              <Line
                key={i}
                data={data}
                type="monotone"
                dataKey="y"
                name={`Series #${i + 1}`}
                stroke={this.props.colors ? this.props.colors[i] : lineColors[i]}
                strokeWidth={activeSeriesIndex === i ? 4 : 2}
                dot={false}
                isAnimationActive={!this.props.disableAnimation && !isSmall}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
    );
  }
}

export default ROCCurve;
