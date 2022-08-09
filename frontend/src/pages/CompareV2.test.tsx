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
import { TEST_ONLY } from './CompareV2';
import { PageProps } from './Page';
import { ApiRunDetail } from 'src/apis/run';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';
import { Struct, Value } from 'google-protobuf/google/protobuf/struct_pb';

const CompareV2 = TEST_ONLY.CompareV2;
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

  function newMockArtifact(
    id: number,
    isConfusionMatrix?: boolean,
    isRocCurve?: boolean,
    displayName?: string,
  ): Artifact {
    const artifact = new Artifact();
    artifact.setId(id);
    const customPropertiesMap: Map<string, Value> = new Map();
    if (isConfusionMatrix) {
      const confusionMatrix: Value = new Value();
      confusionMatrix.setStructValue(
        Struct.fromJavaScript({
          struct: {
            annotationSpecs: [
              { displayName: 'Setosa' },
              { displayName: 'Versicolour' },
              { displayName: 'Virginica' },
            ],
            rows: [{ row: [31, 0, 0] }, { row: [1, 8, 12] }, { row: [0, 0, 23] }],
          },
        }),
      );
      customPropertiesMap.set('confusionMatrix', confusionMatrix);
    }
    if (isRocCurve) {
      const confidenceMetrics: Value = new Value();
      confidenceMetrics.setStructValue(
        Struct.fromJavaScript({
          list: [
            {
              confidenceThreshold: 2,
              falsePositiveRate: 0,
              recall: 0,
            },
            {
              confidenceThreshold: 1,
              falsePositiveRate: 0,
              recall: 0.33962264150943394,
            },
            {
              confidenceThreshold: 0.9,
              falsePositiveRate: 0,
              recall: 0.6037735849056604,
            },
          ],
        }),
      );
      customPropertiesMap.set('confidenceMetrics', confidenceMetrics);
    }
    if (displayName) {
      const displayNameValue = new Value();
      displayNameValue.setStringValue(displayName);
      customPropertiesMap.set('display_name', displayNameValue);
    }
    jest.spyOn(artifact, 'getCustomPropertiesMap').mockReturnValue(customPropertiesMap);
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
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(2), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      // Spies are called twice for each artifact as runs change from undefined to a defined value.
      expect(getContextSpy).toBeCalledTimes(6);
      expect(getExecutionsSpy).toBeCalledTimes(6);
      expect(getArtifactsSpy).toBeCalledTimes(6);
      expect(getEventsSpy).toBeCalledTimes(6);
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
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(2), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

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
    screen.getByText('There are no Parameters available on the selected runs.');
    screen.getByText('Scalar Metrics');

    fireEvent.click(screen.getByText(OVERVIEW_SECTION_NAME));
    expect(screen.queryByText('Filter runs')).toBeNull();

    fireEvent.click(screen.getByText(OVERVIEW_SECTION_NAME));
    screen.getByText('Filter runs');

    fireEvent.click(screen.getByText(PARAMS_SECTION_NAME));
    expect(
      screen.queryByText('There are no Parameters available on the selected runs.'),
    ).toBeNull();

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

  it('Parameters and Scalar metrics tab initially enabled with loading then error, and switch tabs', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    expect(screen.queryAllByRole('circularprogress')).toHaveLength(2);

    await TestUtils.flushPromises();
    await waitFor(() => {
      screen.getByText('There are no Parameters available on the selected runs.');
      screen.getByText('An error is preventing the Scalar Metrics from being displayed.');

      fireEvent.click(screen.getByText('Confusion Matrix'));
      screen.getByText('An error is preventing the Confusion Matrix from being displayed.');
      expect(
        screen.queryByText('An error is preventing the Scalar Metrics from being displayed.'),
      ).toBeNull();

      fireEvent.click(screen.getByText('Confusion Matrix'));
      screen.getByText('An error is preventing the Confusion Matrix from being displayed.');

      fireEvent.click(screen.getByText('Scalar Metrics'));
      screen.getByText('An error is preventing the Scalar Metrics from being displayed.');
      expect(
        screen.queryByText('An error is preventing the Confusion Matrix from being displayed.'),
      ).toBeNull();
    });
  });

  it('Metrics tabs have no content loaded as artifacts are not present', async () => {
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

    await waitFor(() => {
      screen.getByText('There are no Scalar Metrics artifacts available on the selected runs.');

      fireEvent.click(screen.getByText('Confusion Matrix'));
      screen.getByText('There are no Confusion Matrix artifacts available on the selected runs.');

      fireEvent.click(screen.getByText('HTML'));
      screen.getByText('There are no HTML artifacts available on the selected runs.');

      fireEvent.click(screen.getByText('Markdown'));
      screen.getByText('There are no Markdown artifacts available on the selected runs.');

      fireEvent.click(screen.getByText('ROC Curve'));
      screen.getByText('There are no ROC Curve artifacts available on the selected runs.');
    });
  });

  it('Confusion matrix shown on select, stays after tab change or section collapse', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = jest.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, true, false, 'artifactName'),
      newMockArtifact(3),
    ];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(200, 'artifactName'), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

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

    await waitFor(() => expect(filterLinkedArtifactsByTypeSpy).toHaveBeenCalledTimes(15));

    expect(screen.queryByText(/Confusion matrix: artifactName/)).toBeNull();

    fireEvent.click(screen.getByText('Confusion Matrix'));
    fireEvent.click(screen.getByText('Choose a first Confusion Matrix artifact'));

    // Get the second element that has run text: first will be the run list.
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    fireEvent.click(screen.getByText(/artifactName/));
    screen.getByText(/Confusion Matrix: artifactName/);
    screen.getByText(/200/);

    // Change the tab and return, ensure that the confusion matrix and selected item are present.
    fireEvent.click(screen.getByText('HTML'));
    fireEvent.click(screen.getByText('Confusion Matrix'));
    screen.getByText(/Confusion Matrix: artifactName/);
    screen.getByText(/200/);

    // Collapse and expand Metrics, ensure that the confusion matrix and selected item are present.
    fireEvent.click(screen.getByText('Metrics'));
    fireEvent.click(screen.getByText('Metrics'));
    screen.getByText(/Confusion Matrix: artifactName/);
    screen.getByText(/200/);
  });

  it('Confusion matrix shown on select and removed after run is de-selected', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = jest.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, true, false, 'artifactName'),
      newMockArtifact(3),
    ];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(200, 'artifactName'), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

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

    expect(screen.queryByText(/Confusion matrix: artifactName/)).toBeNull();

    fireEvent.click(screen.getByText('Confusion Matrix'));
    fireEvent.click(screen.getByText('Choose a first Confusion Matrix artifact'));

    // Get the second element that has run text: first will be the run list.
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    fireEvent.click(screen.getByText(/artifactName/));
    screen.getByText(/Confusion Matrix: artifactName/);
    screen.getByText(/200/);

    // De-selecting the relevant run will remove the confusion matrix display.
    const runCheckboxes = screen
      .queryAllByRole('checkbox', { checked: true })
      .filter(r => r.nodeName === 'INPUT');
    fireEvent.click(runCheckboxes[1]);
    screen.getByText(/Confusion Matrix: artifactName/);
    fireEvent.click(runCheckboxes[2]);
    expect(screen.queryByText(/Confusion Matrix: artifactName/)).toBeNull();
  });

  it('One ROC Curve shown on select, hidden on run de-select', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = jest.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, false, true, 'artifactName'),
      newMockArtifact(3),
    ];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [newMockEvent(1), newMockEvent(200, 'artifactName'), newMockEvent(3)];
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

    fireEvent.click(screen.getByText('ROC Curve'));
    screen.getByText('ROC Curve: artifactName');

    const runCheckboxes = screen
      .queryAllByRole('checkbox', { checked: true })
      .filter(r => r.nodeName === 'INPUT');
    fireEvent.click(runCheckboxes[1]);
    screen.getByText('ROC Curve: artifactName');
    fireEvent.click(runCheckboxes[2]);
    expect(screen.queryByText('ROC Curve: artifactName')).toBeNull();
  });

  it('Multiple ROC Curves shown on select', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 300),
    ];
    const getContextSpy = jest.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(300)]];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, false, true, 'firstArtifactName'),
      newMockArtifact(300, false, true, 'secondArtifactName'),
    ];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [
      newMockEvent(1),
      newMockEvent(200, 'firstArtifactName'),
      newMockEvent(300, 'secondArtifactName'),
    ];
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

    fireEvent.click(screen.getByText('ROC Curve'));
    screen.getByText('ROC Curve: multiple artifacts');
    screen.getByText('Filter artifacts');
  });
});
