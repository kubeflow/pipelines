/*
 * Copyright 2022 The Kubeflow Authors
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
import { CommonTestWrapper } from 'src/TestWrapper';
import { Apis } from 'src/lib/Apis';
import { PageProps } from './Page';
import { RoutePage, QUERY_PARAMS, RouteParams } from 'src/components/Router';
import { ApiRunDetail } from 'src/apis/run';
import Compare, { CompareProps } from './Compare';
import * as features from 'src/features';

describe('Switch between v1 and v2 Compare runs pages', () => {
  const updateToolbarSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');

  const MOCK_RUN_1_ID = 'mock-run-1-id';
  const MOCK_RUN_2_ID = 'mock-run-2-id';
  const MOCK_RUN_3_ID = 'mock-run-3-id';

  function generateProps(): CompareProps {
    const pageProps: PageProps = {
      history: { push: historyPushSpy } as any,
      location: {
        search: `?${QUERY_PARAMS.runlist}=${MOCK_RUN_1_ID},${MOCK_RUN_2_ID},${MOCK_RUN_3_ID}`
      } as any,
      match: '' as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
    return Object.assign(pageProps, {
      collapseSections: {},
      fullscreenViewerConfig: null,
      metricsCompareProps: { rows: [], xLabels: [], yLabels: [] },
      paramsCompareProps: { rows: [], xLabels: [], yLabels: [] },
      runs: [],
      selectedIds: [],
      viewersMap: new Map(),
      workflowObjects: [],
    });
  }

  let runs: ApiRunDetail[] = [];

  function newMockRun(id?: string, v2?: boolean): ApiRunDetail {
    return {
      pipeline_runtime: {
        workflow_manifest: '{}',
      },
      run: {
        id: id || 'test-run-id',
        name: 'test run ' + id,
        pipeline_spec: v2 ? { pipeline_manifest: '' } : { workflow_manifest: '' },
      },
    };
  }

  beforeEach(async () => {
    // Reset mocks
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();
    historyPushSpy.mockReset();
  });

  afterEach(async () => {
    jest.resetAllMocks();
  });

  it('getRun is called with query param IDs', async () => {
    // TODO: When this first test displays the CompareV2 page, it messes up the remaining tests, and I don't understand why.
    runs.push(newMockRun(MOCK_RUN_1_ID, false), newMockRun(MOCK_RUN_2_ID, true), newMockRun(MOCK_RUN_3_ID, true));
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });

    render(
      <CommonTestWrapper>
        <Compare {...generateProps()} />
      </CommonTestWrapper>
    );

    // TODO: This method is called more than three times - 9 in total, I believe due to page refreshes. Should I look into this?
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_1_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_2_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_3_ID);
  });

  it('Show v1 page if all runs are v1 and the v2 feature flag is enabled', async () => {
    // All runs are v1.
    runs = [newMockRun(MOCK_RUN_1_ID, false), newMockRun(MOCK_RUN_2_ID, false), newMockRun(MOCK_RUN_3_ID, false)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });

    render(
      <CommonTestWrapper>
        <Compare {...generateProps()} />
      </CommonTestWrapper>
    );

    // TODO: I will remove these additional calls once the first test works along with the rest.
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_1_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_2_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_3_ID);
    waitFor(() => screen.getByTestId('compare-runs-v1'));
  });

  it('Show v1 page if some runs are v1 and the v2 feature flag is enabled', async () => {
    // The last two runs are v1.
    runs = [newMockRun(MOCK_RUN_1_ID, false), newMockRun(MOCK_RUN_2_ID, true), newMockRun(MOCK_RUN_3_ID, true)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });

    render(
      <CommonTestWrapper>
        <Compare {...generateProps()} />
      </CommonTestWrapper>
    );

    waitFor(() => screen.getByTestId('compare-runs-v1'));
  });

  it('Show v2 page if all runs are v2 and the v2 feature flag is enabled', async () => {
    // All runs are v2. 
    runs = [newMockRun(MOCK_RUN_1_ID, true), newMockRun(MOCK_RUN_2_ID, true), newMockRun(MOCK_RUN_3_ID, true)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });

    render(
      <CommonTestWrapper>
        <Compare {...generateProps()} />
      </CommonTestWrapper>
    );

    waitFor(() => screen.getByTestId('compare-runs-v2'));
  });

  it('Show v1 page if some runs are v1 and the v2 feature flag is disabled', async () => {
    // The last two runs are v1.
    runs = [newMockRun(MOCK_RUN_1_ID, false), newMockRun(MOCK_RUN_2_ID, true), newMockRun(MOCK_RUN_3_ID, true)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(_ => {
      return false;
    });

    render(
      <CommonTestWrapper>
        <Compare {...generateProps()} />
      </CommonTestWrapper>
    );

    waitFor(() => screen.getByTestId('compare-runs-v1'));
  });

  it('Show v1 page if all runs are v2 and the v2 feature flag is disabled', async () => {
    // All runs are v2. 
    runs = [newMockRun(MOCK_RUN_1_ID, true), newMockRun(MOCK_RUN_2_ID, true), newMockRun(MOCK_RUN_3_ID, true)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(_ => {
      return false;
    });

    render(
      <CommonTestWrapper>
        <Compare {...generateProps()} />
      </CommonTestWrapper>
    );

    waitFor(() => screen.getByTestId('compare-runs-v1'));
  });
});
