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
import { shallow, ShallowWrapper, ReactWrapper } from 'enzyme';
import { PageProps } from './Page';
import { Apis } from '../lib/Apis';
import { RoutePage, RouteParams, QUERY_PARAMS } from '../components/Router';
import { ApiExperiment } from '../apis/experiment';
import { ApiPipeline } from '../apis/pipeline';
import { ApiResourceType, ApiRunDetail, ApiParameter, ApiRelationship } from '../apis/run';

class TestNewRun extends NewRun {
  public _experimentSelectorClosed = super._experimentSelectorClosed;
  public _pipelineSelectorClosed = super._pipelineSelectorClosed;
  public _updateRecurringRunState = super._updateRecurringRunState;
  public _handleParamChange = super._handleParamChange;
}

describe('NewRun', () => {

  let tree: ReactWrapper | ShallowWrapper;

  const consoleErrorSpy = jest.spyOn(console, 'error');
  const startJobSpy = jest.spyOn(Apis.jobServiceApi, 'createJob');
  const startRunSpy = jest.spyOn(Apis.runServiceApi, 'createRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const historyPushSpy = jest.fn();
  const historyReplaceSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();

  let MOCK_EXPERIMENT = newMockExperiment();
  let MOCK_PIPELINE = newMockPipeline();
  let MOCK_RUN_DETAIL = newMockRunDetail();
  let MOCK_RUN_WITH_EMBEDDED_PIPELINE = newMockRunWithEmbeddedPipeline();

  function newMockExperiment(): ApiExperiment {
    return {
      description: 'mock experiment description',
      id: 'some-mock-experiment-id',
      name: 'some mock experiment name',
    };
  }

  function newMockPipeline(): ApiPipeline {
    return {
      id: 'original-run-pipeline-id',
      name: 'original mock pipeline name',
      parameters: [],
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
          pipeline_id: 'original-run-pipeline-id',
          workflow_manifest: '{}',
        },
      },
    };
  }

  function newMockRunWithEmbeddedPipeline(): ApiRunDetail {
    const runDetail = newMockRunDetail();
    delete runDetail.run!.pipeline_spec!.pipeline_id;
    runDetail.run!.pipeline_spec!.workflow_manifest = '{"metadata": {"name": "embedded"}, "parameters": []}';
    return runDetail;
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
    jest.clearAllMocks();

    consoleErrorSpy.mockImplementation(() => null);
    startRunSpy.mockImplementation(() => ({ id: 'new-run-id' }));
    getExperimentSpy.mockImplementation(() => MOCK_EXPERIMENT);
    getPipelineSpy.mockImplementation(() => MOCK_PIPELINE);
    getRunSpy.mockImplementation(() => MOCK_RUN_DETAIL);

    MOCK_EXPERIMENT = newMockExperiment();
    MOCK_PIPELINE = newMockPipeline();
    MOCK_RUN_DETAIL = newMockRunDetail();
    MOCK_RUN_WITH_EMBEDDED_PIPELINE = newMockRunWithEmbeddedPipeline();
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    await tree.unmount();
    jest.resetAllMocks();
  });

  it('renders the new run page', async () => {
    tree = shallow(<TestNewRun {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
  });

  it('does not include any action buttons in the toolbar', async () => {
    const props = generateProps();
    // Clear the experiment ID from the query params, as it used at some point to update the
    // breadcrumb, and we cover that in a later test.
    props.location.search = '';

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: {},
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'Start a run',
    });
  });

  it('clears the banner when refresh is called', async () => {
    tree = shallow(<TestNewRun {...generateProps() as any} />);
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    (tree.instance() as TestNewRun).refresh();
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('clears the banner when load is called', async () => {
    tree = shallow(<TestNewRun {...generateProps() as any} />);
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    (tree.instance() as TestNewRun).load();
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('allows updating the run name', async () => {
    tree = shallow(<TestNewRun {...generateProps() as any} />);
    await TestUtils.flushPromises();

    (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'run name' } });

    expect(tree.state()).toHaveProperty('runName', 'run name');
  });

  it('allows updating the run description', async () => {
    tree = shallow(<TestNewRun {...generateProps() as any} />);
    await TestUtils.flushPromises();
    (tree.instance() as TestNewRun).handleChange('description')({ target: { value: 'run description' } });

    expect(tree.state()).toHaveProperty('description', 'run description');
  });

  it('changes title and form if the new run will recur, based on the radio buttons', async () => {
    // Default props do not include isRecurring in query params
    tree = shallow(<TestNewRun {...generateProps() as any} />);
    await TestUtils.flushPromises();

    (tree.instance() as TestNewRun)._updateRecurringRunState(true);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
  });

  it('changes title and form to default state if the new run is a one-off, based on the radio buttons', async () => {
    // Modify props to set page to recurring run form
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.isRecurring}=1`;
    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    (tree.instance() as TestNewRun)._updateRecurringRunState(false);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
  });

  it('exits to the AllRuns page if there is no associated experiment', async () => {
    const props = generateProps();
    // Clear query params which might otherwise include an experiment ID.
    props.location.search = '';

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();
    tree.find('#exitNewRunPageBtn').simulate('click');

    expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.RUNS);
  });

  it('fetches the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(getExperimentSpy).toHaveBeenLastCalledWith(MOCK_EXPERIMENT.id);
  });

  it('updates the run\'s state with the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('experiment', MOCK_EXPERIMENT);
    expect(tree.state()).toHaveProperty('experimentName', MOCK_EXPERIMENT.name);
    expect(tree).toMatchSnapshot();
  });

  it('updates the breadcrumb with the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: {},
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        {
          displayName: MOCK_EXPERIMENT.name,
          href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, MOCK_EXPERIMENT.id!),
        },
      ],
      pageTitle: 'Start a run',
    });
  });

  it('exits to the associated experiment\'s details page if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();
    tree.find('#exitNewRunPageBtn').simulate('click');

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, MOCK_EXPERIMENT.id!)
    );
  });

  it('changes the exit button\'s text if query params indicate this is the first run of an experiment', async () => {
    const props = generateProps();
    props.location.search =
      `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
      + `&${QUERY_PARAMS.firstRunInExperiment}=1`;

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
  });

  it('shows a page error if getExperiment fails', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'test error message');

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'test error message',
      message: `Error: failed to retrieve associated experiment: ${MOCK_EXPERIMENT.id}. Click Details for more information.`,
      mode: 'error',
    }));
  });

  it('fetches the associated pipeline if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('pipeline', MOCK_PIPELINE);
    expect(tree.state()).toHaveProperty('pipelineName', MOCK_PIPELINE.name);
    expect(tree.state()).toHaveProperty('errorMessage', 'Run name is required');
    expect(tree).toMatchSnapshot();
  });

  it('shows a page error if getPipeline fails', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

    tree = shallow(<TestNewRun {...props} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'test error message',
      message: `Error: failed to retrieve pipeline: ${MOCK_PIPELINE.id}. Click Details for more information.`,
      mode: 'error',
    }));
  });

  describe('choosing a pipeline', () => {
    it('opens up the pipeline selector modal when users clicks \'Choose\'', async () => {
      tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      tree.find('#choosePipelineBtn').at(0).simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('pipelineSelectorOpen')).toBe(true);
    });

    it('closes the pipeline selector modal', async () => {
      tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      tree.find('#choosePipelineBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(true);

      tree.find('#cancelPipelineSelectionBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
    });

    it('sets the pipeline from the selector modal when confirmed', async () => {
      tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
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
      tree.setState({ unconfirmedSelectedPipeline: newPipeline });

      // Confirm pipeline selector
      tree.find('#usePipelineBtn').at(0).simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('pipelineSelectorOpen')).toBe(false);

      expect(tree.state('pipeline')).toEqual(newPipeline);
      expect(tree.state('pipelineName')).toEqual(newPipeline.name);
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
    });

    it('does not set the pipeline from the selector modal when cancelled', async () => {
      tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
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
      tree.setState({ unconfirmedSelectedPipeline: newPipeline });

      // Cancel pipeline selector
      tree.find('#cancelPipelineSelectionBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(false);

      expect(tree.state('pipeline')).toEqual(oldPipeline);
      expect(tree.state('pipelineName')).toEqual(oldPipeline.name);
      expect(tree.state('pipelineSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
    });
  });

  describe('choosing an experiment', () => {
    it('opens up the experiment selector modal when users clicks \'Choose\'', async () => {
      tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      tree.find('#chooseExperimentBtn').at(0).simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('experimentSelectorOpen')).toBe(true);
    });

    it('closes the experiment selector modal', async () => {
      tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      tree.find('#chooseExperimentBtn').at(0).simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(true);

      tree.find('#cancelExperimentSelectionBtn').at(0).simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(false);
    });

    it('sets the experiment from the selector modal when confirmed', async () => {
      tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      const oldExperiment = newMockExperiment();
      oldExperiment.id = 'old-experiment-id';
      oldExperiment.name = 'old-experiment-name';
      const newExperiment = newMockExperiment();
      newExperiment.id = 'new-experiment-id';
      newExperiment.name = 'new-experiment-name';
      getExperimentSpy.mockImplementation(() => newExperiment);
      tree.setState({ experiment: oldExperiment, experimentName: oldExperiment.name });

      tree.find('#chooseExperimentBtn').at(0).simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(true);

      // Simulate selecting experiment
      tree.setState({ unconfirmedSelectedExperiment: newExperiment });

      // Confirm experiment selector
      tree.find('#useExperimentBtn').at(0).simulate('click');
      await TestUtils.flushPromises();
      expect(tree.state('experimentSelectorOpen')).toBe(false);

      expect(tree.state('experiment')).toEqual(newExperiment);
      expect(tree.state('experimentName')).toEqual(newExperiment.name);
      expect(tree.state('experimentSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
    });

    it('does not set the experiment from the selector modal when cancelled', async () => {
      tree = TestUtils.mountWithRouter(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      const oldExperiment = newMockExperiment();
      oldExperiment.id = 'old-experiment-id';
      oldExperiment.name = 'old-experiment-name';
      const newExperiment = newMockExperiment();
      newExperiment.id = 'new-experiment-id';
      newExperiment.name = 'new-experiment-name';
      getExperimentSpy.mockImplementation(() => newExperiment);
      tree.setState({ experiment: oldExperiment, experimentName: oldExperiment.name });

      tree.find('#chooseExperimentBtn').at(0).simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(true);

      // Simulate selecting experiment
      tree.setState({ unconfirmedSelectedExperiment: newExperiment });

      // Cancel experiment selector
      tree.find('#cancelExperimentSelectionBtn').at(0).simulate('click');
      expect(tree.state('experimentSelectorOpen')).toBe(false);

      expect(tree.state('experiment')).toEqual(oldExperiment);
      expect(tree.state('experimentName')).toEqual(oldExperiment.name);
      expect(tree.state('experimentSelectorOpen')).toBe(false);
      await TestUtils.flushPromises();
    });
  });

  // TODO: Add test for when dialog is dismissed. Due to the particulars of how the Dialog element
  // works, this will not be possible until it's wrapped in some manner, like UploadPipelineDialog
  // in PipelineList

  describe('cloning from a run', () => {
    it('fetches the original run if an ID is present in the query params', async () => {
      const run = newMockRunDetail().run!;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${run.id}`;

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(run.id);
    });

    it('automatically generates the new run name based on the original run\'s name', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.name = '-original run-';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('runName')).toBe('Clone of -original run-');
    });

    it('automatically generates the new clone name if the original run was a clone', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.name = 'Clone of some run';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('runName')).toBe('Clone (2) of some run');
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

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(runDetail.run!.id);
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
      expect(getExperimentSpy).toHaveBeenLastCalledWith(experiment.id);
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

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(runDetail.run!.id);
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
      expect(getExperimentSpy).toHaveBeenLastCalledWith(originalRunExperimentId);
    });

    it('retrieves the pipeline from the original run, even if there is a pipeline ID in the query params', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.pipeline_spec = { pipeline_id: 'original-run-pipeline-id' };
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`
        + `&${QUERY_PARAMS.pipelineId}=some-other-pipeline-id`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
      expect(getPipelineSpy).toHaveBeenLastCalledWith(runDetail.run!.pipeline_spec!.pipeline_id);
    });

    it('shows a page error if getPipeline fails to find the pipeline from the original run', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${MOCK_RUN_DETAIL.run!.id}`;

      TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'test error message',
        message:
          'Error: failed to find a pipeline corresponding to that of the original run:'
          + ` ${MOCK_RUN_DETAIL.run!.id}. Click Details for more information.`,
        mode: 'error',
      }));
    });

    it('shows an error if getPipeline fails to find the pipeline from the original run', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.pipeline_spec!.pipeline_id = undefined;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        message: 'Error: failed to read the clone run\'s pipeline definition. Click Details for more information.',
        mode: 'error',
      }));
    });

    it('does not call getPipeline if original run has pipeline spec instead of id', async () => {
      const runDetail = newMockRunDetail();
      delete runDetail.run!.pipeline_spec!.pipeline_id;
      runDetail.run!.pipeline_spec!.workflow_manifest = 'test workflow yaml';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(getPipelineSpy).not.toHaveBeenCalled();
    });

    it('shows a page error if parsing embedded pipeline yaml fails', async () => {
      const runDetail = newMockRunDetail();
      delete runDetail.run!.pipeline_spec!.pipeline_id;
      runDetail.run!.pipeline_spec!.workflow_manifest = '!definitely not yaml';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        message: 'Error: failed to read the clone run\'s pipeline definition. Click Details for more information.',
        mode: 'error',
      }));
    });

    it('loads and selects embedded pipeline from run', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id}`;

      getRunSpy.mockImplementation(() => MOCK_RUN_WITH_EMBEDDED_PIPELINE);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenCalledTimes(1);
      expect(tree.state('workflowFromRun')).toEqual({ metadata: { name: 'embedded' }, parameters: [] });
      expect(tree.state('useWorkflowFromRun')).toBe(true);
    });

    it('shows a page error if the original run\'s workflow_manifest is undefined', async () => {
      const runDetail = newMockRunDetail();
      runDetail.run!.pipeline_spec!.workflow_manifest = undefined;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        message: `Error: run ${runDetail.run!.id} had no workflow manifest`,
        mode: 'error',
      }));
    });

    it('shows a page error if the original run\'s workflow_manifest is invalid JSON', async () => {
      const runDetail = newMockRunWithEmbeddedPipeline();
      runDetail.run!.pipeline_spec!.workflow_manifest = 'not json';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        message: 'Error: failed to read the clone run\'s pipeline definition. Click Details for more information.',
        mode: 'error',
      }));
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

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(tree.state('parameters')).toEqual(originalRunPipelineParams);
    });


    it('shows a page error if getRun fails', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${MOCK_RUN_DETAIL.run!.id}`;

      TestUtils.makeErrorResponseOnce(getRunSpy, 'test error message');

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'test error message',
        message: `Error: failed to retrieve original run: ${MOCK_RUN_DETAIL.run!.id}. Click Details for more information.`,
        mode: 'error',
      }));
    });
  });

  describe('arriving from pipeline details page', () => {
    let mockEmbeddedPipelineProps: PageProps;
    beforeEach(() => {
      mockEmbeddedPipelineProps = generateProps();
      mockEmbeddedPipelineProps.location.search =
        `?${QUERY_PARAMS.fromRunId}=${MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id}`;
      getRunSpy.mockImplementationOnce(() => MOCK_RUN_WITH_EMBEDDED_PIPELINE);
    });

    it('indicates that a pipeline is preselected and provides a means of selecting a different pipeline', async () => {
      tree = shallow(<TestNewRun {...mockEmbeddedPipelineProps as any} />);
      await TestUtils.flushPromises();

      expect(tree.state('useWorkflowFromRun')).toBe(true);
      expect(tree.state('usePipelineFromRunLabel')).toBe('Using pipeline from previous page');
      expect(tree).toMatchSnapshot();
    });

    it('retrieves the run with the embedded pipeline', async () => {
      tree = shallow(<TestNewRun {...mockEmbeddedPipelineProps as any} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenLastCalledWith(MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id);
    });

    it('parses the embedded workflow and stores it in state', async () => {
      MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.pipeline_spec!.workflow_manifest = JSON.stringify(MOCK_PIPELINE);

      tree = shallow(<TestNewRun {...mockEmbeddedPipelineProps as any} />);
      await TestUtils.flushPromises();

      expect(tree.state('workflowFromRun')).toEqual(MOCK_PIPELINE);
      expect(tree.state('parameters')).toEqual(MOCK_PIPELINE.parameters);
      expect(tree.state('useWorkflowFromRun')).toBe(true);
    });

    it('displays a page error if it fails to parse the embedded pipeline', async () => {
      MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.pipeline_spec!.workflow_manifest = 'not JSON';

      tree = shallow(<TestNewRun {...mockEmbeddedPipelineProps as any} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'Unexpected token o in JSON at position 1',
        message: 'Error: failed to parse the embedded pipeline\'s spec: not JSON. Click Details for more information.',
        mode: 'error',
      }));
    });

    it('displays a page error if referenced run has no embedded pipeline', async () => {
      // Remove workflow_manifest entirely
      delete MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.pipeline_spec!.workflow_manifest;

      tree = shallow(<TestNewRun {...mockEmbeddedPipelineProps as any} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        message: `Error: somehow the run provided in the query params: ${MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id} had no embedded pipeline.`,
        mode: 'error',
      }));
    });

    it('displays a page error if it fails to retrieve the run containing the embedded pipeline', async () => {
      getRunSpy.mockReset();
      TestUtils.makeErrorResponseOnce(getRunSpy, 'test - error!');

      tree = shallow(<TestNewRun {...mockEmbeddedPipelineProps as any} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'test - error!',
        message: `Error: failed to retrieve the specified run: ${MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id}. Click Details for more information.`,
        mode: 'error',
      }));
    });
  });

  describe('starting a new run', () => {

    it('disables \'Start\' new run button by default', async () => {
      tree = shallow(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      expect(tree.find('#startNewRunBtn').props()).toHaveProperty('disabled', true);
    });

    it('enables the \'Start\' new run button if pipeline ID in query params and run name entered', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'run name' } });
      await TestUtils.flushPromises();

      expect(tree.find('#startNewRunBtn').props()).toHaveProperty('disabled', false);
    });

    it('re-disables the \'Start\' new run button if pipeline ID in query params and run name entered then cleared', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'run name' } });
      await TestUtils.flushPromises();
      expect(tree.find('#startNewRunBtn').props()).toHaveProperty('disabled', false);

      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: '' } });
      expect(tree.find('#startNewRunBtn').props()).toHaveProperty('disabled', true);
    });

    it('sends a request to Start a run when \'Start\' is clicked', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      (tree.instance() as TestNewRun).handleChange('description')({ target: { value: 'test run description' } });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(1);
      expect(startRunSpy).toHaveBeenLastCalledWith({
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
    });

    it('updates the parameters in state on handleParamChange', async () => {
      const props = generateProps();
      const pipeline = newMockPipeline();
      pipeline.parameters = [
        { name: 'param-1', value: '' },
        { name: 'param-2', value: 'prefilled value' },
      ];
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${pipeline.id}`;

      getPipelineSpy.mockImplementation(() => pipeline);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      // Fill in the first pipeline parameter
      (tree.instance() as TestNewRun)._handleParamChange(0, 'test param value');

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(1);
      expect(startRunSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        pipeline_spec: {
          parameters: [
            { name: 'param-1', value: 'test param value' },
            { name: 'param-2', value: 'prefilled value' },
          ],
          pipeline_id: pipeline.id,
        },
      }));
      expect(tree).toMatchSnapshot();
    });

    it('copies pipeline from run in the start API call when cloning a run with embedded pipeline', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.id}`;

      getRunSpy.mockImplementation(() => MOCK_RUN_WITH_EMBEDDED_PIPELINE);

      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(1);
      expect(startRunSpy).toHaveBeenLastCalledWith({
        description: '',
        name: 'Clone of ' + MOCK_RUN_WITH_EMBEDDED_PIPELINE.run!.name,
        pipeline_spec: {
          parameters: [],
          pipeline_id: undefined,
          workflow_manifest: '{"metadata":{"name":"embedded"},"parameters":[]}',
        },
        resource_references: [],
      });
      expect(tree).toMatchSnapshot();
    });

    it('updates the pipeline params as user selects different pipelines', async () => {
      tree = shallow(<TestNewRun {...generateProps()} />);
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
      tree.setState({ unconfirmedSelectedPipeline: pipelineWithParams });
      const instance = tree.instance() as TestNewRun;
      instance._pipelineSelectorClosed(true);
      await TestUtils.flushPromises();
      expect(tree).toMatchSnapshot();

      // Select a new pipeline with no parameters
      const noParamsPipeline = newMockPipeline();
      noParamsPipeline.id = 'no-params-pipeline';
      noParamsPipeline.parameters = [];
      getPipelineSpy.mockImplementationOnce(() => noParamsPipeline);
      tree.setState({ unconfirmedSelectedPipeline: noParamsPipeline });
      instance._pipelineSelectorClosed(true);
      await TestUtils.flushPromises();
      expect(tree).toMatchSnapshot();
    });

    it('trims whitespace from the pipeline params', async () => {
      tree = shallow(<TestNewRun {...generateProps()} />);
      await TestUtils.flushPromises();

      // Select a pipeline with parameters
      const pipelineWithParams = newMockPipeline();
      pipelineWithParams.id = 'pipeline-with-params';
      pipelineWithParams.parameters = [
        { name: 'param-1', value: '  whitespace on either side  ' },
        { name: 'param-2', value: 'value 2' },
      ];
      getPipelineSpy.mockImplementationOnce(() => pipelineWithParams);
      tree.setState({ unconfirmedSelectedPipeline: pipelineWithParams });
      const instance = tree.instance() as TestNewRun;
      instance._pipelineSelectorClosed(true);
      tree.find('#startNewRunBtn').simulate('click');
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(1);
      expect(startRunSpy).toHaveBeenLastCalledWith({
        description: '',
        name: '',
        pipeline_spec: {
          parameters: [
            { name: 'param-1', value: 'whitespace on either side' },
            { name: 'param-2', value: 'value 2' },
          ],
          pipeline_id: 'pipeline-with-params',
        },
        resource_references: [{
          key: {
            id: MOCK_EXPERIMENT.id,
            type: ApiResourceType.EXPERIMENT,
          },
          relationship: ApiRelationship.OWNER,
        }]
      });
    });

    it('sets the page to a busy state upon clicking \'Start\'', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');

      expect(tree.state('isBeingStarted')).toBe(true);
    });

    it('navigates to the ExperimentDetails page upon successful start if there was an experiment', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(historyPushSpy).toHaveBeenCalledWith(
        RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, MOCK_EXPERIMENT.id!));
    });

    it('navigates to the AllRuns page upon successful start if there was not an experiment', async () => {
      const props = generateProps();
      // No experiment in query params
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.RUNS);
    });

    it('shows an error dialog if Starting the new run fails', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      TestUtils.makeErrorResponseOnce(startRunSpy, 'test error message');

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(updateDialogSpy).toHaveBeenCalledTimes(1);
      expect(updateDialogSpy.mock.calls[0][0]).toMatchObject({
        content: 'test error message',
        title: 'Run creation failed',
      });
    });

    it('shows an error dialog if \'Start\' is clicked and the new run somehow has no pipeline', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`;

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(updateDialogSpy).toHaveBeenCalledTimes(1);
      expect(updateDialogSpy.mock.calls[0][0]).toMatchObject({
        content: 'Cannot start run without pipeline',
        title: 'Run creation failed',
      });
    });

    it('unsets the page to a busy state if starting run fails', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      TestUtils.makeErrorResponseOnce(startRunSpy, 'test error message');

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(tree.state('isBeingStarted')).toBe(false);
    });

    it('shows snackbar confirmation after experiment is started', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
      (tree.instance() as TestNewRun).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
        message: 'Successfully started new Run: test run name',
        open: true,
      });
    });
  });

  describe('starting a new recurring run', () => {

    it('changes the title if the new run will recur, based on query param', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.isRecurring}=1`;
      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(updateToolbarSpy).toHaveBeenLastCalledWith({
        actions: {},
        breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
        pageTitle: 'Start a recurring run',
      });
    });

    it('includes additional trigger input fields if run will be recurring', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.isRecurring}=1`;
      tree = shallow(<TestNewRun {...props} />);
      await TestUtils.flushPromises();

      expect(tree).toMatchSnapshot();
    });

    it('sends a request to start a new recurring run with default periodic schedule when \'Start\' is clicked', async () => {

      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1`
        + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = TestUtils.mountWithRouter(<TestNewRun {...props} />);
      const instance = tree.instance() as TestNewRun;
      instance.handleChange('runName')({ target: { value: 'test run name' } });
      instance.handleChange('description')({ target: { value: 'test run description' } });
      await TestUtils.flushPromises();

      tree.find('#startNewRunBtn').at(0).simulate('click');
      // The start APIs are called in a callback triggered by clicking 'Start', so we wait again
      await TestUtils.flushPromises();

      expect(startRunSpy).toHaveBeenCalledTimes(0);
      expect(startJobSpy).toHaveBeenCalledTimes(1);
      expect(startJobSpy).toHaveBeenLastCalledWith({
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
    });

    it('displays an error message if periodic schedule end date/time is earlier than start date/time', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1`
        + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
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
    });

    it('displays an error message if cron schedule end date/time is earlier than start date/time', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1`
        + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
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
    });

    it('displays an error message if max concurrent runs is negative', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1`
        + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
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
    });

    it('displays an error message if max concurrent runs is not a number', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1`
        + `&${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.id}`
        + `&${QUERY_PARAMS.pipelineId}=${MOCK_PIPELINE.id}`;

      tree = shallow(<TestNewRun {...props} />);
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
    });

  });
});
