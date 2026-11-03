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

import { fireEvent, render, screen, within } from '@testing-library/react';
import { forceRenderStyles } from 'typestyle';
import { color } from '../../Css';
import PagedTable from './PagedTable';
import { PlotType } from './Viewer';
import { invokeAndFlush } from '../../TestUtils';
import { stableMuiSnapshotFragment } from 'src/testUtils/muiSnapshot';

describe('PagedTable', () => {
  it('does not break on no config', () => {
    const { asFragment } = render(<PagedTable configs={[]} />);
    expect(stableMuiSnapshotFragment(asFragment())).toMatchSnapshot();
  });

  it('does not break on empty data', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data: [], labels: [], type: PlotType.TABLE }]} />,
    );
    expect(stableMuiSnapshotFragment(asFragment())).toMatchSnapshot();
  });

  const data = [
    ['col1', 'col2', 'col3'],
    ['col4', 'col5', 'col6'],
  ];
  const labels = ['field1', 'field2', 'field3'];

  it('renders simple data', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />,
    );
    expect(stableMuiSnapshotFragment(asFragment())).toMatchSnapshot();
  });

  it.each([undefined, 400])(
    'renders a left border on each data row at maxDimension %s',
    (maxDimension) => {
      render(
        <PagedTable
          configs={[
            {
              data: [
                ['accuracy', '0.93'],
                ['rows', '3'],
              ],
              labels: ['name', 'value'],
              type: PlotType.TABLE,
            },
          ]}
          maxDimension={maxDimension}
        />,
      );
      forceRenderStyles();

      for (const name of ['accuracy', 'rows']) {
        const row = screen.getByRole('cell', { name }).closest('tr')!;
        const cells = within(row).getAllByRole('cell');
        expect(cells[0]).toHaveStyle({ borderLeft: `1px solid ${color.divider}` });
        for (const cell of cells) {
          expect(cell).toHaveStyle({ borderRight: `1px solid ${color.divider}` });
        }
        expect(cells[1]).not.toHaveStyle({ borderLeft: `1px solid ${color.divider}` });
      }
      for (const header of screen.getAllByRole('columnheader')) {
        expect(header).not.toHaveStyle({ borderLeft: `1px solid ${color.divider}` });
      }
      expect(screen.getByRole('cell', { name: '' })).not.toHaveStyle({
        borderLeft: `1px solid ${color.divider}`,
      });
    },
  );

  it.each([undefined, 400])(
    'renders a top border only on the first label-free data row at maxDimension %s',
    (maxDimension) => {
      const data = Array.from({ length: 12 }, (_, index) => [
        `metric-${String(index).padStart(2, '0')}`,
        String(index),
      ]);
      const { rerender } = render(
        <PagedTable
          configs={[{ data, labels: [], type: PlotType.TABLE }]}
          maxDimension={maxDimension}
        />,
      );
      forceRenderStyles();

      const expectRowTopBorder = (name: string, bordered: boolean) => {
        const row = screen.getByRole('cell', { name }).closest('tr')!;
        for (const cell of within(row).getAllByRole('cell')) {
          if (bordered) {
            expect(cell).toHaveStyle({ borderTop: `1px solid ${color.divider}` });
          } else {
            expect(cell).not.toHaveStyle({ borderTop: `1px solid ${color.divider}` });
          }
        }
      };

      expect(screen.getByRole('table').querySelector('thead')).toBeNull();
      expectRowTopBorder('metric-00', true);
      expectRowTopBorder('metric-01', false);
      fireEvent.click(screen.getByRole('button', { name: 'Go to next page' }));
      expectRowTopBorder('metric-10', true);
      expectRowTopBorder('metric-11', false);
      expect(screen.getByRole('cell', { name: '' })).not.toHaveStyle({
        borderTop: `1px solid ${color.divider}`,
      });

      rerender(
        <PagedTable
          configs={[{ data, labels: ['name', 'value'], type: PlotType.TABLE }]}
          maxDimension={maxDimension}
        />,
      );
      expectRowTopBorder('metric-10', false);
      for (const header of screen.getAllByRole('columnheader')) {
        expect(header).not.toHaveStyle({ borderTop: `1px solid ${color.divider}` });
      }

      rerender(
        <PagedTable
          configs={[{ data: [], labels: [], type: PlotType.TABLE }]}
          maxDimension={maxDimension}
        />,
      );
      expect(screen.getByRole('cell', { name: '' })).not.toHaveStyle({
        borderTop: `1px solid ${color.divider}`,
      });
    },
  );

  it('renders updated table data when configs change', () => {
    const { rerender } = render(
      <PagedTable configs={[{ data: [['initial']], labels: ['value'], type: PlotType.TABLE }]} />,
    );
    expect(screen.getByText('initial')).toBeVisible();

    rerender(
      <PagedTable configs={[{ data: [['recovered']], labels: ['value'], type: PlotType.TABLE }]} />,
    );

    expect(screen.queryByText('initial')).toBeNull();
    expect(screen.getByText('recovered')).toBeVisible();
  });

  it('clamps the current page when updated table data has fewer pages', () => {
    const initialData = Array.from({ length: 15 }, (_, index) => [`row-${index}`]);
    const { rerender } = render(
      <PagedTable configs={[{ data: initialData, labels: ['value'], type: PlotType.TABLE }]} />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Go to next page' }));
    expect(screen.getByText('11–15 of 15')).toBeVisible();

    rerender(
      <PagedTable configs={[{ data: [['only-row']], labels: ['value'], type: PlotType.TABLE }]} />,
    );

    expect(screen.getByText('only-row')).toBeVisible();
    expect(screen.getByText('1–1 of 1')).toBeVisible();
  });

  it('renders simple data without labels', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels: [], type: PlotType.TABLE }]} />,
    );
    expect(stableMuiSnapshotFragment(asFragment())).toMatchSnapshot();
  });

  it('sorts on first column descending', async () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />,
    );
    await invokeAndFlush(() => {
      fireEvent.click(screen.getByText(labels[0]));
    });
    expect(stableMuiSnapshotFragment(asFragment())).toMatchSnapshot();
  });

  it('sorts on first column ascending', async () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />,
    );
    // Once for descending.
    await invokeAndFlush(() => {
      fireEvent.click(screen.getByText(labels[0]));
    });
    // Once for ascending.
    await invokeAndFlush(() => {
      fireEvent.click(screen.getByText(labels[0]));
    });
    expect(stableMuiSnapshotFragment(asFragment())).toMatchSnapshot();
  });

  it('does not emit DOM nesting warnings when rendering pagination', () => {
    const consoleErrorSpy = vi.spyOn(console, 'error');
    try {
      render(<PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />);

      const hasDomNestingWarning = consoleErrorSpy.mock.calls.some((args) =>
        args.some((arg) => typeof arg === 'string' && arg.includes('validateDOMNesting')),
      );
      expect(hasDomNestingWarning).toBe(false);
    } finally {
      consoleErrorSpy.mockRestore();
    }
  });

  it('returns a user friendly display name', () => {
    expect(PagedTable.prototype.getDisplayName()).toBe('Table');
  });
});
