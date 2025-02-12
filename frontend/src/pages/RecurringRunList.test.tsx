/*
 * Copyright 2021 Arrikto Inc.
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
import * as Utils from 'src/lib/Utils';
import RecurringRunList, { RecurringRunListProps } from './RecurringRunList';
import TestUtils from 'src/TestUtils';
import produce from 'immer';
import { Apis, JobSortKeys, ListRequest } from 'src/lib/Apis';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { range } from 'lodash';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';

class RecurringRunListTest extends RecurringRunList {
  public _loadRecurringRuns(request: ListRequest): Promise<string> {
    return super._loadRecurringRuns(request);
  }
}

describe('RecurringRunList', () => {
  let tree: ShallowWrapper | ReactWrapper;

  const onErrorSpy = jest.fn();
  const listRecurringRunsSpy = jest.spyOn(Apis.recurringRunServiceApi, 'listRecurringRuns');
  const getRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun');
  const listExperimentsSpy = jest.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test environments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');

  function generateProps(): RecurringRunListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
      refreshCount: 1,
    };
  }

  function mockNRecurringRuns(n: number, recurringRunTemplate: Partial<V2beta1RecurringRun>): void {
    getRecurringRunSpy.mockImplementation(id =>
      Promise.resolve(
        produce(recurringRunTemplate, draft => {
          draft.recurring_run_id = id;
          draft.display_name = 'recurring run with id: ' + id;
        }),
      ),
    );

    listRecurringRunsSpy.mockImplementation(() =>
      Promise.resolve({
        recurringRuns: range(1, n + 1).map(i => {
          if (recurringRunTemplate) {
            return produce(recurringRunTemplate as Partial<V2beta1RecurringRun>, draft => {
              draft.recurring_run_id = 'testrecurringrun' + i;
              draft.display_name = 'recurring run with id: testrecurringrun' + i;
            });
          }
          return {
            recurring_run_id: 'testrecurringrun' + i,
            display_name: 'recurring run with id: testrecurringrun' + i,
          } as V2beta1RecurringRun;
        }),
      }),
    );

    listExperimentsSpy.mockImplementation(() => ({ display_name: 'some experiment' }));
  }

  function getMountedInstance(): RecurringRunList {
    tree = TestUtils.mountWithRouter(<RecurringRunList {...generateProps()} />);
    return tree.instance() as RecurringRunList;
  }

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
    onErrorSpy.mockClear();
    listRecurringRunsSpy.mockClear();
    getRecurringRunSpy.mockClear();
    listExperimentsSpy.mockClear();
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    if (tree) {
      await tree.unmount();
    }
    jest.resetAllMocks();
  });

  it('renders the empty experience', () => {
    expect(shallow(<RecurringRunList {...generateProps()} />)).toMatchInlineSnapshot(`
      <div>
        <CustomTable
          columns={
            Array [
              Object {
                "customRenderer": [Function],
                "flex": 1.5,
                "label": "Recurring Run Name",
                "sortKey": "name",
              },
              Object {
                "customRenderer": [Function],
                "flex": 0.5,
                "label": "Status",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Trigger",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Experiment",
              },
              Object {
                "flex": 1,
                "label": "Created at",
                "sortKey": "created_at",
              },
            ]
          }
          emptyMessage="No available recurring runs found."
          filterLabel="Filter recurring runs"
          initialSortColumn="created_at"
          reload={[Function]}
          rows={Array []}
        />
      </div>
    `);
  });

  it('loads one recurring run', async () => {
    mockNRecurringRuns(1, {});
    const props = generateProps();
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(Apis.recurringRunServiceApi.listRecurringRuns).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchInlineSnapshot(`
      <div>
        <CustomTable
          columns={
            Array [
              Object {
                "customRenderer": [Function],
                "flex": 1.5,
                "label": "Recurring Run Name",
                "sortKey": "name",
              },
              Object {
                "customRenderer": [Function],
                "flex": 0.5,
                "label": "Status",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Trigger",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Experiment",
              },
              Object {
                "flex": 1,
                "label": "Created at",
                "sortKey": "created_at",
              },
            ]
          }
          emptyMessage="No available recurring runs found."
          filterLabel="Filter recurring runs"
          initialSortColumn="created_at"
          reload={[Function]}
          rows={
            Array [
              Object {
                "error": undefined,
                "id": "testrecurringrun1",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun1",
                  undefined,
                  undefined,
                  undefined,
                  "-",
                ],
              },
            ]
          }
        />
      </div>
    `);
  });

  it('reloads the recurring run when refresh is called', async () => {
    mockNRecurringRuns(0, {});
    const props = generateProps();
    tree = TestUtils.mountWithRouter(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunList).refresh();
    tree.update();
    expect(Apis.recurringRunServiceApi.listRecurringRuns).toHaveBeenCalledTimes(2);
    expect(Apis.recurringRunServiceApi.listRecurringRuns).toHaveBeenLastCalledWith(
      '',
      10,
      JobSortKeys.CREATED_AT + ' desc',
      undefined,
      '',
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
  });

  it('loads multiple recurring runs', async () => {
    mockNRecurringRuns(5, {});
    const props = generateProps();
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchInlineSnapshot(`
      <div>
        <CustomTable
          columns={
            Array [
              Object {
                "customRenderer": [Function],
                "flex": 1.5,
                "label": "Recurring Run Name",
                "sortKey": "name",
              },
              Object {
                "customRenderer": [Function],
                "flex": 0.5,
                "label": "Status",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Trigger",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Experiment",
              },
              Object {
                "flex": 1,
                "label": "Created at",
                "sortKey": "created_at",
              },
            ]
          }
          emptyMessage="No available recurring runs found."
          filterLabel="Filter recurring runs"
          initialSortColumn="created_at"
          reload={[Function]}
          rows={
            Array [
              Object {
                "error": undefined,
                "id": "testrecurringrun1",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun1",
                  undefined,
                  undefined,
                  undefined,
                  "-",
                ],
              },
              Object {
                "error": undefined,
                "id": "testrecurringrun2",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun2",
                  undefined,
                  undefined,
                  undefined,
                  "-",
                ],
              },
              Object {
                "error": undefined,
                "id": "testrecurringrun3",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun3",
                  undefined,
                  undefined,
                  undefined,
                  "-",
                ],
              },
              Object {
                "error": undefined,
                "id": "testrecurringrun4",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun4",
                  undefined,
                  undefined,
                  undefined,
                  "-",
                ],
              },
              Object {
                "error": undefined,
                "id": "testrecurringrun5",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun5",
                  undefined,
                  undefined,
                  undefined,
                  "-",
                ],
              },
            ]
          }
        />
      </div>
    `);
  });

  it('calls error callback when loading recurring runs fails', async () => {
    TestUtils.makeErrorResponseOnce(
      jest.spyOn(Apis.recurringRunServiceApi, 'listRecurringRuns'),
      'bad stuff happened',
    );
    const props = generateProps();
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).toHaveBeenLastCalledWith(
      'Error: failed to fetch recurring runs.',
      new Error('bad stuff happened'),
    );
  });

  it('loads recurring runs for a given experiment id', async () => {
    mockNRecurringRuns(1, {});
    const props = generateProps();
    props.experimentIdMask = 'experiment1';
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.recurringRunServiceApi.listRecurringRuns).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      'experiment1',
    );
  });

  it('loads recurring runs for a given namespace', async () => {
    mockNRecurringRuns(1, {});
    const props = generateProps();
    props.namespaceMask = 'namespace1';
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.recurringRunServiceApi.listRecurringRuns).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      'namespace1',
      undefined,
      undefined,
    );
  });

  it('loads given list of recurring runs only', async () => {
    mockNRecurringRuns(5, {});
    const props = generateProps();
    props.recurringRunIdListMask = ['recurring run1', 'recurring run2'];
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.recurringRunServiceApi.listRecurringRuns).not.toHaveBeenCalled();
    expect(Apis.recurringRunServiceApi.getRecurringRun).toHaveBeenCalledTimes(2);
    expect(Apis.recurringRunServiceApi.getRecurringRun).toHaveBeenCalledWith('recurring run1');
    expect(Apis.recurringRunServiceApi.getRecurringRun).toHaveBeenCalledWith('recurring run2');
  });

  it('shows recurring run status', async () => {
    mockNRecurringRuns(1, {
      status: V2beta1RecurringRunStatus.ENABLED,
    });
    const props = generateProps();
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchInlineSnapshot(`
      <div>
        <CustomTable
          columns={
            Array [
              Object {
                "customRenderer": [Function],
                "flex": 1.5,
                "label": "Recurring Run Name",
                "sortKey": "name",
              },
              Object {
                "customRenderer": [Function],
                "flex": 0.5,
                "label": "Status",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Trigger",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Experiment",
              },
              Object {
                "flex": 1,
                "label": "Created at",
                "sortKey": "created_at",
              },
            ]
          }
          emptyMessage="No available recurring runs found."
          filterLabel="Filter recurring runs"
          initialSortColumn="created_at"
          reload={[Function]}
          rows={
            Array [
              Object {
                "error": undefined,
                "id": "testrecurringrun1",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun1",
                  "ENABLED",
                  undefined,
                  undefined,
                  "-",
                ],
              },
            ]
          }
        />
      </div>
    `);
  });

  it('shows trigger periodic', async () => {
    mockNRecurringRuns(1, {
      trigger: { periodic_schedule: { interval_second: '3600' } },
    });
    const props = generateProps();
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchInlineSnapshot(`
      <div>
        <CustomTable
          columns={
            Array [
              Object {
                "customRenderer": [Function],
                "flex": 1.5,
                "label": "Recurring Run Name",
                "sortKey": "name",
              },
              Object {
                "customRenderer": [Function],
                "flex": 0.5,
                "label": "Status",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Trigger",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Experiment",
              },
              Object {
                "flex": 1,
                "label": "Created at",
                "sortKey": "created_at",
              },
            ]
          }
          emptyMessage="No available recurring runs found."
          filterLabel="Filter recurring runs"
          initialSortColumn="created_at"
          reload={[Function]}
          rows={
            Array [
              Object {
                "error": undefined,
                "id": "testrecurringrun1",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun1",
                  undefined,
                  Object {
                    "periodic_schedule": Object {
                      "interval_second": "3600",
                    },
                  },
                  undefined,
                  "-",
                ],
              },
            ]
          }
        />
      </div>
    `);
  });

  it('shows trigger cron', async () => {
    mockNRecurringRuns(1, {
      trigger: { cron_schedule: { cron: '0 * * * * ?' } },
    });
    const props = generateProps();
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchInlineSnapshot(`
      <div>
        <CustomTable
          columns={
            Array [
              Object {
                "customRenderer": [Function],
                "flex": 1.5,
                "label": "Recurring Run Name",
                "sortKey": "name",
              },
              Object {
                "customRenderer": [Function],
                "flex": 0.5,
                "label": "Status",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Trigger",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Experiment",
              },
              Object {
                "flex": 1,
                "label": "Created at",
                "sortKey": "created_at",
              },
            ]
          }
          emptyMessage="No available recurring runs found."
          filterLabel="Filter recurring runs"
          initialSortColumn="created_at"
          reload={[Function]}
          rows={
            Array [
              Object {
                "error": undefined,
                "id": "testrecurringrun1",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun1",
                  undefined,
                  Object {
                    "cron_schedule": Object {
                      "cron": "0 * * * * ?",
                    },
                  },
                  undefined,
                  "-",
                ],
              },
            ]
          }
        />
      </div>
    `);
  });

  it('shows experiment name', async () => {
    mockNRecurringRuns(1, {
      experiment_id: 'test-experiment-id',
    });
    listExperimentsSpy.mockImplementationOnce(() => ({
      experiments: [
        {
          experiment_id: 'test-experiment-id',
          display_name: 'test experiment',
        },
      ],
    }));
    const props = generateProps();
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchInlineSnapshot(`
      <div>
        <CustomTable
          columns={
            Array [
              Object {
                "customRenderer": [Function],
                "flex": 1.5,
                "label": "Recurring Run Name",
                "sortKey": "name",
              },
              Object {
                "customRenderer": [Function],
                "flex": 0.5,
                "label": "Status",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Trigger",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Experiment",
              },
              Object {
                "flex": 1,
                "label": "Created at",
                "sortKey": "created_at",
              },
            ]
          }
          emptyMessage="No available recurring runs found."
          filterLabel="Filter recurring runs"
          initialSortColumn="created_at"
          reload={[Function]}
          rows={
            Array [
              Object {
                "error": undefined,
                "id": "testrecurringrun1",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun1",
                  undefined,
                  undefined,
                  Object {
                    "displayName": "test experiment",
                    "id": "test-experiment-id",
                  },
                  "-",
                ],
              },
            ]
          }
        />
      </div>
    `);
  });

  it('hides experiment name if instructed', async () => {
    mockNRecurringRuns(1, {
      experiment_id: 'test-experiment-id',
    });
    listExperimentsSpy.mockImplementationOnce(() => ({ display_name: 'test experiment' }));
    const props = generateProps();
    props.hideExperimentColumn = true;
    tree = shallow(<RecurringRunList {...props} />);
    await (tree.instance() as RecurringRunListTest)._loadRecurringRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchInlineSnapshot(`
      <div>
        <CustomTable
          columns={
            Array [
              Object {
                "customRenderer": [Function],
                "flex": 1.5,
                "label": "Recurring Run Name",
                "sortKey": "name",
              },
              Object {
                "customRenderer": [Function],
                "flex": 0.5,
                "label": "Status",
              },
              Object {
                "customRenderer": [Function],
                "flex": 1,
                "label": "Trigger",
              },
              Object {
                "flex": 1,
                "label": "Created at",
                "sortKey": "created_at",
              },
            ]
          }
          emptyMessage="No available recurring runs found."
          filterLabel="Filter recurring runs"
          initialSortColumn="created_at"
          reload={[Function]}
          rows={
            Array [
              Object {
                "error": undefined,
                "id": "testrecurringrun1",
                "otherFields": Array [
                  "recurring run with id: testrecurringrun1",
                  undefined,
                  undefined,
                  "-",
                ],
              },
            ]
          }
        />
      </div>
    `);
  });

  it('renders recurring run trigger in seconds', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '42' } },
        id: 'recurring run-id',
      }),
    ).toMatchInlineSnapshot(`
      <div>
        Every 
        42
         seconds
      </div>
    `);
  });

  it('renders recurring run trigger in minutes', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '120' } },
        id: 'recurring run-id',
      }),
    ).toMatchInlineSnapshot(`
      <div>
        Every 
        2
         minutes
      </div>
    `);
  });

  it('renders recurring run trigger in hours', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '7200' } },
        id: 'recurring run-id',
      }),
    ).toMatchInlineSnapshot(`
      <div>
        Every 
        2
         hours
      </div>
    `);
  });

  it('renders recurring run trigger in days', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '86400' } },
        id: 'recurring run-id',
      }),
    ).toMatchInlineSnapshot(`
      <div>
        Every 
        1
         days
      </div>
    `);
  });

  it('renders recurring run trigger as cron', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { cron_schedule: { cron: '0 * * * * ?' } },
        id: 'recurring run-id',
      }),
    ).toMatchInlineSnapshot(`
      <div>
        Cron: 
        0 * * * * ?
      </div>
    `);
  });

  it('renders status enabled', () => {
    expect(
      getMountedInstance()._statusCustomRenderer({
        value: 'Enabled',
        id: 'recurring run-id',
      }),
    ).toMatchInlineSnapshot(`
      <div
        style={
          Object {
            "color": "#d50000",
          }
        }
      >
        Enabled
      </div>
    `);
  });

  it('renders status disabled', () => {
    expect(
      getMountedInstance()._statusCustomRenderer({
        value: 'Disabled',
        id: 'recurring run-id',
      }),
    ).toMatchInlineSnapshot(`
      <div
        style={
          Object {
            "color": "#d50000",
          }
        }
      >
        Disabled
      </div>
    `);
  });

  it('renders status unknown', () => {
    expect(
      getMountedInstance()._statusCustomRenderer({
        value: 'Unknown Status',
        id: 'recurring run-id',
      }),
    ).toMatchInlineSnapshot(`
      <div
        style={
          Object {
            "color": "#d50000",
          }
        }
      >
        Unknown Status
      </div>
    `);
  });
});
