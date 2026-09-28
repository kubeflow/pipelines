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
import './ComparisonViewers.css';

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
      <div className='kfp-confusion-matrix' data-small={small || undefined}>
        <table>
          <tbody>
            {!small && (
              <tr>
                <td className='kfp-confusion-y-axis-label'>{yAxisLabel}</td>
              </tr>
            )}
            {uiData.map((row, r) => (
              <tr key={r}>
                {!small && (
                  <td>
                    <div
                      className={`kfp-confusion-ylabel ${r === activeRow ? 'kfp-confusion-active-label' : ''}`}
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
                    className='kfp-confusion-cell'
                    style={{
                      backgroundColor: `color-mix(in srgb, var(--primary, #2563d9) ${opacities[r][c] * 25}%, var(--card, #fff))`,
                      color: 'var(--foreground, #15171e)',
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
                      className='kfp-confusion-overlay'
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
              <>
                <tr>
                  <th />
                  {config.labels.map((label, i) => (
                    <th key={i} scope='col' className='kfp-confusion-xlabel'>
                      <div
                        className={i === activeCol ? 'kfp-confusion-active-label' : ''}
                        style={{ overflowWrap: 'anywhere', width: cellDimension }}
                      >
                        {label}
                      </div>
                    </th>
                  ))}
                </tr>
                <tr>
                  <td />
                  <td colSpan={config.labels.length} className='kfp-confusion-x-axis-label'>
                    {xAxisLabel}
                  </td>
                </tr>
              </>
            )}
          </tbody>
        </table>

        {!small && (
          <div
            className='kfp-confusion-legend'
            style={{ height: 0.75 * config.data.length * cellDimension }}
          >
            <div className='kfp-confusion-legend-notch' style={{ top: 0 }}>
              <span className='kfp-confusion-legend-label'>{max}</span>
            </div>
            {new Array(legendNotches).fill(0).map((_, i) => (
              <div
                key={i}
                className='kfp-confusion-legend-notch'
                style={{ top: ((legendNotches - i) / legendNotches) * 100 + '%' }}
              >
                <span className='kfp-confusion-legend-label'>
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
      opacities: uiData.map((row) =>
        row.map((value) => (max > 0 && Number.isFinite(value) ? +value / max : 0)),
      ),
      uiData,
    };
  }

  private _isSmall(): boolean {
    return !!this.props.maxDimension && this.props.maxDimension < this._shrinkThreshold;
  }
}

export default ConfusionMatrix;
