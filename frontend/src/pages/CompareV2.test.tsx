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
import { Artifact, Context, Event, Execution } from 'src/third_party/mlmd';
import { Apis } from 'src/lib/Apis';
import { QUERY_PARAMS } from 'src/components/Router';
import * as mlmdUtils from 'src/mlmd/MlmdUtils';
import * as Utils from 'src/lib/Utils';
import CompareV2 from './CompareV2';
import { PageProps } from './Page';
import { ApiRunDetail } from 'src/apis/run';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';
import { Value } from 'google-protobuf/google/protobuf/struct_pb';

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

  function newMockRun(id?: string, hideName?: boolean): ApiRunDetail {
    return {
      pipeline_runtime: {
        workflow_manifest: '{}',
      },
      run: {
        id: id || 'test-run-id',
        name: hideName ? undefined : 'test run ' + id,
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

  function newMockExecution(id: number, displayName?: string): Execution {
    const execution = new Execution();
    execution.setId(id);
    if (displayName) {
      const customPropertiesMap: Map<string, Value> = new Map();
      const displayNameValue = new Value();
      displayNameValue.setStringValue(displayName);
      customPropertiesMap.set('display_name', displayNameValue);
      jest.spyOn(execution, 'getCustomPropertiesMap').mockReturnValue(customPropertiesMap);
    }
    return execution;
  }

  function newMockEvent(id: number, displayName?: string): Event {
    const event = new Event();
    event.setArtifactId(id);
    event.setExecutionId(id);
    event.setType(Event.Type.OUTPUT);
    if (displayName) {
      const path = new Event.Path();
      const step = new Event.Path.Step();
      step.setKey(displayName);
      path.addSteps(step);
      event.setPath(path);
    }
    return event;
  }

  function newMockArtifact(id: number, isConfusionMatrix?: boolean): Artifact {
    const artifact = new Artifact();
    artifact.setId(id);
    if (isConfusionMatrix) {
      const customPropertiesMap: Map<string, Value> = new Map();
      customPropertiesMap.set('confusionMatrix', new Value());
      jest.spyOn(artifact, 'getCustomPropertiesMap').mockReturnValue(customPropertiesMap);
    }
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

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

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
      expect(getArtifactTypesSpy).toBeCalledTimes(1);
      expect(updateBannerSpy).toHaveBeenLastCalledWith({});
    });
  });

  it('Log warning when artifact with specified ID is not found', async () => {
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

    const artifacts = [newMockArtifact(1), newMockArtifact(3)];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [newMockEvent(1), newMockEvent(2), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    const warnSpy = jest.spyOn(Utils.logger, 'warn');

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(warnSpy).toHaveBeenLastCalledWith(
        'The artifact with the following ID was not found: 2',
      );
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

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        additionalInfo: 'test error',
        message: 'Error: failed loading 3 runs. Click Details for more information.',
        mode: 'error',
      }),
    );
  });

  it('Failed MLMD request creates error banner', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));
    jest
      .spyOn(mlmdUtils, 'getKfpV2RunContext')
      .mockRejectedValue(new Error('Not connected to MLMD'));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        additionalInfo: 'Not connected to MLMD',
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
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockRejectedValue(new Error('Not connected to MLMD'));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        additionalInfo: 'Not connected to MLMD',
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
    screen.getByText('Scalar Metrics');

    fireEvent.click(screen.getByText(OVERVIEW_SECTION_NAME));
    expect(screen.queryByText('Filter runs')).toBeNull();

    fireEvent.click(screen.getByText(OVERVIEW_SECTION_NAME));
    screen.getByText('Filter runs');

    fireEvent.click(screen.getByText(PARAMS_SECTION_NAME));
    expect(screen.queryByText('Parameter Section V2')).toBeNull();

    fireEvent.click(screen.getByText(METRICS_SECTION_NAME));
    expect(screen.queryByText('Scalar Metrics')).toBeNull();
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

  it('Scalar metrics tab initially enabled, and switch tabs', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    screen.getByText('This is the Scalar Metrics tab.');

    fireEvent.click(screen.getByText('ROC Curve'));
    screen.getByText('This is the ROC Curve tab.');
    expect(screen.queryByText('This is the Scalar Metrics Tab')).toBeNull();

    fireEvent.click(screen.getByText('ROC Curve'));
    screen.getByText('This is the ROC Curve tab.');

    fireEvent.click(screen.getByText('Scalar Metrics'));
    screen.getByText('This is the Scalar Metrics tab.');
    expect(screen.queryByText('This is the ROC Curve Tab')).toBeNull();
  });

  it('Two-panel tabs have no dropdown loaded as content is not present', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    jest.spyOn(mlmdUtils, 'getKfpV2RunContext').mockReturnValue(new Context());
    jest.spyOn(mlmdUtils, 'getExecutionsFromContext').mockReturnValue([]);
    jest.spyOn(mlmdUtils, 'getArtifactsFromContext').mockReturnValue([]);
    jest.spyOn(mlmdUtils, 'getEventsByExecutions').mockReturnValue([]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue([]);

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('Confusion Matrix'));
    screen.getByText('There are no Confusion Matrix artifacts available on the selected runs.');

    fireEvent.click(screen.getByText('HTML'));
    screen.getByText('There are no HTML artifacts available on the selected runs.');

    fireEvent.click(screen.getByText('Markdown'));
    screen.getByText('There are no Markdown artifacts available on the selected runs.');
  });

  it('Only confusion matrix tab has dropdown loaded with content', async () => {
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

    const executions = [
      [newMockExecution(1)],
      [newMockExecution(2, 'executionName')],
      [newMockExecution(3)],
    ];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [newMockArtifact(1), newMockArtifact(2, true), newMockArtifact(3)];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [newMockEvent(1), newMockEvent(2, 'artifactName'), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    // Simulate all artifacts as type "ClassificationMetrics" (Confusion Matrix or ROC Curve).
    const filterLinkedArtifactsByTypeSpy = jest.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
    filterLinkedArtifactsByTypeSpy.mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.ClassificationMetrics' ? linkedArtifacts : [],
    );

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => expect(filterLinkedArtifactsByTypeSpy).toHaveBeenCalledTimes(9));

    fireEvent.click(screen.getByText('Confusion Matrix'));
    screen.getByText('Choose a first Confusion Matrix artifact');

    fireEvent.click(screen.getByText('HTML'));
    screen.getByText('There are no HTML artifacts available on the selected runs.');

    fireEvent.click(screen.getByText('Markdown'));
    screen.getByText('There are no Markdown artifacts available on the selected runs.');
  });

  it('Log warnings when specified run, execution, or artifact does not have a name', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID, true), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
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

    const executions = [
      [newMockExecution(1)],
      [newMockExecution(2)],
      [newMockExecution(3, 'executionName')],
    ];
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

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    const filterLinkedArtifactsByTypeSpy = jest.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
    filterLinkedArtifactsByTypeSpy.mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.HTML' ? linkedArtifacts : [],
    );

    const warnSpy = jest.spyOn(Utils.logger, 'warn');

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('HTML'));
    await waitFor(() => {
      expect(warnSpy).toHaveBeenNthCalledWith(
        1,
        `Failed to fetch the name of the run with the following ID: ${MOCK_RUN_1_ID}`,
      );
      expect(warnSpy).toHaveBeenNthCalledWith(
        2,
        'Failed to fetch the display name of the execution with the following ID: 2',
      );
      expect(warnSpy).toHaveBeenLastCalledWith(
        'Failed to fetch the display name of the artifact with the following ID: 3',
      );
    });
  });
});
