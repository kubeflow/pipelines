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
import { render, screen, waitFor } from '@testing-library/react';
import { range } from 'lodash';
import ResourceSelector, { BaseResource, ResourceSelectorProps } from './ResourceSelector';
import TestUtils from '../TestUtils';
import { CommonTestWrapper } from '../TestWrapper';
import { NameWithTooltip } from '../components/CustomTableNameColumn';
import { logger } from '../lib/Utils';
import { vi } from 'vitest';

describe('ResourceSelector', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let resourceSelectorRef: React.RefObject<ResourceSelector> | null = null;

  const updateDialogSpy = vi.fn();
  const selectionChangedCbSpy = vi.fn();
  const listResourceSpy = vi.fn();
  const RESOURCES: BaseResource[] = [
    {
      created_at: new Date(2018, 1, 2, 3, 4, 5),
      description: 'test-1 description',
      id: 'some-id-1',
      name: 'test-1 name',
    },
    {
      created_at: new Date(2018, 10, 9, 8, 7, 6),
      description: 'test-2 description',
      id: 'some-2-id',
      name: 'test-2 name',
    },
  ];

  const selectorColumns = [
    { label: 'Resource name', flex: 1, sortKey: 'name', customRenderer: NameWithTooltip },
    { label: 'Description', flex: 1.5 },
    { label: 'Uploaded on', flex: 1, sortKey: 'created_at' },
  ];

  const testEmptyMessage = 'Test - Sorry, no resources.';
  const testTitle = 'A test selector';

  function generateProps(): ResourceSelectorProps {
    return {
      columns: selectorColumns,
      emptyMessage: testEmptyMessage,
      filterLabel: 'test filter label',
      history: {} as any,
      initialSortColumn: 'created_at',
      listApi: listResourceSpy as any,
      location: '' as any,
      match: {} as any,
      selectionChanged: selectionChangedCbSpy,
      title: testTitle,
      updateDialog: updateDialogSpy,
    };
  }

  function getResourceSelectorState(): ResourceSelector['state'] | undefined {
    return resourceSelectorRef?.current?.state;
  }

  function getRowById(id: string): HTMLElement {
    const rows = screen.getAllByTestId('table-row');
    const row = rows.find(element => element.getAttribute('data-row-id') === id);
    if (!row) {
      throw new Error(`Row not found: ${id}`);
    }
    return row;
  }

  async function renderResourceSelector(
    customProps?: Partial<ResourceSelectorProps>,
  ): Promise<void> {
    resourceSelectorRef = React.createRef<ResourceSelector>();
    const props = { ...generateProps(), ...customProps } as ResourceSelectorProps;
    renderResult = render(
      <CommonTestWrapper>
        <ResourceSelector ref={resourceSelectorRef} {...props} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(listResourceSpy).toHaveBeenCalled();
    });
    await TestUtils.flushPromises();
  }

  beforeEach(() => {
    listResourceSpy.mockReset();
    listResourceSpy.mockImplementation(() => ({
      nextPageToken: 'test-next-page-token',
      resources: RESOURCES,
    }));
    updateDialogSpy.mockReset();
    selectionChangedCbSpy.mockReset();
  });

  afterEach(() => {
    if (renderResult) {
      renderResult.unmount();
      renderResult = null;
    }
    resourceSelectorRef = null;
    vi.restoreAllMocks();
  });

  it('displays resource selector', async () => {
    await renderResourceSelector();

    expect(listResourceSpy).toHaveBeenCalledTimes(1);
    expect(listResourceSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc', '');
    expect(getResourceSelectorState()).toHaveProperty('resources', RESOURCES);
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('converts resources into a table rows', async () => {
    const resources: BaseResource[] = [
      {
        created_at: new Date(2018, 1, 2, 3, 4, 5),
        description: 'a description',
        id: 'an-id',
        name: 'a name',
      },
    ];
    listResourceSpy.mockImplementationOnce(() => ({ resources, nextPageToken: '' }));

    await renderResourceSelector();

    expect(getResourceSelectorState()).toHaveProperty('rows', [
      {
        id: 'an-id',
        otherFields: [
          { display_name: undefined, name: 'a name' },
          'a description',
          '2/2/2018, 3:04:05 AM',
        ],
      },
    ]);
  });

  it('shows error dialog if listing fails', async () => {
    TestUtils.makeErrorResponseOnce(listResourceSpy as any, 'woops!');
    vi.spyOn(console, 'error').mockImplementation(() => null);

    await renderResourceSelector();

    expect(listResourceSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'List request failed with:\nwoops!',
        title: 'Error retrieving resources',
      }),
    );
    expect(getResourceSelectorState()).toHaveProperty('resources', []);
  });

  it('calls selection callback when a resource is selected', async () => {
    await renderResourceSelector();

    expect(getResourceSelectorState()).toHaveProperty('selectedIds', []);

    const row = getRowById(RESOURCES[1].id!);
    row.click();

    await waitFor(() => {
      expect(selectionChangedCbSpy).toHaveBeenLastCalledWith(RESOURCES[1].id!);
      expect(getResourceSelectorState()).toHaveProperty('selectedIds', [RESOURCES[1].id]);
    });
  });

  it('logs error if more than one resource is selected', async () => {
    await renderResourceSelector();
    const loggerSpy = vi.spyOn(logger, 'error').mockImplementation(() => null);

    expect(getResourceSelectorState()).toHaveProperty('selectedIds', []);

    (resourceSelectorRef?.current as any)._selectionChanged([RESOURCES[0].id!, RESOURCES[1].id!]);

    expect(selectionChangedCbSpy).not.toHaveBeenCalled();
    expect(getResourceSelectorState()).toHaveProperty('selectedIds', []);
    expect(loggerSpy).toHaveBeenLastCalledWith('2 resources were selected somehow', [
      RESOURCES[0].id,
      RESOURCES[1].id,
    ]);
  });
});
