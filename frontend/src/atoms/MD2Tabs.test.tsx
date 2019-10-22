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
import MD2Tabs from './MD2Tabs';
import toJson from 'enzyme-to-json';
import { logger } from '../lib/Utils';
import { shallow, mount } from 'enzyme';

describe('Input', () => {
  const buttonSelector = 'WithStyles(Button)';
  it('renders with the right styles by default', () => {
    const tree = shallow(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={0} />);
    expect(toJson(tree)).toMatchSnapshot();
  });

  it('does not try to call the onSwitch handler if it is not defined', () => {
    const tree = shallow(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={0} />);
    tree
      .find(buttonSelector)
      .at(1)
      .simulate('click');
  });

  it('calls the onSwitch function if an unselected button is clicked', () => {
    const switchHandler = jest.fn();
    const tree = shallow(
      <MD2Tabs tabs={['tab1', 'tab2']} selectedTab={0} onSwitch={switchHandler} />,
    );
    tree
      .find(buttonSelector)
      .at(1)
      .simulate('click');
    expect(switchHandler).toHaveBeenCalled();
  });

  it('does not the onSwitch function if the already selected button is clicked', () => {
    const switchHandler = jest.fn();
    const tree = shallow(
      <MD2Tabs tabs={['tab1', 'tab2']} selectedTab={1} onSwitch={switchHandler} />,
    );
    tree
      .find(buttonSelector)
      .at(1)
      .simulate('click');
    expect(switchHandler).not.toHaveBeenCalled();
  });

  it('gracefully handles an out of bound selectedTab value', () => {
    logger.error = jest.fn();
    const tree = mount(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={100} />);
    (tree.instance() as any)._updateIndicator();
    expect(toJson(tree)).toMatchSnapshot();
  });

  it('recalculates indicator styles when props are updated', () => {
    const spy = jest.fn();
    jest.useFakeTimers();
    jest.spyOn(MD2Tabs.prototype as any, '_updateIndicator').mockImplementationOnce(spy);
    const tree = mount(<MD2Tabs tabs={['tab1', 'tab2']} selectedTab={0} />);
    tree.instance().componentDidUpdate!({}, {});
    jest.runAllTimers();
    expect(spy).toHaveBeenCalled();
  });
});
