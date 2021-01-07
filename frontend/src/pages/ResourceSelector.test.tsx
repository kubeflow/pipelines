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
import ResourceSelector, { ResourceSelectorProps, BaseResource } from './ResourceSelector';
import TestUtils from '../TestUtils';
import { ListRequest } from '../lib/Apis';
import { shallow, ReactWrapper, ShallowWrapper } from 'enzyme';
import { Row } from '../components/CustomTable';

let mockedValue = '';
jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate HoC receive the t function as a prop
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: () => mockedValue };
    return Component;
  },
}));

class TestResourceSelector extends ResourceSelector {
  public async _load(request: ListRequest): Promise<string> {
    return super._load(request);
  }

  public _selectionChanged(selectedIds: string[]): void {
    return super._selectionChanged(selectedIds);
  }

  public _resourcesToRow(resources: BaseResource[]): Row[] {
    return super._resourcesToRow(resources);
  }
}

describe('ResourceSelector', () => {
  let tree: ReactWrapper | ShallowWrapper;

  const updateDialogSpy = jest.fn();
  const selectionChangedCbSpy = jest.fn();
  const listResourceSpy = jest.fn();
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
    { label: 'Resource name', flex: 1, sortKey: 'name' },
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
  beforeEach(() => {
    listResourceSpy.mockReset();
    listResourceSpy.mockImplementation(() => ({
      nextPageToken: 'test-next-page-token',
      resources: RESOURCES,
    }));
    updateDialogSpy.mockReset();
    selectionChangedCbSpy.mockReset();
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    await tree.unmount();
  });

  it('displays resource selector', async () => {
    tree = shallow(<TestResourceSelector {...generateProps()} />);
    await (tree.instance() as TestResourceSelector)._load({});

    expect(listResourceSpy).toHaveBeenCalledTimes(1);
    expect(listResourceSpy).toHaveBeenLastCalledWith(undefined, undefined, undefined, undefined);
    expect(tree.state('resources')).toEqual(RESOURCES);
    expect(tree).toMatchSnapshot();
  });

  it('converts resources into a table rows', async () => {
    const props = generateProps();
    const resources: BaseResource[] = [
      {
        created_at: new Date(2018, 1, 2, 3, 4, 5),
        description: 'a description',
        id: 'an-id',
        name: 'a name',
      },
    ];
    listResourceSpy.mockImplementationOnce(() => ({ resources, nextPageToken: '' }));
    props.listApi = listResourceSpy as any;

    tree = shallow(<TestResourceSelector {...props} />);
    await (tree.instance() as TestResourceSelector)._load({});

    expect(tree.state('rows')).toEqual([
      {
        id: 'an-id',
        otherFields: ['a name', 'a description', '2/2/2018, 3:04:05 AM'],
      },
    ]);
  });

  it('shows error dialog if listing fails', async () => {
    mockedValue = 'mockedT-value';
    TestUtils.makeErrorResponseOnce(listResourceSpy, 'woops!');
    jest.spyOn(console, 'error').mockImplementation();

    tree = shallow(<TestResourceSelector {...generateProps()} />);
    await (tree.instance() as TestResourceSelector)._load({});

    expect(listResourceSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'mockedT-value:\nwoops!',
        title: 'mockedT-value',
      }),
    );
    expect(tree.state('resources')).toEqual([]);
  });

  it('calls selection callback when a resource is selected', async () => {
    tree = shallow(<TestResourceSelector {...generateProps()} />);
    await (tree.instance() as TestResourceSelector)._load({});

    expect(tree.state('selectedIds')).toEqual([]);
    (tree.instance() as TestResourceSelector)._selectionChanged([RESOURCES[1].id!]);
    expect(selectionChangedCbSpy).toHaveBeenLastCalledWith(RESOURCES[1]);
    expect(tree.state('selectedIds')).toEqual([RESOURCES[1].id]);
  });

  it('logs error if more than one resource is selected', async () => {
    mockedValue = 'resources were selected somehow';
    tree = shallow(<TestResourceSelector {...generateProps()} />);
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    await (tree.instance() as TestResourceSelector)._load({});

    expect(tree.state('selectedIds')).toEqual([]);

    (tree.instance() as TestResourceSelector)._selectionChanged([
      RESOURCES[0].id!,
      RESOURCES[1].id!,
    ]);

    expect(selectionChangedCbSpy).not.toHaveBeenCalled();
    expect(tree.state('selectedIds')).toEqual([]);
    expect(consoleSpy).toHaveBeenLastCalledWith('2 resources were selected somehow', [
      RESOURCES[0].id,
      RESOURCES[1].id,
    ]);
  });

  it('logs error if selected resource ID is not found in list', async () => {
    mockedValue = 'Somehow no resource was found with ID';
    tree = shallow(<TestResourceSelector {...generateProps()} />);
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    await (tree.instance() as TestResourceSelector)._load({});

    expect(tree.state('selectedIds')).toEqual([]);

    (tree.instance() as TestResourceSelector)._selectionChanged(['id-not-in-list']);

    expect(selectionChangedCbSpy).not.toHaveBeenCalled();
    expect(tree.state('selectedIds')).toEqual([]);
    expect(consoleSpy).toHaveBeenLastCalledWith(
      'Somehow no resource was found with ID: id-not-in-list',
    );
  });
});
