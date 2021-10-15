/*
 * Copyright 2021 The Kubeflow Authors
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

import { act, queryByText, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import * as React from 'react';
import { ApiRelationship, ApiResourceType } from 'src/apis/run';
import { RoutePage, RouteParams } from 'src/components/Router';
import * as v2PipelineSpec from 'src/data/test/mock_lightweight_python_functions_v2_pipeline.json';
import { Apis } from 'src/lib/Apis';
import { Api } from 'src/mlmd/Api';
import { KFP_V2_RUN_CONTEXT_TYPE } from 'src/mlmd/MlmdUtils';
import { mockResizeObserver, testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import {
  Context,
  GetContextByTypeAndNameRequest,
  GetContextByTypeAndNameResponse,
  GetExecutionsByContextResponse,
} from 'src/third_party/mlmd';
import {
  GetArtifactsByContextResponse,
  GetEventsByExecutionIDsResponse,
} from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';
import { PageProps } from './Page';
import { RunDetailsInternalProps } from './RunDetails';
import { RunDetailsV2 } from './RunDetailsV2';

testBestPractices();
describe('RunDetailsV2', () => {
  const RUN_ID = '1';

  let updateBannerSpy: any;
  let updateDialogSpy: any;
  let updateSnackbarSpy: any;
  let updateToolbarSpy: any;
  let historyPushSpy: any;

  function generateProps(): RunDetailsInternalProps & PageProps {
    const pageProps: PageProps = {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: {
        params: {
          [RouteParams.runId]: RUN_ID,
        },
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
    return Object.assign(pageProps, {
      gkeMetadata: {},
    });
  }
  const TEST_RUN = {
    pipeline_runtime: {
      workflow_manifest: '{}',
    },
    run: {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      scheduled_at: new Date(2018, 8, 6, 4, 3, 2),
      finished_at: new Date(2018, 8, 7, 4, 3, 2),
      description: 'test run description',
      id: 'test-run-id',
      name: 'test run',
      pipeline_spec: {
        parameters: [{ name: 'param1', value: 'value1' }],
        pipeline_id: 'some-pipeline-id',
      },
      resource_references: [
        {
          key: { id: 'some-experiment-id', type: ApiResourceType.EXPERIMENT },
          name: 'some experiment',
          relationship: ApiRelationship.OWNER,
        },
        {
          key: { id: 'test-run-id', type: ApiResourceType.PIPELINEVERSION },
          name: 'default',
          relationship: ApiRelationship.CREATOR,
        },
      ],
      status: 'Succeeded',
    },
  };
  const TEST_EXPERIMENT = {
    created_at: '2021-01-24T18:03:08Z',
    description: 'All runs will be grouped here.',
    id: 'some-experiment-id',
    name: 'Default',
    storage_state: 'STORAGESTATE_AVAILABLE',
  };
  beforeEach(() => {
    mockResizeObserver();

    updateBannerSpy = jest.fn();
    updateToolbarSpy = jest.fn();
  });

  it('Render detail page with reactflow', async () => {
    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={JSON.stringify(v2PipelineSpec)}
          runDetail={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );
    expect(screen.getByTestId('DagCanvas')).not.toBeNull();
  });

  it('Shows error banner when disconnected from MLMD', async () => {
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName')
      .mockRejectedValue(new Error('Not connected to MLMD'));

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={JSON.stringify(v2PipelineSpec)}
          runDetail={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo:
            'Cannot find context with {"typeName":"system.PipelineRun","contextName":"1"}: Not connected to MLMD',
          message: 'Cannot get MLMD objects from Metadata store.',
          mode: 'error',
        }),
      ),
    );
  });

  it('Shows no banner when connected from MLMD', async () => {
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName')
      .mockImplementation((request: GetContextByTypeAndNameRequest) => {
        const response = new GetContextByTypeAndNameResponse();
        if (
          request.getTypeName() === KFP_V2_RUN_CONTEXT_TYPE &&
          request.getContextName() === RUN_ID
        ) {
          response.setContext(new Context());
        }
        return response;
      });
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getExecutionsByContext')
      .mockResolvedValue(new GetExecutionsByContextResponse());
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByContext')
      .mockResolvedValue(new GetArtifactsByContextResponse());
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getEventsByExecutionIDs')
      .mockResolvedValue(new GetEventsByExecutionIDsResponse());

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={JSON.stringify(v2PipelineSpec)}
          runDetail={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await waitFor(() => expect(updateBannerSpy).toHaveBeenLastCalledWith({}));
  });

  it("shows run title and experiments' links", async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    getRunSpy.mockResolvedValue(TEST_RUN);
    const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
    getExperimentSpy.mockResolvedValue(TEST_EXPERIMENT);

    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName')
      .mockImplementation((request: GetContextByTypeAndNameRequest) => {
        return new GetContextByTypeAndNameResponse();
      });
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getExecutionsByContext')
      .mockResolvedValue(new GetExecutionsByContextResponse());
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByContext')
      .mockResolvedValue(new GetArtifactsByContextResponse());
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getEventsByExecutionIDs')
      .mockResolvedValue(new GetEventsByExecutionIDsResponse());

    await act(async () => {
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={JSON.stringify(v2PipelineSpec)}
            runDetail={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );
    });

    await waitFor(() =>
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          pageTitleTooltip: 'test run',
        }),
      ),
    );
    await waitFor(() =>
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          breadcrumbs: [
            { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
            {
              displayName: 'Default',
              href: `/experiments/details/some-experiment-id`,
            },
          ],
        }),
      ),
    );
  });

  it('shows top bar buttons', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    getRunSpy.mockResolvedValue(TEST_RUN);
    const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
    getExperimentSpy.mockResolvedValue(TEST_EXPERIMENT);

    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getContextByTypeAndName')
      .mockResolvedValue(new GetContextByTypeAndNameResponse());
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getExecutionsByContext')
      .mockResolvedValue(new GetExecutionsByContextResponse());
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByContext')
      .mockResolvedValue(new GetArtifactsByContextResponse());
    jest
      .spyOn(Api.getInstance().metadataStoreService, 'getEventsByExecutionIDs')
      .mockResolvedValue(new GetEventsByExecutionIDsResponse());

    await act(async () => {
      render(
        <CommonTestWrapper>
          <RunDetailsV2
            pipeline_job={JSON.stringify(v2PipelineSpec)}
            runDetail={TEST_RUN}
            {...generateProps()}
          ></RunDetailsV2>
        </CommonTestWrapper>,
      );
    });

    await waitFor(() =>
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          actions: expect.objectContaining({
            archive: expect.objectContaining({ disabled: false, title: 'Archive' }),
            retry: expect.objectContaining({ disabled: true, title: 'Retry' }),
            terminateRun: expect.objectContaining({ disabled: true, title: 'Terminate' }),
            cloneRun: expect.objectContaining({ disabled: false, title: 'Clone run' }),
          }),
        }),
      ),
    );
  });

  it('shows Execution Sidepanel', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    getRunSpy.mockResolvedValue(TEST_RUN);
    const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
    getExperimentSpy.mockResolvedValue(TEST_EXPERIMENT);

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={JSON.stringify(v2PipelineSpec)}
          runDetail={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    // Default view has no side panel.
    expect(screen.queryByText('Input/Output')).toBeNull();
    expect(screen.queryByText('Task Details')).toBeNull();

    // Select execution to open side panel.
    userEvent.click(screen.getByText('preprocess'));
    screen.getByText('Input/Output');
    screen.getByText('Task Details');

    // Close side panel.
    userEvent.click(screen.getByLabelText('close'));
    expect(screen.queryByText('Input/Output')).toBeNull();
    expect(screen.queryByText('Task Details')).toBeNull();
  });

  it('shows Artifact Sidepanel', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    getRunSpy.mockResolvedValue(TEST_RUN);
    const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
    getExperimentSpy.mockResolvedValue(TEST_EXPERIMENT);

    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={JSON.stringify(v2PipelineSpec)}
          runDetail={TEST_RUN}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    // Default view has no side panel.
    expect(screen.queryByText('Artifact Info')).toBeNull();
    expect(screen.queryByText('Visualization')).toBeNull();

    // Select artifact to open side panel.
    userEvent.click(screen.getByText('model'));
    screen.getByText('Artifact Info');
    screen.getByText('Visualization');

    // Close side panel.
    userEvent.click(screen.getByLabelText('close'));
    expect(screen.queryByText('Artifact Info')).toBeNull();
    expect(screen.queryByText('Visualization')).toBeNull();
  });
});
