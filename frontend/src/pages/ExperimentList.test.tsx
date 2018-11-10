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
import ExperimentList from './ExperimentList';
import TestUtils from '../TestUtils';
import { ApiResourceType } from '../apis/run';
import { Apis } from '../lib/Apis';
import { ExpandState } from '../components/CustomTable';
import { NodePhase } from './Status';
import { PageProps } from './Page';
import { QUERY_PARAMS } from '../lib/URLParser';
import { RoutePage } from '../components/Router';
import { range } from 'lodash';
import { shallow, ReactWrapper } from 'enzyme';

describe('ExperimentList', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const listExperimentsSpy = jest.spyOn(Apis.experimentServiceApi, 'listExperiment');
  const listRunsSpy = jest.spyOn(Apis.runServiceApi, 'listRuns');

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: '' as any,
      toolbarProps: ExperimentList.prototype.getInitialToolbarState(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  async function mountWithNExperiments(n: number, nRuns: number): Promise<ReactWrapper> {
    listExperimentsSpy.mockImplementation(() => ({
      experiments: range(n).map(i => ({ id: 'test-experiment-id' + i, name: 'test experiment name' + i })),
    }));
    listRunsSpy.mockImplementation(() => ({
      runs: range(nRuns).map(i => ({ id: 'test-run-id' + i, name: 'test run name' + i })),
    }));
    const tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    await listExperimentsSpy;
    await listRunsSpy;
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
    listExperimentsSpy.mockReset();
    listRunsSpy.mockReset();
  });

  it('renders an empty list with empty state message', () => {
    const tree = shallow(<ExperimentList {...generateProps()} />);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('renders a list of one experiment', async () => {
    const tree = shallow(<ExperimentList {...generateProps()} />);
    tree.setState({
      displayExperiments: [{
        description: 'test experiment description',
        expandState: ExpandState.COLLAPSED,
        name: 'test experiment name',
      }]
    });
    await listExperimentsSpy;
    await listRunsSpy;
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('renders a list of one experiment with no description', async () => {
    const tree = shallow(<ExperimentList {...generateProps()} />);
    tree.setState({
      experiments: [{
        expandState: ExpandState.COLLAPSED,
        name: 'test experiment name',
      }]
    });
    await listExperimentsSpy;
    await listRunsSpy;
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('renders a list of one experiment with error', async () => {
    const tree = shallow(<ExperimentList {...generateProps()} />);
    tree.setState({
      experiments: [{
        description: 'test experiment description',
        error: 'oops! could not load experiment',
        expandState: ExpandState.COLLAPSED,
        name: 'test experiment name',
      }]
    });
    await listExperimentsSpy;
    await listRunsSpy;
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('calls Apis to list experiments, sorted by creation time in descending order', async () => {
    const tree = await mountWithNExperiments(1, 1);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(listRunsSpy).toHaveBeenLastCalledWith(undefined, 5, 'created_at desc',
      ApiResourceType.EXPERIMENT.toString(), 'test-experiment-id0');
    expect(tree.state()).toHaveProperty('displayExperiments', [{
      expandState: ExpandState.COLLAPSED,
      id: 'test-experiment-id0',
      last5Runs: [{ id: 'test-run-id0', name: 'test run name0' }],
      name: 'test experiment name0',
    }]);
    tree.unmount();
  });

  it('has a Refresh button, clicking it refreshes the experiment list', async () => {
    const tree = await mountWithNExperiments(1, 1);
    const instance = tree.instance() as ExperimentList;
    expect(listExperimentsSpy.mock.calls.length).toBe(1);
    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    await refreshBtn!.action();
    expect(listExperimentsSpy.mock.calls.length).toBe(2);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
    tree.unmount();
  });

  it('shows error banner when listing experiments fails', async () => {
    TestUtils.makeErrorResponseOnce(listExperimentsSpy, 'bad stuff happened');
    const tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    await listExperimentsSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of experiments. Click Details for more information.',
      mode: 'error',
    }));
    tree.unmount();
  });

  it('shows error next to experiment when listing its last 5 runs fails', async () => {
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementation();

    listExperimentsSpy.mockImplementationOnce(() => ({ experiments: [{ name: 'exp1' }] }));
    TestUtils.makeErrorResponseOnce(listRunsSpy, 'bad stuff happened');
    const tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    await listExperimentsSpy;
    await TestUtils.flushPromises();
    expect(tree.state()).toHaveProperty('displayExperiments', [{
      error: 'Failed to load the last 5 runs of this experiment',
      expandState: 0,
      name: 'exp1',
    }]);
    tree.unmount();
  });

  it('shows error banner when listing experiments fails after refresh', async () => {
    const tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    const instance = tree.instance() as ExperimentList;
    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    TestUtils.makeErrorResponseOnce(listExperimentsSpy, 'bad stuff happened');
    await refreshBtn!.action();
    expect(listExperimentsSpy.mock.calls.length).toBe(2);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of experiments. Click Details for more information.',
      mode: 'error',
    }));
    tree.unmount();
  });

  it('hides error banner when listing experiments fails then succeeds', async () => {
    TestUtils.makeErrorResponseOnce(listExperimentsSpy, 'bad stuff happened');
    const tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    const instance = tree.instance() as ExperimentList;
    await listExperimentsSpy;
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'bad stuff happened',
      message: 'Error: failed to retrieve list of experiments. Click Details for more information.',
      mode: 'error',
    }));
    updateBannerSpy.mockReset();

    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    listExperimentsSpy.mockImplementationOnce(() => ({ experiments: [{ name: 'experiment1' }] }));
    listRunsSpy.mockImplementationOnce(() => ({ runs: [{ name: 'run1' }] }));
    await refreshBtn!.action();
    expect(listExperimentsSpy.mock.calls.length).toBe(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
    tree.unmount();
  });

  it('can expand an experiment to see its runs', async () => {
    const tree = await mountWithNExperiments(1, 1);
    tree.find('.tableRow button').at(0).simulate('click');
    expect(tree.state()).toHaveProperty('displayExperiments', [{
      expandState: ExpandState.EXPANDED,
      id: 'test-experiment-id0',
      last5Runs: [{ id: 'test-run-id0', name: 'test run name0' }],
      name: 'test experiment name0',
    }]);
    tree.unmount();
  });

  it('renders a list of runs for given experiment', async () => {
    const tree = shallow(<ExperimentList {...generateProps()} />);
    tree.setState({ displayExperiments: [{ last5Runs: [{ id: 'run1id' }, { id: 'run2id' }] }] });
    const runListTree = (tree.instance() as any)._getExpandedExperimentComponent(0);
    expect(runListTree.props.runIdListMask).toEqual(['run1id', 'run2id']);
  });

  it('navigates to new experiment page when Create experiment button is clicked', async () => {
    const tree = TestUtils.mountWithRouter(<ExperimentList {...generateProps()} />);
    const createBtn = (tree.instance() as ExperimentList)
      .getInitialToolbarState().actions.find(b => b.title === 'Create experiment');
    await createBtn!.action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.NEW_EXPERIMENT);
  });

  it('always has new experiment button enabled', async () => {
    const tree = await mountWithNExperiments(1, 1);
    const calls = updateToolbarSpy.mock.calls[0];
    expect(calls[0].actions.find((b: any) => b.title === 'Create experiment')).not.toHaveProperty('disabled');
    tree.unmount();
  });

  it('enables clone button when one run is selected', async () => {
    const tree = await mountWithNExperiments(1, 1);
    (tree.instance() as any)._runSelectionChanged(['run1']);
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    expect(updateToolbarSpy.mock.calls[0][0].actions.find((b: any) => b.title === 'Clone run'))
      .toHaveProperty('disabled', true);
    expect(updateToolbarSpy.mock.calls[1][0].actions.find((b: any) => b.title === 'Clone run'))
      .toHaveProperty('disabled', false);
    tree.unmount();
  });

  it('disables clone button when more than one run is selected', async () => {
    const tree = await mountWithNExperiments(1, 1);
    (tree.instance() as any)._runSelectionChanged(['run1', 'run2']);
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    expect(updateToolbarSpy.mock.calls[0][0].actions.find((b: any) => b.title === 'Clone run'))
      .toHaveProperty('disabled', true);
    expect(updateToolbarSpy.mock.calls[1][0].actions.find((b: any) => b.title === 'Clone run'))
      .toHaveProperty('disabled', true);
    tree.unmount();
  });

  it('enables compare runs button only when more than one is selected', async () => {
    const tree = await mountWithNExperiments(1, 1);
    (tree.instance() as any)._runSelectionChanged(['run1']);
    (tree.instance() as any)._runSelectionChanged(['run1', 'run2']);
    (tree.instance() as any)._runSelectionChanged(['run1', 'run2', 'run3']);
    expect(updateToolbarSpy).toHaveBeenCalledTimes(4);
    expect(updateToolbarSpy.mock.calls[0][0].actions.find((b: any) => b.title === 'Compare runs'))
      .toHaveProperty('disabled', true);
    expect(updateToolbarSpy.mock.calls[1][0].actions.find((b: any) => b.title === 'Compare runs'))
      .toHaveProperty('disabled', true);
    expect(updateToolbarSpy.mock.calls[2][0].actions.find((b: any) => b.title === 'Compare runs'))
      .toHaveProperty('disabled', false);
    expect(updateToolbarSpy.mock.calls[3][0].actions.find((b: any) => b.title === 'Compare runs'))
      .toHaveProperty('disabled', false);
    tree.unmount();
  });

  it('navigates to compare page with the selected run ids', async () => {
    const tree = await mountWithNExperiments(1, 1);
    (tree.instance() as any)._runSelectionChanged(['run1', 'run2', 'run3']);
    const compareBtn = (tree.instance() as ExperimentList)
      .getInitialToolbarState().actions.find(b => b.title === 'Compare runs');
    await compareBtn!.action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      `${RoutePage.COMPARE}?${QUERY_PARAMS.runlist}=run1,run2,run3`);
  });

  it('navigates to new run page with the selected run id for cloning', async () => {
    const tree = await mountWithNExperiments(1, 1);
    (tree.instance() as any)._runSelectionChanged(['run1']);
    const cloneBtn = (tree.instance() as ExperimentList)
      .getInitialToolbarState().actions.find(b => b.title === 'Clone run');
    await cloneBtn!.action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      `${RoutePage.NEW_RUN}?${QUERY_PARAMS.cloneFromRun}=run1`);
  });

  it('renders experiment names as links to their details pages', async () => {
    const tree = TestUtils.mountWithRouter((ExperimentList.prototype as any)
      ._nameCustomRenderer('experiment name', 'experiment-id'));
    expect(tree).toMatchSnapshot();
  });

  it('renders last 5 runs statuses', async () => {
    const tree = shallow((ExperimentList.prototype as any)
      ._last5RunsCustomRenderer([
        { status: NodePhase.SUCCEEDED },
        { status: NodePhase.PENDING },
        { status: NodePhase.FAILED },
        { status: NodePhase.UNKNOWN },
        { status: NodePhase.SUCCEEDED },
      ]));
    expect(tree).toMatchSnapshot();
  });
});
