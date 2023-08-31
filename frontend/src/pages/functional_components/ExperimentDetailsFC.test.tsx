/*
 * Copyright 2023 The Kubeflow Authors
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

import { render, screen, waitFor } from '@testing-library/react';
import * as React from 'react';
import EnhancedExperimentDetails, { ExperimentDetails } from 'src/pages/ExperimentDetails';
import { ExperimentDetailsFC } from './ExperimentDetailsFC';
import TestUtils from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { V2beta1Experiment, V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import { PageProps } from 'src/pages/Page';
import { RoutePage, RouteParams, QUERY_PARAMS } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { range } from 'lodash';
import { ButtonKeys } from 'src/lib/Buttons';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { Router } from 'react-router-dom';
import { createMemoryHistory } from 'history';
import { V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';

describe('ExperimentDetails', () => {
  const TEST_EXPERIMENT_ID = 'test-experiment-id';
  const TEST_EXPERIMENT_NAME = 'test-experiment-name';
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
      description: 'test-experiment-description',
      display_name: TEST_EXPERIMENT_NAME,
      experiment_id: TEST_EXPERIMENT_ID,
    };
  }

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: {
        params: { [RouteParams.experimentId]: MOCK_EXPERIMENT.experiment_id },
        isExact: true,
        path: '',
        url: '',
      },
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
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
    jest.clearAllMocks();
    getExperimentSpy.mockImplementation(() => newMockExperiment());
    // await mockNRecurringRuns(0);
    // await mockNRuns(0);
  });

  it('renders a page with no runs or recurring runs', async () => {
    listRunsSpy.mockImplementation(() => {}); // listRuns return nothing
    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
      expect(listRunsSpy).toHaveBeenCalled();
    });

    screen.getByText('No available runs found for this experiment.');
  });

  it('uses the experiment ID in props as the page title if the experiment has no name', async () => {
    const experiment = newMockExperiment();
    experiment.display_name = '';
    getExperimentSpy.mockImplementationOnce(() => experiment);

    const props = generateProps();
    props.match = { params: { [RouteParams.experimentId]: 'test-no-name-exp-id' } } as any;
    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...props} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
    });

    expect(updateBannerSpy).toHaveBeenCalledWith({
      pageTitle: 'test-no-name-exp-id',
    });
  });

  it('uses the experiment name as the page title', async () => {
    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
    });

    expect(updateBannerSpy).toHaveBeenCalledWith({
      pageTitle: TEST_EXPERIMENT_NAME,
    });
  });

  it('removes all description text after second newline and replaces with an ellipsis', async () => {
    const experiment = newMockExperiment();
    experiment.description = 'Line 1\nLine 2\nLine 3\nLine 4';
    getExperimentSpy.mockImplementationOnce(() => experiment);

    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
    });

    screen.getByText('Line 1');
  });

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});

  //   it('', () => {});
});
