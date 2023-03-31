/*
 * Copyright 2018 The Kubeflow Authors
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
import EnhancedExperimentDetails, { ExperimentDetails } from './ExperimentDetails';
import TestUtils from 'src/TestUtils';
import { V2beta1Experiment, V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import { PageProps } from './Page';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { RoutePage, RouteParams, QUERY_PARAMS } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { range } from 'lodash';
import { ButtonKeys } from 'src/lib/Buttons';
import { render, screen } from '@testing-library/react';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { Router } from 'react-router-dom';
import { createMemoryHistory } from 'history';
import { V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';

describe('ExperimentDetails', () => {
  let tree: ReactWrapper | ShallowWrapper;

  const consoleLogSpy = jest.spyOn(console, 'log').mockImplementation(() => null);
  const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => null);

  const updateToolbarSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
  const listRecurringRunsSpy = jest.spyOn(Apis.recurringRunServiceApi, 'listRecurringRuns');
  const listRunsSpy = jest.spyOn(Apis.runServiceApiV2, 'listRuns');

  const MOCK_EXPERIMENT = newMockExperiment();

  function newMockExperiment(): V2beta1Experiment {
    return {
      description: 'mock experiment description',
      experiment_id: 'some-mock-experiment-id',
      display_name: 'some mock experiment name',
    };
  }

  function generateProps(): PageProps {
    const match = { params: { [RouteParams.experimentId]: MOCK_EXPERIMENT.experiment_id } } as any;
    return TestUtils.generatePageProps(
      ExperimentDetails,
      {} as any,
      match,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  async function mockNRecurringRuns(n: number): Promise<void> {
    listRecurringRunsSpy.mockImplementation(() => ({
      recurringRuns: range(n).map(i => ({
        display_name: 'test job name' + i,
        recurring_run_id: 'test-recurringrun-id' + i,
        status: V2beta1RecurringRunStatus.ENABLED,
      })),
    }));
    await listRecurringRunsSpy;
    await TestUtils.flushPromises();
  }

  async function mockNRuns(n: number): Promise<void> {
    listRunsSpy.mockImplementation(() => ({
      runs: range(n).map(i => ({ run_id: 'test-run-id' + i, display_name: 'test run name' + i })),
    }));
    await listRunsSpy;
    await TestUtils.flushPromises();
  }

  beforeEach(async () => {
    // Reset mocks
    consoleLogSpy.mockReset();
    consoleErrorSpy.mockReset();
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();
    getExperimentSpy.mockReset();
    historyPushSpy.mockReset();
    listRecurringRunsSpy.mockReset();
    listRunsSpy.mockReset();

    getExperimentSpy.mockImplementation(() => newMockExperiment());

    await mockNRecurringRuns(0);
    await mockNRuns(0);
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    if (tree.exists()) {
      await tree.unmount();
    }
  });

  it('renders a page with no runs or recurring runs', async () => {
    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
    expect(tree).toMatchSnapshot();
  });

  it('uses the experiment ID in props as the page title if the experiment has no name', async () => {
    const experiment = newMockExperiment();
    experiment.display_name = '';

    const props = generateProps();
    props.match = { params: { [RouteParams.experimentId]: 'test exp ID' } } as any;

    getExperimentSpy.mockImplementationOnce(() => experiment);

    tree = shallow(<ExperimentDetails {...props} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        pageTitle: 'test exp ID',
        pageTitleTooltip: 'test exp ID',
      }),
    );
  });

  it('uses the experiment name as the page title', async () => {
    const experiment = newMockExperiment();
    experiment.display_name = 'A Test Experiment';

    getExperimentSpy.mockImplementationOnce(() => experiment);

    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        pageTitle: 'A Test Experiment',
        pageTitleTooltip: 'A Test Experiment',
      }),
    );
  });

  it('uses an empty string if the experiment has no description', async () => {
    const experiment = newMockExperiment();
    delete experiment.description;

    getExperimentSpy.mockImplementationOnce(() => experiment);

    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('removes all description text after second newline and replaces with an ellipsis', async () => {
    const experiment = newMockExperiment();
    experiment.description = 'Line 1\nLine 2\nLine 3\nLine 4';

    getExperimentSpy.mockImplementationOnce(() => experiment);

    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('opens the expanded description modal when the expand button is clicked', async () => {
    tree = TestUtils.mountWithRouter(<ExperimentDetails {...(generateProps() as any)} />);
    await TestUtils.flushPromises();

    tree.update();

    tree
      .find('#expandExperimentDescriptionBtn')
      .at(0)
      .simulate('click');
    await TestUtils.flushPromises();
    expect(updateDialogSpy).toHaveBeenCalledWith({
      content: MOCK_EXPERIMENT.description,
      title: 'Experiment description',
    });
  });

  it('calls getExperiment with the experiment ID in props', async () => {
    const props = generateProps();
    props.match = { params: { [RouteParams.experimentId]: 'test exp ID' } } as any;
    tree = shallow(<ExperimentDetails {...props} />);
    await TestUtils.flushPromises();
    expect(getExperimentSpy).toHaveBeenCalledTimes(1);
    expect(getExperimentSpy).toHaveBeenCalledWith('test exp ID');
  });

  it('shows an error banner if fetching the experiment fails', async () => {
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'test error');

    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'test error',
        message:
          'Error: failed to retrieve experiment: ' +
          MOCK_EXPERIMENT.experiment_id +
          '. Click Details for more information.',
        mode: 'error',
      }),
    );
    expect(consoleErrorSpy.mock.calls[0][0]).toBe(
      'Error loading experiment: ' + MOCK_EXPERIMENT.experiment_id,
    );
  });

  it('shows a list of available runs', async () => {
    await mockNRecurringRuns(1);
    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(tree.find('RunListsRouter').prop('storageState')).toBe(
      V2beta1RunStorageState.AVAILABLE,
      // TODO(jlyaoyuli): Change to v2 storage state after run integration
    );
  });

  it('shows a list of archived runs', async () => {
    await mockNRecurringRuns(1);

    getExperimentSpy.mockImplementation(() => {
      let apiExperiment = newMockExperiment();
      apiExperiment['storage_state'] = V2beta1ExperimentStorageState.ARCHIVED;
      return apiExperiment;
    });

    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(tree.find('RunListsRouter').prop('storageState')).toBe(
      V2beta1RunStorageState.ARCHIVED,
      // TODO(jlyaoyuli): Change to v2 storage state after run integration
    );
  });

  it("fetches this experiment's recurring runs", async () => {
    await mockNRecurringRuns(1);

    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1);
    expect(listRecurringRunsSpy).toHaveBeenLastCalledWith(
      undefined,
      100,
      '',
      undefined,
      undefined,
      MOCK_EXPERIMENT.experiment_id,
    );
    expect(tree.state('activeRecurringRunsCount')).toBe(1);
    expect(tree).toMatchSnapshot();
  });

  it("shows an error banner if fetching the experiment's recurring runs fails", async () => {
    TestUtils.makeErrorResponseOnce(listRecurringRunsSpy, 'test error');

    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'test error',
        message:
          'Error: failed to retrieve recurring runs for experiment: ' +
          MOCK_EXPERIMENT.experiment_id +
          '. Click Details for more information.',
        mode: 'error',
      }),
    );
    expect(consoleErrorSpy.mock.calls[0][0]).toBe(
      'Error fetching recurring runs for experiment: ' + MOCK_EXPERIMENT.experiment_id,
    );
  });

  it('only counts enabled recurring runs as active', async () => {
    const recurringRuns = [
      {
        recurring_run_id: 'enabled-recurringrun-1-id',
        status: V2beta1RecurringRunStatus.ENABLED,
        display_name: 'enabled-recurringrun-1',
      },
      {
        recurring_run_id: 'enabled-recurringrun-2-id',
        status: V2beta1RecurringRunStatus.ENABLED,
        display_name: 'enabled-recurringrun-2',
      },
      {
        recurring_run_id: 'disabled-recurringrun-1-id',
        status: V2beta1RecurringRunStatus.DISABLED,
        display_name: 'disabled-recurringrun-1',
      },
    ];
    listRecurringRunsSpy.mockImplementationOnce(() => ({ recurringRuns }));
    await listRecurringRunsSpy;

    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    expect(tree.state('activeRecurringRunsCount')).toBe(2);
  });

  it("opens the recurring run manager modal when 'manage' is clicked", async () => {
    await mockNRecurringRuns(1);
    tree = TestUtils.mountWithRouter(<ExperimentDetails {...(generateProps() as any)} />);
    await TestUtils.flushPromises();

    tree.update();

    tree
      .find('#manageExperimentRecurringRunsBtn')
      .at(0)
      .simulate('click');
    await TestUtils.flushPromises();
    expect(tree.state('recurringRunsManagerOpen')).toBeTruthy();
  });

  it('closes the recurring run manager modal', async () => {
    await mockNRecurringRuns(1);
    tree = TestUtils.mountWithRouter(<ExperimentDetails {...(generateProps() as any)} />);
    await TestUtils.flushPromises();

    tree.update();

    tree
      .find('#manageExperimentRecurringRunsBtn')
      .at(0)
      .simulate('click');
    await TestUtils.flushPromises();
    expect(tree.state('recurringRunsManagerOpen')).toBeTruthy();

    tree
      .find('#closeExperimentRecurringRunManagerBtn')
      .at(0)
      .simulate('click');
    await TestUtils.flushPromises();
    expect(tree.state('recurringRunsManagerOpen')).toBeFalsy();
  });

  it('refreshes the number of active recurring runs when the recurring run manager is closed', async () => {
    await mockNRecurringRuns(1);
    tree = TestUtils.mountWithRouter(<ExperimentDetails {...(generateProps() as any)} />);
    await TestUtils.flushPromises();

    tree.update();

    // Called when the page initially loads to display the number of active recurring runs
    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1);

    tree
      .find('#manageExperimentRecurringRunsBtn')
      .at(0)
      .simulate('click');
    await TestUtils.flushPromises();
    expect(tree.state('recurringRunsManagerOpen')).toBeTruthy();

    // Called in the recurring run manager to list the recurring runs
    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(2);

    tree
      .find('#closeExperimentRecurringRunManagerBtn')
      .at(0)
      .simulate('click');
    await TestUtils.flushPromises();
    expect(tree.state('recurringRunsManagerOpen')).toBeFalsy();

    // Called a third time when the manager is closed to update the number of active recurring runs
    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(3);
  });

  it('clears the error banner on refresh', async () => {
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'test error');

    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();

    // Verify that error banner is being shown
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({ mode: 'error' }));

    (tree.instance() as ExperimentDetails).refresh();

    // Error banner should be cleared
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('navigates to the compare runs page', async () => {
    const runs = [
      { run_id: 'run-1-id', display_name: 'run-1' },
      { run_id: 'run-2-id', display_name: 'run-2' },
    ];
    listRunsSpy.mockImplementation(() => ({ runs }));
    await listRunsSpy;

    tree = TestUtils.mountWithRouter(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    tree
      .find('.tableRow')
      .at(0)
      .simulate('click');
    tree
      .find('.tableRow')
      .at(1)
      .simulate('click');

    const compareBtn = (tree.state('runListToolbarProps') as ToolbarProps).actions[
      ButtonKeys.COMPARE
    ];
    await compareBtn!.action();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.COMPARE + `?${QUERY_PARAMS.runlist}=run-1-id,run-2-id`,
    );
  });

  it('navigates to the new run page and passes this experiments ID as a query param', async () => {
    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    const newRunBtn = (tree.state('runListToolbarProps') as ToolbarProps).actions[
      ButtonKeys.NEW_RUN
    ];
    await newRunBtn!.action();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.experiment_id}`,
    );
  });

  it('navigates to the new run page with query param indicating it will be a recurring run', async () => {
    tree = shallow(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    const newRecurringRunBtn = (tree.state('runListToolbarProps') as ToolbarProps).actions[
      ButtonKeys.NEW_RECURRING_RUN
    ];
    await newRecurringRunBtn!.action();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.experiment_id}` +
        `&${QUERY_PARAMS.isRecurring}=1`,
    );
  });

  it('supports cloning a selected run', async () => {
    const runs = [{ run_id: 'run-1-id', display_name: 'run-1' }];
    listRunsSpy.mockImplementation(() => ({ runs }));
    await listRunsSpy;

    tree = TestUtils.mountWithRouter(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    // Select the run to clone
    tree.find('.tableRow').simulate('click');

    const cloneBtn = (tree.state('runListToolbarProps') as ToolbarProps).actions[
      ButtonKeys.CLONE_RUN
    ];
    await cloneBtn!.action();

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.cloneFromRun}=run-1-id`,
    );
  });

  it('enables the compare runs button only when between 2 and 10 runs are selected', async () => {
    await mockNRuns(12);

    tree = TestUtils.mountWithRouter(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    for (let i = 0; i < 12; i++) {
      const compareBtn = (tree.state('runListToolbarProps') as ToolbarProps).actions[
        ButtonKeys.COMPARE
      ];
      if (i < 2 || i > 10) {
        expect(compareBtn!.disabled).toBeTruthy();
      } else {
        expect(compareBtn!.disabled).toBeFalsy();
      }
      tree
        .find('.tableRow')
        .at(i)
        .simulate('click');
    }
  });

  it('enables the clone run button only when 1 run is selected', async () => {
    await mockNRuns(4);

    tree = TestUtils.mountWithRouter(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    for (let i = 0; i < 4; i++) {
      const cloneBtn = (tree.state('runListToolbarProps') as ToolbarProps).actions[
        ButtonKeys.CLONE_RUN
      ];
      if (i === 1) {
        expect(cloneBtn!.disabled).toBeFalsy();
      } else {
        expect(cloneBtn!.disabled).toBeTruthy();
      }
      tree
        .find('.tableRow')
        .at(i)
        .simulate('click');
    }
  });

  it('enables Archive button when at least one run is selected', async () => {
    await mockNRuns(4);

    tree = TestUtils.mountWithRouter(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    for (let i = 0; i < 4; i++) {
      const archiveButton = (tree.state('runListToolbarProps') as ToolbarProps).actions[
        ButtonKeys.ARCHIVE
      ];
      if (i === 0) {
        expect(archiveButton!.disabled).toBeTruthy();
      } else {
        expect(archiveButton!.disabled).toBeFalsy();
      }
      tree
        .find('.tableRow')
        .at(i)
        .simulate('click');
    }
  });

  it('enables Restore button when at least one run is selected', async () => {
    await mockNRuns(4);

    tree = TestUtils.mountWithRouter(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    tree
      .find('MD2Tabs')
      .find('Button')
      .at(1) // `Archived` tab button
      .simulate('click');
    await TestUtils.flushPromises();

    for (let i = 0; i < 4; i++) {
      const restoreButton = (tree.state('runListToolbarProps') as ToolbarProps).actions[
        ButtonKeys.RESTORE
      ];
      if (i === 0) {
        expect(restoreButton!.disabled).toBeTruthy();
      } else {
        expect(restoreButton!.disabled).toBeFalsy();
      }
      tree
        .find('.tableRow')
        .at(i)
        .simulate('click');
    }
  });

  it('switches to another tab will change Archive/Restore button', async () => {
    await mockNRuns(4);

    tree = TestUtils.mountWithRouter(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    tree
      .find('MD2Tabs')
      .find('Button')
      .at(1) // `Archived` tab button
      .simulate('click');
    await TestUtils.flushPromises();
    expect(
      (tree.state('runListToolbarProps') as ToolbarProps).actions[ButtonKeys.ARCHIVE],
    ).toBeUndefined();
    expect(
      (tree.state('runListToolbarProps') as ToolbarProps).actions[ButtonKeys.RESTORE],
    ).toBeDefined();

    tree
      .find('MD2Tabs')
      .find('Button')
      .at(0) // `Active` tab button
      .simulate('click');
    await TestUtils.flushPromises();
    expect(
      (tree.state('runListToolbarProps') as ToolbarProps).actions[ButtonKeys.ARCHIVE],
    ).toBeDefined();
    expect(
      (tree.state('runListToolbarProps') as ToolbarProps).actions[ButtonKeys.RESTORE],
    ).toBeUndefined();
  });

  it('switches to active/archive tab will show active/archive runs', async () => {
    await mockNRuns(4);
    tree = TestUtils.mountWithRouter(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();
    expect(tree.find('.tableRow').length).toEqual(4);

    await mockNRuns(2);
    tree
      .find('MD2Tabs')
      .find('Button')
      .at(1) // `Archived` tab button
      .simulate('click');
    await TestUtils.flushPromises();
    tree.update();
    expect(tree.find('.tableRow').length).toEqual(2);
  });

  it('switches to another tab will change Archive/Restore button', async () => {
    await mockNRuns(4);

    tree = TestUtils.mountWithRouter(<ExperimentDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    tree
      .find('MD2Tabs')
      .find('Button')
      .at(1) // `Archived` tab button
      .simulate('click');
    await TestUtils.flushPromises();
    expect(
      (tree.state('runListToolbarProps') as ToolbarProps).actions[ButtonKeys.ARCHIVE],
    ).toBeUndefined();
    expect(
      (tree.state('runListToolbarProps') as ToolbarProps).actions[ButtonKeys.RESTORE],
    ).toBeDefined();

    tree
      .find('MD2Tabs')
      .find('Button')
      .at(0) // `Active` tab button
      .simulate('click');
    await TestUtils.flushPromises();
    expect(
      (tree.state('runListToolbarProps') as ToolbarProps).actions[ButtonKeys.ARCHIVE],
    ).toBeDefined();
    expect(
      (tree.state('runListToolbarProps') as ToolbarProps).actions[ButtonKeys.RESTORE],
    ).toBeUndefined();
  });

  describe('EnhancedExperimentDetails', () => {
    it('renders ExperimentDetails initially', () => {
      render(<EnhancedExperimentDetails {...generateProps()}></EnhancedExperimentDetails>);
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
    });

    it('redirects to ExperimentList page if namespace changes', () => {
      const history = createMemoryHistory();
      const { rerender } = render(
        <Router history={history}>
          <NamespaceContext.Provider value='test-ns-1'>
            <EnhancedExperimentDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      rerender(
        <Router history={history}>
          <NamespaceContext.Provider value='test-ns-2'>
            <EnhancedExperimentDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual(RoutePage.EXPERIMENTS);
    });
  });
});
