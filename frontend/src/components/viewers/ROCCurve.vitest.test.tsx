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
import renderer from 'react-test-renderer';
import { LineSeries } from 'react-vis';
import { PlotType } from './Viewer';
import ROCCurve from './ROCCurve';

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
    const tree = renderer.create(
      <ROCCurve configs={[{ data, type: PlotType.ROC }]} disableAnimation />,
    );
    const lineSeries = tree.root.findAllByType(LineSeries);
    expect(lineSeries.length).toBeGreaterThanOrEqual(2);
  });

  it('renders an ROC curve using three configs', () => {
    const config = { data, type: PlotType.ROC };
    const { asFragment } = render(<ROCCurve configs={[config, config, config]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders three lines with three different colors', () => {
    const config = { data, type: PlotType.ROC };
    const tree = renderer.create(<ROCCurve configs={[config, config, config]} disableAnimation />);
    const lineSeries = tree.root.findAllByType(LineSeries);
    expect(lineSeries.length).toBeGreaterThanOrEqual(4); // +1 for baseline
    const dataLineColors = lineSeries.map(series => series.props.color).filter(Boolean);
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
});
