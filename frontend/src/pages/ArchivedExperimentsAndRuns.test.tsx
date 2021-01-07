/*
 * Copyright 2020 Google LLC
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
import ArchivedExperimentsAndRuns, {
  ArchivedExperimentAndRunsProps,
  ArchivedExperimentsAndRunsTab,
} from './ArchivedExperimentsAndRuns';
import { shallow } from 'enzyme';
import { TFunction } from 'i18next';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate HoC receive the t function as a prop
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: () => '' };
    return Component;
  },
}));
let t: TFunction = (key: string) => key;
function generateProps(): ArchivedExperimentAndRunsProps {
  return {
    history: {} as any,
    location: '' as any,
    match: '' as any,
    toolbarProps: {} as any,
    updateBanner: () => null,
    updateDialog: jest.fn(),
    updateSnackbar: jest.fn(),
    updateToolbar: () => null,
    view: ArchivedExperimentsAndRunsTab.RUNS,
    t,
  };
}

describe('ArchivedExperimentsAndRuns', () => {
  it('renders archived runs page', () => {
    expect(shallow(<ArchivedExperimentsAndRuns {...(generateProps() as any)} />)).toMatchSnapshot();
  });

  it('renders archived experiments page', () => {
    const props = generateProps();
    props.view = ArchivedExperimentsAndRunsTab.EXPERIMENTS;
    expect(shallow(<ArchivedExperimentsAndRuns {...(props as any)} />)).toMatchSnapshot();
  });

  it('switches to clicked page by pushing to history', () => {
    const spy = jest.fn();
    const props = generateProps();
    props.history.push = spy;
    const tree = shallow(<ArchivedExperimentsAndRuns {...(props as any)} />);

    tree.find('MD2Tabs').simulate('switch', ArchivedExperimentsAndRunsTab.RUNS);
    expect(spy).toHaveBeenCalledWith('/archive/runs');

    tree.find('MD2Tabs').simulate('switch', ArchivedExperimentsAndRunsTab.EXPERIMENTS);
    expect(spy).toHaveBeenCalledWith('/archive/experiments');
  });
});
