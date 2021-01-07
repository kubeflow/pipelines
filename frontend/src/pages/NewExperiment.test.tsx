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
import { NewExperiment } from './NewExperiment';
import TestUtils, { defaultToolbarProps } from '../TestUtils';
import { shallow, ReactWrapper, ShallowWrapper } from 'enzyme';
import { Apis } from '../lib/Apis';
import { RoutePage, QUERY_PARAMS } from '../components/Router';
import { ApiResourceType, ApiRelationship } from 'src/apis/experiment';
import { TFunction } from 'i18next';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate hook can use it without a warning being shown
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: (key: string) => key };
    return Component;
  },
}));

describe('NewExperiment', () => {
  let tree: ReactWrapper | ShallowWrapper;
  const createExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'createExperiment');
  const historyPushSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  function generateProps(search?: string): any {
    return {
      history: { push: historyPushSpy } as any,
      location: { pathname: RoutePage.NEW_EXPERIMENT } as any,
      match: '' as any,
      toolbarProps: defaultToolbarProps(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  // Used by tests that don't care about exact experiment name
  function fillAnyExperimentName() {
    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'a-random-experiment-name-DO-NOT-VERIFY-THIS' },
    });
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

  afterEach(() => tree.unmount());

  it('renders the new experiment page', () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);
    expect(tree).toMatchSnapshot();
  });
  it('does not include any action buttons in the toolbar', () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);
    // the received match the expected values but the test fail

    let t: TFunction = (key: string) => key as any;
    let object = {
      actions: {},
      breadcrumbs: [{ displayName: 'common:experiments', href: '/experiments' }],
      pageTitle: 'newExperiment',
      t,
    };
    expect(updateToolbarSpy).lastCalledWith(
      expect.objectContaining({
        actions: {},
        breadcrumbs: [{ displayName: 'common:experiments', href: '/experiments' }],
        pageTitle: 'newExperiment',
        t: expect.any(Function),
      }),
    );
  });

  it("enables the 'Next' button when an experiment name is entered", () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', true);

    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment name' },
    });

    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', false);
    expect(tree).toMatchSnapshot();
  });

  it("re-disables the 'Next' button when an experiment name is cleared after having been entered", () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', true);

    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment name' },
    });
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', false);

    (tree.instance() as any).handleChange('experimentName')({ target: { value: '' } });
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('disabled', true);
    expect(tree).toMatchSnapshot();
  });

  it('updates the experiment name', () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);
    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment name' },
    });

    expect(tree.state()).toEqual({
      description: '',
      experimentName: 'experiment name',
      isbeingCreated: false,
      validationError: '',
    });
  });

  it('updates the experiment description', () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);
    (tree.instance() as any).handleChange('description')({ target: { value: 'a description!' } });

    expect(tree.state()).toEqual({
      description: 'a description!',
      experimentName: '',
      isbeingCreated: false,
      validationError: 'experimentNameRequired',
    });
  });

  it("sets the page to a busy state upon clicking 'Next'", async () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);

    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment-name' },
    });

    tree.find('#createExperimentBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('isbeingCreated', true);
    expect(tree.find('#createExperimentBtn').props()).toHaveProperty('busy', true);
  });

  it("calls the createExperiment API with the new experiment upon clicking 'Next'", async () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);

    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment name' },
    });
    (tree.instance() as any).handleChange('description')({
      target: { value: 'experiment description' },
    });

    tree.find('#createExperimentBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(createExperimentSpy).toHaveBeenCalledWith({
      description: 'experiment description',
      name: 'experiment name',
    });
  });

  it('calls the createExperimentAPI with namespace when it is provided', async () => {
    tree = shallow(
      <NewExperiment t={(key: any) => key} {...(generateProps() as any)} namespace='test-ns' />,
    );

    fillAnyExperimentName();
    tree.find('#createExperimentBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(createExperimentSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        resource_references: [
          {
            key: {
              id: 'test-ns',
              type: ApiResourceType.NAMESPACE,
            },
            relationship: ApiRelationship.OWNER,
          },
        ],
      }),
    );
  });

  it('navigates to NewRun page upon successful creation', async () => {
    const experimentId = 'test-exp-id-1';
    createExperimentSpy.mockImplementation(() => ({ id: experimentId }));
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);

    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment-name' },
    });

    tree.find('#createExperimentBtn').simulate('click');
    await createExperimentSpy;
    await TestUtils.flushPromises();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN + `?experimentId=${experimentId}` + `&firstRunInExperiment=1`,
    );
  });

  it('includes pipeline ID in NewRun page query params if present', async () => {
    const experimentId = 'test-exp-id-1';
    createExperimentSpy.mockImplementation(() => ({ id: experimentId }));

    const pipelineId = 'some-pipeline-id';
    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.pipelineId}=${pipelineId}`;
    tree = shallow(<NewExperiment t={(key: any) => key} {...(props as any)} />);

    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment-name' },
    });

    tree.find('#createExperimentBtn').simulate('click');
    await createExperimentSpy;
    await TestUtils.flushPromises();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN +
        `?experimentId=${experimentId}` +
        `&pipelineId=${pipelineId}` +
        `&firstRunInExperiment=1`,
    );
  });

  it('shows snackbar confirmation after experiment is created', async () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);

    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment-name' },
    });

    tree.find('#createExperimentBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      autoHideDuration: 10000,
      message: 'newExperimentSuccess: experiment-name',
      open: true,
    });
  });

  it('unsets busy state when creation fails', async () => {
    // Don't actually log to console.
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementation();

    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);

    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment-name' },
    });

    TestUtils.makeErrorResponseOnce(createExperimentSpy, 'test error!');
    tree.find('#createExperimentBtn').simulate('click');
    await createExperimentSpy;
    await TestUtils.flushPromises();

    expect(tree.state()).toHaveProperty('isbeingCreated', false);
  });

  it('shows error dialog when creation fails', async () => {
    // Don't actually log to console.
    // tslint:disable-next-line:no-console
    console.error = jest.spyOn(console, 'error').mockImplementation();

    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);

    (tree.instance() as any).handleChange('experimentName')({
      target: { value: 'experiment-name' },
    });

    TestUtils.makeErrorResponseOnce(createExperimentSpy, 'test error!');
    tree.find('#createExperimentBtn').simulate('click');
    await createExperimentSpy;
    await TestUtils.flushPromises();

    const call = updateDialogSpy.mock.calls[0][0];
    expect(call).toHaveProperty('title', 'experimentCreationFailed');
    expect(call).toHaveProperty('content', 'test error!');
  });

  it('navigates to experiment list page upon cancellation', async () => {
    tree = shallow(<NewExperiment t={(key: any) => key} {...(generateProps() as any)} />);
    tree.find('#cancelNewExperimentBtn').simulate('click');
    await TestUtils.flushPromises();

    expect(historyPushSpy).toHaveBeenCalledWith(RoutePage.EXPERIMENTS);
  });
});
