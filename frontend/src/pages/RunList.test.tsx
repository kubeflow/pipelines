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
import * as Utils from 'src/lib/Utils';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import RunList, { RunListProps } from './RunList';
import TestUtils from 'src/TestUtils';
import produce from 'immer';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import { V2beta1Run, V2beta1RunStorageState, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { Apis, RunSortKeys, ListRequest } from 'src/lib/Apis';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { range } from 'lodash';
import { CommonTestWrapper } from 'src/TestWrapper';

class RunListTest extends RunList {
  public _loadRuns(request: ListRequest): Promise<string> {
    return super._loadRuns(request);
  }
}

describe('RunList', () => {
  let tree: ShallowWrapper | ReactWrapper;

  const onErrorSpy = jest.fn();
  const listRunsSpy = jest.spyOn(Apis.runServiceApiV2, 'listRuns');
  const getRunSpy = jest.spyOn(Apis.runServiceApiV2, 'getRun');
  const getPipelineVersionSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
  const listExperimentsSpy = jest.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');

  function generateProps(): RunListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
    };
  }

  function mockNRuns(n: number, runTemplate: Partial<V2beta1Run>): void {
    getRunSpy.mockImplementation(id => {
      let pipelineVersionRef = {
        pipeline_id: 'testpipeline' + id,
        pipeline_version_id: 'testversion' + id,
      };
      return Promise.resolve(
        produce(runTemplate, draft => {
          draft = draft || {};
          draft.run_id = id;
          draft.display_name = 'run with id: ' + id;
          draft.pipeline_version_reference = pipelineVersionRef;
        }),
      );
    });

    listRunsSpy.mockImplementation(() =>
      Promise.resolve({
        runs: range(1, n + 1).map(i => {
          if (runTemplate) {
            let pipelineVersionRef = {
              pipeline_id: 'testpipeline' + i,
              pipeline_version_id: 'testversion' + i,
            };
            return produce(runTemplate as Partial<V2beta1Run>, draft => {
              draft.run_id = 'testrun' + i;
              draft.display_name = 'run with id: testrun' + i;
              draft.pipeline_version_reference = pipelineVersionRef;
            });
          }
          return {
            run_id: 'testrun' + i,
            display_name: 'run with id: testrun' + i,
            pipeline_version_reference: {
              pipeline_id: 'testpipeline' + i,
              pipeline_version_id: 'testversion' + i,
            },
          } as V2beta1Run;
        }),
      }),
    );

    getPipelineVersionSpy.mockImplementation(() => ({ display_name: 'some pipeline version' }));
    listExperimentsSpy.mockImplementation(() => ({ display_name: 'some experiment' }));
  }

  function getMountedInstance(): RunList {
    tree = TestUtils.mountWithRouter(<RunList {...generateProps()} />);
    return tree.instance() as RunList;
  }

  function getShallowInstance(): RunList {
    tree = shallow(<RunList {...generateProps()} />);
    return tree.instance() as RunList;
  }

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date?: Date | string) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
    onErrorSpy.mockClear();
    listRunsSpy.mockClear();
    getRunSpy.mockClear();
    listExperimentsSpy.mockClear();
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    if (tree && tree.exists()) {
      await tree.unmount();
    }
    jest.resetAllMocks();
  });

  it('renders the empty experience', () => {
    expect(shallow(<RunList {...generateProps()} />)).toMatchSnapshot();
  });

  describe('in archived state', () => {
    it('renders the empty experience', () => {
      const props = generateProps();
      props.storageState = V2beta1RunStorageState.ARCHIVED;
      expect(shallow(<RunList {...props} />)).toMatchSnapshot();
    });

    it('loads runs whose storage state is not ARCHIVED when storage state equals AVAILABLE', async () => {
      mockNRuns(1, {});
      const props = generateProps();
      props.storageState = V2beta1RunStorageState.AVAILABLE;
      tree = shallow(<RunList {...props} />);
      await (tree.instance() as RunListTest)._loadRuns({});
      expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        encodeURIComponent(
          JSON.stringify({
            predicates: [
              {
                key: 'storage_state',
                operation: V2beta1PredicateOperation.NOTEQUALS,
                string_value: V2beta1RunStorageState.ARCHIVED.toString(),
              },
            ],
          } as V2beta1Filter),
        ),
      );
    });

    it('loads runs whose storage state is ARCHIVED when storage state equals ARCHIVED', async () => {
      mockNRuns(1, {});
      const props = generateProps();
      props.storageState = V2beta1RunStorageState.ARCHIVED;
      tree = shallow(<RunList {...props} />);
      await (tree.instance() as RunListTest)._loadRuns({});
      expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        encodeURIComponent(
          JSON.stringify({
            predicates: [
              {
                key: 'storage_state',
                operation: V2beta1PredicateOperation.EQUALS,
                string_value: V2beta1RunStorageState.ARCHIVED.toString(),
              },
            ],
          } as V2beta1Filter),
        ),
      );
    });

    it('augments request filter with storage state predicates', async () => {
      mockNRuns(1, {});
      const props = generateProps();
      props.storageState = V2beta1RunStorageState.ARCHIVED;
      tree = shallow(<RunList {...props} />);
      await (tree.instance() as RunListTest)._loadRuns({
        filter: encodeURIComponent(
          JSON.stringify({
            predicates: [{ key: 'k', op: 'op', string_value: 'val' }],
          }),
        ),
      });
      expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        encodeURIComponent(
          JSON.stringify({
            predicates: [
              {
                key: 'k',
                op: 'op',
                string_value: 'val',
              },
              {
                key: 'storage_state',
                operation: V2beta1PredicateOperation.EQUALS,
                string_value: V2beta1RunStorageState.ARCHIVED.toString(),
              },
            ],
          } as V2beta1Filter),
        ),
      );
    });
  });

  it('loads one run', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    render(
      <CommonTestWrapper>
        <RunList {...props} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
    });

    screen.getByText('run with id: testrun1');
    expect(screen.queryByText('run with id: testrun2')).toBeNull();
  });

  it('reloads the run when refresh is called', async () => {
    mockNRuns(0, {});
    const props = generateProps();
    tree = TestUtils.mountWithRouter(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    tree.update();
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(2);
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      '',
      10,
      RunSortKeys.CREATED_AT + ' desc',
      '',
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('loads multiple runs', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    render(
      <CommonTestWrapper>
        <RunList {...props} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
    });

    screen.getByText('run with id: testrun1');
    screen.getByText('run with id: testrun2');
    screen.getByText('run with id: testrun3');
    screen.getByText('run with id: testrun4');
    screen.getByText('run with id: testrun5');
  });

  it('calls error callback when loading runs fails', async () => {
    TestUtils.makeErrorResponseOnce(
      jest.spyOn(Apis.runServiceApiV2, 'listRuns'),
      'bad stuff happened',
    );
    const props = generateProps();
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).toHaveBeenLastCalledWith(
      'Error: failed to fetch runs.',
      new Error('bad stuff happened'),
    );
  });

  it('displays error in run row if experiment could not be fetched', async () => {
    mockNRuns(1, {
      experiment_id: 'test-experiment-id',
    });
    TestUtils.makeErrorResponseOnce(listExperimentsSpy, 'bad stuff happened');
    const props = generateProps();

    render(
      <CommonTestWrapper>
        <RunList {...props} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
      expect(listExperimentsSpy).toHaveBeenCalled();
    });

    screen.findByText('Failed to get associated experiment: bad stuff happened');
  });

  it('displays error in run row if it failed to parse (run list mask)', async () => {
    TestUtils.makeErrorResponseOnce(
      jest.spyOn(Apis.runServiceApiV2, 'getRun'),
      'bad stuff happened',
    );
    const props = generateProps();
    props.runIdListMask = ['testrun1'];
    render(
      <CommonTestWrapper>
        <RunList {...props} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      // won't call listRuns if specific run id is provided
      expect(listRunsSpy).toHaveBeenCalledTimes(0);
      expect(getRunSpy).toHaveBeenCalledTimes(1);
    });

    screen.findByText('Failed to get associated experiment: bad stuff happened');
  });

  it('shows run time for each run', async () => {
    mockNRuns(1, {
      created_at: new Date(2018, 10, 10, 10, 10, 10),
      finished_at: new Date(2018, 10, 10, 11, 11, 11),
      state: V2beta1RuntimeState.SUCCEEDED,
    });
    const props = generateProps();
    render(
      <CommonTestWrapper>
        <RunList {...props} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
    });

    screen.findByText('1:01:01');
  });

  it('loads runs for a given experiment id', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    props.experimentIdMask = 'experiment1';
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
      undefined,
      'experiment1',
      undefined,
      undefined,
      undefined,
      undefined,
    );
  });

  it('loads runs for a given namespace', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    props.namespaceMask = 'namespace1';
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
      'namespace1',
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    );
  });

  it('loads given list of runs only', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    props.runIdListMask = ['run1', 'run2'];
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.runServiceApiV2.listRuns).not.toHaveBeenCalled();
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledTimes(2);
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledWith('run1');
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledWith('run2');
  });

  it('loads given and filtered list of runs only', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    props.runIdListMask = ['filterRun1', 'filterRun2', 'notincluded'];
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({
      filter: encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'name',
              operation: V2beta1PredicateOperation.ISSUBSTRING,
              string_value: 'filterRun',
            },
          ],
        } as V2beta1Filter),
      ),
    });
    expect(tree.state('runs')).toMatchObject([
      {
        run: { display_name: 'run with id: filterRun1', run_id: 'filterRun1' },
      },
      {
        run: { display_name: 'run with id: filterRun2', run_id: 'filterRun2' },
      },
    ]);
  });

  it('loads given and filtered list of runs only through multiple filters', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    props.runIdListMask = ['filterRun1', 'filterRun2', 'notincluded1'];
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({
      filter: encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'name',
              operation: V2beta1PredicateOperation.ISSUBSTRING,
              string_value: 'filterRun',
            },
            { key: 'name', operation: V2beta1PredicateOperation.ISSUBSTRING, string_value: '1' },
          ],
        } as V2beta1Filter),
      ),
    });
    expect(tree.state('runs')).toMatchObject([
      {
        run: { display_name: 'run with id: filterRun1', run_id: 'filterRun1' },
      },
    ]);
  });

  it('shows pipeline version name', async () => {
    mockNRuns(1, {
      pipeline_version_reference: {
        pipeline_id: 'testpipeline1',
        pipeline_version_id: 'testversion1',
      },
    });
    const props = generateProps();
    render(
      <CommonTestWrapper>
        <RunList {...props} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
      expect(getPipelineVersionSpy).toHaveBeenCalled();
    });

    screen.findByText('some pipeline version');
  });

  //TODO(jlyaoyuli): add back this test (show recurring run config)
  //after the recurring run v2 API integration

  it('shows experiment name', async () => {
    mockNRuns(1, {
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
    render(
      <CommonTestWrapper>
        <RunList {...props} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
      expect(listExperimentsSpy).toHaveBeenCalled();
    });

    screen.getByText('test experiment');
  });

  it('hides experiment name if instructed', async () => {
    mockNRuns(1, {
      experiment_id: 'test-experiment-id',
    });
    listExperimentsSpy.mockImplementationOnce(() => ({ display_name: 'test experiment' }));
    const props = generateProps();
    props.hideExperimentColumn = true;
    render(
      <CommonTestWrapper>
        <RunList {...props} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(listRunsSpy).toHaveBeenCalled();
      expect(listExperimentsSpy).toHaveBeenCalledTimes(0);
    });

    expect(screen.queryByText('test experiment')).toBeNull();
  });

  it('renders run name as link to its details page', () => {
    expect(
      getMountedInstance()._nameCustomRenderer({ value: 'test run', id: 'run-id' }),
    ).toMatchSnapshot();
  });

  it('renders pipeline name as link to its details page', () => {
    expect(
      getMountedInstance()._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: { displayName: 'test pipeline', pipelineId: 'pipeline-id', usePlaceholder: false },
      }),
    ).toMatchSnapshot();
  });

  it('handles no pipeline id given', () => {
    expect(
      getMountedInstance()._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: { displayName: 'test pipeline', usePlaceholder: false },
      }),
    ).toMatchSnapshot();
  });

  it('shows "View pipeline" button if pipeline is embedded in run', () => {
    expect(
      getMountedInstance()._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: { displayName: 'test pipeline', pipelineId: 'pipeline-id', usePlaceholder: true },
      }),
    ).toMatchSnapshot();
  });

  it('handles no pipeline name', () => {
    expect(
      getMountedInstance()._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: { /* no displayName */ usePlaceholder: true },
      }),
    ).toMatchSnapshot();
  });

  it('renders pipeline name as link to its details page', () => {
    expect(
      getMountedInstance()._recurringRunCustomRenderer({
        id: 'run-id',
        value: { id: 'recurring-run-id' },
      }),
    ).toMatchSnapshot();
  });

  it('renders experiment name as link to its details page', () => {
    expect(
      getMountedInstance()._experimentCustomRenderer({
        id: 'run-id',
        value: { displayName: 'test experiment', id: 'experiment-id' },
      }),
    ).toMatchSnapshot();
  });

  it('renders no experiment name', () => {
    expect(
      getMountedInstance()._experimentCustomRenderer({
        id: 'run-id',
        value: { /* no displayName */ id: 'experiment-id' },
      }),
    ).toMatchSnapshot();
  });

  it('renders status as icon', () => {
    expect(
      getShallowInstance()._statusCustomRenderer({
        value: V2beta1RuntimeState.SUCCEEDED,
        id: 'run-id',
      }),
    ).toMatchSnapshot();
  });

  it('renders pipeline version name as link to its details page', () => {
    expect(
      getMountedInstance()._pipelineVersionCustomRenderer({
        id: 'run-id',
        value: {
          displayName: 'test pipeline version',
          pipelineId: 'pipeline-id',
          usePlaceholder: false,
          versionId: 'version-id',
        },
      }),
    ).toMatchSnapshot();
  });
});
