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
import { QUERY_PARAMS } from 'src/components/Router';
import { ApiRunDetail } from 'src/apis/run';
import Compare from './Compare';
import * as features from 'src/features';
import TestUtils, { testBestPractices } from 'src/TestUtils';

testBestPractices();
describe('Switch between v1 and v2 Run Comparison pages', () => {
  const MOCK_RUN_1_ID = 'mock-run-1-id';
  const MOCK_RUN_2_ID = 'mock-run-2-id';
  const MOCK_RUN_3_ID = 'mock-run-3-id';
  const updateBannerSpy = jest.fn();

  function generateProps(): PageProps {
    const pageProps: PageProps = {
      history: {} as any,
      location: {
        search: `?${QUERY_PARAMS.runlist}=${MOCK_RUN_1_ID},${MOCK_RUN_2_ID},${MOCK_RUN_3_ID}`,
      } as any,
      match: {} as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: () => null,
      updateSnackbar: () => null,
      updateToolbar: () => null,
    };
    return pageProps;
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

  it('getRun is called with query param IDs', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [
      newMockRun(MOCK_RUN_1_ID, true),
      newMockRun(MOCK_RUN_2_ID, true),
      newMockRun(MOCK_RUN_3_ID, true),
    ];
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
      </CommonTestWrapper>,
    );

    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_1_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_2_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_3_ID);
  });

  it('Show v1 page if all runs are v1 and the v2 feature flag is enabled', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [
      newMockRun(MOCK_RUN_1_ID, false),
      newMockRun(MOCK_RUN_2_ID, false),
      newMockRun(MOCK_RUN_3_ID, false),
    ];
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
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => expect(screen.queryByText('Scalar Metrics')).toBeNull());
  });

  it('Show mixed version runs page error if run versions are mixed between v1 and v2', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [
      newMockRun(MOCK_RUN_1_ID, false),
      newMockRun(MOCK_RUN_2_ID, true),
      newMockRun(MOCK_RUN_3_ID, true),
    ];
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
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        additionalInfo:
          'The selected runs are a mix of V1 and V2.' +
          ' Please select all V1 or all V2 runs to view the associated Run Comparison page.',
        message:
          'Error: failed loading the Run Comparison page. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show invalid run count page error if there are less than two runs', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID, true)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=${MOCK_RUN_1_ID}`;
    render(
      <CommonTestWrapper>
        <Compare {...props} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        additionalInfo:
          'At least two runs and at most ten runs must be selected to view the Run Comparison page.',
        message:
          'Error: failed loading the Run Comparison page. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show invalid run count page error if there are more than ten runs', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [
      newMockRun('1', true),
      newMockRun('2', true),
      newMockRun('3', true),
      newMockRun('4', true),
      newMockRun('5', true),
      newMockRun('6', true),
      newMockRun('7', true),
      newMockRun('8', true),
      newMockRun('9', true),
      newMockRun('10', true),
      newMockRun('11', true),
    ];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn on.
    jest.spyOn(features, 'isFeatureEnabled').mockImplementation(featureKey => {
      if (featureKey === features.FeatureKey.V2_ALPHA) {
        return true;
      }
      return false;
    });

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=1,2,3,4,5,6,7,8,9,10,11`;
    render(
      <CommonTestWrapper>
        <Compare {...props} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        additionalInfo:
          'At least two runs and at most ten runs must be selected to view the Run Comparison page.',
        message:
          'Error: failed loading the Run Comparison page. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Show no error on v1 page if run versions are mixed between v1 and v2 and feature flag disabled', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [
      newMockRun(MOCK_RUN_1_ID, false),
      newMockRun(MOCK_RUN_2_ID, true),
      newMockRun(MOCK_RUN_3_ID, true),
    ];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(false);

    render(
      <CommonTestWrapper>
        <Compare {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => expect(screen.queryByText('Scalar Metrics')).toBeNull());
  });

  it('Show no error on v1 page if there are less than two runs and v2 feature flag disabled', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID, true)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(false);

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=${MOCK_RUN_1_ID}`;
    render(
      <CommonTestWrapper>
        <Compare {...props} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => expect(screen.queryByText('Scalar Metrics')).toBeNull());
  });

  it('Show v2 page if all runs are v2 and the v2 feature flag is enabled', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [
      newMockRun(MOCK_RUN_1_ID, true),
      newMockRun(MOCK_RUN_2_ID, true),
      newMockRun(MOCK_RUN_3_ID, true),
    ];
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
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => screen.getByText('Scalar Metrics'));
  });

  it('Show v1 page if some runs are v1 and the v2 feature flag is disabled', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [
      newMockRun(MOCK_RUN_1_ID, false),
      newMockRun(MOCK_RUN_2_ID, true),
      newMockRun(MOCK_RUN_3_ID, true),
    ];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(false);

    render(
      <CommonTestWrapper>
        <Compare {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => expect(screen.queryByText('Scalar Metrics')).toBeNull());
  });

  it('Show v1 page if all runs are v2 and the v2 feature flag is disabled', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [
      newMockRun(MOCK_RUN_1_ID, true),
      newMockRun(MOCK_RUN_2_ID, true),
      newMockRun(MOCK_RUN_3_ID, true),
    ];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    // v2 feature is turn off.
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(false);

    render(
      <CommonTestWrapper>
        <Compare {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => expect(screen.queryByText('Scalar Metrics')).toBeNull());
  });

  it('Show page error on page when getRun request fails', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [
      newMockRun(MOCK_RUN_1_ID, true),
      newMockRun(MOCK_RUN_2_ID, true),
      newMockRun(MOCK_RUN_3_ID, true),
    ];
    getRunSpy.mockImplementation(_ => {
      throw {
        text: () => Promise.resolve('test error'),
      };
    });

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
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        additionalInfo: 'test error',
        message: 'Error: failed loading 3 runs. Click Details for more information.',
        mode: 'error',
      }),
    );
  });
});
