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
import { range } from 'lodash';

class TestNewRun extends NewRun {
  public _pipelineSelectionChanged(selectedId: string): void {
    return super._pipelineSelectionChanged(selectedId);
  }

  public async _pipelineSelectorClosed(confirmed: boolean): Promise<void> {
    return await super._pipelineSelectorClosed(confirmed);
  }
}

describe('NewRun', () => {
  const createJobSpy = jest.spyOn(Apis.jobServiceApi, 'createJob');
  const createRunSpy = jest.spyOn(Apis.runServiceApi, 'createRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const historyPushSpy = jest.fn();
  const historyReplaceSpy = jest.fn();
  const listPipelinesSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();

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
        search: `?${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`
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
    createJobSpy.mockReset();
    createRunSpy.mockReset();
    getExperimentSpy.mockReset();
    getPipelineSpy.mockReset();
    getRunSpy.mockReset();
    historyPushSpy.mockReset();
    historyReplaceSpy.mockReset();
    listPipelinesSpy.mockReset();
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();

    createRunSpy.mockImplementation(() => ({ id: 'new-run-id' }));
    getExperimentSpy.mockImplementation(() => newMockExperiment());
    getPipelineSpy.mockImplementation(() => newMockPipeline());
    getRunSpy.mockImplementation(() => newMockRunDetail());
  });

  it('renders the new run page', async () => {
    const tree = shallow(<TestNewRun {...generateProps() as any} />);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('does not include any action buttons in the toolbar', async () => {
    const props = generateProps();
    // Clear the experiment ID from the query params, as it used at some point to update the
    // breadcrumb, and we cover that in a later test.
    props.location.search = '';

    const tree = shallow(<TestNewRun {...props as any} />);
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

    (tree.instance() as any).handleChange('runName')({ target: { value: 'run name' } });

    expect(tree.state()).toHaveProperty('runName', 'run name');
    tree.unmount();
  });

  it('allows updating the run description', async () => {
    const tree = shallow(<TestNewRun {...generateProps() as any} />);
    await TestUtils.flushPromises();
    (tree.instance() as any).handleChange('description')({ target: { value: 'run description' } });

    expect(tree.state()).toHaveProperty('description', 'run description');
    tree.unmount();
  });

  it('exits to the AllRuns page if there is no associated experiment', async () => {
    const props = generateProps();
    // Clear query params which might otherwise include an experiment ID.
    props.location.search = '';

    const tree = shallow(<TestNewRun {...props as any} />);
    await TestUtils.flushPromises();
    tree.find('#exitNewRunPageBtn').simulate('click');

    expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.RUNS);
    tree.unmount();
  });

  it('fetches the associated experiment if one is present in the query params', async () => {
    const experiment = newMockExperiment();
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${experiment.id}`;

    getExperimentSpy.mockImplementation(() => experiment);

    const tree = shallow(<TestNewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(getExperimentSpy).toHaveBeenLastCalledWith(experiment.id);
    tree.unmount();
  });

  it('updates the run\'s state with the associated experiment if one is present in the query params', async () => {
    const experiment = newMockExperiment();
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${experiment.id}`;

    getExperimentSpy.mockImplementation(() => experiment);

    const tree = shallow(<TestNewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('experiment', experiment);
    expect(tree.state()).toHaveProperty('experimentName', experiment.name);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('updates the breadcrumb with the associated experiment if one is present in the query params', async () => {
    const experiment = newMockExperiment();
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${experiment.id}`;

    getExperimentSpy.mockImplementation(() => experiment);

    const tree = shallow(<TestNewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: [],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        {
          displayName: experiment.name,
          href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experiment.id!)
        },
        { displayName: 'Start a new run', href: '' }
      ],
    });
    tree.unmount();
  });

  it('exits to the associated experiment\'s details page if one is present in the query params', async () => {
    const experiment = newMockExperiment();
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${experiment.id}`;

    getExperimentSpy.mockImplementation(() => experiment);

    const tree = shallow(<TestNewRun {...props as any} />);
    await TestUtils.flushPromises();
    tree.find('#exitNewRunPageBtn').simulate('click');

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experiment.id!)
    );
    tree.unmount();
  });

  it('changes the exit button\'s text if query params indicate this is the first run of an experiment', async () => {
    const experiment = newMockExperiment();
    const props = generateProps();
    props.location.search =
      `?${QUERY_PARAMS.experimentId}=${experiment.id}`
      + `&${QUERY_PARAMS.firstRunInExperiment}=1`;

    getExperimentSpy.mockImplementation(() => experiment);

    const tree = shallow(<TestNewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows a page error if getExperiment fails', async () => {
    // Don't actually log to console.
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

    const experiment = newMockExperiment();
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${experiment.id}`;

    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'test error message');

    const tree = shallow(<TestNewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'test error message',
      message: `Error: failed to retrieve associated experiment: ${experiment.id}. Click Details for more information.`,
      mode: 'error',
    }));
    tree.unmount();
  });

  it('fetches the associated pipeline if one is present in the query params', async () => {
    const pipeline = newMockPipeline();
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${pipeline.id}`;

    getPipelineSpy.mockImplementation(() => pipeline);

    const tree = shallow(<TestNewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('pipeline', pipeline);
    expect(tree.state()).toHaveProperty('pipelineName', pipeline.name);
    expect(tree.state()).toHaveProperty('errorMessage', 'Run name is required');
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows a page error if getPipeline fails', async () => {
    // Don't actually log to console.
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

    const pipeline = newMockPipeline();
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${pipeline.id}`;

    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

    const tree = shallow(<TestNewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'test error message',
      message: `Error: failed to retrieve pipeline: ${pipeline.id}. Click Details for more information.`,
      mode: 'error',
    }));
    tree.unmount();
  });

  describe('choosing a pipeline', () => {

    beforeEach(() => {
      listPipelinesSpy.mockImplementation(() => ({
        pipelines: range(5).map(i => ({ id: 'test-pipeline-id' + i, name: 'test pipeline name' + i })),
      }));
    });

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
      const instance = tree.instance() as TestNewRun;
      instance._pipelineSelectionChanged(newPipeline.id);

      tree.find('#choosePipelineBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(true);
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
      const instance = tree.instance() as TestNewRun;
      instance._pipelineSelectionChanged(newPipeline.id);

      tree.find('#choosePipelineBtn').at(0).simulate('click');
      expect(tree.state('pipelineSelectorOpen')).toBe(true);
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

    it('shows a page error if fetching the selected pipeline fails', async () => {
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

      const tree = shallow(<TestNewRun {...generateProps() as any} />);
      await TestUtils.flushPromises();

      TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test getPipeline error');

      const instance = tree.instance() as TestNewRun;
      const pipelineId = 'some-pipeline-id';
      instance._pipelineSelectionChanged(pipelineId);
      // Confirm pipeline selector
      await instance._pipelineSelectorClosed(true);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'test getPipeline error',
        message:
          `Error: failed to retrieve pipeline with ID: ${pipelineId}.`
          + ' Click Details for more information.',
        mode: 'error',
      }));
      tree.unmount();
    });
  });

  describe('cloning from a run', () => {
    it('fetches the original run if an ID is present in the query params', async () => {
      const run = newMockRunDetail().run!;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${run.id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
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

      const tree = shallow(<TestNewRun {...props as any} />);
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

      const tree = shallow(<TestNewRun {...props as any} />);
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

      const tree = shallow(<TestNewRun {...props as any} />);
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

      const tree = shallow(<TestNewRun {...props as any} />);
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

      const tree = shallow(<TestNewRun {...props as any} />);
      await TestUtils.flushPromises();

      expect(getPipelineSpy).toHaveBeenCalledTimes(1);
      expect(getPipelineSpy).toHaveBeenLastCalledWith(runDetail.run!.pipeline_spec!.pipeline_id);
      tree.unmount();
    });

    it('shows a page error if getPipeline fails to find the pipeline from the original run', async () => {
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

      const runDetail = newMockRunDetail();
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);
      TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

      const tree = shallow(<TestNewRun {...props as any} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'test error message',
        message:
          'Error: failed to find a pipeline corresponding to that of the original run:'
          + ` ${runDetail.run!.id}. Click Details for more information.`,
        mode: 'error',
      }));
      tree.unmount();
    });

    it('does not show an error if getPipeline fails to find the pipeline from the original run', async () => {
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.log = jest.spyOn(console, 'log').mockImplementationOnce(() => null);

      const runDetail = newMockRunDetail();
      runDetail.run!.pipeline_spec!.pipeline_id = undefined;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props as any} />);
      await TestUtils.flushPromises();

      // tslint:disable-next-line:no-console
      expect(console.log).toHaveBeenLastCalledWith('Original run did not have an associated pipeline ID');
      tree.unmount();
    });

    it('shows a page error if the original run\'s workflow_manifest is undefined', async () => {
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

      const runDetail = newMockRunDetail();
      runDetail.pipeline_runtime!.workflow_manifest = undefined;
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props as any} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        message: `Error: run ${runDetail.run!.id} had no workflow manifest`,
        mode: 'error',
      }));
      tree.unmount();
    });

    it('shows a page error if the original run\'s workflow_manifest is invalid JSON', async () => {
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

      const runDetail = newMockRunDetail();
      runDetail.pipeline_runtime!.workflow_manifest = 'not json';
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      getRunSpy.mockImplementation(() => runDetail);

      const tree = shallow(<TestNewRun {...props as any} />);
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

      const tree = shallow(<TestNewRun {...props as any} />);
      await TestUtils.flushPromises();

      expect(tree.state('pipeline')).toHaveProperty('parameters', originalRunPipelineParams);
      tree.unmount();
    });


    it('shows a page error if getRun fails', async () => {
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

      const runDetail = newMockRunDetail();
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${runDetail.run!.id}`;

      TestUtils.makeErrorResponseOnce(getRunSpy, 'test error message');

      const tree = shallow(<TestNewRun {...props as any} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'test error message',
        message: `Error: failed to retrieve original run: ${runDetail.run!.id}. Click Details for more information.`,
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
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'run name' } });
      await TestUtils.flushPromises();

      expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', false);
      tree.unmount();
    });

    it('re-disables the \'Create\' new run button if pipeline ID in query params and run name entered then cleared', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'run name' } });
      await TestUtils.flushPromises();
      expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', false);

      (tree.instance() as any).handleChange('runName')({ target: { value: '' } });
      expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', true);

      tree.unmount();
    });

    it('sends a request to create a new run when \'Create\' is clicked', async () => {
      const props = generateProps();
      const experiment = newMockExperiment();
      const pipeline = newMockPipeline();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${experiment.id}`
        + `&${QUERY_PARAMS.pipelineId}=${pipeline.id}`;

      getExperimentSpy.mockImplementation(() => experiment);
      getPipelineSpy.mockImplementation(() => pipeline);

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
      (tree.instance() as any).handleChange('description')({ target: { value: 'test run description' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(createRunSpy).toHaveBeenCalledTimes(1);
      expect(createRunSpy).toHaveBeenLastCalledWith({
        description: 'test run description',
        name: 'test run name',
        pipeline_spec: {
          parameters: pipeline.parameters,
          pipeline_id: pipeline.id,
        },
        resource_references: [{
          key: {
            id: experiment.id,
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

      const tree = shallow(<TestNewRun {...props as any} />);
      await TestUtils.flushPromises();
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
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
      tree.unmount();
    });

    it('sets the page to a busy state upon clicking \'Create\' is clicked', async () => {
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`
        + `&${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(tree.state('isBeingCreated')).toBe(true);
      tree.unmount();
    });

    it('navigates to the ExperimentDetails page upon successful creation if there was an experiment', async () => {
      const props = generateProps();
      const experiment = newMockExperiment();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${experiment.id}`
        + `&${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(historyPushSpy).toHaveBeenCalledWith(
        RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experiment.id!));
      tree.unmount();
    });

    it('navigates to the AllRuns page upon successful creation if there was not an experiment', async () => {
      const props = generateProps();
      // No experiment in query params
      props.location.search = `?${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
      await TestUtils.flushPromises();

      tree.find('#createNewRunBtn').simulate('click');
      // The create APIs are called in a callback triggered by clicking 'Create', so we wait again
      await TestUtils.flushPromises();

      expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.RUNS);
      tree.unmount();
    });

    it('shows an error dialog if creating the new run fails', async () => {
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`
        + `&${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      TestUtils.makeErrorResponseOnce(createRunSpy, 'test error message');

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
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
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
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
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.error = jest.spyOn(console, 'error').mockImplementationOnce(() => null);

      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`
        + `&${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      TestUtils.makeErrorResponseOnce(createRunSpy, 'test error message');

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
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
        `?${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`
        + `&${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
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
      const tree = shallow(<TestNewRun {...props as any} />);
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
      const tree = shallow(<TestNewRun {...props as any} />);
      await TestUtils.flushPromises();

      expect(tree).toMatchSnapshot();
      tree.unmount();
    });

    it('sends a request to create a new recurring run when \'Create\' is clicked', async () => {

      const props = generateProps();
      const experiment = newMockExperiment();
      const pipeline = newMockPipeline();
      props.location.search =
        `?${QUERY_PARAMS.isRecurring}=1`
        + `&${QUERY_PARAMS.experimentId}=${experiment.id}`
        + `&${QUERY_PARAMS.pipelineId}=${pipeline.id}`;

      getExperimentSpy.mockImplementation(() => experiment);
      getPipelineSpy.mockImplementation(() => pipeline);

      const tree = TestUtils.mountWithRouter(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
      (tree.instance() as any).handleChange('description')({ target: { value: 'test run description' } });
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
          parameters: pipeline.parameters,
          pipeline_id: pipeline.id,
        },
        resource_references: [{
          key: {
            id: experiment.id,
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
      + `&${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`
      + `&${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
      // TODO: figure out how to do this by interacting with the Trigger element
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
      + `&${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`
      + `&${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
      // TODO: figure out how to do this by interacting with the Trigger element
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
      + `&${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`
      + `&${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
      // TODO: figure out how to do this by interacting with the Trigger element
      tree.setState({
        maxConcurrentRuns: -1,
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
      + `&${QUERY_PARAMS.experimentId}=${newMockExperiment().id}`
      + `&${QUERY_PARAMS.pipelineId}=${newMockPipeline().id}`;

      const tree = shallow(<TestNewRun {...props as any} />);
      (tree.instance() as any).handleChange('runName')({ target: { value: 'test run name' } });
      // TODO: figure out how to do this by interacting with the Trigger element
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
