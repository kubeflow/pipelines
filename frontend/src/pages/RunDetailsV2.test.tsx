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

import { render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { RouteParams } from 'src/components/Router';
import * as v2PipelineSpec from 'src/data/test/mock_lightweight_python_functions_v2_pipeline.json';
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

  beforeEach(() => {
    mockResizeObserver();

    updateBannerSpy = jest.fn();
  });

  it('Render detail page with reactflow', async () => {
    render(
      <CommonTestWrapper>
        <RunDetailsV2
          pipeline_job={JSON.stringify(v2PipelineSpec)}
          runId={RUN_ID}
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
          runId={RUN_ID}
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
          runId={RUN_ID}
          {...generateProps()}
        ></RunDetailsV2>
      </CommonTestWrapper>,
    );

    await waitFor(() => expect(updateBannerSpy).toHaveBeenLastCalledWith({}));
  });
});
