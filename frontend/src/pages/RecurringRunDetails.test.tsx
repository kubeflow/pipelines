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
import RecurringRunDetails from './RecurringRunDetails';
import TestUtils from '../TestUtils';
import produce from 'immer';
import { ApiJob, ApiResourceType } from '../apis/job';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { RouteParams, RoutePage } from '../components/Router';
import { shallow } from 'enzyme';

describe('RecurringRunDetails', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getJobSpy = jest.spyOn(Apis.jobServiceApi, 'getJob');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');

  const fullTestJob = {
    created_at: new Date(2018, 8, 5, 4, 3, 2),
    description: 'test job description',
    enabled: true,
    id: 'test-job-id',
    max_concurrency: '50',
    name: 'test job',
    pipeline_spec: { pipeline_id: 'some-pipeline-id' },
    trigger: {
      periodic_schedule: {
        end_time: new Date(2018, 10, 9, 8, 7, 6),
        interval_second: '3600',
        start_time: new Date(2018, 9, 8, 7, 6),
      }
    },
  } as ApiJob;

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: { params: { [RouteParams.runId]: fullTestJob.id }, isExact: true, path: '', url: '' },
      toolbarProps: RecurringRunDetails.prototype.getInitialToolbarState(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    updateBannerSpy.mockClear();
    updateDialogSpy.mockClear();
    updateSnackbarSpy.mockClear();
    updateToolbarSpy.mockClear();
    getJobSpy.mockImplementation(() => fullTestJob);
    getExperimentSpy.mockImplementation();
  });

  it('renders a recurring run with periodic schedule', () => {
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    tree.setState({
      run: fullTestJob,
    });
    expect(tree).toMatchSnapshot();
  });

  it('renders a recurring run with cron schedule', () => {
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    tree.setState({
      run: produce(fullTestJob, draft => {
        draft.trigger = {
          cron_schedule: {
            cron: '* * * 0 0 !',
            end_time: new Date(2018, 10, 9, 8, 7, 6),
            start_time: new Date(2018, 9, 8, 7, 6),
          }
        };
      }),
    });
    expect(tree).toMatchSnapshot();
  });

  it('loads the recurring run given its id in query params', async () => {
    fullTestJob.resource_references = [];
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await (tree.instance() as RecurringRunDetails).load();
    expect(getJobSpy).toHaveBeenLastCalledWith(fullTestJob.id);
    expect(getExperimentSpy).not.toHaveBeenCalled();
  });

  it('shows All runs -> run name when there is no experiment', async () => {
    fullTestJob.resource_references = [];
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await (tree.instance() as RecurringRunDetails).load();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      breadcrumbs: [
        { displayName: 'All runs', href: RoutePage.RUNS },
        { displayName: fullTestJob.name, href: '' },
      ],
    }));
  });

  it('loads the recurring run and its experiment if it has one', async () => {
    fullTestJob.resource_references = [{ key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } }];
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await (tree.instance() as RecurringRunDetails).load();
    expect(getJobSpy).toHaveBeenLastCalledWith(fullTestJob.id);
    expect(getExperimentSpy).toHaveBeenLastCalledWith('test-experiment-id');
  });

  it('shows Experiments -> Experiment name -> run name when there is no experiment', async () => {
    fullTestJob.resource_references = [{ key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } }];
    getExperimentSpy.mockImplementation(id => ({ id, name: 'test experiment name' }));
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await (tree.instance() as RecurringRunDetails).load();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        {
          displayName: 'test experiment name',
          href: RoutePage.EXPERIMENT_DETAILS.replace(
            ':' + RouteParams.experimentId, 'test-experiment-id'),
        },
        { displayName: fullTestJob.name, href: '' },
      ],
    }));
  });

  it('shows error banner if run cannot be fetched', async () => {
    fullTestJob.resource_references = [];
    TestUtils.makeErrorResponseOnce(getJobSpy, 'woops!');
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await (tree.instance() as RecurringRunDetails).load();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      `Error: failed to retrieve recurring run: ${fullTestJob.id}.`, {});
  });

});
