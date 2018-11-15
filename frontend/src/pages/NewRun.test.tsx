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
import NewRun from './NewRun';
import TestUtils from '../TestUtils';
import { shallow } from 'enzyme';
import { PageProps } from './Page';
import { Apis } from '../lib/Apis';
import { RoutePage, RouteParams } from '../components/Router';
import { ApiExperiment } from '../apis/experiment';
import { ApiPipeline } from '../apis/pipeline';
import { QUERY_PARAMS } from '../lib/URLParser';
import { ApiResourceType, ApiRunDetail, ApiParameter, ApiRelationship } from '../apis/run';

class TestNewRun extends NewRun {
  public _pipelineSelectionChanged(selectedId: string): void {
    return super._pipelineSelectionChanged(selectedId);
  }

  public async _pipelineSelectorClosed(confirmed: boolean): Promise<void> {
    return await super._pipelineSelectorClosed(confirmed);
  }
}

describe('NewRun', () => {
  const consoleErrorSpy = jest.spyOn(console, 'error');
  const createJobSpy = jest.spyOn(Apis.jobServiceApi, 'createJob');
  const createRunSpy = jest.spyOn(Apis.runServiceApi, 'createRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const historyPushSpy = jest.fn();
  const historyReplaceSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();

  const MOCK_EXPERIMENT = newMockExperiment();
  const MOCK_PIPELINE = newMockPipeline();
  const MOCK_RUN_DETAIL = newMockRunDetail();

  function newMockExperiment(): ApiExperiment {
    return {
      description: 'mock experiment description',
      id: 'some-mock-experiment-id',
      name: 'some mock experiment name',
    };
  }

  function newMockPipeline(): ApiPipeline {
    return {
      id: 'some-mock-pipeline-id',
      name: 'some mock pipeline name',
    };
  }

  function newMockRunDetail(): ApiRunDetail {
    return {
      pipeline_runtime: {
        workflow_manifest: '{}'
      },
      run: {
        id: 'some-mock-run-id',
        name: 'some mock run name',
        pipeline_spec: {
          pipeline_id: 'original-run-pipeline-id'
        },
      },
    };
  }

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: {
        pathname: RoutePage.NEW_RUN,
        // TODO: this should be removed once experiments are no longer required to reach this page.
        search: `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
      } as any,
      match: '' as any,
      toolbarProps: TestNewRun.prototype.getInitialToolbarState(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    // Reset mocks
    consoleErrorSpy.mockReset();
    createJobSpy.mockReset();
    createRunSpy.mockReset();
    getExperimentSpy.mockReset();
    getPipelineSpy.mockReset();
    getRunSpy.mockReset();
    historyPushSpy.mockReset();
    historyReplaceSpy.mockReset();
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();

    consoleErrorSpy.mockImplementation(() => null);
    createRunSpy.mockImplementation(() => ({ id: 'new-run-id' }));
    getExperimentSpy.mockImplementation(() => MOCK_EXPERIMENT);
    getPipelineSpy.mockImplementation(() => MOCK_PIPELINE);
    getRunSpy.mockImplementation(() => MOCK_RUN_DETAIL);
  });

  it('renders the new run page', async () => {
    const tree = shallow(<TestNewRun {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('does not include any action buttons in the toolbar', async () => {
    const props = generateProps();
    // Clear the experiment ID from the query params, as it used at some point to update the
    // breadcrumb, and we cover that in a later test.
    props.location.search = '';

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: [],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        { displayName: 'Start a new run', href: '' }
      ],
    });
    tree.unmount();
  });

  it('clears the banner when refresh is called', async () => {
    const tree = shallow(<TestNewRun {...generateProps() as any} />);
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    (tree.instance() as TestNewRun).refresh();
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('clears the banner when load is called', async () => {
    const tree = shallow(<TestNewRun {...generateProps() as any} />);
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    (tree.instance() as TestNewRun).load();
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('allows updating the run name', async () => {
    const tree = shallow(<TestNewRun {...generateProps() as any} />);
    await TestUtils.flushPromises();

    (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'run name' } });

    expect(tree.state()).toHaveProperty('runName', 'run name');
    tree.unmount();
  });

  it('allows updating the run description', async () => {
    const tree = shallow(<TestNewRun {...generateProps() as any} />);
    await TestUtils.flushPromises();
    (tree.instance() as TestNewRun).handleChange('description')({ target: { value: 'run description' } });

    expect(tree.state()).toHaveProperty('description', 'run description');
    tree.unmount();
  });

  it('exits to the AllRuns page if there is no associated experiment', async () => {
    const props = generateProps();
    // Clear query params which might otherwise include an experiment ID.
    props.location.search = '';

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();
    tree.find('#exitNewRunPageBtn').simulate('click');

    expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.RUNS);
    tree.unmount();
  });

  it('fetches the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(getExperimentSpy).toHaveBeenLastCalledWith(MOCK_EXPERIMENT.id);
    tree.unmount();
  });

  it('updates the run\'s state with the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('experiment', MOCK_EXPERIMENT);
    expect(tree.state()).toHaveProperty('experimentName', MOCK_EXPERIMENT.name);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('updates the breadcrumb with the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: [],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        {
          displayName: MOCK_EXPERIMENT.name,
          href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, MOCK_EXPERIMENT.id!),
        },
        { displayName: 'Start a new run', href: '' }
      ],
    });
    tree.unmount();
  });

  it('exits to the associated experiment\'s details page if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();
    tree.find('#exitNewRunPageBtn').simulate('click');

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, MOCK_EXPERIMENT.id!)
    );
    tree.unmount();
  });

  it('changes the exit button\'s text if query params indicate this is the first run of an experiment', async () => {
    const props = generateProps();
    props.location.search =
      `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
      + `&${QUERY_PARAMS.firstRunInExperiment}=1`;

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows a page error if getExperiment fails', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'test error message');

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'test error message',
      message: `Error: failed to retrieve associated experiment: ${MOCK_EXPERIMENT.id}. Click Details for more information.`,
      mode: 'error',
    }));
    tree.unmount();
  });

  it('fetches the associated pipeline if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('pipeline', MOCK_PIPELINE);
    expect(tree.state()).toHaveProperty('pipelineName', MOCK_PIPELINE.name);
    expect(tree.state()).toHaveProperty('errorMessage', 'Run name is required');
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows a page error if getPipeline fails', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

    const tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'test error message',
      message: `Error: failed to retrieve pipeline: ${MOCK_PIPELINE.id}. Click Details for more information.`,
      mode: 'error',
    }));
    tree.unmount();
  });

  describe('choosing a pipeline', () => {
    it('opens up the pipeline selector modal when users clicks \'Choose\'', async () => {
      const tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      tree.find('#choosePipelineBtn').at(0).simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('pipelineSelectorOpen')).toBe(true);

      // Close and flush this to avoid minor memory leak where PipelineSelector._loadPipelines is
      // called after the component is unmounted by tree.unmount().
      tree.setState({ pipelineSelectorOpen: false });
      await TestUtils.flushPromises();
      tree.unmount();
    });

    it('closes the pipeline selector modal', async () => {
      const tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      tree.find('#choosePipelineBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(true);

      tree.find('#cancelPipelineSelectionBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
      // Flush this to avoid minor memory leak where PipelineSelector._loadPipelines is called after
      // the component is unmounted by tree.unmount().
      await TestUtils.flushPromises();
      tree.unmount();
    });

    it('sets the pipeline ID from the selector modal when confirmed', async () => {
      const tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      const oldPipeline = newMockPipeline();
      oldPipeline.id = 'old-pipeline-id';
      oldPipeline.name = 'old-pipeline-name';
      const newPipeline = newMockPipeline();
      newPipeline.id = 'new-pipeline-id';
      newPipeline.name = 'new-pipeline-name';
      getPipelineSpy.mockImplementation(() => newPipeline);
      tree.setState({ pipeline: oldPipeline, pipelineName: oldPipeline.name });

      tree.find('#choosePipelineBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(true);

      // Simulate selecting pipeline
      const instance = tree.instance() as TestNewRun;
      instance._pipelineSelectionChanged(newPipeline.id);

      // Confirm pipeline selector
      tree.find('#usePipelineBtn').at(0).simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('pipelineSelectorOpen')).toBe(false);

      expect(tree.state('pipeline')).toEqual(newPipeline);
      expect(tree.state('pipelineName')).toEqual(newPipeline.name);
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
      tree.unmount();
    });

    it('does not set the pipeline ID from the selector modal when cancelled', async () => {
      const tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      const oldPipeline = newMockPipeline();
      oldPipeline.id = 'old-pipeline-id';
      oldPipeline.name = 'old-pipeline-name';
      const newPipeline = newMockPipeline();
      newPipeline.id = 'new-pipeline-id';
      newPipeline.name = 'new-pipeline-name';
      getPipelineSpy.mockImplementation(() => newPipeline);
      tree.setState({ pipeline: oldPipeline, pipelineName: oldPipeline.name });

      tree.find('#choosePipelineBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(true);

      // Simulate selecting pipeline
      const instance = tree.instance() as TestNewRun;
      instance._pipelineSelectionChanged(newPipeline.id);

      // Cancel pipeline selector
      tree.find('#cancelPipelineSelectionBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(false);

      expect(tree.state('pipeline')).toEqual(oldPipeline);
      expect(tree.state('pipelineName')).toEqual(oldPipeline.name);
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
      tree.unmount();
    });

    // TODO: Add test for when dialog is dismissed. Due to the particulars of how the Dialog element
    // works, this will not be possible until it's wrapped in some manner, like UploadPipelineDialog
    // in PipelineList

    it('shows an error dialog if fetching the selected pipeline fails', async () => {
      const tree = shallow(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test getPipeline error');

      const instance = tree.instance() as TestNewRun;
      const pipelineId = 'some-pipeline-id';
      instance._pipelineSelectionChanged(pipelineId);
      // Confirm pipeline selector
      await instance._pipelineSelectorClosed(true);
      await TestUtils.flushPromises();

      expect(updateDialogSpy).toHaveBeenCalledTimes(1);
      expect(updateDialogSpy.mock.calls[0][0]).toMatchObject({
        content: 'test getPipeline error',
        title: `Failed to retrieve pipeline with ID: ${pipelineId}`,
      });
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
      tree.unmount();
    });
  });

  describe('cloning from a run', () => {
    it('fetches the original run if an ID is present in the query params', async () => {
      const run = newMockRunDetail().run!;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${run.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(run.id);
      tree.unmount();
    });

    it('automatically generates the new run name based on the original run\'s name', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.name = '-original run-';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('runName')).toBe('Clone of -original run-');
      tree.unmount();
    });

    it('automatically generates the new clone name if the original run was a clone', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.name = 'Clone of some run';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('runName')).toBe('Clone (2) of some run');
      tree.unmount();
    });

    it('uses the query param experiment ID over the one in the original run if an ID is present in both', async () => {
      const experiment = newMockExperiment();
      const runDetail = newMockRunDetail();
      runDetail.run!.resource_references = [{
        key: { id: `${experiment.id}-different`, type: ApiResourceType.EXPERIMENT },
      }];
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`
        + `&${QUERY_PARAMS.experimentId}=${experiment.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(runDetail.run!.id);
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
      expect(getExperimentSpy).toHaveBeenLastCalledWith(experiment.id);
      tree.unmount();
    });

    it('uses the experiment ID in the original run if no experiment ID is present in query params', async () => {
      const originalRunExperimentId = 'original-run-experiment-id';
      const runDetail = newMockRunDetail();
      runDetail.run!.resource_references = [{
        key: { id: originalRunExperimentId, type: ApiResourceType.EXPERIMENT },
      }];
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(runDetail.run!.id);
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
      expect(getExperimentSpy).toHaveBeenLastCalledWith(originalRunExperimentId);
      tree.unmount();
    });

    it('retrieves the pipeline from the original run, even if there is a pipeline ID in the query params', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.pipeline_spec = { pipeline_id: 'original-run-pipeline-id' };
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`
        + `&${QUERY_PARAMS.pipelineId}=some-other-pipeline-id`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
      expect(getPipelineSpy).toHaveBeenLastCalledWith(runDetail.run!.pipeline_spec!.pipeline_id);
      tree.unmount();
    });

    it('shows a page error if getPipeline fails to find the pipeline from the original run', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${MOCK_RUN_DETAIL.run!.id}`;

      TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'test error message',
        message:
          'Error: failed to find a pipeline corresponding to that of the original run:'
          + ` ${MOCK_RUN_DETAIL.run!.id}. Click Details for more information.`,
        mode: 'error',
      }));
      tree.unmount();
    });

    it('logs an error if getPipeline fails to find the pipeline from the original run', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.pipeline_spec!.pipeline_id = undefined;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      // tslint:disable-next-line:no-console
      expect(consoleErrorSpy).toHaveBeenLastCalledWith('Original run did not have an associated pipeline ID');
      tree.unmount();
    });

    it('shows a page error if the original run\'s workflow_manifest is undefined', async () => {
      const runDetail = newMockRunDetail();
      runDetail.pipeline_runtime!.workflow_manifest = undefined;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        message: `Error: run ${runDetail.run!.id} had no workflow manifest`,
        mode: 'error',
      }));
      tree.unmount();
    });

    it('shows a page error if the original run\'s workflow_manifest is invalid JSON', async () => {
      const runDetail = newMockRunDetail();
      runDetail.pipeline_runtime!.workflow_manifest = 'not json';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        message: 'Error: failed to parse the original run\'s runtime. Click Details for more information.',
        mode: 'error',
      }));
      tree.unmount();
    });

    it('gets the pipeline parameter values of the original run\'s pipeline', async () => {
      const runDetail = newMockRunDetail();
      const originalRunPipelineParams: ApiParameter[] =
        [{ name: 'thisTestParam', value: 'thisTestVal' }];
      runDetail.pipeline_runtime!.workflow_manifest =
        JSON.stringify({
          spec: {
            arguments: {
              parameters: originalRunPipelineParams
            },
          },
        });
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('pipeline')).toHaveProperty('parameters', originalRunPipelineParams);
      tree.unmount();
    });


    it('shows a page error if getRun fails', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${MOCK_RUN_DETAIL.run!.id}`;

      TestUtils.makeErrorResponseOnce(getRunSpy, 'test error message');

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'test error message',
        message: `Error: failed to retrieve original run: ${MOCK_RUN_DETAIL.run!.id}. Click Details for more information.`,
        mode: 'error',
      }));
      tree.unmount();
    });

  });

  describe('creating a new run', () => {

    it('disables \'Create\' new run button by default', async () => {
      const tree = shallow(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', true);
      tree.unmount();
    });

    it('enables the \'Create\' new run button if pipeline ID in query params and run name entered', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'run name' } });
      await TestUtils.flushPromises();

      expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', false);
      tree.unmount();
    });

    it('re-disables the \'Create\' new run button if pipeline ID in query params and run name entered then cleared', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'run name' } });
      await TestUtils.flushPromises();
      expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', false);

      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: '' } });
      expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', true);

      tree.unmount();
    });

    it('sends a request to create a new run when \'Create\' is clicked', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
          + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      (tree.instance() as TestNewRun).handleChange('description')({ target: { value: 'test run description' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(createRunSpy).toHaveBeenCalledTimes(1);
      expect(createRunSpy).toHaveBeenLastCalledWith({
        description: 'test run description',
        name: 'test run name',
        pipeline_spec: {
          parameters: MOCK_PIPELINE.parameters,
          pipeline_id: MOCK_PIPELINE.id,
        },
        resource_references: [{
          key: {
            id: MOCK_EXPERIMENT.id,
            type: ApiResourceType.EXPERIMENT,
          },
          relationship: ApiRelationship.OWNER,
        }]
      });
      tree.unmount();
    });

    it('updates the pipeline in state when a user fills in its params', async () => {
      const props = generateProps();
      const pipeline = newMockPipeline();
      pipeline.parameters = [
        { name: 'param-1', value: '' },
        { name: 'param-2', value: 'prefilled value' },
      ];
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${pipeline.id}`;

      getPipelineSpy.mockImplementation(() => pipeline);

      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      // Fill in the first pipeline parameter
      tree.find('#newRunPipelineParam0').simulate('change', { target: { value: 'test param value' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(createRunSpy).toHaveBeenCalledTimes(1);
      expect(createRunSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        pipeline_spec: {
          parameters: [
            { name: 'param-1', value: 'test param value' },
            { name: 'param-2', value: 'prefilled value' },
          ],
          pipeline_id: pipeline.id,
        },
      }));
      expect(tree).toMatchSnapshot();
      tree.unmount();
    });

    it('updates the pipeline params as user selects different pipelines', async () => {
      const tree = shallow(<TestNewRun {...generateProps()} />);
      await TestUtils.flushPromises();

      // No parameters should be showing
      expect(tree).toMatchSnapshot();

      // Select a pipeline with parameters
      const pipelineWithParams = newMockPipeline();
      pipelineWithParams.id = 'pipeline-with-params';
      pipelineWithParams.parameters = [
        { name: 'param-1', value: 'prefilled value 1' },
        { name: 'param-2', value: 'prefilled value 2' },
      ];
      getPipelineSpy.mockImplementationOnce(() => pipelineWithParams);
      const instance = tree.instance() as TestNewRun;
      instance._pipelineSelectionChanged(pipelineWithParams.id!);
      instance._pipelineSelectorClosed(true);
      await TestUtils.flushPromises();
      expect(tree).toMatchSnapshot();

      // Select a new pipeline with no parameters
      const noParamsPipeline = newMockPipeline();
      noParamsPipeline.id = 'no-params-pipeline';
      noParamsPipeline.parameters = [];
      getPipelineSpy.mockImplementationOnce(() => noParamsPipeline);
      instance._pipelineSelectionChanged(noParamsPipeline.id!);
      instance._pipelineSelectorClosed(true);
      await TestUtils.flushPromises();
      expect(tree).toMatchSnapshot();
      tree.unmount();
    });

    it('sets the page to a busy state upon clicking \'Create\'', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(tree.state('isBeingCreated')).toBe(true);
      tree.unmount();
    });

    it('navigates to the ExperimentDetails page upon successful creation if there was an experiment', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(historyPushSpy).toHaveBeenCalledWith(
        RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, MOCK_EXPERIMENT.id!));
      tree.unmount();
    });

    it('navigates to the AllRuns page upon successful creation if there was not an experiment', async () => {
      const props = generateProps();
      // No experiment in query params
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.RUNS);
      tree.unmount();
    });

    it('shows an error dialog if creating the new run fails', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      TestUtils.makeErrorResponseOnce(createRunSpy, 'test error message');

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(updateDialogSpy).toHaveBeenCalledTimes(1);
      expect(updateDialogSpy.mock.calls[0][0]).toMatchObject({
        content: 'test error message',
        title: 'Run creation failed',
      });
      tree.unmount();
    });

    it('shows an error dialog if \'Create\' is clicked and the new run somehow has no pipeline', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(updateDialogSpy).toHaveBeenCalledTimes(1);
      expect(updateDialogSpy.mock.calls[0][0]).toMatchObject({
        content: 'Cannot create run without pipeline',
        title: 'Run creation failed',
      });
      tree.unmount();
    });

    it('unsets the page to a busy state if creation fails', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      TestUtils.makeErrorResponseOnce(createRunSpy, 'test error message');

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(tree.state('isBeingCreated')).toBe(false);
      tree.unmount();
    });

    it('shows snackbar confirmation after experiment is created', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
        message: 'Successfully created new Run: test run name',
        open: true,
      });
      tree.unmount();
    });

  });

  describe('creating a new recurring run', () => {

    it('changes the title if the new run will recur, based on query param', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.isRecurring}=1`;
      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateToolbarSpy).toHaveBeenLastCalledWith({
        actions: [],
        breadcrumbs: [
          { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
          { displayName: 'Start a recurring run', href: '' }
        ],
      });
      tree.unmount();
    });

    it('includes additional trigger input fields if run will be recurring', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.isRecurring}=1`;
      const tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(tree).toMatchSnapshot();
      tree.unmount();
    });

    it('sends a request to create a new recurring run with default periodic schedule when \'Create\' is clicked', async () => {

      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1`
        + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = TestUtils.mountWithRouter(<TestNewRun {...props} />);
      const instance = tree.instance() as TestNewRun;
      instance.handleChange('runName')({ target: { value: 'test run name' } });
      instance.handleChange('description')({ target: { value: 'test run description' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').at(0).simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(createRunSpy).toHaveBeenCalledTimes(0);
      expect(createJobSpy).toHaveBeenCalledTimes(1);
      expect(createJobSpy).toHaveBeenLastCalledWith({
        description: 'test run description',
        enabled: true,
        max_concurrency: '10',
        name: 'test run name',
        pipeline_spec: {
          parameters: MOCK_PIPELINE.parameters,
          pipeline_id: MOCK_PIPELINE.id,
        },
        resource_references: [{
          key: {
            id: MOCK_EXPERIMENT.id,
            type: ApiResourceType.EXPERIMENT,
          },
          relationship: ApiRelationship.OWNER,
        }],
        // Default trigger
        trigger: {
          periodic_schedule: {
            end_time: undefined,
            interval_second: '60',
            start_time: undefined,
          },
        },
      });
      tree.unmount();
    });

    it('displays an error message if periodic schedule end date/time is earlier than start date/time', async () => {
      const props = generateProps();
      props.location.search =
      `?${QUERY_PARAMS.isRecurring}=1`
      + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
      + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      tree.setState({
        trigger: {
          periodic_schedule: {
            end_time: new Date(2018, 4, 1),
            start_time: new Date(2018, 5, 1),
          },
        }
      });
      await TestUtils.flushPromises();

      expect(tree.state('errorMessage')).toBe('End date/time cannot be earlier than start date/time');
      tree.unmount();
    });

    it('displays an error message if cron schedule end date/time is earlier than start date/time', async () => {
      const props = generateProps();
      props.location.search =
      `?${QUERY_PARAMS.isRecurring}=1`
      + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
      + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      tree.setState({
        trigger: {
          cron_schedule: {
            end_time: new Date(2018, 4, 1),
            start_time: new Date(2018, 5, 1),
          },
        }
      });
      await TestUtils.flushPromises();

      expect(tree.state('errorMessage')).toBe('End date/time cannot be earlier than start date/time');
      tree.unmount();
    });

    it('displays an error message if max concurrent runs is negative', async () => {
      const props = generateProps();
      props.location.search =
      `?${QUERY_PARAMS.isRecurring}=1`
      + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
      + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      tree.setState({
        maxConcurrentRuns: '-1',
        trigger: {
          periodic_schedule: {
            interval_second: '60',
          }
        },
      });
      await TestUtils.flushPromises();

      expect(tree.state('errorMessage')).toBe('For triggered runs, maximum concurrent runs must be a positive number');
      tree.unmount();
    });

    it('displays an error message if max concurrent runs is not a number', async () => {
      const props = generateProps();
      props.location.search =
      `?${QUERY_PARAMS.isRecurring}=1`
      + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
      + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      const tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      tree.setState({
        maxConcurrentRuns: 'not a number',
        trigger: {
          periodic_schedule: {
            interval_second: '60',
          }
        },
      });
      await TestUtils.flushPromises();

      expect(tree.state('errorMessage')).toBe('For triggered runs, maximum concurrent runs must be a positive number');
      tree.unmount();
    });

  });
});
