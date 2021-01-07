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
import PlotCard from './PlotCard';
import { ViewerConfig, PlotType } from './viewers/Viewer';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate hook can use it without a warning being shown
  withTranslation: () => (component: React.ComponentClass) => {
    component.defaultProps = { ...component.defaultProps, t: (key: string) => key };
    return component;
  },
}));

describe('PlotCard', () => {
  it('handles no configs', () => {
    expect(shallow(<PlotCard title='' configs={[]} maxDimension={100} />)).toMatchSnapshot();
  });

  const config: ViewerConfig = { type: PlotType.CONFUSION_MATRIX };

  it('renders on confusion matrix viewer card', () => {
    const tree = shallow(<PlotCard title='test title' configs={[config]} maxDimension={100} />);
    expect(tree).toMatchSnapshot();
  });

  it('pops out a full screen view of the viewer', () => {
    const tree = shallow(<PlotCard title='' configs={[config]} maxDimension={100} />);
    tree.find('.popOutButton').simulate('click');
    expect(tree).toMatchSnapshot();
  });

  it('close button closes full screen dialog', () => {
    const tree = shallow(<PlotCard title='' configs={[config]} maxDimension={100} />);
    tree.find('.popOutButton').simulate('click');
    tree.find('.fullscreenCloseButton').simulate('click');
    expect(tree).toMatchSnapshot();
  });

  it('clicking outside full screen dialog closes it', () => {
    const tree = shallow(<PlotCard title='' configs={[config]} maxDimension={100} />);
    tree.find('.popOutButton').simulate('click');
    tree.find('WithStyles(Dialog)').simulate('close');
    expect(tree).toMatchSnapshot();
  });
});
