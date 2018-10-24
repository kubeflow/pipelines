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
import JobsAndRuns, { JobsAndRunsTab } from './JobsAndRuns';
import { shallow } from 'enzyme';

describe('JobsAndRuns', () => {
  it('renders jobs page', () => {
    const routerProps = { view: JobsAndRunsTab.JOBS, someOtherProps: 'any' };
    expect(shallow(<JobsAndRuns {...routerProps as any} />)).toMatchSnapshot();
  });

  it('renders runs page', () => {
    const routerProps = { view: JobsAndRunsTab.RUNS, someOtherProps: 'any' };
    expect(shallow(<JobsAndRuns {...routerProps as any} />)).toMatchSnapshot();
  });

  it('switches to clicked page by pushing to history', () => {
    const spy = jest.fn();
    const routerProps = { view: JobsAndRunsTab.JOBS, history: { push: spy } };
    const tree = shallow(<JobsAndRuns {...routerProps as any} />);

    tree.find('MD2Tabs').simulate('switch', 1);
    expect(spy).toHaveBeenCalledWith('/runs');

    tree.find('MD2Tabs').simulate('switch', 0);
    expect(spy).toHaveBeenCalledWith('/jobs');
  });
});
