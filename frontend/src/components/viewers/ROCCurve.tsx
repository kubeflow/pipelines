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
  CartesianGrid,
  Line,
  LineChart,
  ReferenceArea,
  ReferenceLine,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import Viewer, { ViewerConfig } from './Viewer';
import { color, commonCss, fontsize } from '../../Css';
import { stylesheet } from 'typestyle';

const axisTickStyle = {
  fill: color.grey,
  fontSize: fontsize.small,
  fontWeight: 400,
} as const;

const axisLabelStyle = {
  fill: color.strong,
  fontSize: fontsize.small + 1,
  fontWeight: 500,
} as const;

const css = stylesheet({
  crosshair: {
    backgroundColor: color.tooltipBg,
    borderRadius: 5,
    boxShadow: `1px 1px 5px ${color.tooltipShadow}`,
    color: color.background,
    padding: 10,
  },
  crosshairLabel: {
    color: color.background,
    fontWeight: 'bold',
    whiteSpace: 'nowrap',
  },
  legendItem: {
    alignItems: 'center',
    display: 'flex',
    marginRight: 12,
  },
  legendSwatch: {
    borderRadius: 2,
    display: 'inline-block',
    height: 10,
    marginRight: 6,
    width: 10,
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

interface TooltipPayloadEntry {
  payload?: { x?: number | string };
}

interface TooltipContentProps {
  active?: boolean;
  label?: number | string;
  payload?: ReadonlyArray<TooltipPayloadEntry>;
}

interface TooltipRow {
  index: number;
  label: string;
}

export const findNearestDisplayPoint = (
  data: DisplayPoint[],
  targetX: number,
): DisplayPoint | null => {
  if (!data.length) {
    return null;
  }
  let nearestPoint = data[0];
  let nearestDistance = Math.abs(data[0].x - targetX);
  for (let i = 1; i < data.length; i++) {
    const distance = Math.abs(data[i].x - targetX);
    if (distance < nearestDistance) {
      nearestPoint = data[i];
      nearestDistance = distance;
    }
  }
  return nearestPoint;
};

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
  lastDrawLocation: { left: number; right: number } | null;
  highlightIndex: number;
  refAreaLeft: number | null;
  refAreaRight: number | null;
}

class ROCCurve extends Viewer<ROCCurveProps, ROCCurveState> {
  private cachedConfigs: ROCCurveConfig[] | null = null;
  private cachedChartData: Array<Record<string, number | null>> = [];
  private readonly xTicks = Array.from({ length: 20 }, (_, i) => Number((i * 0.05).toFixed(2)));
  private readonly yTicks = Array.from({ length: 10 }, (_, i) => Number((i * 0.1).toFixed(1)));

  constructor(props: any) {
    super(props);

    this.state = {
      lastDrawLocation: null,
      highlightIndex: -1, // -1 indicates no curve is highlighted
      refAreaLeft: null,
      refAreaRight: null,
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
    const configs = this.props.configs;
    const datasets = configs.map(d => d.data);
    const labels = configs.map((_, i) => `threshold (Series #${i + 1})`);
    const { lastDrawLocation, highlightIndex, refAreaLeft, refAreaRight } = this.state;
    const chartData = this.getChartData(configs, datasets);
    const xDomain = lastDrawLocation
      ? ([lastDrawLocation.left, lastDrawLocation.right] as [number, number])
      : ([0, 1] as [number, number]);
    const colors = this.props.colors || lineColors;
    const showLegend = this.props.forceLegend || datasets.length > 1;

    return (
      <div>
        <LineChart
          width={width}
          height={height}
          data={chartData}
          margin={{ top: 5, right: 5, bottom: 20, left: 32 }}
          onMouseDown={
            !isSmall
              ? e => {
                  if (e?.activeLabel == null) {
                    return;
                  }
                  this.setState({ refAreaLeft: Number(e.activeLabel), refAreaRight: null });
                }
              : undefined
          }
          onMouseMove={
            !isSmall && refAreaLeft != null
              ? e => {
                  if (e?.activeLabel == null) {
                    return;
                  }
                  this.setState({ refAreaRight: Number(e.activeLabel) });
                }
              : undefined
          }
          onMouseUp={
            !isSmall && refAreaLeft != null
              ? () => {
                  if (refAreaRight == null || refAreaLeft === refAreaRight) {
                    this.setState({ refAreaLeft: null, refAreaRight: null });
                    return;
                  }
                  const left = Math.min(refAreaLeft, refAreaRight);
                  const right = Math.max(refAreaLeft, refAreaRight);
                  this.setState({
                    lastDrawLocation: { left, right },
                    refAreaLeft: null,
                    refAreaRight: null,
                  });
                }
              : undefined
          }
        >
          <CartesianGrid />
          <XAxis
            type='number'
            dataKey='x'
            domain={xDomain}
            ticks={this.xTicks}
            interval={0}
            tickFormatter={(value: number) => value.toFixed(2)}
            tick={axisTickStyle}
            label={{
              value: 'fpr',
              position: 'insideBottom',
              offset: 0,
              ...axisLabelStyle,
            }}
          />
          <YAxis
            type='number'
            domain={[0, 1]}
            ticks={this.yTicks}
            interval={0}
            tickFormatter={(value: number) => value.toFixed(1)}
            tick={axisTickStyle}
            label={{
              value: 'tpr',
              angle: -90,
              position: 'insideLeft',
              offset: 14,
              ...axisLabelStyle,
            }}
          />
          <ReferenceLine
            segment={[
              { x: 0, y: 0 },
              { x: 1, y: 1 },
            ]}
            stroke={color.disabledBg}
            strokeWidth={1}
            strokeDasharray='4 4'
          />
          {datasets.map((_, i) => (
            <Line
              key={i}
              type='basis'
              dataKey={`y${i}`}
              stroke={colors[i] || colors[colors.length - 1]}
              strokeWidth={highlightIndex === i ? 4 : 2}
              dot={false}
              isAnimationActive={!this.props.disableAnimation && !isSmall}
              connectNulls={true}
            />
          ))}
          {!isSmall && refAreaLeft != null && refAreaRight != null && (
            <ReferenceArea
              x1={Math.min(refAreaLeft, refAreaRight)}
              x2={Math.max(refAreaLeft, refAreaRight)}
              strokeOpacity={0.1}
            />
          )}
          {!isSmall && (
            <Tooltip
              cursor={{ stroke: color.weak }}
              content={tooltipProps => this.renderTooltipContent(tooltipProps, datasets, labels)}
            />
          )}
        </LineChart>

        <div className={commonCss.flex}>
          {/* Legend */}
          {showLegend && (
            <div style={{ flexGrow: 1 }}>
              <div className={commonCss.flex}>
                {datasets.map((_, i) => (
                  <div
                    key={`legend-${i}`}
                    className={css.legendItem}
                    onMouseEnter={() => this.setState({ highlightIndex: i })}
                    onMouseLeave={() => this.setState({ highlightIndex: -1 })}
                  >
                    <span
                      className={css.legendSwatch}
                      style={{
                        backgroundColor: colors[i] || colors[colors.length - 1],
                      }}
                    />
                    <span>{`Series #${i + 1}`}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {lastDrawLocation && (
            <button
              type='button'
              onClick={() => this.setState({ lastDrawLocation: null })}
              style={{ cursor: 'pointer' }}
            >
              Click to reset zoom
            </button>
          )}
        </div>
      </div>
    );
  }

  private buildChartData(datasets: DisplayPoint[][]): Array<Record<string, number | null>> {
    const pointsBySeries = datasets.map(data => {
      const map = new Map<number, DisplayPoint>();
      data.forEach(point => map.set(point.x, point));
      return map;
    });
    const xValues = new Set<number>();
    pointsBySeries.forEach(map => map.forEach((_value, key) => xValues.add(key)));
    const sortedX = Array.from(xValues).sort((a, b) => a - b);
    return sortedX.map(xValue => {
      const row: Record<string, number | null> = { x: xValue };
      pointsBySeries.forEach((map, index) => {
        const point = map.get(xValue);
        row[`y${index}`] = point ? point.y : null;
      });
      return row;
    });
  }

  private getChartData(
    configs: ROCCurveConfig[],
    datasets: DisplayPoint[][],
  ): Array<Record<string, number | null>> {
    if (this.cachedConfigs !== configs) {
      this.cachedConfigs = configs;
      this.cachedChartData = this.buildChartData(datasets);
    }
    return this.cachedChartData;
  }

  private resolveHoveredX({ label, payload }: TooltipContentProps): number | null {
    const labelNumber = typeof label === 'number' ? label : Number(label);
    if (Number.isFinite(labelNumber)) {
      return labelNumber;
    }
    const payloadPoint = payload?.[0]?.payload;
    const payloadX = payloadPoint?.x != null ? Number(payloadPoint.x) : NaN;
    return Number.isFinite(payloadX) ? payloadX : null;
  }

  private buildTooltipRows(datasets: DisplayPoint[][], hoveredX: number): TooltipRow[] {
    return datasets
      .map((dataset, index) => {
        const nearest = findNearestDisplayPoint(dataset, hoveredX);
        if (!nearest) {
          return null;
        }
        return { index, label: nearest.label };
      })
      .filter((row): row is TooltipRow => !!row);
  }

  private renderTooltipRows(rows: TooltipRow[], labels: string[]): JSX.Element {
    return (
      <div className={css.crosshair}>
        {rows.map(row => (
          <div key={row.index} className={css.crosshairLabel}>
            {`${labels[row.index]}: ${row.label}`}
          </div>
        ))}
      </div>
    );
  }

  private renderTooltipContent(
    tooltipProps: TooltipContentProps,
    datasets: DisplayPoint[][],
    labels: string[],
  ): React.ReactNode {
    if (!tooltipProps.active || !tooltipProps.payload?.length) {
      return null;
    }
    const hoveredX = this.resolveHoveredX(tooltipProps);
    if (hoveredX == null) {
      return null;
    }
    const rows = this.buildTooltipRows(datasets, hoveredX);
    if (!rows.length) {
      return null;
    }
    return this.renderTooltipRows(rows, labels);
  }
}

export default ROCCurve;
