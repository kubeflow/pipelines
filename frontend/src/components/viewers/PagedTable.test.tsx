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
import { fireEvent, render, screen } from '@testing-library/react';
import PagedTable from './PagedTable';
import { PlotType } from './Viewer';
import TestUtils from '../../TestUtils';

const normalizeMuiIds = (fragment: DocumentFragment) => {
  const idMap = new Map<string, string>();
  let nextId = 0;
  fragment.querySelectorAll('[id^="mui-"]').forEach(el => {
    const oldId = el.getAttribute('id');
    if (!oldId) {
      return;
    }
    if (!idMap.has(oldId)) {
      idMap.set(oldId, `mui-id-${nextId++}`);
    }
    el.setAttribute('id', idMap.get(oldId)!);
  });

  const updateAttr = (el: Element, attr: string) => {
    const value = el.getAttribute(attr);
    if (!value) {
      return;
    }
    const parts = value.split(' ');
    let changed = false;
    const updated = parts.map(part => {
      const mapped = idMap.get(part);
      if (mapped) {
        changed = true;
        return mapped;
      }
      return part;
    });
    if (changed) {
      el.setAttribute(attr, updated.join(' '));
    }
  };

  fragment.querySelectorAll('[aria-labelledby]').forEach(el => updateAttr(el, 'aria-labelledby'));
  fragment.querySelectorAll('[for]').forEach(el => updateAttr(el, 'for'));
  fragment.querySelectorAll('[aria-describedby]').forEach(el => updateAttr(el, 'aria-describedby'));
};

const expectStableSnapshot = (fragment: DocumentFragment) => {
  normalizeMuiIds(fragment);
  expect(fragment).toMatchSnapshot();
};

describe('PagedTable', () => {
  it('does not break on no config', () => {
    const { asFragment } = render(<PagedTable configs={[]} />);
    expectStableSnapshot(asFragment());
  });

  it('does not break on empty data', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data: [], labels: [], type: PlotType.TABLE }]} />,
    );
    expectStableSnapshot(asFragment());
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
    expectStableSnapshot(asFragment());
  });

  it('renders simple data without labels', () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels: [], type: PlotType.TABLE }]} />,
    );
    expectStableSnapshot(asFragment());
  });

  it('sorts on first column descending', async () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />,
    );
    fireEvent.click(screen.getByText(labels[0]));
    await TestUtils.flushPromises();
    expectStableSnapshot(asFragment());
  });

  it('sorts on first column ascending', async () => {
    const { asFragment } = render(
      <PagedTable configs={[{ data, labels, type: PlotType.TABLE }]} />,
    );
    // Once for descending.
    fireEvent.click(screen.getByText(labels[0]));
    // Once for ascending.
    fireEvent.click(screen.getByText(labels[0]));
    await TestUtils.flushPromises();
    expectStableSnapshot(asFragment());
  });

  it('returns a user friendly display name', () => {
    expect(PagedTable.prototype.getDisplayName()).toBe('Table');
  });
});
