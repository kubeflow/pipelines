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
import PagedTable from './PagedTable';
import { PlotType } from './Viewer';
import { TFunction } from 'i18next';
import { componentMap } from './ViewerContainer';

describe('PagedTable', () => {
  let t: TFunction = (key: string) => key;
  it('does not break on no config', () => {
    const tree = shallow(<PagedTable configs={[]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does not break on empty data', () => {
    const tree = shallow(<PagedTable configs={[{ data: [], labels: [], type: PlotType.TABLE }]} />);
    expect(tree).toMatchSnapshot();
  });

  const data = [
    ['col1', 'col2', 'col3'],
    ['col4', 'col5', 'col6'],
  ];
  const labels = ['field1', 'field2', 'field3'];

  it('renders simple data', () => {
    const tree = shallow(<PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders simple data without labels', () => {
    const tree = shallow(<PagedTable configs={[{ data, labels: [], type: PlotType.TABLE }]} />);
    expect(tree).toMatchSnapshot();
  });

  it('sorts on first column descending', () => {
    const tree = shallow(<PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />);
    tree
      .find('WithStyles(TableSortLabel)')
      .at(0)
      .simulate('click');
    expect(tree).toMatchSnapshot();
  });

  it('sorts on first column ascending', () => {
    const tree = shallow(<PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />);
    // Once for descending
    tree
      .find('WithStyles(TableSortLabel)')
      .at(0)
      .simulate('click');
    // Once for ascending
    tree
      .find('WithStyles(TableSortLabel)')
      .at(0)
      .simulate('click');
    expect(tree).toMatchSnapshot();
  });

  it('returns a user friendly display name', () => {
    expect((componentMap[PlotType.TABLE].displayNameKey = 'common:table'));
  });
});
