/*
 * Copyright 2018 Google LLC
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
import { ViewerConfig } from './Viewer';
import { color, fontsize, commonCss } from '../../Css';
import { stylesheet } from 'typestyle';
import { TFunction } from 'i18next';

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

const lineColors = [
  '#4285f4',
  '#efb4a3',
  '#684e91',
  '#d74419',
  '#7fa6c4',
  '#ffdc10',
  '#d7194d',
  '#6b2f49',
  '#f9e27c',
  '#633a70',
  '#5ec4ec',
];

interface DisplayPoint {
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
  t: TFunction;
}

interface ROCCurveState {
  hoveredValues: DisplayPoint[];
  lastDrawLocation: { left: number; right: number } | null;
}

class ROCCurve extends React.Component<ROCCurveProps, ROCCurveState> {
  constructor(props: any) {
    super(props);

    this.state = {
      hoveredValues: new Array(this.props.configs.length).fill(''),
      lastDrawLocation: null,
    };
  }

  public render(): JSX.Element {
    const width = this.props.maxDimension || 800;
    const height = width * 0.65;
    const isSmall = width < 600;
    const datasets = this.props.configs.map(d => d.data);
    const numLines = datasets.length;
    const labels = this.props.configs.map((_, i) => `threshold (Series #${i})`);
    const baseLineData = Array.from(Array(100).keys()).map(x => ({ x: x / 100, y: x / 100 }));

    const { hoveredValues, lastDrawLocation } = this.state;

    return (
      <div>
        <XYPlot
          width={width}
          height={height}
          animation={!isSmall}
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
          {datasets.map((data, i) => (
            <LineSeries
              key={i}
              color={lineColors[i] || lineColors[lineColors.length - 1]}
              strokeWidth={2}
              data={data}
              onNearestX={(d: any) => this._lineHovered(i, d)}
              curve='curveBasis'
            />
          ))}

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
          {datasets.length > 1 && (
            <div style={{ flexGrow: 1 }}>
              <DiscreteColorLegend
                items={datasets.map((_, i) => ({
                  color: lineColors[i],
                  title: 'Series #' + (i + 1),
                }))}
                orientation='horizontal'
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
