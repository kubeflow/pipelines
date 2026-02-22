/*
 * Copyright 2019 The Kubeflow Authors
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
import { render } from '@testing-library/react';
import { vi } from 'vitest';
import Metric from './Metric';
import { RunMetricFormat } from '../apis/run';

describe('Metric', () => {
  it('renders an empty metric when there is no metric', () => {
    const { asFragment } = render(<Metric />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders an empty metric when metric has no value', () => {
    const { asFragment } = render(<Metric metric={{}} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a metric when metric has value and percentage format', () => {
    const { asFragment } = render(
      <Metric metric={{ format: RunMetricFormat.PERCENTAGE, number_value: 0.54 }} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders an empty metric when metric has no metadata and unspecified format', () => {
    const { asFragment } = render(
      <Metric metric={{ format: RunMetricFormat.UNSPECIFIED, number_value: 0.54 }} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders an empty metric when metric has no metadata and raw format', () => {
    const { asFragment } = render(
      <Metric metric={{ format: RunMetricFormat.RAW, number_value: 0.54 }} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a metric when metric has max and min value of 0', () => {
    const { asFragment } = render(
      <Metric
        metadata={{ name: 'some metric', count: 1, maxValue: 0, minValue: 0 }}
        metric={{ format: RunMetricFormat.RAW, number_value: 0.54 }}
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a metric and does not log an error when metric is between max and min value', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const { asFragment } = render(
      <Metric
        metadata={{ name: 'some metric', count: 1, maxValue: 1, minValue: 0 }}
        metric={{ format: RunMetricFormat.RAW, number_value: 0.54 }}
      />,
    );
    expect(consoleSpy).not.toHaveBeenCalled();
    expect(asFragment()).toMatchSnapshot();
    consoleSpy.mockRestore();
  });

  it('renders a metric and logs an error when metric has value less than min value', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const { asFragment } = render(
      <Metric
        metadata={{ name: 'some metric', count: 1, maxValue: 1, minValue: 0 }}
        metric={{ format: RunMetricFormat.RAW, number_value: -0.54 }}
      />,
    );
    expect(consoleSpy).toHaveBeenCalled();
    expect(asFragment()).toMatchSnapshot();
    consoleSpy.mockRestore();
  });

  it('renders a metric and logs an error when metric has value greater than max value', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const { asFragment } = render(
      <Metric
        metadata={{ name: 'some metric', count: 1, maxValue: 1, minValue: 0 }}
        metric={{ format: RunMetricFormat.RAW, number_value: 2 }}
      />,
    );
    expect(consoleSpy).toHaveBeenCalled();
    expect(asFragment()).toMatchSnapshot();
    consoleSpy.mockRestore();
  });
});
