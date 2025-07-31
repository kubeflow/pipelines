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

import React, { useState } from 'react';
import {
  VictoryChart,
  VictoryLine,
  VictoryAxis,
  VictoryLegend,
  VictoryTooltip,
  VictoryVoronoiContainer,
  VictoryGroup,
} from 'victory';
import { ViewerConfig } from './Viewer';
import { color, fontsize, commonCss } from '../../Css';
import { stylesheet } from 'typestyle';

const css = stylesheet({
  axis: {
    fontSize: fontsize.medium,
    fontWeight: 'bolder',
  },
  crosshair: {
    backgroundColor: '#1d2744',
    borderRadius: 5,
    boxShadow: '1px 1px 5px #aaa',
    padding: 10,
  },
  crosshairLabel: {
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

const ROCCurve: React.FC<ROCCurveProps> = ({
  configs,
  maxDimension,
  colors,
  forceLegend,
  disableAnimation,
}) => {
  const width = maxDimension || 800;
  const height = width * 0.65;
  const isSmall = width < 600;
  const datasets = configs.map(d => d.data);
  // const numLines = datasets.length;
  // const labels = configs.map((_, i) => `threshold (Series #${i + 1})`);
  const baseLineData = Array.from(Array(100).keys()).map(x => ({ x: x / 100, y: x / 100 }));

  const [highlightIndex, setHighlightIndex] = useState<number>(-1);
  const [zoomDomain, setZoomDomain] = useState<any>(undefined);

  // Tooltip state
  const [, setActivePoints] = useState<any[]>([]);

  return (
    <div>
      <VictoryChart
        width={width}
        height={height}
        containerComponent={
          <VictoryVoronoiContainer
            labels={({ datum }: { datum: any; seriesIndex: number }) => {
              if (datum && datum.label) {
                return `${datum.label}`;
              }
              return `(${datum.x.toFixed(2)}, ${datum.y.toFixed(2)})`;
            }}
            labelComponent={<VictoryTooltip flyoutStyle={{ fill: '#1d2744', color: '#fff' }} />}
            onActivated={(points: any[]) => setActivePoints(points)}
            voronoiDimension='x'
          />
        }
        domain={zoomDomain}
        animate={disableAnimation || isSmall ? undefined : { duration: 500 }}
        style={{ parent: { margin: 'auto' } }}
      >
        {/* Axes */}
        <VictoryAxis
          label='fpr'
          style={{
            axisLabel: css.axis as React.CSSProperties,
            tickLabels: { fontSize: fontsize.medium },
          }}
        />
        <VictoryAxis
          dependentAxis
          label='tpr'
          style={{
            axisLabel: css.axis as React.CSSProperties,
            tickLabels: { fontSize: fontsize.medium },
          }}
        />
        {/* Reference line */}
        <VictoryLine
          data={baseLineData}
          style={{ data: { stroke: color.disabledBg, strokeDasharray: '5,5', strokeWidth: 1 } }}
        />
        {/* ROC Curves */}
        <VictoryGroup>
          {datasets.map((data, i) => (
            <VictoryLine
              key={i}
              data={data}
              style={{
                data: {
                  stroke: colors?.[i] || lineColors[i] || lineColors[lineColors.length - 1],
                  strokeWidth: highlightIndex === i ? 5 : 2,
                  opacity: highlightIndex === -1 || highlightIndex === i ? 1 : 0.3,
                },
              }}
              interpolation='basis'
              events={[
                {
                  target: 'data',
                  eventHandlers: {
                    onMouseOver: () => {
                      setHighlightIndex(i);
                      return null;
                    },
                    onMouseOut: () => {
                      setHighlightIndex(-1);
                      return null;
                    },
                  },
                },
              ]}
            />
          ))}
        </VictoryGroup>
      </VictoryChart>
      <div className={commonCss.flex}>
        {/* Legend */}
        {(forceLegend || datasets.length > 1) && (
          <div style={{ flexGrow: 1 }}>
            <VictoryLegend
              orientation='horizontal'
              gutter={20}
              data={datasets.map((_, i) => ({
                name: 'Series #' + (i + 1),
                symbol: {
                  fill: colors?.[i] || lineColors[i] || lineColors[lineColors.length - 1],
                  type: 'square',
                },
              }))}
              style={{
                labels: { fontSize: fontsize.medium },
              }}
              events={[
                {
                  target: 'data',
                  eventHandlers: {
                    onMouseOver: (_: React.SyntheticEvent, props: { index: number }) => {
                      setHighlightIndex(props.index);
                      return null;
                    },
                    onMouseOut: () => {
                      setHighlightIndex(-1);
                      return null;
                    },
                  },
                },
              ]}
            />
          </div>
        )}
        {zoomDomain && (
          <span
            style={{ marginLeft: 16, cursor: 'pointer' }}
            onClick={() => setZoomDomain(undefined)}
          >
            Click to reset zoom
          </span>
        )}
      </div>
    </div>
  );
};

export default ROCCurve;
