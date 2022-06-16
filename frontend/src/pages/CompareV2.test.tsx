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

import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import * as React from 'react';
import { CommonTestWrapper } from 'src/TestWrapper';
import TestUtils, { testBestPractices } from 'src/TestUtils';
import { Artifact, Context, Event, Execution, Value } from 'src/third_party/mlmd';
import { Apis } from 'src/lib/Apis';
import { QUERY_PARAMS } from 'src/components/Router';
import * as mlmdUtils from 'src/mlmd/MlmdUtils';
import CompareV2 from './CompareV2';
import { PageProps } from './Page';
import { ApiRunDetail } from 'src/apis/run';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';

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

  function newMockContext(name: string, id: number): Execution {
    const context = new Context();
    context.setName(name);
    context.setId(id);
    return context;
  }

  function newMockExecution(id: number): Execution {
    const execution = new Execution();
    execution.setId(id);
    return execution;
  }

  function newMockEvent(id: number): Event {
    const event = new Event();
    event.setArtifactId(id);
    event.setExecutionId(id);
    event.setType(Event.Type.OUTPUT);
    return event;
  }

  function newMockArtifact(id: number): Artifact {
    const artifact = new Artifact();
    artifact.setId(id);
    return artifact;
  }

  it('Render Compare v2 page', async () => {
    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    screen.getByText(OVERVIEW_SECTION_NAME);
  });

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

  it('Clear banner when getRun and MLMD requests succeed', async () => {
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

    const artifacts = [newMockArtifact(1), newMockArtifact(2), newMockArtifact(3)];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [newMockEvent(1), newMockEvent(2), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue([]);

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(getContextSpy).toBeCalledTimes(3);
      expect(getExecutionsSpy).toBeCalledTimes(3);
      expect(getArtifactsSpy).toBeCalledTimes(3);
      expect(getEventsSpy).toBeCalledTimes(3);
      expect(updateBannerSpy).toHaveBeenLastCalledWith({});
    });
  });

  it('Show page error on page when getRun request fails', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
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

    expect(updateBannerSpy).toHaveBeenLastCalledWith({
      additionalInfo: 'test error',
      message: 'Error: failed loading 3 runs. Click Details for more information.',
      mode: 'error',
    });
  });

  it('Failed MLMD request creates error banner', async () => {
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

  it('Failed getArtifactTypes request creates error banner', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    jest.spyOn(mlmdUtils, 'getKfpV2RunContext').mockReturnValue(new Context());
    jest.spyOn(mlmdUtils, 'getExecutionsFromContext').mockReturnValue([]);
    jest.spyOn(mlmdUtils, 'getArtifactsFromContext').mockReturnValue([]);
    jest.spyOn(mlmdUtils, 'getEventsByExecutions').mockReturnValue([]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockRejectedValue('test error');

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        message: 'Cannot get Artifact Types for MLMD.',
        mode: 'error',
      });
    });
  });


  it('Allows individual sections to be collapsed and expanded', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    screen.getByText('Filter runs');
    screen.getByText('Parameter Section V2');
    screen.getByText('Metrics Section V2');

    fireEvent.click(screen.getByText(OVERVIEW_SECTION_NAME));
    expect(screen.queryByText('Filter runs')).toBeNull();

    fireEvent.click(screen.getByText(OVERVIEW_SECTION_NAME));
    screen.getByText('Filter runs');

    fireEvent.click(screen.getByText(PARAMS_SECTION_NAME));
    expect(screen.queryByText('Parameter Section V2')).toBeNull();

    fireEvent.click(screen.getByText(METRICS_SECTION_NAME));
    expect(screen.queryByText('Metrics Section V2')).toBeNull();
  });

  it('All runs are initially selected', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    // Four checkboxes: three runs and one table header
    let runCheckboxes = screen.queryAllByRole('checkbox', { checked: true });
    expect(runCheckboxes.filter(r => r.nodeName === 'INPUT')).toHaveLength(4);

    // Uncheck all run checkboxes
    fireEvent.click(runCheckboxes[0]);
    runCheckboxes = screen.queryAllByRole('checkbox', { checked: true });
    expect(runCheckboxes.filter(r => r.nodeName === 'INPUT')).toHaveLength(0);
  });
});
