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
import { ViewerConfig, PlotType } from './Viewer';
import { color, commonCss, fontsize } from '../../Css';
import { classes, stylesheet } from 'typestyle';
import { TFunction } from 'i18next';

const legendNotches = 5;

export interface ConfusionMatrixConfig extends ViewerConfig {
  data: number[][];
  axes: string[];
  labels: string[];
  type: PlotType;
}

interface ConfusionMatrixProps {
  configs: ConfusionMatrixConfig[];
  maxDimension?: number;
}

interface ConfusionMatrixState {
  activeCell: [number, number];
}

class ConfusionMatrix extends React.Component<ConfusionMatrixProps, ConfusionMatrixState> {
  private _opacities: number[][] = [];
  private _config = this.props.configs[0];
  private _max =
    this._config &&
    Math.max(...this._config.data.map(d => d.map(n => +n)).map(d => Math.max(...d)));
  private _minRegularCellDimension = 15;
  private _maxRegularCellDimension = 80;
  private _cellDimension = this._config
    ? Math.max(
        Math.min(
          (this.props.maxDimension || 700) / this._config.data.length,
          this._maxRegularCellDimension,
        ),
        this._minRegularCellDimension,
      ) - 1
    : 0;
  private _shrinkThreshold = 600;
  private _uiData: number[][] = [];

  private _css = stylesheet({
    activeLabel: {
      borderRadius: 5,
      color: color.theme,
      fontWeight: 'bold',
    },
    cell: {
      border: 'solid 1px ' + color.background,
      fontSize: this._isSmall() ? fontsize.small : fontsize.base,
      height: this._cellDimension,
      minHeight: this._cellDimension,
      minWidth: this._cellDimension,
      position: 'relative',
      textAlign: 'center',
      verticalAlign: 'middle',
      width: this._cellDimension,
    },
    legend: {
      background: `linear-gradient(${color.theme}, ${color.background})`,
      borderRight: 'solid 1px #777',
      marginLeft: 20,
      minWidth: 10,
      position: 'relative',
      width: 10,
    },
    legendLabel: {
      left: 15,
      position: 'absolute',
      top: -7,
    },
    legendNotch: {
      borderTop: 'solid 1px #777',
      left: '100%',
      paddingLeft: 5,
      position: 'absolute',
      width: 5,
    },
    overlay: {
      backgroundColor: '#000',
      bottom: 0,
      left: 0,
      opacity: 0,
      position: 'absolute',
      right: 0,
      top: 0,
    },
    root: {
      flexGrow: 1,
      justifyContent: 'center',
      margin: 'auto',
      pointerEvents: this._isSmall() ? 'none' : 'initial', // Disable interaction for snapshot view
      position: 'relative',
      width: 'fit-content',
    },
    xAxisLabel: {
      color: color.foreground,
      fontSize: 15,
      fontWeight: 'bold',
      paddingLeft: 20,
      position: 'absolute',
    },
    xlabel: {
      marginLeft: 15,
      overflow: 'hidden',
      position: 'absolute',
      textAlign: 'left',
      textOverflow: 'ellipsis',
      transform: 'rotate(60deg)',
      transformOrigin: 'left',
      whiteSpace: 'nowrap',
      width: 150,
    },
    yAxisLabel: {
      color: color.foreground,
      fontSize: 15,
      height: 25,
      paddingRight: 20,
      textAlign: 'right',
    },
    ylabel: {
      lineHeight: `${this._cellDimension}px`,
      marginRight: 10,
      minWidth: this._cellDimension,
      textAlign: 'right',
      whiteSpace: 'nowrap',
    },
  });

