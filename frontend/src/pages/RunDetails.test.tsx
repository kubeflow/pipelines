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
import RunDetails from './RunDetails';
import TestUtils from '../TestUtils';
import WorkflowParser from '../lib/WorkflowParser';
import { ApiRunDetail, ApiResourceType } from '../apis/run';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { QUERY_PARAMS } from '../lib/URLParser';
import { RouteParams, RoutePage } from '../components/Router';
import { graphlib } from 'dagre';
import { shallow } from 'enzyme';

describe('RunDetails', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const getPodLogsSpy = jest.spyOn(Apis, 'getPodLogs');

  let testRun: ApiRunDetail = {};

  function generateProps(): PageProps {
    const pageProps: PageProps = {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: { params: { [RouteParams.runId]: testRun.run!.id }, isExact: true, path: '', url: '' },
      toolbarProps: { actions: [], breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
    return Object.assign(pageProps, {
      toolbarProps: new RunDetails(pageProps).getInitialToolbarState(),
    });
  }

  beforeAll(() => jest.spyOn(console, 'error').mockImplementation());
  afterAll(() => jest.resetAllMocks());

  beforeEach(() => {
    testRun = {
      pipeline_runtime: {
        workflow_manifest: '{}',
      },
      run: {
        created_at: new Date(2018, 8, 5, 4, 3, 2),
        description: 'test run description',
        id: 'test-run-id',
        name: 'test run',
        pipeline_spec: {
          parameters: [{ name: 'param1', value: 'value1' }],
          pipeline_id: 'some-pipeline-id',
        },
      },
    };
    historyPushSpy.mockClear();
    updateBannerSpy.mockClear();
    updateDialogSpy.mockClear();
    updateSnackbarSpy.mockClear();
    updateToolbarSpy.mockClear();
    getRunSpy.mockReset();
    getRunSpy.mockImplementation(() => Promise.resolve(testRun));
    getExperimentSpy.mockClear();
    getExperimentSpy.mockImplementation(() => Promise.resolve('{}'));
    getPodLogsSpy.mockClear();
    getPodLogsSpy.mockImplementation(() => new graphlib.Graph());
  });

  it('has a refresh button, clicking it calls get run API again', async () => {
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    expect(getRunSpy).toHaveBeenCalledTimes(1);
    await refreshBtn!.action();
    expect(getRunSpy).toHaveBeenCalledTimes(2);
  });

  it('has a clone button, clicking it navigates to new run page', async () => {
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    const instance = tree.instance() as RunDetails;
    const cloneBtn = instance.getInitialToolbarState().actions.find(
      b => b.title === 'Clone');
    expect(cloneBtn).toBeDefined();
    await cloneBtn!.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.cloneFromRun}=${testRun.run!.id}`);
  });

  it('renders an empty run', async () => {
    const tree = shallow(<RunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('calls the get run API once to load it', async () => {
    shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(getRunSpy).toHaveBeenCalledTimes(1);
    expect(getRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
  });

  it('calls the get experiment API once to load it if the run has its reference', async () => {
    testRun.run!.resource_references = [{ key: { id: 'experiment1', type: ApiResourceType.EXPERIMENT } }];
    shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(getRunSpy).toHaveBeenCalledTimes(1);
    expect(getRunSpy).toHaveBeenLastCalledWith(testRun.run!.id);
    expect(getExperimentSpy).toHaveBeenCalledTimes(1);
    expect(getExperimentSpy).toHaveBeenLastCalledWith('experiment1');
  });

  it('shows workflow errors as page error', async () => {
    jest.spyOn(WorkflowParser, 'getWorkflowError').mockImplementationOnce(() => 'some error message');
    const tree = shallow(<RunDetails {...generateProps()} />);
    await getRunSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear on init, once for error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'some error message',
      message: 'Error: found errors when executing run: ' + testRun.run!.id + '. Click Details for more information.',
      mode: 'error',
    }));
    tree.unmount();
  });

});
