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

import { shallow } from 'enzyme';
import { statusToIcon, NodePhase } from './Status';


describe('Status', () => {
  const startDate = new Date('Wed Jan 9 2019 13:23:22 GMT-0800');
  const endDate = new Date('Fri Jan 11 2019 15:36:01 GMT-0800');

  it('handles an unknown phase', () => {
    const consoleSpy = jest.spyOn(console, 'log').mockImplementation(() => null);
    const tree = shallow(statusToIcon('bad phase' as any));
    expect(tree).toMatchSnapshot();
    expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'bad phase');
  });

  it('displays start and end dates if both are provided', () => {
    const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, startDate, endDate));
    expect(tree).toMatchSnapshot();
  });

  it('does not display a end date if none was provided', () => {
    const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, startDate));
    expect(tree).toMatchSnapshot();
  });

  it('does not display a start date if none was provided', () => {
    const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, undefined, endDate));
    expect(tree).toMatchSnapshot();
  });

  it('does not display any dates if neither was provided', () => {
    const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, /* No dates */));
    expect(tree).toMatchSnapshot();
  });

  Object.keys(NodePhase).map(status => (
    it('renders an icon with tooltip for phase: ' + status, () => {
      const tree = shallow(statusToIcon(NodePhase[status]));
      expect(tree).toMatchSnapshot();
    })
  ));
});
