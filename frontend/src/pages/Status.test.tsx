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
  it('handles an unknown phase', () => {
    const consoleSpy = jest.spyOn(console, 'log').mockImplementation(() => null);
    const tree = shallow(statusToIcon('bad phase' as any));
    expect(tree).toMatchSnapshot();
    expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'bad phase');
  });

  it('does not display a date if none was provided', () => {
    const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, /* No date */));
    expect(tree).toMatchSnapshot();
  });

  Object.keys(NodePhase).map(status => (
    it('renders an icon with tooltip for phase: ' + status, () => {
      const tree = shallow(statusToIcon(NodePhase[status], new Date('Fri Jan 11 2019 15:36:01 GMT-0800')));
      expect(tree).toMatchSnapshot();
    })
  ));
});
