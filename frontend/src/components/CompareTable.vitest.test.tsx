/*
 * Copyright 2018 The Kubeflow Authors
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
import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import CompareTable from './CompareTable';
import { logger } from 'src/lib/Utils';

let loggerSpy: ReturnType<typeof vi.spyOn>;

const rows = [
  ['1', '2', '3'],
  ['4', '5', '6'],
  ['cell7', 'cell8', 'cell9'],
];
const xLabels = ['col1', 'col2', 'col3'];
const yLabels = ['row1', 'row2', 'row3'];

describe('CompareTable', () => {
  beforeEach(() => {
    loggerSpy = vi.spyOn(logger, 'error').mockImplementation(() => undefined);
  });

  afterEach(() => {
    loggerSpy.mockRestore();
  });

  it('renders no data', () => {
    const { asFragment } = render(<CompareTable rows={[]} xLabels={[]} yLabels={[]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('logs error if ylabels and rows have different lengths', () => {
    render(<CompareTable rows={[rows[0], rows[1]]} xLabels={xLabels} yLabels={yLabels} />);
    expect(loggerSpy).toHaveBeenCalledWith(
      'Number of rows (2) should match the number of Y labels (3).',
    );
  });

  it('renders one row with three columns', () => {
    const { asFragment } = render(<CompareTable rows={rows} xLabels={xLabels} yLabels={yLabels} />);
    expect(asFragment()).toMatchSnapshot();
  });
});

describe('CompareTable xParentLabels', () => {
  beforeEach(() => {
    loggerSpy = vi.spyOn(logger, 'error').mockImplementation(() => undefined);
  });

  afterEach(() => {
    loggerSpy.mockRestore();
  });
  const xParentLabels = [
    { colSpan: 2, label: 'parent1' },
    { colSpan: 1, label: 'parent2' },
  ];

  it('Hide x-axis parent labels if xlabels and xparentlabels have different lengths', () => {
    render(
      <CompareTable
        rows={rows}
        xLabels={xLabels}
        yLabels={yLabels}
        xParentLabels={[xParentLabels[0]]}
      />,
    );

    expect(loggerSpy).toHaveBeenCalledWith(
      'Number of columns with data (3) should match the aggregated length of parent columns (2).',
    );
    expect(screen.queryByText(xParentLabels[0].label)).toBeNull();
  });

  it('X-axis parent labels present if xlabels and xparentlabels have appropriate lengths', () => {
    render(
      <CompareTable
        rows={rows}
        xLabels={xLabels}
        yLabels={yLabels}
        xParentLabels={xParentLabels}
      />,
    );

    screen.getByText(xParentLabels[0].label);
    screen.getByText(xParentLabels[1].label);
  });
});
