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
import { render, screen } from '@testing-library/react';
import { PlotType } from './Viewer';
import ROCCurve, { findNearestDisplayPoint } from './ROCCurve';

describe('ROCCurve', () => {
  const data = [
    { x: 0, y: 0, label: '1' },
    { x: 0.2, y: 0.3, label: '2' },
    { x: 0.5, y: 0.7, label: '3' },
    { x: 0.9, y: 0.9, label: '4' },
    { x: 1, y: 1, label: '5' },
  ];

  it('does not break on no config', () => {
    const { asFragment } = render(<ROCCurve configs={[]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('does not break on empty data', () => {
    const { asFragment } = render(<ROCCurve configs={[{ data: [], type: PlotType.ROC }]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a simple ROC curve given one config', () => {
    const { asFragment } = render(<ROCCurve configs={[{ data, type: PlotType.ROC }]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a reference base line series', () => {
    const { container } = render(<ROCCurve configs={[{ data, type: PlotType.ROC }]} />);
    const referenceLines = container.querySelectorAll('.recharts-reference-line-line');
    expect(referenceLines.length).toBeGreaterThanOrEqual(1);
  });

  it('renders an ROC curve using three configs', () => {
    const config = { data, type: PlotType.ROC };
    const { asFragment } = render(<ROCCurve configs={[config, config, config]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders three lines with three different colors', () => {
    const config = { data, type: PlotType.ROC };
    const { container } = render(<ROCCurve configs={[config, config, config]} />);
    const dataLines = Array.from(container.querySelectorAll('.recharts-line-curve'));
    expect(dataLines.length).toBeGreaterThanOrEqual(3);
    const dataLineColors = dataLines
      .map(line => line.getAttribute('stroke'))
      .filter((stroke): stroke is string => !!stroke);
    expect(new Set(dataLineColors).size).toBeGreaterThanOrEqual(3);
  });

  it('does not render a legend when there is only one config', () => {
    const config = { data, type: PlotType.ROC };
    render(<ROCCurve configs={[config]} />);
    expect(screen.queryByText('Series #1')).toBeNull();
  });

  it('renders a legend when there is more than one series', () => {
    const config = { data, type: PlotType.ROC };
    render(<ROCCurve configs={[config, config, config]} />);
    expect(screen.getByText('Series #1')).toBeInTheDocument();
    expect(screen.getByText('Series #2')).toBeInTheDocument();
    expect(screen.getByText('Series #3')).toBeInTheDocument();
  });

  it('returns friendly display name', () => {
    expect(ROCCurve.prototype.getDisplayName()).toBe('ROC Curve');
  });

  it('is aggregatable', () => {
    expect(ROCCurve.prototype.isAggregatable()).toBeTruthy();
  });

  it('force legend display even with one config', () => {
    const config = { data, type: PlotType.ROC };
    render(<ROCCurve configs={[config]} forceLegend />);
    screen.getByText('Series #1');
  });

  it('finds nearest point when series x values do not align', () => {
    const sparseSeries = [
      { x: 0.2, y: 0.1, label: 'near-start' },
      { x: 0.8, y: 0.9, label: 'near-end' },
    ];
    expect(findNearestDisplayPoint(sparseSeries, 0.05)?.label).toBe('near-start');
    expect(findNearestDisplayPoint(sparseSeries, 0.95)?.label).toBe('near-end');
  });

  it('returns null for nearest point when no data exists', () => {
    expect(findNearestDisplayPoint([], 0.5)).toBeNull();
  });

  it('returns the only point when nearest-point series has one item', () => {
    const singlePointSeries = [{ x: 0.5, y: 0.5, label: 'only' }];
    expect(findNearestDisplayPoint(singlePointSeries, 0.9)?.label).toBe('only');
  });

  it('uses payload x for tooltip matching when label is not numeric', () => {
    const rocCurve = new ROCCurve({ configs: [] } as any);
    const hoveredX = (rocCurve as any).resolveHoveredX({
      label: 'not-a-number',
      payload: [{ payload: { x: '0.42' } }],
    });
    expect(hoveredX).toBeCloseTo(0.42);
  });

  it('renders tooltip rows using nearest points for each series', () => {
    const rocCurve = new ROCCurve({ configs: [] } as any);
    const labels = ['threshold (Series #1)', 'threshold (Series #2)'];
    const tooltip = (rocCurve as any).renderTooltipContent(
      {
        active: true,
        label: 'invalid',
        payload: [{ payload: { x: '0.39' } }],
      },
      [
        [
          { x: 0.2, y: 0.3, label: 'left' },
          { x: 0.4, y: 0.6, label: 'right' },
        ],
        [{ x: 0.2, y: 0.7, label: 'single' }],
      ],
      labels,
    );
    expect(tooltip).not.toBeNull();
    render(<>{tooltip}</>);
    expect(screen.getByText('threshold (Series #1): right')).toBeInTheDocument();
    expect(screen.getByText('threshold (Series #2): single')).toBeInTheDocument();
  });
});
