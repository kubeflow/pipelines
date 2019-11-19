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
import ViewerContainer from './ViewerContainer';
import { PlotType } from './Viewer';

describe('ViewerContainer', () => {
  it('does not break on empty configs', () => {
    const tree = shallow(<ViewerContainer configs={[]} />);
    expect(tree).toMatchSnapshot();
  });

  Object.keys(PlotType).map(type =>
    it('renders a viewer of type ' + type, () => {
      const tree = shallow(<ViewerContainer configs={[{ type: PlotType[type] }]} />);
      expect(tree).toMatchSnapshot();
    }),
  );
});
