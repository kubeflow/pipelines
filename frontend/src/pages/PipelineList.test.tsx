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
import PipelineList from './PipelineList';
import TestUtils from '../TestUtils';
import { ApiPipeline } from '../apis/pipeline';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { shallow } from 'enzyme';

describe('PipelineList', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const listPipelinesSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');

  function generateProps(): PageProps {
    return {
      history: {} as any,
      location: '' as any,
      match: '' as any,
      toolbarProps: { actions: [], breadcrumbs: [] },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    // Reset mocks
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();
    listPipelinesSpy.mockReset();
  });

  it('renders an empty list with empty state message', () => {
    expect(shallow(<PipelineList {...generateProps()} />)).toMatchSnapshot();
  });

  it('renders a list of one pipeline', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({
      pipelines: [{
        created_at: new Date(2018, 8, 22, 11, 5, 48),
        description: 'test pipeline description',
        name: 'pipeline1',
        parameters: [],
      } as ApiPipeline]
    });
    await listPipelinesSpy;
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one pipeline with no description or created date', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({
      pipelines: [{
        created_at: new Date(2018, 8, 22, 11, 5, 48),
        description: 'test pipeline description',
        name: 'pipeline1',
        parameters: [],
      } as ApiPipeline]
    });
    await listPipelinesSpy;
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one pipeline with error', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({
      pipelines: [{
        created_at: new Date(2018, 8, 22, 11, 5, 48),
        description: 'test pipeline description',
        error: 'oops! could not load pipeline',
        name: 'pipeline1',
        parameters: [],
      } as ApiPipeline]
    });
    await listPipelinesSpy;
    expect(tree).toMatchSnapshot();
  });

  it('calls Apis to list pipelines, sorted by creation time in descending order', async () => {
    listPipelinesSpy.mockImplementationOnce(() => ({ pipelines: [{ name: 'pipeline1' }] }));
    const tree = TestUtils.mountWithRouter(PipelineList, generateProps());
    await listPipelinesSpy;
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(tree.state()).toHaveProperty('pipelines', [{ name: 'pipeline1' }]);
  });

  it('has a Refresh button, clicking it refreshes the pipeline list', async () => {
    listPipelinesSpy.mockImplementation(() => ({ pipelines: [{ name: 'pipeline1' }] }));
    const instance = TestUtils.mountWithRouter(PipelineList, generateProps())
      .instance() as PipelineList;
    await listPipelinesSpy;
    expect(listPipelinesSpy.mock.calls.length).toBe(1);
    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    await refreshBtn!.action();
    expect(listPipelinesSpy.mock.calls.length).toBe(2);
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('shows error banner when listing pipelines fails', async () => {
    listPipelinesSpy.mockImplementationOnce(() => {
      throw {
        text: () => Promise.resolve('bad stuff happened'),
      };
    });
    TestUtils.mountWithRouter(PipelineList, generateProps());
    await listPipelinesSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of pipelines. Click Details for more information.',
      mode: 'error',
    }));
  });

  it('shows error banner when listing pipelines fails after refresh', async () => {
    const instance = TestUtils.mountWithRouter(PipelineList, generateProps())
      .instance() as PipelineList;
    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    listPipelinesSpy.mockImplementationOnce(() => {
      throw {
        text: () => Promise.resolve('bad stuff happened'),
      };
    });
    await refreshBtn!.action();
    expect(listPipelinesSpy.mock.calls.length).toBe(2);
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of pipelines. Click Details for more information.',
      mode: 'error',
    }));
  });

  it('hides error banner when listing pipelines fails then succeeds', async () => {
    listPipelinesSpy.mockImplementationOnce(() => {
      throw {
        text: () => Promise.resolve('bad stuff happened'),
      };
    });
    const instance = TestUtils.mountWithRouter(PipelineList, generateProps())
      .instance() as PipelineList;
    await listPipelinesSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of pipelines. Click Details for more information.',
      mode: 'error',
    }));
    updateBannerSpy.mockReset();

    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    listPipelinesSpy.mockImplementationOnce(() => ({ pipelines: [{ name: 'pipeline1' }] }));
    await refreshBtn!.action();
    expect(listPipelinesSpy.mock.calls.length).toBe(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

});
