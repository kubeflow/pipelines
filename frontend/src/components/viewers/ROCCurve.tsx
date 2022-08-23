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
  Crosshair,
  DiscreteColorLegend,
  Highlight,
  HorizontalGridLines,
  LineSeries,
  VerticalGridLines,
  XAxis,
  XYPlot,
  YAxis,
  // @ts-ignore
} from 'react-vis';
import 'react-vis/dist/style.css';
import Viewer, { ViewerConfig } from './Viewer';
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

interface ROCCurveState {
  hoveredValues: DisplayPoint[];
  lastDrawLocation: { left: number; right: number } | null;
  highlightIndex: number;
}

class ROCCurve extends Viewer<ROCCurveProps, ROCCurveState> {
  constructor(props: any) {
    super(props);

    this.state = {
      hoveredValues: new Array(this.props.configs.length).fill(''),
      lastDrawLocation: null,
      highlightIndex: -1, // -1 indicates no curve is highlighted
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
    const labels = this.props.configs.map((_, i) => `threshold (Series #${i + 1})`);
    const baseLineData = Array.from(Array(100).keys()).map(x => ({ x: x / 100, y: x / 100 }));

    const { hoveredValues, lastDrawLocation, highlightIndex } = this.state;

    return (
      <div>
        <XYPlot
          width={width}
          height={height}
          animation={!this.props.disableAnimation && !isSmall}
          classes={{ root: css.root }}
          onMouseLeave={() => this.setState({ hoveredValues: new Array(numLines).fill('') })}
          xDomain={lastDrawLocation && [lastDrawLocation.left, lastDrawLocation.right]}
        >
          <VerticalGridLines />
          <HorizontalGridLines />

          {/* Draw the axes from the first config in case there are several */}
          <XAxis title={'fpr'} className={css.axis} />
          <YAxis title={'tpr'} className={css.axis} />

          {/* Reference line */}
          <LineSeries
            color={color.disabledBg}
            strokeWidth={1}
            data={baseLineData}
            strokeStyle='dashed'
          />

          {/* Lines */}
          {datasets.map(
            (data, i) =>
              highlightIndex !== i && (
                <LineSeries
                  key={i}
                  color={
                    this.props.colors
                      ? this.props.colors[i]
                      : lineColors[i] || lineColors[lineColors.length - 1]
                  }
                  strokeWidth={2}
                  data={data}
                  onNearestX={(d: any) => this._lineHovered(i, d)}
                  curve='curveBasis'
                />
              ),
          )}

          {/* Highlighted line, if present */}
          {highlightIndex >= 0 && (
            <LineSeries
              key={highlightIndex}
              color={
                this.props.colors
                  ? this.props.colors[highlightIndex]
                  : lineColors[highlightIndex] || lineColors[lineColors.length - 1]
              }
              strokeWidth={5}
              data={datasets[highlightIndex]}
              onNearestX={(d: any) => this._lineHovered(highlightIndex, d)}
              curve='curveBasis'
            />
          )}

          {!isSmall && (
            <Highlight
              onBrushEnd={(area: any) => this.setState({ lastDrawLocation: area })}
              enableY={false}
              onDrag={(area: any) =>
                this.setState({
                  lastDrawLocation: {
                    left: (lastDrawLocation ? lastDrawLocation.left : 0) - (area.right - area.left),
                    right:
                      (lastDrawLocation ? lastDrawLocation.right : 0) - (area.right - area.left),
                  },
                })
              }
            />
          )}

          {/* Hover effect to show labels */}
          {!isSmall && (
            <Crosshair values={hoveredValues}>
              <div className={css.crosshair}>
                {hoveredValues.map((value, i) => (
                  <div key={i} className={css.crosshairLabel}>{`${labels[i]}: ${value.label}`}</div>
                ))}
              </div>
            </Crosshair>
          )}
        </XYPlot>

        <div className={commonCss.flex}>
          {/* Legend */}
          {(this.props.forceLegend || datasets.length > 1) && (
            <div style={{ flexGrow: 1 }}>
              <DiscreteColorLegend
                items={datasets.map((_, i) => ({
                  color: this.props.colors ? this.props.colors[i] : lineColors[i],
                  title: 'Series #' + (i + 1),
                }))}
                orientation='horizontal'
                onItemMouseEnter={(_: any, i: number) => {
                  this.setState({ highlightIndex: i });
                }}
                onItemMouseLeave={() => {
                  this.setState({ highlightIndex: -1 });
                }}
              />
            </div>
          )}

          {lastDrawLocation && <span>Click to reset zoom</span>}
        </div>
      </div>
    );
  }

  private _lineHovered(lineIdx: number, data: any): void {
    const hoveredValues = this.state.hoveredValues;
    hoveredValues[lineIdx] = data;
    this.setState({ hoveredValues });
  }
}

export default ROCCurve;
