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

import { render, screen, waitFor, fireEvent, within } from '@testing-library/react';
import * as React from 'react';
import { CommonTestWrapper } from 'src/TestWrapper';
import TestUtils, { expectErrors, testBestPractices } from 'src/TestUtils';
import { Artifact, Context, Event, Execution } from 'src/third_party/mlmd';
import { Apis } from 'src/lib/Apis';
import { QUERY_PARAMS } from 'src/components/Router';
import * as mlmdUtils from 'src/mlmd/MlmdUtils';
import * as Utils from 'src/lib/Utils';
import { TEST_ONLY } from './CompareV2';
import { PageProps } from './Page';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';
import { Struct, Value } from 'google-protobuf/google/protobuf/struct_pb';
import { V2beta1Run, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { vi } from 'vitest';

const CompareV2 = TEST_ONLY.CompareV2;
testBestPractices();
describe('CompareV2', () => {
  const MOCK_RUN_1_ID = 'mock-run-1-id';
  const MOCK_RUN_2_ID = 'mock-run-2-id';
  const MOCK_RUN_3_ID = 'mock-run-3-id';
  const updateBannerSpy = vi.fn();
  const getBodyText = (): string => (document.body.textContent || '').replace(/\s+/g, ' ').trim();

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

  let runs: V2beta1Run[] = [];

  beforeEach(() => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id) || newMockRun(id));

    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockResolvedValue(new Context());
    vi.spyOn(mlmdUtils, 'getExecutionsFromContext').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getArtifactsFromContext').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getEventsByExecutions').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([]);
  });

  function newMockRun(id?: string, hideName?: boolean): V2beta1Run {
    return {
      run_id: id || 'test-run-id',
      display_name: hideName ? undefined : 'test run ' + id,
      pipeline_spec: { pipeline_manifest: '' },
      state: V2beta1RuntimeState.SUCCEEDED,
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
    const resolvedDisplayName = displayName || `execution-${id}`;
    if (resolvedDisplayName) {
      const customPropertiesMap: Map<string, Value> = new Map();
      const displayNameValue = new Value();
      displayNameValue.setStringValue(resolvedDisplayName);
      customPropertiesMap.set('display_name', displayNameValue);
      vi.spyOn(execution, 'getCustomPropertiesMap').mockReturnValue(customPropertiesMap);
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
    vi.spyOn(artifact, 'getCustomPropertiesMap').mockReturnValue(customPropertiesMap);
    return artifact;
  }

  function getRunListContainer(): HTMLElement {
    const filterInput = screen.getByLabelText('Filter runs');
    return (filterInput.closest('[class*="pageOverflowHidden"]') as HTMLElement) || document.body;
  }

  function getHeaderCheckbox(): HTMLInputElement {
    const runListContainer = getRunListContainer();
    const headerCheckbox = runListContainer.querySelector(
      '[class*="header"] input[type="checkbox"]',
    ) as HTMLInputElement | null;
    if (!headerCheckbox) {
      throw new Error('Header checkbox not found in run list.');
    }
    return headerCheckbox;
  }

  async function waitForRunCheckboxes(expectedCount: number): Promise<HTMLElement[]> {
    let runCheckboxes: HTMLElement[] = [];
    await waitFor(() => {
      const runListContainer = getRunListContainer();
      runCheckboxes = Array.from(
        runListContainer.querySelectorAll('[data-testid="table-row"][aria-checked="true"]'),
      );
      expect(runCheckboxes).toHaveLength(expectedCount);
    });
    return runCheckboxes;
  }

  async function waitForRunLabel(id: string): Promise<void> {
    await waitFor(() => {
      expect(screen.queryAllByText(`test run ${id}`).length).toBeGreaterThan(1);
    });
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
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

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
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 2),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    const executions = [[newMockExecution(1)], [newMockExecution(2)], [newMockExecution(3)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [newMockArtifact(1), newMockArtifact(2), newMockArtifact(3)];
    const getArtifactsSpy = vi.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(2), newMockEvent(3)];
    const getEventsSpy = vi.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

    const getArtifactTypesSpy = vi.spyOn(mlmdUtils, 'getArtifactTypes');
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
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 2),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    const executions = [[newMockExecution(1)], [newMockExecution(2)], [newMockExecution(3)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [newMockArtifact(1), newMockArtifact(3)];
    const getArtifactsSpy = vi.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(2), newMockEvent(3)];
    const getEventsSpy = vi.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

    const getArtifactTypesSpy = vi.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    const warnSpy = vi.spyOn(Utils.logger, 'warn').mockImplementation(() => undefined);

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
    const expectError = expectErrors();
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
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
    expectError();
  });

  it('Failed MLMD request creates error banner', async () => {
    const expectError = expectErrors();
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));
    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockRejectedValue(new Error('Not connected to MLMD'));

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
    expectError();
  });

  it('Failed getArtifactTypes request creates error banner', async () => {
    const expectError = expectErrors();
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockReturnValue(new Context());
    vi.spyOn(mlmdUtils, 'getExecutionsFromContext').mockReturnValue([]);
    vi.spyOn(mlmdUtils, 'getArtifactsFromContext').mockReturnValue([]);
    vi.spyOn(mlmdUtils, 'getEventsByExecutions').mockReturnValue([]);
    vi.spyOn(mlmdUtils, 'getArtifactTypes').mockRejectedValue(new Error('Not connected to MLMD'));

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
    expectError();
  });

  it('Allows individual sections to be collapsed and expanded', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    screen.getByLabelText('Filter runs');
    screen.getByText('There are no Parameters available on the selected runs.');
    screen.getByText('Scalar Metrics');

    const getSectionToggle = (name: string) =>
      screen.getByText(name).closest('button') as HTMLButtonElement;

    fireEvent.click(getSectionToggle(OVERVIEW_SECTION_NAME));
    await waitFor(() => {
      expect(screen.queryByLabelText('Filter runs')).toBeNull();
    });

    fireEvent.click(getSectionToggle(OVERVIEW_SECTION_NAME));
    await screen.findByLabelText('Filter runs');

    fireEvent.click(getSectionToggle(PARAMS_SECTION_NAME));
    await waitFor(() => {
      expect(
        screen.queryByText('There are no Parameters available on the selected runs.'),
      ).toBeNull();
    });

    fireEvent.click(getSectionToggle(METRICS_SECTION_NAME));
    await waitFor(() => {
      expect(screen.queryByText('Scalar Metrics')).toBeNull();
    });
  });

  it('All runs are initially selected', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitForRunCheckboxes(3);
    const headerCheckbox = getHeaderCheckbox();
    expect(headerCheckbox).toBeChecked();

    // Uncheck all run checkboxes.
    fireEvent.click(headerCheckbox);
    await waitForRunCheckboxes(0);
  });

  it('Parameters and Scalar metrics tab initially enabled with loading then error, and switch tabs', async () => {
    const expectError = expectErrors();
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));
    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockRejectedValue(new Error('Not connected to MLMD'));
    vi.spyOn(mlmdUtils, 'getExecutionsFromContext').mockRejectedValue(
      new Error('Not connected to MLMD'),
    );
    vi.spyOn(mlmdUtils, 'getArtifactsFromContext').mockRejectedValue(
      new Error('Not connected to MLMD'),
    );
    vi.spyOn(mlmdUtils, 'getEventsByExecutions').mockRejectedValue(
      new Error('Not connected to MLMD'),
    );
    vi.spyOn(mlmdUtils, 'getArtifactTypes').mockRejectedValue(new Error('Not connected to MLMD'));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    expect(screen.queryAllByRole('circularprogress')).toHaveLength(2);

    await TestUtils.flushPromises();
    await waitFor(() => {
      screen.getByText('There are no Parameters available on the selected runs.');
      expect(getBodyText()).toContain(
        'An error is preventing the Scalar Metrics from being displayed.',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Confusion Matrix' }));
    await waitFor(() => {
      expect(getBodyText()).toContain(
        'An error is preventing the Confusion Matrix from being displayed.',
      );
      expect(getBodyText()).not.toContain(
        'An error is preventing the Scalar Metrics from being displayed.',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Confusion Matrix' }));
    await waitFor(() => {
      expect(getBodyText()).toContain(
        'An error is preventing the Confusion Matrix from being displayed.',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Scalar Metrics' }));
    await waitFor(() => {
      expect(getBodyText()).toContain(
        'An error is preventing the Scalar Metrics from being displayed.',
      );
      expect(getBodyText()).not.toContain(
        'An error is preventing the Confusion Matrix from being displayed.',
      );
    });
    expectError();
  });

  it('Metrics tabs have no content loaded as artifacts are not present', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockResolvedValue(new Context());
    vi.spyOn(mlmdUtils, 'getExecutionsFromContext').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getArtifactsFromContext').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getEventsByExecutions').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([]);

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(getBodyText()).toContain(
        'There are no Scalar Metrics artifacts available on the selected runs.',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Confusion Matrix' }));
    await waitFor(() => {
      expect(getBodyText()).toContain(
        'There are no Confusion Matrix artifacts available on the selected runs.',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'HTML' }));
    await waitFor(() => {
      expect(getBodyText()).toContain(
        'There are no HTML artifacts available on the selected runs.',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'Markdown' }));
    await waitFor(() => {
      expect(getBodyText()).toContain(
        'There are no Markdown artifacts available on the selected runs.',
      );
    });

    fireEvent.click(screen.getByRole('button', { name: 'ROC Curve' }));
    await waitFor(() => {
      expect(getBodyText()).toContain(
        'There are no ROC Curve artifacts available on the selected runs.',
      );
    });
  });

  it('Confusion matrix shown on select, stays after tab change or section collapse', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, true, false, 'artifactName'),
      newMockArtifact(3),
    ];
    const getArtifactsSpy = vi.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(200, 'artifactName'), newMockEvent(3)];
    const getEventsSpy = vi.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

    const getArtifactTypesSpy = vi.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    // Simulate all artifacts as type "ClassificationMetrics" (Confusion Matrix or ROC Curve).
    const filterLinkedArtifactsByTypeSpy = vi.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
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

    fireEvent.click(screen.getByRole('button', { name: 'Confusion Matrix' }));
    fireEvent.click(screen.getByText('Choose a first Confusion Matrix artifact'));

    // Get the second element that has run text: first will be the run list.
    await waitForRunLabel(MOCK_RUN_2_ID);
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    fireEvent.click(screen.getByText(/artifactName/));
    await waitFor(() => {
      screen.getByText(/Confusion Matrix: artifactName/);
      screen.getByText(/200/);
    });

    // Change the tab and return, ensure that the confusion matrix and selected item are present.
    fireEvent.click(screen.getByText('HTML'));
    fireEvent.click(screen.getByRole('button', { name: 'Confusion Matrix' }));
    await waitFor(() => {
      screen.getByText(/Confusion Matrix: artifactName/);
      screen.getByText(/200/);
    });

    // Collapse and expand Metrics, ensure that the confusion matrix and selected item are present.
    fireEvent.click(screen.getByText('Metrics'));
    fireEvent.click(screen.getByText('Metrics'));
    await waitFor(() => {
      screen.getByText(/Confusion Matrix: artifactName/);
      screen.getByText(/200/);
    });
  });

  it('Confusion matrix shown on select and removed after run is de-selected', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, true, false, 'artifactName'),
      newMockArtifact(3),
    ];
    const getArtifactsSpy = vi.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(200, 'artifactName'), newMockEvent(3)];
    const getEventsSpy = vi.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

    const getArtifactTypesSpy = vi.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    // Simulate all artifacts as type "ClassificationMetrics" (Confusion Matrix or ROC Curve).
    const filterLinkedArtifactsByTypeSpy = vi.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
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

    fireEvent.click(screen.getByRole('button', { name: 'Confusion Matrix' }));
    fireEvent.click(screen.getByText('Choose a first Confusion Matrix artifact'));

    // Get the second element that has run text: first will be the run list.
    await waitForRunLabel(MOCK_RUN_2_ID);
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    fireEvent.click(screen.getByText(/artifactName/));
    await waitFor(() => {
      screen.getByText(/Confusion Matrix: artifactName/);
      screen.getByText(/200/);
    });

    // De-selecting the relevant run will remove the confusion matrix display.
    let runCheckboxes = await waitForRunCheckboxes(3);
    fireEvent.click(runCheckboxes[0]);
    await waitFor(() => screen.getByText(/Confusion Matrix: artifactName/));
    runCheckboxes = await waitForRunCheckboxes(2);
    fireEvent.click(runCheckboxes[0]);
    await waitFor(() => expect(screen.queryByText(/Confusion Matrix: artifactName/)).toBeNull());
  }, 20000);

  it('One ROC Curve shown on select, hidden on run de-select', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, false, true, 'artifactName'),
      newMockArtifact(3),
    ];
    const getArtifactsSpy = vi.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [newMockEvent(1), newMockEvent(200, 'artifactName'), newMockEvent(3)];
    const getEventsSpy = vi.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    const getArtifactTypesSpy = vi.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    // Simulate all artifacts as type "ClassificationMetrics" (Confusion Matrix or ROC Curve).
    const filterLinkedArtifactsByTypeSpy = vi.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
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

    fireEvent.click(screen.getByRole('button', { name: 'ROC Curve' }));
    await waitFor(() => screen.getByText('ROC Curve: artifactName'));

    let runCheckboxes = await waitForRunCheckboxes(3);
    fireEvent.click(runCheckboxes[0]);
    await waitFor(() => screen.getByText('ROC Curve: artifactName'));
    runCheckboxes = await waitForRunCheckboxes(2);
    fireEvent.click(runCheckboxes[0]);
    await waitFor(() => expect(screen.queryByText('ROC Curve: artifactName')).toBeNull());
  });

  it('Multiple ROC Curves shown on select', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run_id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 300),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(300)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, false, true, 'firstArtifactName'),
      newMockArtifact(300, false, true, 'secondArtifactName'),
    ];
    const getArtifactsSpy = vi.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [
      newMockEvent(1),
      newMockEvent(200, 'firstArtifactName'),
      newMockEvent(300, 'secondArtifactName'),
    ];
    const getEventsSpy = vi.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    const getArtifactTypesSpy = vi.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    // Simulate all artifacts as type "ClassificationMetrics" (Confusion Matrix or ROC Curve).
    const filterLinkedArtifactsByTypeSpy = vi.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
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

    fireEvent.click(screen.getByRole('button', { name: 'ROC Curve' }));
    await screen.findByText('ROC Curve: multiple artifacts');
    await screen.findByLabelText('Filter artifacts');
  });
});
