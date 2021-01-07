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
import { PlotType } from './Viewer';
import ROCCurve from './ROCCurve';
import { TFunction } from 'i18next';
import { componentMap } from './ViewerContainer';

describe('ROCCurve', () => {
  let t: TFunction = (key: string) => key;
  it('does not break on no config', () => {
    const tree = shallow(<ROCCurve t={t} configs={[]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does not break on empty data', () => {
    const tree = shallow(<ROCCurve t={t} configs={[{ data: [], type: PlotType.ROC }]} />);
    expect(tree).toMatchSnapshot();
  });

  const data = [
    { x: 0, y: 0, label: '1' },
    { x: 0.2, y: 0.3, label: '2' },
    { x: 0.5, y: 0.7, label: '3' },
    { x: 0.9, y: 0.9, label: '4' },
    { x: 1, y: 1, label: '5' },
  ];

  it('renders a simple ROC curve given one config', () => {
    const tree = shallow(<ROCCurve t={t} configs={[{ data, type: PlotType.ROC }]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders a reference base line series', () => {
    const tree = shallow(<ROCCurve t={t} configs={[{ data, type: PlotType.ROC }]} />);
    expect(tree.find('LineSeries').length).toBe(2);
  });

  it('renders an ROC curve using three configs', () => {
    const config = { data, type: PlotType.ROC };
    const tree = shallow(<ROCCurve t={t} configs={[config, config, config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders three lines with three different colors', () => {
    const config = { data, type: PlotType.ROC };
    const tree = shallow(<ROCCurve t={t} configs={[config, config, config]} />);
    expect(tree.find('LineSeries').length).toBe(4); // +1 for baseline
    const [line1Color, line2Color, line3Color] = [
      (tree
        .find('LineSeries')
        .at(1)
        .props() as any).color,
      (tree
        .find('LineSeries')
        .at(2)
        .props() as any).color,
      (tree
        .find('LineSeries')
        .at(3)
        .props() as any).color,
    ];
    expect(line1Color !== line2Color && line1Color !== line3Color && line2Color !== line3Color);
  });

  it('does not render a legend when there is only one config', () => {
    const config = { data, type: PlotType.ROC };
    const tree = shallow(<ROCCurve t={t} configs={[config]} />);
    expect(tree.find('DiscreteColorLegendItem').length).toBe(0);
  });

  it('renders a legend when there is more than one series', () => {
    const config = { data, type: PlotType.ROC };
    const tree = shallow(<ROCCurve t={t} configs={[config, config, config]} />);
    expect(tree.find('DiscreteColorLegendItem').length).toBe(1);
    const legendItems = (tree
      .find('DiscreteColorLegendItem')
      .at(0)
      .props() as any).items;
    expect(legendItems.length).toBe(3);
    legendItems.map((item: any, i: number) => expect(item.title).toBe('Series #' + (i + 1)));
  });

  it('returns friendly display name', () => {
    expect((componentMap[PlotType.ROC].displayNameKey = 'common:rocCurve'));
  });

  it('is aggregatable', () => {
    expect((componentMap[PlotType.ROC].isAggregatable = true));
  });
});
