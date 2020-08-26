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
import CompareTable from './CompareTable';

// tslint:disable-next-line:no-console
const consoleErrorBackup = console.error;
let consoleSpy: jest.Mock;

describe('CompareTable', () => {
  beforeAll(() => {
    consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => null);
  });

  afterAll(() => {
    // tslint:disable-next-line:no-console
    console.error = consoleErrorBackup;
  });

  it('renders no data', () => {
    const tree = shallow(<CompareTable rows={[]} xLabels={[]} yLabels={[]} />);
    expect(tree).toMatchSnapshot();
  });

  const rows = [
    ['1', '2', '3'],
    ['4', '5', '6'],
    ['cell7', 'cell8', 'cell9'],
  ];
  const xLabels = ['col1', 'col2', 'col3'];
  const yLabels = ['row1', 'row2', 'row3'];

  it('logs error if ylabels and rows have different lengths', () => {
    shallow(<CompareTable rows={[rows[0], rows[1]]} xLabels={xLabels} yLabels={yLabels} />);
    expect(consoleSpy).toHaveBeenCalledWith(
      'Number of rows (2) should match the number of Y labels (3).',
    );
  });

  it('renders one row with three columns', () => {
    const tree = shallow(<CompareTable rows={rows} xLabels={xLabels} yLabels={yLabels} />);
    expect(tree).toMatchSnapshot();
  });
});