  constructor(props: any) {
    super(props);

    if (!this._config) {
      return;
    }

    this.state = {
      activeCell: [-1, -1],
    };
    // Raw data:
    // [
    //   [1, 2],
    //   [3, 4],
    // ]
    // converts to UI data:
    // y-axis
    // ^
    // |
    // 1  [2, 4],
    // |
    // 0  [1, 3],
    // |
    // *---0--1---> x-axis
    if (!this._config || !this._config.labels || !this._config.data) {
      this._uiData = [];
    } else {
      const labelCount = this._config.labels.length;
      const uiData: number[][] = new Array(labelCount)
        .fill(undefined)
        .map(() => new Array(labelCount));
      for (let i = 0; i < labelCount; ++i) {
        for (let j = 0; j < labelCount; ++j) {
          uiData[labelCount - 1 - j][i] = this._config.data[i]?.[j];
        }
      }
      this._uiData = uiData;
    }

    for (const i of this._uiData) {
      const row = [];
      for (const j of i) {
        row.push(+j / this._max);
      }
      this._opacities.push(row);
    }
  }

  public getDisplayName(t: TFunction): string {
    return t('common:confusionMatrix');
  }

  public render(): JSX.Element | null {
    if (!this._config) {
      return null;
    }

    const [activeRow, activeCol] = this.state.activeCell;
    const [xAxisLabel, yAxisLabel] = this._config.axes;
    const small = this._isSmall();

    return (
      <div className={classes(commonCss.flex, this._css.root)}>
        <table>
          <tbody>
            {!small && (
              <tr>
                <td className={this._css.yAxisLabel}>{yAxisLabel}</td>
              </tr>
            )}
            {this._uiData.map((row, r) => (
              <tr key={r}>
                {!small && (
                  <td>
                    <div
                      className={classes(
                        this._css.ylabel,
                        r === activeRow ? this._css.activeLabel : '',
                      )}
                    >
                      {
                        this._config.labels[
                          this._config.labels.length - 1 - r
                        ] /* uiData's ith's row corresponds to the reverse ordered label */
                      }
                    </div>
                  </td>
                )}
                {row.map((cell, c) => (
                  <td
                    key={c}
                    className={this._css.cell}
                    style={{
                      backgroundColor: `rgba(41, 121, 255, ${this._opacities[r][c]})`,
                      color: this._opacities[r][c] < 0.6 ? color.foreground : color.background,
                    }}
                    onMouseOver={() => this.setState({ activeCell: [r, c] })}
                    onMouseLeave={() =>
                      this.setState(state => ({
                        // Remove active cell if it's still the one active
                        activeCell:
                          state.activeCell[0] === r && state.activeCell[1] === c
                            ? [-1, -1]
                            : state.activeCell,
                      }))
                    }
                  >
                    <div
                      className={this._css.overlay}
                      style={{
                        opacity: r === activeRow || c === activeCol ? 0.05 : 0,
                      }}
                    />
                    {cell}
                  </td>
                ))}
              </tr>
            ))}

            {/* Footer */}
            {!small && (
              <tr>
                <th className={this._css.xlabel} />
                {this._config.labels.map((label, i) => (
                  <th key={i}>
                    <div
                      className={classes(
                        i === activeCol ? this._css.activeLabel : '',
                        this._css.xlabel,
                      )}
                    >
                      {label}
                    </div>
                  </th>
                ))}
                <td className={this._css.xAxisLabel}>{xAxisLabel}</td>
              </tr>
            )}
          </tbody>
        </table>

        {!small && (
          <div
            className={this._css.legend}
            style={{ height: 0.75 * this._config.data.length * this._cellDimension }}
          >
            <div className={this._css.legendNotch} style={{ top: 0 }}>
              <span className={this._css.legendLabel}>{this._max}</span>
            </div>
            {new Array(legendNotches).fill(0).map((_, i) => (
              <div
                key={i}
                className={this._css.legendNotch}
                style={{ top: ((legendNotches - i) / legendNotches) * 100 + '%' }}
              >
                <span className={this._css.legendLabel}>
                  {Math.floor((i / legendNotches) * this._max)}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    );
  }

  private _isSmall(): boolean {
    return !!this.props.maxDimension && this.props.maxDimension < this._shrinkThreshold;
  }
}

export default ConfusionMatrix;
