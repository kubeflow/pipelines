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
import { ApiRun, ApiResourceType } from '../apis/run';

describe('NewRun', () => {
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

  const mockExperiment: ApiExperiment = {
    description: 'mock experiment description',
    id: 'some-mock-experiment-id',
    name: 'some mock experiment name',
  };

  const mockPipeline: ApiPipeline = {
    id: 'some-mock-pipeline-id',
    name: 'some mock pipeline name',
  };

  const mockRun: ApiRun = {
    id: 'some-mock-run-id',
    name: 'some mock run name',
  };

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy, replace: historyReplaceSpy } as any,
      location: { pathname: RoutePage.NEW_RUN } as any,
      match: '' as any,
      toolbarProps: NewRun.prototype.getInitialToolbarState(),
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
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();

    createRunSpy.mockImplementation(() => ({ id: 'new-run-id' }));
    getExperimentSpy.mockImplementation(() =>({ mockExperiment }));
    getPipelineSpy.mockImplementation(() =>({ mockPipeline }));
    getRunSpy.mockImplementation(() =>({ mockRun }));
  });

  it('renders the new run page', () => {
    const tree = shallow(<NewRun {...generateProps() as any} />);

    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('disables \'Create\' new run button by default', () => {
    const tree = shallow(<NewRun {...generateProps() as any} />);

    expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', true);
    tree.unmount();
  });

  it('does not include any action buttons in the toolbar', () => {
    const tree = shallow(<NewRun {...generateProps() as any} />);

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: [],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        { displayName: 'Start a new run', href: '' }
      ],
    });
    tree.unmount();
  });

  it('changes the title if the new run will recur, based on query param', () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.isRecurring}=1`;
    const tree = shallow(<NewRun {...props as any} />);

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: [],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        { displayName: 'Start a recurring run', href: '' }
      ],
    });
    tree.unmount();
  });

  it('clears the banner when refresh is called', () => {
    const tree = shallow(<NewRun {...generateProps() as any} />);
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    (tree.instance() as NewRun).refresh();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('clears the banner when load is called', () => {
    const tree = shallow(<NewRun {...generateProps() as any} />);
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    (tree.instance() as NewRun).load();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('updates the run name', () => {
    const tree = shallow(<NewRun {...generateProps() as any} />);
    (tree.instance() as any).handleChange('runName')({ target: { value: 'run name' } });

    expect(tree.state()).toHaveProperty('runName', 'run name');
    tree.unmount();
  });

  it('updates the run description', () => {
    const tree = shallow(<NewRun {...generateProps() as any} />);
    (tree.instance() as any).handleChange('description')({ target: { value: 'run description' } });

    expect(tree.state()).toHaveProperty('description', 'run description');
    tree.unmount();
  });

  it('fetches the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${mockExperiment.id}`;

    getExperimentSpy.mockImplementation(() => mockExperiment);

    const tree = shallow(<NewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(getExperimentSpy).toHaveBeenLastCalledWith(mockExperiment.id);
    tree.unmount();
  });

  it('updates the run\'s state with the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${mockExperiment.id}`;

    getExperimentSpy.mockImplementation(() => mockExperiment);

    const tree = shallow(<NewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('experiment', mockExperiment);
    expect(tree.state()).toHaveProperty('experimentName', mockExperiment.name);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('updates the breadcrumb with the associated experiment if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${mockExperiment.id}`;

    getExperimentSpy.mockImplementation(() => mockExperiment);

    const tree = shallow(<NewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(updateToolbarSpy).toHaveBeenLastCalledWith({
      actions: [],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        {
          displayName: mockExperiment.name,
          href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, mockExperiment.id!)
        },
        { displayName: 'Start a new run', href: ''}
      ],
    });
    tree.unmount();
  });

  it('shows a page error if getExperiment fails', async () => {
    // Don't actually log to console.
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementation();

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.experimentId}=${mockExperiment.id}`;

    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'test error message');

    const tree = shallow(<NewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'test error message',
      message: `Error: failed to retrieve associated experiment: ${mockExperiment.id}. Click Details for more information.`,
      mode: 'error',
    }));
    tree.unmount();
  });

  it('fetches the associated pipeline if one is present in the query params', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${mockPipeline.id}`;

    getPipelineSpy.mockImplementation(() => mockPipeline);

    const tree = shallow(<NewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('pipeline', mockPipeline);
    expect(tree.state()).toHaveProperty('pipelineName', mockPipeline.name);
    expect(tree.state()).toHaveProperty('errorMessage', 'Run name is required');
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('enables the \'Create\' new run button if pipeline ID in query params and run name entered', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${mockPipeline.id}`;

    const tree = shallow(<NewRun {...props as any} />);
    (tree.instance() as any).handleChange('runName')({ target: { value: 'run name' } });
    await TestUtils.flushPromises();

    expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', false);
    tree.unmount();
  });

  it('re-disables the \'Create\' new run button if pipeline ID in query params and run name entered then cleared', async () => {
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${mockPipeline.id}`;

    const tree = shallow(<NewRun {...props as any} />);
    (tree.instance() as any).handleChange('runName')({ target: { value: 'run name' } });
    await TestUtils.flushPromises();
    expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', false);

    (tree.instance() as any).handleChange('runName')({ target: { value: '' } });
    expect(tree.find('#createNewRunBtn').props()).toHaveProperty('disabled', true);

    tree.unmount();
  });

  it('shows a page error if getPipeline fails', async () => {
    // Don't actually log to console.
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementation();

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${mockPipeline.id}`;

    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

    const tree = shallow(<NewRun {...props as any} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'test error message',
      message: `Error: failed to retrieve pipeline: ${mockPipeline.id}. Click Details for more information.`,
      mode: 'error',
    }));
    tree.unmount();
  });

  describe('cloning from a run', () => {
    it('fetches the original run if an ID is present in the query params', async () => {
      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${mockRun.id}`;

      const tree = shallow(<NewRun {...props as any} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(mockRun.id);
      tree.unmount();
    });

    it('uses the query param experiment ID over the one in the original run if an ID is present in both', async () => {
      const run: ApiRun = {
        id: 'some-mock-run-id',
        name: 'some mock run name',
        resource_references: [{
          key: { id: `${mockExperiment.id}-different`, type: ApiResourceType.EXPERIMENT },
        }],
      };
      const props = generateProps();
      props.location.search =
        `?${QUERY_PARAMS.cloneFromRun}=${run.id}`
        + `&${QUERY_PARAMS.experimentId}=${mockExperiment.id}`;

      getRunSpy.mockImplementation(() => run);

      const tree = shallow(<NewRun {...props as any} />);
      await TestUtils.flushPromises();

      expect(getRunSpy).toHaveBeenCalledTimes(1);
      expect(getRunSpy).toHaveBeenLastCalledWith(run.id);
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
      expect(getExperimentSpy).toHaveBeenLastCalledWith(mockExperiment.id);
      expect(tree.state('experiment')).toHaveProperty('id', mockExperiment.id);
      tree.unmount();
    });

    it('shows a page error if getRun fails', async () => {
      // Don't actually log to console.
      // tslint:disable-next-line:no-console
      console.error = jest.spyOn(console, 'error').mockImplementation();

      const props = generateProps();
      props.location.search = `?${QUERY_PARAMS.cloneFromRun}=${mockRun.id}`;

      TestUtils.makeErrorResponseOnce(getRunSpy, 'test error message');

      const tree = shallow(<NewRun {...props as any} />);
      await TestUtils.flushPromises();

      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
        additionalInfo: 'test error message',
        message: `Error: failed to retrieve original run: ${mockRun.id}. Click Details for more information.`,
        mode: 'error',
      }));
      tree.unmount();
    });

  });

  // it('clears the pipeline ID query param if getPipeline fails', async () => {
  //   const props = generateProps();
  //   props.location.search = `?${QUERY_PARAMS.pipelineId}=${mockPipeline.id}`;

  //   TestUtils.makeErrorResponseOnce(getPipelineSpy, 'test error message');

  //   const tree = TestUtils.mountWithRouter(<NewRun {...props as any} />);
  //   await TestUtils.flushPromises();

  //   expect((tree.instance().props as any).location.search).toBe('test');
  //   tree.unmount();
  // });
  // it('Updates the experiment description', () => {
  //   const tree = shallow(<NewRun {...generateProps() as any} />);
  //   (tree.instance() as any).handleChange('description')({ target: { value: 'a description!' } });

  //   expect(tree.state()).toEqual({
  //     description: 'a description!',
  //     experimentName: '',
  //     isbeingCreated: false,
  //     validationError: 'Experiment name is required',
  //   });
  //   tree.unmount();
  // });

  // it('sets the page to a busy state upon clicking \'Next\'', async () => {
  //   const tree = shallow(<NewRun {...generateProps() as any} />);

  //   (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

  //   tree.find('#createExperimentBtn').simulate('click');
  //   await TestUtils.flushPromises();

  //   expect(tree.state()).toHaveProperty('isbeingCreated', true);
  //   tree.unmount();
  // });

  // it('calls the createExperiment API with the new experiment upon clicking \'Next\'', async () => {
  //   const tree = shallow(<NewRun {...generateProps() as any} />);

  //   (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment name' } });
  //   (tree.instance() as any).handleChange('description')({ target: { value: 'experiment description' } });

  //   tree.find('#createExperimentBtn').simulate('click');
  //   await TestUtils.flushPromises();

  //   expect(createRunSpy).toHaveBeenCalledWith({
  //     description: 'experiment description',
  //     name: 'experiment name',
  //   });
  //   tree.unmount();
  // });

  // it('navigates to NewRun page upon successful creation', async () => {
  //   const experimentId = 'test-exp-id-1';
  //   createRunSpy.mockImplementation(() => ({ id: experimentId }));
  //   const tree = shallow(<NewRun {...generateProps() as any} />);

  //   (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

  //   tree.find('#createExperimentBtn').simulate('click');
  //   await TestUtils.flushPromises();

  //   expect(historyPushSpy).toHaveBeenCalledWith(
  //     RoutePage.NEW_RUN
  //     + `?experimentId=${experimentId}`
  //     + `&firstRunInExperiment=1`);
  //   tree.unmount();
  // });

  // it('includes pipeline ID in NewRun page query params if present', async () => {
  //   const experimentId = 'test-exp-id-1';
  //   createRunSpy.mockImplementation(() => ({ id: experimentId }));

  //   const pipelineId = 'pipelineId=some-pipeline-id';
  //   const props = generateProps();
  //   props.location.search = `?pipelineId=${pipelineId}`;
  //   const tree = shallow(<NewRun {...props as any} />);

  //   (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

  //   tree.find('#createExperimentBtn').simulate('click');
  //   await TestUtils.flushPromises();

  //   expect(historyPushSpy).toHaveBeenCalledWith(
  //     RoutePage.NEW_RUN
  //     + `?experimentId=${experimentId}`
  //     + `&pipelineId=${pipelineId}`
  //     + `&firstRunInExperiment=1`);
  //   tree.unmount();
  // });

  // it('shows snackbar confirmation after experiment is created', async () => {
  //   const tree = shallow(<NewRun {...generateProps() as any} />);

  //   (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

  //   tree.find('#createExperimentBtn').simulate('click');
  //   await TestUtils.flushPromises();

  //   expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
  //     autoHideDuration: 10000,
  //     message: 'Successfully created new Experiment: experiment-name',
  //     open: true,
  //   });
  //   tree.unmount();
  // });

  // it('unsets busy state when creation fails', async () => {
  //   // tslint:disable-next-line:no-console
  //   console.error = jest.spyOn(console, 'error').mockImplementation();

  //   const tree = shallow(<NewRun {...generateProps() as any} />);

  //   (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

  //   TestUtils.makeErrorResponseOnce(createRunSpy, 'test error!');
  //   tree.find('#createExperimentBtn').simulate('click');
  //   await TestUtils.flushPromises();

  //   expect(tree.state()).toHaveProperty('isbeingCreated', false);
  //   tree.unmount();
  // });

  // it('shows error dialog when creation fails', async () => {
  //   // tslint:disable-next-line:no-console
  //   console.error = jest.spyOn(console, 'error').mockImplementation();

  //   const tree = shallow(<NewRun {...generateProps() as any} />);

  //   (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

  //   TestUtils.makeErrorResponseOnce(createRunSpy, 'test error!');
  //   tree.find('#createExperimentBtn').simulate('click');
  //   await TestUtils.flushPromises();

  //   const call = updateDialogSpy.mock.calls[0][0];
  //   expect(call).toHaveProperty('title', 'Experiment creation failed');
  //   expect(call).toHaveProperty('content', 'test error!');
  //   tree.unmount();
  // });

  // it('navigates to experiment list page upon cancellation', async () => {
  //   const tree = shallow(<NewRun {...generateProps() as any} />);
  //   tree.find('#cancelNewRunBtn').simulate('click');
  //   await TestUtils.flushPromises();

  //   expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.EXPERIMENTS);
  //   tree.unmount();
  // });
});
