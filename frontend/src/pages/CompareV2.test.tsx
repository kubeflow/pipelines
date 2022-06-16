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
import TestUtils, { testBestPractices } from 'src/TestUtils';
import { Artifact, Context, Event, Execution, LinkedArtifact, Value } from 'src/third_party/mlmd';
import { Apis } from 'src/lib/Apis';
import * as mlmdUtils from 'src/mlmd/MlmdUtils';
import CompareV2 from './CompareV2';
import { QUERY_PARAMS } from 'src/components/Router';
import { PageProps } from './Page';
import { ApiRunDetail } from 'src/apis/run';

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

  function newMockLinkedArtifact(id?: number): LinkedArtifact {
    const event = new Event();
    event.setArtifactId(id);
    event.setType(Event.Type.OUTPUT);

    const artifact = new Artifact();
    artifact.setId(id);
    return {
      event,
      artifact,
    } as LinkedArtifact;
  }

  it('getRun is called with query param IDs', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );

    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_1_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_2_ID);
    expect(getRunSpy).toHaveBeenCalledWith(MOCK_RUN_3_ID);
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

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
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

  it('Successful MLMD requests clear the banner', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 2),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = jest.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    const executions = [[newMockExecution(1)], [newMockExecution(2)], [newMockExecution(3)]];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const linkedArtifacts = [
      [newMockLinkedArtifact(1)],
      [newMockLinkedArtifact(2)],
      [newMockLinkedArtifact(3)],
    ];
    const getLinkedArtifactsSpy = jest.spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution');
    getLinkedArtifactsSpy.mockImplementation((execution: Execution) =>
      Promise.resolve(linkedArtifacts.find(l => l[0].artifact.getId() === execution.getId())),
    );

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(getContextSpy).toBeCalledTimes(3);
      expect(getExecutionsSpy).toBeCalledTimes(3);
      expect(getLinkedArtifactsSpy).toBeCalledTimes(3);
      expect(updateBannerSpy).toHaveBeenLastCalledWith({});
    });
  });

  it('Failed MLMD request create error banner', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        additionalInfo:
          'Cannot find context with ' +
          '{"typeName":"system.PipelineRun","contextName":"mock-run-1-id"}: Incomplete response',
        message: 'Cannot get MLMD objects from Metadata store.',
        mode: 'error',
      });
    });
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
