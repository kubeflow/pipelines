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
import { shallow } from 'enzyme';
import ConfusionMatrix, { ConfusionMatrixConfig } from './ConfusionMatrix';
import { PlotType } from './Viewer';
import { TFunction } from 'i18next';

describe('ConfusionMatrix', () => {
  let t: TFunction = (key: string) => key;
  it('does not break on empty data', () => {
    const tree = shallow(<ConfusionMatrix configs={[]} />);
    expect(tree).toMatchSnapshot();
  });

  const data = [
    [0, 1, 2],
    [3, 4, 5],
    [6, 7, 8],
  ];
  const config: ConfusionMatrixConfig = {
    axes: ['test x axis', 'test y axis'],
    data,
    labels: ['label1', 'label2'],
    type: PlotType.CONFUSION_MATRIX,
  };
  it('renders a basic confusion matrix', () => {
    const tree = shallow(<ConfusionMatrix configs={[config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does not break on asymetric data', () => {
    const testConfig = { ...config };
    testConfig.data = data.slice(1);
    const tree = shallow(<ConfusionMatrix configs={[testConfig]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders only one of the given list of configs', () => {
    const tree = shallow(<ConfusionMatrix configs={[config, config, config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders a small confusion matrix snapshot, with no labels or footer', () => {
    const tree = shallow(<ConfusionMatrix configs={[config]} maxDimension={100} />);
    expect(tree).toMatchSnapshot();
  });

  it('activates row/column on cell hover', () => {
    const tree = shallow(<ConfusionMatrix configs={[config]} />);
    tree
      .find('td')
      .at(2)
      .simulate('mouseOver');
    expect(tree.state()).toHaveProperty('activeCell', [0, 0]);
  });

  it('returns a user friendly display name', () => {
    expect(ConfusionMatrix.prototype.getDisplayName(t)).toBe('common:confusionMatrix');
  });
});
