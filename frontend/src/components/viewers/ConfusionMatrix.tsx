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

import type * as React from 'react';
import Viewer, { ViewerConfig, PlotType } from './Viewer';
import { color, commonCss, fontsize } from '../../Css';
import { classes, stylesheet } from 'typestyle';

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

class ConfusionMatrix extends Viewer<ConfusionMatrixProps, ConfusionMatrixState> {
  private _minRegularCellDimension = 15;
  private _maxRegularCellDimension = 80;
  private _shrinkThreshold = 600;

  private _css = stylesheet({
    activeLabel: {
      borderRadius: 5,
      color: color.theme,
      fontWeight: 'bold',
    },
    cell: {
      border: 'solid 1px ' + color.background,
      fontSize: this._isSmall() ? fontsize.small : fontsize.base,
      position: 'relative',
      textAlign: 'center',
      verticalAlign: 'middle',
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
      marginRight: 10,
      textAlign: 'right',
      whiteSpace: 'nowrap',
    },
  });

  constructor(props: any) {
    super(props);
    this.state = {
      activeCell: [-1, -1],
    };
  }

  public getDisplayName(): string {
    return 'Confusion matrix';
  }

  public render(): React.JSX.Element | null {
    const config = this.props.configs[0];
    if (!config) {
      return null;
    }

    const { cellDimension, max, opacities, uiData } = this._buildViewModel(config);
    const [activeRow, activeCol] = this.state.activeCell;
    const [xAxisLabel, yAxisLabel] = config.axes;
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
            {uiData.map((row, r) => (
              <tr key={r}>
                {!small && (
                  <td>
                    <div
                      className={classes(
                        this._css.ylabel,
                        r === activeRow ? this._css.activeLabel : '',
                      )}
                      style={{ lineHeight: `${cellDimension}px`, minWidth: cellDimension }}
                    >
                      {
                        config.labels[
                          config.labels.length - 1 - r
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
                      backgroundColor: `rgba(41, 121, 255, ${opacities[r][c]})`,
                      color: opacities[r][c] < 0.6 ? color.foreground : color.background,
                      height: cellDimension,
                      minHeight: cellDimension,
                      minWidth: cellDimension,
                      width: cellDimension,
                    }}
                    onMouseOver={() => this.setState({ activeCell: [r, c] })}
                    onMouseLeave={() =>
                      this.setState((state) => ({
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
                {config.labels.map((label, i) => (
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
            style={{ height: 0.75 * config.data.length * cellDimension }}
          >
            <div className={this._css.legendNotch} style={{ top: 0 }}>
              <span className={this._css.legendLabel}>{max}</span>
            </div>
            {new Array(legendNotches).fill(0).map((_, i) => (
              <div
                key={i}
                className={this._css.legendNotch}
                style={{ top: ((legendNotches - i) / legendNotches) * 100 + '%' }}
              >
                <span className={this._css.legendLabel}>
                  {Math.floor((i / legendNotches) * max)}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    );
  }

  private _buildViewModel(config: ConfusionMatrixConfig): {
    cellDimension: number;
    max: number;
    opacities: number[][];
    uiData: number[][];
  } {
    const max = Math.max(...config.data.map((row) => Math.max(...row.map((value) => +value))));
    const cellDimension =
      Math.max(
        Math.min(
          (this.props.maxDimension || 700) / config.data.length,
          this._maxRegularCellDimension,
        ),
        this._minRegularCellDimension,
      ) - 1;
    const labelCount = config.labels?.length || 0;
    const uiData: number[][] = new Array(labelCount)
      .fill(undefined)
      .map(() => new Array(labelCount));
    for (let i = 0; i < labelCount; ++i) {
      for (let j = 0; j < labelCount; ++j) {
        uiData[labelCount - 1 - j][i] = config.data[i]?.[j];
      }
    }
    return {
      cellDimension,
      max,
      opacities: uiData.map((row) => row.map((value) => +value / max)),
      uiData,
    };
  }

  private _isSmall(): boolean {
    return !!this.props.maxDimension && this.props.maxDimension < this._shrinkThreshold;
  }
}

export default ConfusionMatrix;
