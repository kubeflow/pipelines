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
import * as StaticGraphParser from '../lib/StaticGraphParser';
import PipelineDetails from './PipelineDetails';
import TestUtils from '../TestUtils';
import { ApiPipeline } from '../apis/pipeline';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { RouteParams } from '../components/Router';
import { graphlib } from 'dagre';
import { shallow } from 'enzyme';

describe('PipelineDetails', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getTemplateSpy = jest.spyOn(Apis.pipelineServiceApi, 'getTemplate');
  const createGraphSpy = jest.spyOn(StaticGraphParser, 'createGraph');

  let testPipeline: ApiPipeline = {};

  function generateProps(): PageProps {
    // getInitialToolbarState relies on page props having been populated, so fill those first
    const pageProps: PageProps = {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: { params: { [RouteParams.pipelineId]: testPipeline.id }, isExact: true, path: '', url: '' },
      toolbarProps: { actions: [], breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
    return Object.assign(pageProps, {
      toolbarProps: new PipelineDetails(pageProps).getInitialToolbarState(),
    });
  }

  beforeEach(() => {
    testPipeline = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test job description',
      enabled: true,
      id: 'test-job-id',
      max_concurrency: '50',
      name: 'test job',
      pipeline_spec: {
        parameters: [{ name: 'param1', value: 'value1' }],
        pipeline_id: 'some-pipeline-id',
      },
      trigger: {
        periodic_schedule: {
          end_time: new Date(2018, 10, 9, 8, 7, 6),
          interval_second: '3600',
          start_time: new Date(2018, 9, 8, 7, 6),
        }
      },
    } as ApiPipeline;

    historyPushSpy.mockClear();
    updateBannerSpy.mockClear();
    updateDialogSpy.mockClear();
    updateSnackbarSpy.mockClear();
    updateToolbarSpy.mockClear();
    getPipelineSpy.mockImplementation(() => Promise.resolve(testPipeline));
    getPipelineSpy.mockClear();
    getTemplateSpy.mockImplementation(() => Promise.resolve('{}'));
    getTemplateSpy.mockClear();
    createGraphSpy.mockImplementation(() => new graphlib.Graph());
    createGraphSpy.mockClear();
  });

  it('shows empty pipeline details with no graph graph', async () => {
    TestUtils.makeErrorResponseOnce(createGraphSpy, 'bad graph');
    const tree = shallow(<PipelineDetails {...generateProps()} />);
    const instance = tree.instance() as PipelineDetails;
    TestUtils.makeErrorResponseOnce(createGraphSpy, 'bad graph');
    await instance.load();
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows no graph error banner when failing to parse graph', async () => {
    TestUtils.makeErrorResponseOnce(createGraphSpy, 'bad graph');
    const tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad graph',
      message: 'Error: failed to generate Pipeline graph. Click Details for more information.',
      mode: 'error',
    }));
    tree.unmount();
  });

  it('shows load error banner when failing to get pipeline', async () => {
    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'woops');
    const tree = shallow(<PipelineDetails {...generateProps()} />);
    await getPipelineSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear banner, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'woops',
      message: 'Cannot retrieve pipeline details',
      mode: 'error',
    }));
    tree.unmount();
  });

  it('shows empty pipeline details with empty graph', async () => {
    const tree = shallow(<PipelineDetails {...generateProps()} />);
    const instance = tree.instance() as PipelineDetails;
    await instance.load();
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

});
