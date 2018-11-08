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
import { RoutePage, RouteParams } from '../components/Router';
import { shallow, ReactWrapper } from 'enzyme';
import { range } from 'lodash';

describe('PipelineList', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const listPipelinesSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');
  const deletePipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'deletePipeline');
  const uploadPipelineSpy = jest.spyOn(Apis, 'uploadPipeline');

  function generateProps(): PageProps {
    return {
      history: {} as any,
      location: '' as any,
      match: '' as any,
      toolbarProps: PipelineList.prototype.getInitialToolbarState(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  async function mountWithNPipelines(n: number): Promise<ReactWrapper> {
    listPipelinesSpy.mockImplementationOnce(() => ({
      pipelines: range(n).map(i => ({ id: 'test-pipeline-id' + i, name: 'test pipeline name' + i })),
    }));
    const tree = TestUtils.mountWithRouter(<PipelineList {...generateProps()} />);
    await listPipelinesSpy;
    await TestUtils.flushPromises();
    tree.update(); // Make sure the tree is updated before returning it
    return tree;
  }

  beforeEach(() => {
    // Reset mocks
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();
    listPipelinesSpy.mockReset();
    deletePipelineSpy.mockReset();
    uploadPipelineSpy.mockReset();
  });

  it('renders an empty list with empty state message', () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    expect(tree).toMatchSnapshot();
    tree.unmount();
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
    tree.unmount();
  });

  it('renders a list of one pipeline with no description or created date', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({
      pipelines: [{
        name: 'pipeline1',
        parameters: [],
      } as ApiPipeline]
    });
    await listPipelinesSpy;
    expect(tree).toMatchSnapshot();
    tree.unmount();
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
    tree.unmount();
  });

  it('calls Apis to list pipelines, sorted by creation time in descending order', async () => {
    listPipelinesSpy.mockImplementationOnce(() => ({ pipelines: [{ name: 'pipeline1' }] }));
    const tree = TestUtils.mountWithRouter(<PipelineList {...generateProps()} />);
    await listPipelinesSpy;
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(tree.state()).toHaveProperty('pipelines', [{ name: 'pipeline1' }]);
    tree.unmount();
  });

  it('has a Refresh button, clicking it refreshes the pipeline list', async () => {
    const tree = await mountWithNPipelines(1);
    const instance = tree.instance() as PipelineList;
    expect(listPipelinesSpy.mock.calls.length).toBe(1);
    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    await refreshBtn!.action();
    expect(listPipelinesSpy.mock.calls.length).toBe(2);
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
    tree.unmount();
  });

  it('shows error banner when listing pipelines fails', async () => {
    TestUtils.makeErrorResponseOnce(listPipelinesSpy, 'bad stuff happened');
    const tree = TestUtils.mountWithRouter(<PipelineList {...generateProps()} />);
    await listPipelinesSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of pipelines. Click Details for more information.',
      mode: 'error',
    }));
    tree.unmount();
  });

  it('shows error banner when listing pipelines fails after refresh', async () => {
    const tree = TestUtils.mountWithRouter(<PipelineList {...generateProps()} />);
    const instance = tree.instance() as PipelineList;
    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    TestUtils.makeErrorResponseOnce(listPipelinesSpy, 'bad stuff happened');
    await refreshBtn!.action();
    expect(listPipelinesSpy.mock.calls.length).toBe(2);
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of pipelines. Click Details for more information.',
      mode: 'error',
    }));
    tree.unmount();
  });

  it('hides error banner when listing pipelines fails then succeeds', async () => {
    TestUtils.makeErrorResponseOnce(listPipelinesSpy, 'bad stuff happened');
    const tree = TestUtils.mountWithRouter(<PipelineList {...generateProps()} />);
    const instance = tree.instance() as PipelineList;
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
    tree.unmount();
  });

  it('renders pipeline names as links to their details pages', async () => {
    const tree = await mountWithNPipelines(1);
    const link = tree.find('a[children="test pipeline name0"]');
    expect(link).toHaveLength(1);
    expect(link.prop('href')).toBe(RoutePage.PIPELINE_DETAILS.replace(
      ':' + RouteParams.pipelineId, 'test-pipeline-id0'
    ));
    tree.unmount();
  });

  it('always has upload pipeline button enabled', async () => {
    const tree = await mountWithNPipelines(1);
    const calls = updateToolbarSpy.mock.calls[0];
    expect(calls[0].actions.find((b: any) => b.title === 'Upload pipeline')).not.toHaveProperty('disabled');
    tree.unmount();
  });

  it('enables delete button when one pipeline is selected', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').simulate('click');
    expect(updateToolbarSpy.mock.calls).toHaveLength(2); // Initial call, then selection update
    const calls = updateToolbarSpy.mock.calls[1];
    expect(calls[0].actions.find((b: any) => b.title === 'Delete')).toHaveProperty('disabled', false);
    tree.unmount();
  });

  it('enables delete button when two pipelines are selected', async () => {
    const tree = await mountWithNPipelines(2);
    tree.find('.tableRow').at(0).simulate('click');
    tree.find('.tableRow').at(1).simulate('click');
    expect(updateToolbarSpy.mock.calls).toHaveLength(3); // Initial call, then selection updates
    const calls = updateToolbarSpy.mock.calls[2];
    expect(calls[0].actions.find((b: any) => b.title === 'Delete')).toHaveProperty('disabled', false);
    tree.unmount();
  });

  it('re-disables delete button pipelines are unselected', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').at(0).simulate('click');
    tree.find('.tableRow').at(0).simulate('click');
    expect(updateToolbarSpy.mock.calls).toHaveLength(3); // Initial call, then selection updates
    const calls = updateToolbarSpy.mock.calls[2];
    expect(calls[0].actions.find((b: any) => b.title === 'Delete')).toHaveProperty('disabled', true);
    tree.unmount();
  });

  it('shows delete dialog when delete button is clicked', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').at(0).simulate('click');
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    expect(call).toHaveProperty('title', 'Delete 1 pipeline?');
    tree.unmount();
  });

  it('shows delete dialog when delete button is clicked, indicating several pipelines to delete', async () => {
    const tree = await mountWithNPipelines(5);
    tree.find('.tableRow').at(0).simulate('click');
    tree.find('.tableRow').at(2).simulate('click');
    tree.find('.tableRow').at(3).simulate('click');
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    expect(call).toHaveProperty('title', 'Delete 3 pipelines?');
    tree.unmount();
  });

  it('does not call delete API for selected pipeline when delete dialog is canceled', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').at(0).simulate('click');
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((b: any) => b.text === 'Cancel');
    await cancelBtn.onClick();
    expect(deletePipelineSpy).not.toHaveBeenCalled();
    tree.unmount();
  });

  it('calls delete API for selected pipeline after delete dialog is confirmed', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').at(0).simulate('click');
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deletePipelineSpy).toHaveBeenLastCalledWith('test-pipeline-id0');
    tree.unmount();
  });

  it('updates the selected indices after a pipeline is deleted', async () => {
    const tree = await mountWithNPipelines(5);
    tree.find('.tableRow').at(0).simulate('click');
    expect(tree.state()).toHaveProperty('selectedIds', ['test-pipeline-id0']);
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(tree.state()).toHaveProperty('selectedIds', []);
    tree.unmount();
  });

  it('updates the selected indices after multiple pipelines are deleted', async () => {
    const tree = await mountWithNPipelines(5);
    tree.find('.tableRow').at(0).simulate('click');
    tree.find('.tableRow').at(3).simulate('click');
    expect(tree.state()).toHaveProperty('selectedIds', ['test-pipeline-id0', 'test-pipeline-id3']);
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(tree.state()).toHaveProperty('selectedIds', []);
    tree.unmount();
  });

  it('calls delete API for all selected pipelines after delete dialog is confirmed', async () => {
    const tree = await mountWithNPipelines(5);
    tree.find('.tableRow').at(0).simulate('click');
    tree.find('.tableRow').at(1).simulate('click');
    tree.find('.tableRow').at(4).simulate('click');
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deletePipelineSpy).toHaveBeenCalledTimes(3);
    expect(deletePipelineSpy).toHaveBeenCalledWith('test-pipeline-id0');
    expect(deletePipelineSpy).toHaveBeenCalledWith('test-pipeline-id1');
    expect(deletePipelineSpy).toHaveBeenCalledWith('test-pipeline-id4');
    tree.unmount();
  });

  it('shows snackbar confirmation after pipeline is deleted', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').at(0).simulate('click');
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      message: 'Successfully deleted 1 pipeline!',
      open: true,
    });
    tree.unmount();
  });

  it('shows error dialog when pipeline deletion fails', async () => {
    const tree = await mountWithNPipelines(1);
    tree.find('.tableRow').at(0).simulate('click');
    TestUtils.makeErrorResponseOnce(deletePipelineSpy, 'woops, failed');
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    const lastCall = updateDialogSpy.mock.calls[1][0];
    expect(lastCall).toMatchObject({
      content: 'Deleting pipeline: test pipeline name0 failed with error: "woops, failed"',
      title: 'Failed to delete 1 pipeline',
    });
    tree.unmount();
  });

  it('shows error dialog when multiple pipeline deletions fail', async () => {
    const tree = await mountWithNPipelines(5);
    tree.find('.tableRow').at(0).simulate('click');
    tree.find('.tableRow').at(2).simulate('click');
    tree.find('.tableRow').at(1).simulate('click');
    tree.find('.tableRow').at(3).simulate('click');
    deletePipelineSpy.mockImplementation(id => {
      if (id.indexOf(3) === -1 && id.indexOf(2) === -1) {
        throw {
          text: () => Promise.resolve('woops, failed!'),
        };
      }
    });
    const deleteBtn = (tree.instance() as PipelineList)
      .getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    // Should show only one error dialog for both pipelines (plus once for confirmation)
    expect(updateDialogSpy).toHaveBeenCalledTimes(2);
    const lastCall = updateDialogSpy.mock.calls[1][0];
    expect(lastCall).toMatchObject({
      content: 'Deleting pipeline: test pipeline name0 failed with error: "woops, failed!"\n\n' +
        'Deleting pipeline: test pipeline name1 failed with error: "woops, failed!"',
      title: 'Failed to delete 2 pipelines',
    });

    // Should show snackbar for the one successful deletion
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      message: 'Successfully deleted 2 pipelines!',
      open: true,
    });
    tree.unmount();
  });

  it('shows upload dialog when upload button is clicked', async () => {
    const tree = await mountWithNPipelines(0);
    const instance = tree.instance() as PipelineList;
    const uploadBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Upload pipeline');
    expect(uploadBtn).toBeDefined();
    await uploadBtn!.action();
    expect(instance.state).toHaveProperty('uploadDialogOpen', true);
    tree.unmount();
  });

  it('can dismiss upload dialog', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({ uploadDialogOpen: true });
    tree.find('UploadPipelineDialog').simulate('close');
    tree.update();
    expect(tree.state()).toHaveProperty('uploadDialogOpen', false);
    tree.unmount();
  });

  it('does not try to upload if no file is returned from upload dialog', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    const handlerSpy = jest.spyOn(tree.instance() as any, '_uploadDialogClosed');
    tree.setState({ uploadDialogOpen: true });
    tree.find('UploadPipelineDialog').simulate('close', 'some name', null);
    expect(handlerSpy).toHaveBeenLastCalledWith('some name', null);
    expect(uploadPipelineSpy).not.toHaveBeenCalled();
    tree.unmount();
  });

  it('tries to upload if a file is returned from upload dialog', async () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({ uploadDialogOpen: true });
    tree.find('UploadPipelineDialog').simulate('close', 'some name', { body: 'something' });
    tree.update();
    await uploadPipelineSpy;
    expect(uploadPipelineSpy).toHaveBeenLastCalledWith('some name', { body: 'something' });

    // Check the dialog is closed
    expect(tree.state()).toHaveProperty('uploadDialogOpen', false);
    tree.unmount();
  });

  it('shows error dialog and does not dismiss upload dialog when upload fails', async () => {
    TestUtils.makeErrorResponseOnce(uploadPipelineSpy, 'woops, could not upload');
    const tree = shallow(<PipelineList {...generateProps()} />);
    tree.setState({ uploadDialogOpen: true });
    tree.find('UploadPipelineDialog').simulate('close', 'some name', { body: 'something' });
    tree.update();
    await uploadPipelineSpy;
    await TestUtils.flushPromises();
    expect(uploadPipelineSpy).toHaveBeenLastCalledWith('some name', { body: 'something' });
    expect(updateDialogSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      content: 'woops, could not upload',
      title: 'Failed to upload pipeline',
    }));
    // Check the dialog is not closed
    expect(tree.state()).toHaveProperty('uploadDialogOpen', true);
    tree.unmount();
  });

});
