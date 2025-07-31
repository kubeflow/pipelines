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

import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import * as React from 'react';
import PagedTable from './PagedTable';
import { PlotType } from './Viewer';

describe('PagedTable', () => {
  it('does not break on no config', () => {
    const { asFragment } = render(<PagedTable configs={[]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it.skip('does not break on empty data', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data: [], labels: [], type: PlotType.TABLE }]} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  const data = [
    ['col1', 'col2', 'col3'],
    ['col4', 'col5', 'col6'],
  ];
  const labels = ['field1', 'field2', 'field3'];

  it.skip('renders simple data', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it.skip('renders simple data without labels', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels: [], type: PlotType.TABLE }]} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it.skip('sorts on first column descending', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />,
    );

    const firstColumnSort = screen.getByTestId('table-sort-label-0');
    fireEvent.click(firstColumnSort);

    expect(asFragment()).toMatchSnapshot();
  });

  it.skip('sorts on first column ascending', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />,
    );

    const firstColumnSort = screen.getByTestId('table-sort-label-0');
    // Once for descending
    fireEvent.click(firstColumnSort);
    // Once for ascending
    fireEvent.click(firstColumnSort);

    expect(asFragment()).toMatchSnapshot();
  });

  it('returns a user friendly display name', () => {
    expect(PagedTable.prototype.getDisplayName()).toBe('Table');
  });
});
