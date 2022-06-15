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

import { render, screen } from '@testing-library/react';
import * as React from 'react';
import { CommonTestWrapper } from 'src/TestWrapper';
import { testBestPractices } from 'src/TestUtils';
import {
  Context,
  Execution,
  GetContextByTypeAndNameRequest,
  GetContextByTypeAndNameResponse,
  GetExecutionsByContextRequest,
  GetExecutionsByContextResponse,
  Value,
} from 'src/third_party/mlmd';
import { Apis } from 'src/lib/Apis';
import { Api } from 'src/mlmd/Api';
import CompareV2 from './CompareV2';
import { QUERY_PARAMS } from 'src/components/Router';
import { PageProps } from './Page';
import { ApiRunDetail } from 'src/apis/run';
import TestUtils from 'src/TestUtils';

testBestPractices();
describe('CompareV2', () => {
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

  function newMockRun(id?: string): ApiRunDetail {
    return {
      pipeline_runtime: {
        workflow_manifest: '{}',
      },
      run: {
        id: id || 'test-run-id',
        name: 'test run ' + id,
        pipeline_spec: { pipeline_manifest: '' },
      },
    };
  }

  function newMockContext(name?: string, id?: number): Execution {
    const context = new Context();
    context.setName(name);
    context.setId(id);
    return context;
  }

  function newMockExecution(id?: number): Execution {
    const execution = new Execution();
    execution.setId(id);
    execution
      .getCustomPropertiesMap()
      .set('display_name', new Value().setStringValue(`execution${id}`));
    return execution;
  }

  it('Ensure MLMD requests are accurately made', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const getContextSpy = jest.spyOn(
      Api.getInstance().metadataStoreService,
      'getContextByTypeAndName',
    );
    const getExecutionsSpy = jest.spyOn(
      Api.getInstance().metadataStoreService,
      'getExecutionsByContext',
    );

    const contextResponses = [
      new GetContextByTypeAndNameResponse().setContext(newMockContext(MOCK_RUN_1_ID, 1)),
      new GetContextByTypeAndNameResponse().setContext(newMockContext(MOCK_RUN_2_ID, 2)),
      new GetContextByTypeAndNameResponse().setContext(newMockContext(MOCK_RUN_3_ID, 3)),
    ];
    getContextSpy.mockImplementation((request: GetContextByTypeAndNameRequest) =>
      contextResponses.find(c => c.getContext().getName() === request.getContextName())
    );

    const executionResponses = [
      new GetExecutionsByContextResponse().setExecutionsList([newMockExecution(1)]),
      new GetExecutionsByContextResponse().setExecutionsList([newMockExecution(2)]),
      new GetExecutionsByContextResponse().setExecutionsList([newMockExecution(3)]),
    ]
    getExecutionsSpy.mockImplementation((request: GetExecutionsByContextRequest) =>
      executionResponses.find(e => e.getExecutionsList()[0].getId() === 1)
    );

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_1_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_2_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_3_ID);

    expect(getContextSpy).toHaveBeenCalledWith(expect.objectContaining({ array: ['system.PipelineRun', MOCK_RUN_1_ID] }));
    expect(getContextSpy).toHaveBeenCalledWith(expect.objectContaining({ array: ['system.PipelineRun', MOCK_RUN_2_ID] }));
    expect(getContextSpy).toHaveBeenCalledWith(expect.objectContaining({ array: ['system.PipelineRun', MOCK_RUN_3_ID] }));

    expect(getExecutionsSpy).toHaveBeenCalledWith(expect.objectContaining({ array: [1] }));
    expect(getExecutionsSpy).toHaveBeenCalledWith(expect.objectContaining({ array: [2] }));
    expect(getExecutionsSpy).toHaveBeenCalledWith(expect.objectContaining({ array: [3] }));
  });

  it('Render Compare v2 page', async () => {
    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    screen.getByText('This is the V2 Run Comparison page.');
  });
});
