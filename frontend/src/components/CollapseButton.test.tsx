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
import CollapseButton from './CollapseButton';
import { shallow } from 'enzyme';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate hook can use it without a warning being shown
  withTranslation: () => (component: React.ComponentClass) => {
    component.defaultProps = { ...component.defaultProps, t: (key: string) => key };
    return component;
  },
}));

describe('CollapseButton', () => {
  const compareComponent = {
    setState: jest.fn(),
    state: {
      collapseSections: {},
    },
  } as any;

  afterEach(() => (compareComponent.state.collapseSections = {}));

  it('initial render', () => {
    const tree = shallow(
      <CollapseButton
        collapseSections={compareComponent.state.collapseSections}
        compareSetState={compareComponent.setState}
        sectionName='testSection'
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('renders the button collapsed if in collapsedSections', () => {
    compareComponent.state.collapseSections.testSection = true;
    const tree = shallow(
      <CollapseButton
        collapseSections={compareComponent.state.collapseSections}
        compareSetState={compareComponent.setState}
        sectionName='testSection'
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('collapses given section when clicked', () => {
    const tree = shallow(
      <CollapseButton
        collapseSections={compareComponent.state.collapseSections}
        compareSetState={compareComponent.setState}
        sectionName='testSection'
      />,
    );
    tree.find('WithStyles(Button)').simulate('click');
    expect(compareComponent.setState).toHaveBeenCalledWith({
      collapseSections: { testSection: true },
    });
  });

  it('expands given section when clicked if it is collapsed', () => {
    compareComponent.state.collapseSections.testSection = true;
    const tree = shallow(
      <CollapseButton
        collapseSections={compareComponent.state.collapseSections}
        compareSetState={compareComponent.setState}
        sectionName='testSection'
      />,
    );
    tree.find('WithStyles(Button)').simulate('click');
    expect(compareComponent.setState).toHaveBeenCalledWith({
      collapseSections: { testSection: false },
    });
  });
});
