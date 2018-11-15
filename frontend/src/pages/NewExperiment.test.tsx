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
import NewExperiment from './NewExperiment';
import TestUtils from '../TestUtils';
import { shallow } from 'enzyme';
import { PageProps } from './Page';
import { Apis } from '../lib/Apis';
import { RoutePage } from '../components/Router';
import { QUERY_PARAMS } from '../lib/URLParser';

describe('NewExperiment', () => {
  const createExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'createExperiment');
  const historyPushSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: { pathname: RoutePage.NEW_EXPERIMENT } as any,
      match: '' as any,
      toolbarProps: NewExperiment.prototype.getInitialToolbarState(),
      updateBanner: () => null,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    // Reset mocks
    createExperimentSpy.mockReset();
    historyPushSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();

    createExperimentSpy.mockImplementation(() => ({ id: 'new-experiment-id' }));
  });

  it('renders the new experiment page', () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);

    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('does not include any action buttons in the toolbar', () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);

    expect(updateToolbarSpy).toHaveBeenCalledWith({
      actions: [],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        { displayName: 'New experiment', href: RoutePage.NEW_EXPERIMENT }
      ],
    });
    tree.unmount();
  });

  it('enables the \'Next\' button when an experiment name is entered', () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', true);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment name' } });

    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', false);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('re-disables the \'Next\' button when an experiment name is cleared after having been entered', () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', true);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment name' } });
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', false);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: '' } });
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', true);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('updates the experiment name', () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);
    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment name' } });

    expect(tree.state()).toEqual({
      description: '',
      experimentName: 'experiment name',
      isbeingCreated: false,
      validationError: '',
    });
    tree.unmount();
  });

  it('updates the experiment description', () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);
    (tree.instance() as any).handleChange('description')({ target: { value: 'a description!' } });

    expect(tree.state()).toEqual({
      description: 'a description!',
      experimentName: '',
      isbeingCreated: false,
      validationError: 'Experiment name is required',
    });
    tree.unmount();
  });

  it('sets the page to a busy state upon clicking \'Next\'', async () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

    tree.find('#createExperimentBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('isbeingCreated', true);
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('busy', true);
    tree.unmount();
  });

  it('calls the createExperiment API with the new experiment upon clicking \'Next\'', async () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment name' } });
    (tree.instance() as any).handleChange('description')({ target: { value: 'experiment description' } });

    tree.find('#createExperimentBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(createExperimentSpy).toHaveBeenCalledWith({
      description: 'experiment description',
      name: 'experiment name',
    });
    tree.unmount();
  });

  it('navigates to NewRun page upon successful creation', async () => {
    const experimentId = 'test-exp-id-1';
    createExperimentSpy.mockImplementation(() => ({ id: experimentId }));
    const tree = shallow(<NewExperiment {...generateProps() as any} />);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

    tree.find('#createExperimentBtn').simulate('click');
    await createExperimentSpy;
    await TestUtils.flushPromises();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN
      + `?experimentId=${experimentId}`
      + `&firstRunInExperiment=1`);
    tree.unmount();
  });

  it('includes pipeline ID in NewRun page query params if present', async () => {
    const experimentId = 'test-exp-id-1';
    createExperimentSpy.mockImplementation(() => ({ id: experimentId }));

    const pipelineId = 'some-pipeline-id';
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${pipelineId}`;
    const tree = shallow(<NewExperiment {...props as any} />);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

    tree.find('#createExperimentBtn').simulate('click');
    await createExperimentSpy;
    await TestUtils.flushPromises();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN
      + `?experimentId=${experimentId}`
      + `&pipelineId=${pipelineId}`
      + `&firstRunInExperiment=1`);
    tree.unmount();
  });

  it('shows snackbar confirmation after experiment is created', async () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

    tree.find('#createExperimentBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      autoHideDuration: 10000,
      message: 'Successfully created new Experiment: experiment-name',
      open: true,
    });
    tree.unmount();
  });

  it('unsets busy state when creation fails', async () => {
    // Don't actually log to console.
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementation();

    const tree = shallow(<NewExperiment {...generateProps() as any} />);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

    TestUtils.makeErrorResponseOnce(createExperimentSpy, 'test error!');
    tree.find('#createExperimentBtn').simulate('click');
    await createExperimentSpy;
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('isbeingCreated', false);
    tree.unmount();
  });

  it('shows error dialog when creation fails', async () => {
    // Don't actually log to console.
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementation();

    const tree = shallow(<NewExperiment {...generateProps() as any} />);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: 'experiment-name' } });

    TestUtils.makeErrorResponseOnce(createExperimentSpy, 'test error!');
    tree.find('#createExperimentBtn').simulate('click');
    await createExperimentSpy;
    await TestUtils.flushPromises();

    const call = updateDialogSpy.mock.calls[0][0];
    expect(call).toHaveProperty('title', 'Experiment creation failed');
    expect(call).toHaveProperty('content', 'test error!');
    tree.unmount();
  });

  it('navigates to experiment list page upon cancellation', async () => {
    const tree = shallow(<NewExperiment {...generateProps() as any} />);
    tree.find('#cancelNewExperimentBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.EXPERIMENTS);
    tree.unmount();
  });
});
