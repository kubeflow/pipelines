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
import * as Utils from '../lib/Utils';
import JobList, { JobListProps } from './JobList';
import TestUtils from '../TestUtils';
import produce from 'immer';
import { ApiJob, ApiResourceType } from '../apis/job';
import { Apis, JobSortKeys, ListRequest } from '../lib/Apis';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { range } from 'lodash';

class JobListTest extends JobList {
  public _loadJobs(request: ListRequest): Promise<string> {
    return super._loadJobs(request);
  }
}

describe('JobList', () => {
  let tree: ShallowWrapper | ReactWrapper;

  const onErrorSpy = jest.fn();
  const listJobsSpy = jest.spyOn(Apis.jobServiceApi, 'listJobs');
  const getJobSpy = jest.spyOn(Apis.jobServiceApi, 'getJob');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test environments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');

  function generateProps(): JobListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
      refreshCount: 1,
    };
  }

  function mockNJobs(n: number, jobTemplate: Partial<ApiJob>): void {
    getJobSpy.mockImplementation(id =>
      Promise.resolve(
        produce(jobTemplate, draft => {
          draft.id = id;
          draft.name = 'job with id: ' + id;
        }),
      ),
    );

    listJobsSpy.mockImplementation(() =>
      Promise.resolve({
        jobs: range(1, n + 1).map(i => {
          if (jobTemplate) {
            return produce(jobTemplate as Partial<ApiJob>, draft => {
              draft.id = 'testjob' + i;
              draft.name = 'job with id: testjob' + i;
            });
          }
          return {
            id: 'testjob' + i,
            name: 'job with id: testjob' + i,
          } as ApiJob;
        }),
      }),
    );

    getExperimentSpy.mockImplementation(() => ({ name: 'some experiment' }));
  }

  function getMountedInstance(): JobList {
    tree = TestUtils.mountWithRouter(<JobList {...generateProps()} />);
    return tree.instance() as JobList;
  }

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
    onErrorSpy.mockClear();
    listJobsSpy.mockClear();
    getJobSpy.mockClear();
    getExperimentSpy.mockClear();
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
    expect(shallow(<JobList {...generateProps()} />)).toMatchSnapshot();
  });

  it('loads one job', async () => {
    mockNJobs(1, {});
    const props = generateProps();
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(Apis.jobServiceApi.listJobs).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('reloads the job when refresh is called', async () => {
    mockNJobs(0, {});
    const props = generateProps();
    tree = TestUtils.mountWithRouter(<JobList {...props} />);
    await (tree.instance() as JobList).refresh();
    tree.update();
    expect(Apis.jobServiceApi.listJobs).toHaveBeenCalledTimes(2);
    expect(Apis.jobServiceApi.listJobs).toHaveBeenLastCalledWith(
      '',
      10,
      JobSortKeys.CREATED_AT + ' desc',
      undefined,
      undefined,
      '',
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('loads multiple jobs', async () => {
    mockNJobs(5, {});
    const props = generateProps();
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('calls error callback when loading jobs fails', async () => {
    TestUtils.makeErrorResponseOnce(
      jest.spyOn(Apis.jobServiceApi, 'listJobs'),
      'bad stuff happened',
    );
    const props = generateProps();
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).toHaveBeenLastCalledWith(
      'Error: failed to fetch jobs.',
      new Error('bad stuff happened'),
    );
  });

  it('loads jobs for a given experiment id', async () => {
    mockNJobs(1, {});
    const props = generateProps();
    props.experimentIdMask = 'experiment1';
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.jobServiceApi.listJobs).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      'EXPERIMENT',
      'experiment1',
      undefined,
    );
  });

  it('loads jobs for a given namespace', async () => {
    mockNJobs(1, {});
    const props = generateProps();
    props.namespaceMask = 'namespace1';
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.jobServiceApi.listJobs).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      'NAMESPACE',
      'namespace1',
      undefined,
    );
  });

  it('loads given list of jobs only', async () => {
    mockNJobs(5, {});
    const props = generateProps();
    props.jobIdListMask = ['job1', 'job2'];
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.jobServiceApi.listJobs).not.toHaveBeenCalled();
    expect(Apis.jobServiceApi.getJob).toHaveBeenCalledTimes(2);
    expect(Apis.jobServiceApi.getJob).toHaveBeenCalledWith('job1');
    expect(Apis.jobServiceApi.getJob).toHaveBeenCalledWith('job2');
  });

  it('shows job status', async () => {
    mockNJobs(1, {
      status: 'ENABLED',
    });
    const props = generateProps();
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('shows trigger periodic', async () => {
    mockNJobs(1, {
      trigger: { periodic_schedule: { interval_second: '3600' } },
    });
    const props = generateProps();
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('shows trigger cron', async () => {
    mockNJobs(1, {
      trigger: { cron_schedule: { cron: '0 * * * * ?' } },
    });
    const props = generateProps();
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('shows experiment name', async () => {
    mockNJobs(1, {
      resource_references: [
        {
          key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT },
        },
      ],
    });
    getExperimentSpy.mockImplementationOnce(() => ({ name: 'test experiment' }));
    const props = generateProps();
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('hides experiment name if instructed', async () => {
    mockNJobs(1, {
      resource_references: [
        {
          key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT },
        },
      ],
    });
    getExperimentSpy.mockImplementationOnce(() => ({ name: 'test experiment' }));
    const props = generateProps();
    props.hideExperimentColumn = true;
    tree = shallow(<JobList {...props} />);
    await (tree.instance() as JobListTest)._loadJobs({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('renders job trigger in seconds', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '42' } },
        id: 'job-id',
      }),
    ).toMatchSnapshot();
  });

  it('renders job trigger in minutes', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '120' } },
        id: 'job-id',
      }),
    ).toMatchSnapshot();
  });

  it('renders job trigger in hours', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '7200' } },
        id: 'job-id',
      }),
    ).toMatchSnapshot();
  });

  it('renders job trigger in days', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '86400' } },
        id: 'job-id',
      }),
    ).toMatchSnapshot();
  });

  it('renders job trigger as cron', () => {
    expect(
      getMountedInstance()._triggerCustomRenderer({
        value: { cron_schedule: { cron: '0 * * * * ?' } },
        id: 'job-id',
      }),
    ).toMatchSnapshot();
  });

  it('renders status enabled', () => {
    expect(
      getMountedInstance()._statusCustomRenderer({
        value: 'Enabled',
        id: 'job-id',
      }),
    ).toMatchSnapshot();
  });

  it('renders status disabled', () => {
    expect(
      getMountedInstance()._statusCustomRenderer({
        value: 'Disabled',
        id: 'job-id',
      }),
    ).toMatchSnapshot();
  });

  it('renders status unknown', () => {
    expect(
      getMountedInstance()._statusCustomRenderer({
        value: 'Unknown Status',
        id: 'job-id',
      }),
    ).toMatchSnapshot();
  });
});
