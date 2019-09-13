/*
 * Copyright 2019 Google LLC
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
import { Column, Row } from './CustomTable';
import TestUtils from '../TestUtils';
import { shallow } from 'enzyme';
import { CustomTableRow } from './CustomTableRow';

describe('CustomTable', () => {
  const props = {
    columns: [],
    row: [],
  };

  const columns: Column[] = [
    {
      customRenderer: undefined,
      label: 'col1',
    },
    {
      customRenderer: undefined,
      label: 'col2',
    },
  ];

  const row: Row = {
    id: 'row',
    otherFields: ['cell1', 'cell2'],
  };

  it('renders some rows using a custom renderer', async () => {
    columns[0].customRenderer = () => (<span>this is custom output</span>) as any;
    const tree = shallow(<CustomTableRow {...props} row={row} columns={columns} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
    columns[0].customRenderer = undefined;
  });

  it('displays warning icon with tooltip if row has error', async () => {
    row.error = 'dummy error';
    const tree = shallow(<CustomTableRow {...props} row={row} columns={columns} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
    row.error = undefined;
  });
});
