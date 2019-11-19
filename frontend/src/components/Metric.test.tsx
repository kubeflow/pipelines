/*
 * Copyright 2019 Google LLC
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
import Metric from './Metric';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { RunMetricFormat } from '../apis/run';

describe('Metric', () => {
  let tree: ShallowWrapper | ReactWrapper;

  const onErrorSpy = jest.fn();

  beforeEach(() => {
    onErrorSpy.mockClear();
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    if (tree) {
      await tree.unmount();
    }
    jest.resetAllMocks();
  });

  it('renders an empty metric when there is no metric', () => {
    tree = shallow(<Metric />);
    expect(tree).toMatchSnapshot();
  });

  it('renders an empty metric when metric has no value', () => {
    tree = shallow(<Metric metric={{}} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders a metric when metric has value and percentage format', () => {
    tree = shallow(<Metric metric={{ format: RunMetricFormat.PERCENTAGE, number_value: 0.54 }} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders an empty metric when metric has no metadata and unspecified format', () => {
    tree = shallow(<Metric metric={{ format: RunMetricFormat.UNSPECIFIED, number_value: 0.54 }} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders an empty metric when metric has no metadata and raw format', () => {
    tree = shallow(<Metric metric={{ format: RunMetricFormat.RAW, number_value: 0.54 }} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders a metric when metric has max and min value of 0', () => {
    tree = shallow(
      <Metric
        metadata={{ name: 'some metric', count: 1, maxValue: 0, minValue: 0 }}
        metric={{ format: RunMetricFormat.RAW, number_value: 0.54 }}
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders a metric and does not log an error when metric is between max and min value', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    tree = shallow(
      <Metric
        metadata={{ name: 'some metric', count: 1, maxValue: 1, minValue: 0 }}
        metric={{ format: RunMetricFormat.RAW, number_value: 0.54 }}
      />,
    );
    expect(consoleSpy).toHaveBeenCalledTimes(0);
    expect(tree).toMatchSnapshot();
  });

  it('renders a metric and logs an error when metric has value less than min value', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    tree = shallow(
      <Metric
        metadata={{ name: 'some metric', count: 1, maxValue: 1, minValue: 0 }}
        metric={{ format: RunMetricFormat.RAW, number_value: -0.54 }}
      />,
    );
    expect(consoleSpy).toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('renders a metric and logs an error when metric has value greater than max value', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    tree = shallow(
      <Metric
        metadata={{ name: 'some metric', count: 1, maxValue: 1, minValue: 0 }}
        metric={{ format: RunMetricFormat.RAW, number_value: 2 }}
      />,
    );
    expect(consoleSpy).toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });
});
