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

import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { useEffect, useState } from 'react';
import { CommonTestWrapper } from 'src/TestWrapper';
import TestUtils, { expectErrors, testBestPractices } from 'src/TestUtils';
import { Artifact, Context, Event, Execution } from 'src/third_party/mlmd';
import { Apis } from 'src/lib/Apis';
import { ButtonKeys } from 'src/lib/Buttons';
import { QUERY_PARAMS } from 'src/components/Router';
import { ArtifactType } from 'src/mlmd';
import { LinkedArtifact } from 'src/mlmd/MlmdUtils';
import * as mlmdUtils from 'src/mlmd/MlmdUtils';
import * as Utils from 'src/lib/Utils';
import * as metricsVisualizations from 'src/components/viewers/MetricsVisualizations';
import { TEST_ONLY } from './CompareV2';
import { PageProps } from './Page';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';
import { Struct, Value } from 'google-protobuf/google/protobuf/struct_pb';
import { V2beta1Run, V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { MetricsType } from 'src/lib/v2/CompareUtils';
import { vi } from 'vitest';

const CompareV2 = TEST_ONLY.CompareV2;
testBestPractices();
describe('CompareV2', () => {
  const MOCK_RUN_1_ID = 'mock-run-1-id';
  const MOCK_RUN_2_ID = 'mock-run-2-id';
  const MOCK_RUN_3_ID = 'mock-run-3-id';
  const updateBannerSpy = vi.fn();
  const updateToolbarSpy = vi.fn();
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
      updateToolbar: updateToolbarSpy,
    };
    return pageProps;
  }

  let runs: V2beta1Run[] = [];

  beforeEach(() => {
    vi.clearAllMocks();
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    getRunSpy.mockImplementation(
      (id: string) => runs.find((r) => r.run_id === id) || newMockRun(id),
    );

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

  function getRunRow(id: string): HTMLElement {
    const runRows = within(getRunListContainer()).getAllByTestId('table-row');
    const runRow = runRows.find((row) => row.textContent?.includes(`test run ${id}`));
    if (!runRow) {
      throw new Error(`Run row not found for ${id}`);
    }
    return runRow;
  }

  async function waitForRunCheckboxes(expectedCount: number): Promise<HTMLElement[]> {
    let runCheckboxes: HTMLElement[] = [];
    await waitFor(() => {
      const allRows = within(getRunListContainer()).getAllByTestId('table-row');
      runCheckboxes = allRows.filter((row) => row.getAttribute('aria-checked') === 'true');
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

  it('does not mark ROC selection initialized before ROC artifacts are available', () => {
    const initialSelection = TEST_ONLY.createInitialRocCurveSelectionState();

    expect(TEST_ONLY.reconcileRocCurveSelectionState(initialSelection, [], new Set())).toBe(
      initialSelection,
    );
  });

  it('initializes ROC selection once and only prunes invalid ids on later refreshes', async () => {
    const initialLinkedArtifacts: LinkedArtifact[] = [
      {
        artifact: newMockArtifact(100, false, true, 'artifact-100'),
        event: newMockEvent(100, 'artifact-100'),
      },
      {
        artifact: newMockArtifact(200, false, true, 'artifact-200'),
        event: newMockEvent(200, 'artifact-200'),
      },
      {
        artifact: newMockArtifact(300, false, true, 'artifact-300'),
        event: newMockEvent(300, 'artifact-300'),
      },
    ];
    const refreshedLinkedArtifacts: LinkedArtifact[] = [
      {
        artifact: newMockArtifact(50, false, true, 'artifact-50'),
        event: newMockEvent(50, 'artifact-50'),
      },
      ...initialLinkedArtifacts,
    ];

    function RocCurveSelectionHarness() {
      const [linkedArtifacts, setLinkedArtifacts] = useState(initialLinkedArtifacts);
      const [selection, setSelection] = useState(TEST_ONLY.createInitialRocCurveSelectionState);

      useEffect(() => {
        setSelection((currentSelection) =>
          TEST_ONLY.reconcileRocCurveSelectionState(
            currentSelection,
            linkedArtifacts,
            new Set(
              linkedArtifacts.map(
                (linkedArtifact) =>
                  `${linkedArtifact.event.getExecutionId()}-${linkedArtifact.artifact.getId()}`,
              ),
            ),
          ),
        );
      }, [linkedArtifacts]);

      return (
        <>
          <div data-testid='roc-selected-ids'>{selection.selectedIds.join(',')}</div>
          <button onClick={() => setLinkedArtifacts(refreshedLinkedArtifacts)}>refresh ROC</button>
        </>
      );
    }

    render(
      <CommonTestWrapper>
        <RocCurveSelectionHarness />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('roc-selected-ids')).toHaveTextContent('100-100,200-200,300-300');
    });

    fireEvent.click(screen.getByRole('button', { name: 'refresh ROC' }));

    await waitFor(() => {
      expect(screen.getByTestId('roc-selected-ids')).toHaveTextContent('100-100,200-200,300-300');
    });
  });

  it('reconciles invalid two-panel artifact selections against the available run artifacts', () => {
    const selectedArtifactsMap = {
      [MetricsType.CONFUSION_MATRIX]: [
        {
          selectedItem: {
            itemName: `test run ${MOCK_RUN_2_ID}`,
            subItemName: 'artifactName',
          },
        },
        {
          selectedItem: {
            itemName: 'missing run',
            subItemName: 'staleArtifact',
          },
        },
      ],
      [MetricsType.HTML]: [
        {
          selectedItem: {
            itemName: `test run ${MOCK_RUN_1_ID}`,
            subItemName: 'firstHtmlArtifact',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
      [MetricsType.MARKDOWN]: [
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
    };

    const reconciledArtifactsMap = TEST_ONLY.reconcileSelectedArtifactsMap(selectedArtifactsMap, {
      scalarMetricsTableData: undefined,
      confusionMatrixRunArtifacts: [
        { run: newMockRun(MOCK_RUN_2_ID), executionArtifacts: [] as any },
      ],
      htmlRunArtifacts: [{ run: newMockRun(MOCK_RUN_1_ID), executionArtifacts: [] as any }],
      markdownRunArtifacts: [],
      rocCurveRunArtifacts: [],
    });

    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][0].selectedItem).toEqual({
      itemName: `test run ${MOCK_RUN_2_ID}`,
      subItemName: 'artifactName',
    });
    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][1].selectedItem).toEqual({
      itemName: '',
      subItemName: '',
    });
    expect(reconciledArtifactsMap[MetricsType.HTML][0].selectedItem).toEqual({
      itemName: `test run ${MOCK_RUN_1_ID}`,
      subItemName: 'firstHtmlArtifact',
    });
    expect(reconciledArtifactsMap[MetricsType.MARKDOWN][0].selectedItem).toEqual({
      itemName: '',
      subItemName: '',
    });
    expect(reconciledArtifactsMap[MetricsType.MARKDOWN][1].selectedItem).toEqual({
      itemName: '',
      subItemName: '',
    });
  });

  it('keeps both two-panel selections when they reference the same run', () => {
    const selectedArtifactsMap = {
      [MetricsType.CONFUSION_MATRIX]: [
        {
          selectedItem: {
            itemName: `test run ${MOCK_RUN_2_ID}`,
            subItemName: 'firstArtifact',
          },
        },
        {
          selectedItem: {
            itemName: `test run ${MOCK_RUN_2_ID}`,
            subItemName: 'secondArtifact',
          },
        },
      ],
      [MetricsType.HTML]: [
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
      [MetricsType.MARKDOWN]: [
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
    };

    const reconciledArtifactsMap = TEST_ONLY.reconcileSelectedArtifactsMap(selectedArtifactsMap, {
      scalarMetricsTableData: undefined,
      confusionMatrixRunArtifacts: [
        { run: newMockRun(MOCK_RUN_2_ID), executionArtifacts: [] as any },
      ],
      htmlRunArtifacts: [],
      markdownRunArtifacts: [],
      rocCurveRunArtifacts: [],
    });

    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][0].selectedItem).toEqual({
      itemName: `test run ${MOCK_RUN_2_ID}`,
      subItemName: 'firstArtifact',
    });
    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][1].selectedItem).toEqual({
      itemName: `test run ${MOCK_RUN_2_ID}`,
      subItemName: 'secondArtifact',
    });
  });

  it('keeps runId-bearing selection when runId matches a valid run', () => {
    const selectedArtifactsMap = {
      [MetricsType.CONFUSION_MATRIX]: [
        {
          selectedItem: {
            runId: MOCK_RUN_2_ID,
            itemName: `test run ${MOCK_RUN_2_ID}`,
            subItemName: 'artifactName',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
      [MetricsType.HTML]: [
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
      [MetricsType.MARKDOWN]: [
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
    };

    const reconciledArtifactsMap = TEST_ONLY.reconcileSelectedArtifactsMap(selectedArtifactsMap, {
      scalarMetricsTableData: undefined,
      confusionMatrixRunArtifacts: [
        { run: newMockRun(MOCK_RUN_2_ID), executionArtifacts: [] as any },
      ],
      htmlRunArtifacts: [],
      markdownRunArtifacts: [],
      rocCurveRunArtifacts: [],
    });

    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][0].selectedItem).toEqual({
      runId: MOCK_RUN_2_ID,
      itemName: `test run ${MOCK_RUN_2_ID}`,
      subItemName: 'artifactName',
    });
    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][1].selectedItem).toEqual({
      itemName: '',
      subItemName: '',
    });
  });

  it('falls back to display_name and clears runId when runId is stale but display_name still matches', () => {
    const selectedArtifactsMap = {
      [MetricsType.CONFUSION_MATRIX]: [
        {
          selectedItem: {
            runId: 'stale-run-id',
            itemName: `test run ${MOCK_RUN_2_ID}`,
            subItemName: 'artifactName',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
      [MetricsType.HTML]: [
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
      [MetricsType.MARKDOWN]: [
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
    };

    const reconciledArtifactsMap = TEST_ONLY.reconcileSelectedArtifactsMap(selectedArtifactsMap, {
      scalarMetricsTableData: undefined,
      confusionMatrixRunArtifacts: [
        { run: newMockRun(MOCK_RUN_2_ID), executionArtifacts: [] as any },
      ],
      htmlRunArtifacts: [],
      markdownRunArtifacts: [],
      rocCurveRunArtifacts: [],
    });

    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][0].selectedItem).toEqual({
      runId: undefined,
      itemName: `test run ${MOCK_RUN_2_ID}`,
      subItemName: 'artifactName',
    });
    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][1].selectedItem).toEqual({
      itemName: '',
      subItemName: '',
    });
  });

  it('clears selection when both runId and display_name are invalid', () => {
    const selectedArtifactsMap = {
      [MetricsType.CONFUSION_MATRIX]: [
        {
          selectedItem: {
            runId: 'stale-run-id',
            itemName: 'missing run',
            subItemName: 'staleArtifact',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
      [MetricsType.HTML]: [
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
      [MetricsType.MARKDOWN]: [
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
        {
          selectedItem: {
            itemName: '',
            subItemName: '',
          },
        },
      ],
    };

    const reconciledArtifactsMap = TEST_ONLY.reconcileSelectedArtifactsMap(selectedArtifactsMap, {
      scalarMetricsTableData: undefined,
      confusionMatrixRunArtifacts: [
        { run: newMockRun(MOCK_RUN_2_ID), executionArtifacts: [] as any },
      ],
      htmlRunArtifacts: [],
      markdownRunArtifacts: [],
      rocCurveRunArtifacts: [],
    });

    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][0].selectedItem).toEqual({
      itemName: '',
      subItemName: '',
    });
    expect(reconciledArtifactsMap[MetricsType.CONFUSION_MATRIX][1].selectedItem).toEqual({
      itemName: '',
      subItemName: '',
    });
  });

  it('getRun is called with query param IDs', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find((r) => r.run_id === id));

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
    getRunSpy.mockImplementation((id: string) =>
      Promise.resolve(runs.find((r) => r.run_id === id)),
    );

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 2),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find((c) => c.getName() === runID)),
    );

    const executions = [[newMockExecution(1)], [newMockExecution(2)], [newMockExecution(3)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find((e) => e[0].getId() === context.getId())),
    );

    const artifacts = [newMockArtifact(1), newMockArtifact(2), newMockArtifact(3)];
    const getArtifactsSpy = vi.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(2), newMockEvent(3)];
    const getEventsSpy = vi.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

    const getArtifactTypesSpy = vi.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockResolvedValue([]);

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    // Wait for runs to render (indicates all queries resolved) before banner clears
    await waitForRunCheckboxes(3);
    await waitFor(
      () => {
        expect(updateBannerSpy).toHaveBeenCalledWith({});
      },
      { timeout: 10000 },
    );
  });

  it('Log warning when artifact with specified ID is not found', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) =>
      Promise.resolve(runs.find((r) => r.run_id === id)),
    );

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 2),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find((c) => c.getName() === runID)),
    );

    const executions = [[newMockExecution(1)], [newMockExecution(2)], [newMockExecution(3)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find((e) => e[0].getId() === context.getId())),
    );

    const artifacts = [newMockArtifact(1), newMockArtifact(3)];
    const getArtifactsSpy = vi.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(2), newMockEvent(3)];
    const getEventsSpy = vi.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockResolvedValue(events);

    const getArtifactTypesSpy = vi.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockResolvedValue([]);

    const warnSpy = vi.spyOn(Utils.logger, 'warn').mockImplementation(() => undefined);

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();
    await waitForRunCheckboxes(3);

    await waitFor(
      () => {
        expect(warnSpy).toHaveBeenLastCalledWith(
          'The artifact with the following ID was not found: 2',
        );
      },
      { timeout: 10000 },
    );
  });

  it('Show page error on page when getRun request fails', async () => {
    const expectError = expectErrors();
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((_) => {
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
    getRunSpy.mockImplementation((id: string) =>
      Promise.resolve(runs.find((r) => r.run_id === id)),
    );
    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockRejectedValue(new Error('Not connected to MLMD'));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(
      () => {
        expect(updateBannerSpy).toHaveBeenLastCalledWith({
          additionalInfo: 'Not connected to MLMD',
          message: 'Cannot get MLMD objects from Metadata store.',
          mode: 'error',
        });
      },
      { timeout: 10000 },
    );
    expectError();
  });

  it('Failed getArtifactTypes request creates error banner', async () => {
    const expectError = expectErrors();
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) =>
      Promise.resolve(runs.find((r) => r.run_id === id)),
    );

    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockResolvedValue(new Context());
    vi.spyOn(mlmdUtils, 'getExecutionsFromContext').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getArtifactsFromContext').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getEventsByExecutions').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getArtifactTypes').mockRejectedValue(new Error('Not connected to MLMD'));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(
      () => {
        expect(updateBannerSpy).toHaveBeenLastCalledWith({
          additionalInfo: 'Not connected to MLMD',
          message: 'Cannot get Artifact Types for MLMD.',
          mode: 'error',
        });
      },
      { timeout: 10000 },
    );
    expectError();
  });

  it('Allows individual sections to be collapsed and expanded', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find((r) => r.run_id === id));

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
    getRunSpy.mockImplementation((id: string) => runs.find((r) => r.run_id === id));

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

  it('updates the selected run count when a single run is toggled', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find((r) => r.run_id === id));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitForRunCheckboxes(3);
    fireEvent.click(getRunRow(MOCK_RUN_2_ID));
    await TestUtils.flushPromises();

    await waitForRunCheckboxes(2);
    expect(getHeaderCheckbox()).not.toBeChecked();
  });

  it('preserves a manual run selection when the toolbar refresh returns the same run ids', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => ({ ...runs.find((r) => r.run_id === id)! }));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(updateToolbarSpy).toHaveBeenCalled();
    });
    await waitForRunCheckboxes(3);

    fireEvent.click(getRunRow(MOCK_RUN_2_ID));
    await TestUtils.flushPromises();
    await waitForRunCheckboxes(2);
    expect(getRunRow(MOCK_RUN_2_ID)).toHaveAttribute('aria-checked', 'false');

    const refreshAction = updateToolbarSpy.mock.lastCall?.[0].actions[ButtonKeys.REFRESH]
      .action as () => Promise<void>;
    await act(async () => {
      await refreshAction();
    });
    await TestUtils.flushPromises();

    await waitForRunCheckboxes(2);
    expect(getRunRow(MOCK_RUN_1_ID)).toHaveAttribute('aria-checked', 'true');
    expect(getRunRow(MOCK_RUN_2_ID)).toHaveAttribute('aria-checked', 'false');
    expect(getRunRow(MOCK_RUN_3_ID)).toHaveAttribute('aria-checked', 'true');
    expect(getHeaderCheckbox()).not.toBeChecked();
  });

  it('drops stale manual selections when the toolbar refresh returns different fetched run ids', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    const refreshedRunIds = new Set<string>();
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => {
      if (refreshedRunIds.has(id)) {
        return {
          ...newMockRun(`replacement-${id}`),
          display_name: `test run ${id}`,
        };
      }
      return { ...runs.find((r) => r.run_id === id)! };
    });

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(updateToolbarSpy).toHaveBeenCalled();
    });
    await waitForRunCheckboxes(3);

    fireEvent.click(getRunRow(MOCK_RUN_2_ID));
    await TestUtils.flushPromises();
    await waitForRunCheckboxes(2);

    refreshedRunIds.add(MOCK_RUN_3_ID);
    const refreshAction = updateToolbarSpy.mock.lastCall?.[0].actions[ButtonKeys.REFRESH]
      .action as () => Promise<void>;
    await act(async () => {
      await refreshAction();
    });
    await TestUtils.flushPromises();

    await waitForRunCheckboxes(1);
    expect(getRunRow(MOCK_RUN_1_ID)).toHaveAttribute('aria-checked', 'true');
    expect(getRunRow(MOCK_RUN_2_ID)).toHaveAttribute('aria-checked', 'false');
    expect(getRunRow(MOCK_RUN_3_ID)).toHaveAttribute('aria-checked', 'false');
    expect(getHeaderCheckbox()).not.toBeChecked();
  });

  it('reinitializes selection to the new runlist after a route change', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => ({ ...runs.find((r) => r.run_id === id)! }));

    const renderResult = render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitForRunCheckboxes(3);
    fireEvent.click(getRunRow(MOCK_RUN_2_ID));
    await TestUtils.flushPromises();
    await waitForRunCheckboxes(2);

    const nextProps = generateProps();
    nextProps.location.search = `?${QUERY_PARAMS.runlist}=${MOCK_RUN_2_ID},${MOCK_RUN_3_ID}`;
    renderResult.rerender(
      <CommonTestWrapper>
        <CompareV2 {...nextProps} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitForRunCheckboxes(2);
    expect(getRunRow(MOCK_RUN_2_ID)).toHaveAttribute('aria-checked', 'true');
    expect(getRunRow(MOCK_RUN_3_ID)).toHaveAttribute('aria-checked', 'true');
  });

  it('refreshes the selected HTML artifact after a route change with the same visible labels', async () => {
    const ORIGINAL_ROUTE_RUN_ID = 'html-route-original';
    const UPDATED_ROUTE_RUN_ID = 'html-route-updated';
    const SHARED_RUN_NAME = 'shared run';
    const SHARED_EXECUTION_NAME = 'shared execution';
    const SHARED_ARTIFACT_NAME = 'shared artifact';

    runs = [
      {
        ...newMockRun(ORIGINAL_ROUTE_RUN_ID),
        display_name: SHARED_RUN_NAME,
      },
      {
        ...newMockRun(UPDATED_ROUTE_RUN_ID),
        display_name: SHARED_RUN_NAME,
      },
    ];

    const originalContext = newMockContext(ORIGINAL_ROUTE_RUN_ID, 200);
    const updatedContext = newMockContext(UPDATED_ROUTE_RUN_ID, 300);
    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockImplementation((runID: string) =>
      Promise.resolve(
        [originalContext, updatedContext].find((context) => context.getName() === runID),
      ),
    );

    const originalExecution = newMockExecution(200, SHARED_EXECUTION_NAME);
    const updatedExecution = newMockExecution(300, SHARED_EXECUTION_NAME);
    vi.spyOn(mlmdUtils, 'getExecutionsFromContext').mockImplementation((context: Context) =>
      Promise.resolve(
        context.getId() === originalContext.getId() ? [originalExecution] : [updatedExecution],
      ),
    );

    const originalArtifact = newMockArtifact(200, false, false, SHARED_ARTIFACT_NAME);
    const updatedArtifact = newMockArtifact(300, false, false, SHARED_ARTIFACT_NAME);
    vi.spyOn(mlmdUtils, 'getArtifactsFromContext').mockImplementation((context: Context) =>
      Promise.resolve(
        context.getId() === originalContext.getId() ? [originalArtifact] : [updatedArtifact],
      ),
    );

    vi.spyOn(mlmdUtils, 'getEventsByExecutions').mockImplementation((executions: Execution[]) =>
      Promise.resolve(
        executions[0]?.getId() === originalExecution.getId()
          ? [newMockEvent(200, SHARED_ARTIFACT_NAME)]
          : [newMockEvent(300, SHARED_ARTIFACT_NAME)],
      ),
    );

    vi.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'filterLinkedArtifactsByType').mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.HTML' ? linkedArtifacts : [],
    );

    const getHtmlViewerConfigSpy = vi.spyOn(metricsVisualizations, 'getHtmlViewerConfig');
    getHtmlViewerConfigSpy.mockResolvedValue([]);

    const initialProps = generateProps();
    initialProps.location.search = `?${QUERY_PARAMS.runlist}=${ORIGINAL_ROUTE_RUN_ID}`;

    const renderResult = render(
      <CommonTestWrapper>
        <CompareV2 {...initialProps} />
      </CommonTestWrapper>,
    );
    await waitForRunCheckboxes(1);

    fireEvent.click(screen.getByRole('button', { name: 'HTML' }));
    fireEvent.click(await screen.findByText('Choose a first HTML artifact', { timeout: 10000 }));
    fireEvent.mouseEnter(screen.getAllByText(SHARED_RUN_NAME)[1]);
    fireEvent.click(screen.getByLabelText(`${SHARED_EXECUTION_NAME} > ${SHARED_ARTIFACT_NAME}`));

    await waitFor(() => {
      const lastCall = getHtmlViewerConfigSpy.mock.lastCall;
      expect(lastCall?.[0]?.[0]?.artifact.getId()).toBe(originalArtifact.getId());
      expect(lastCall?.[1]).toBeUndefined();
    });

    const nextProps = generateProps();
    nextProps.location.search = `?${QUERY_PARAMS.runlist}=${UPDATED_ROUTE_RUN_ID}`;
    renderResult.rerender(
      <CommonTestWrapper>
        <CompareV2 {...nextProps} />
      </CommonTestWrapper>,
    );

    await TestUtils.flushPromises();
    await waitFor(() => {
      const lastCall = getHtmlViewerConfigSpy.mock.lastCall;
      expect(lastCall?.[0]?.[0]?.artifact.getId()).toBe(updatedArtifact.getId());
      expect(lastCall?.[1]).toBeUndefined();
    });
  });

  it('Parameters and Scalar metrics tab initially enabled with loading then error, and switch tabs', async () => {
    const expectError = expectErrors();
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find((r) => r.run_id === id));
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
    getRunSpy.mockImplementation((id: string) =>
      Promise.resolve(runs.find((r) => r.run_id === id)),
    );

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
    await waitForRunCheckboxes(3);

    await waitFor(
      () => {
        expect(getBodyText()).toContain(
          'There are no Scalar Metrics artifacts available on the selected runs.',
        );
      },
      { timeout: 10000 },
    );

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
    await waitFor(
      () => {
        expect(getBodyText()).toContain(
          'There are no ROC Curve artifacts available on the selected runs.',
        );
      },
      { timeout: 10000 },
    );
  });

  it('Confusion matrix shown on select, stays after tab change or section collapse', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) =>
      Promise.resolve(runs.find((r) => r.run_id === id)),
    );

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find((c) => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find((e) => e[0].getId() === context.getId())),
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
    await waitForRunCheckboxes(3);

    await waitFor(() => expect(filterLinkedArtifactsByTypeSpy).toHaveBeenCalled());

    expect(screen.queryByText(/Confusion matrix: artifactName/)).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Confusion Matrix' }));
    fireEvent.click(
      await screen.findByText('Choose a first Confusion Matrix artifact', { timeout: 10000 }),
    );

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
    getRunSpy.mockImplementation((id: string) =>
      Promise.resolve(runs.find((r) => r.run_id === id)),
    );

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = vi.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find((c) => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    const getExecutionsSpy = vi.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find((e) => e[0].getId() === context.getId())),
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
    await waitForRunCheckboxes(3);

    expect(screen.queryByText(/Confusion matrix: artifactName/)).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Confusion Matrix' }));
    fireEvent.click(
      await screen.findByText('Choose a first Confusion Matrix artifact', { timeout: 10000 }),
    );

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
    getRunSpy.mockImplementation((id: string) =>
      Promise.resolve(runs.find((r) => r.run_id === id)),
    );

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockImplementation((runID: string) =>
      Promise.resolve(contexts.find((c) => c.getName() === runID)),
    );

    // Use same pattern as Confusion Matrix test: shared artifacts/events, context-specific executions
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    vi.spyOn(mlmdUtils, 'getExecutionsFromContext').mockImplementation((context: Context) =>
      Promise.resolve(executions.find((e) => e[0].getId() === context.getId()) ?? []),
    );

    const artifact200 = newMockArtifact(200, false, true, 'artifactName');
    const artifacts = [newMockArtifact(1), artifact200, newMockArtifact(3)];
    vi.spyOn(mlmdUtils, 'getArtifactsFromContext').mockResolvedValue(artifacts);

    const events = [newMockEvent(1), newMockEvent(200, 'artifactName'), newMockEvent(3)];
    vi.spyOn(mlmdUtils, 'getEventsByExecutions').mockResolvedValue(events);

    vi.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([]);

    vi.spyOn(mlmdUtils, 'filterLinkedArtifactsByType').mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.ClassificationMetrics' ? linkedArtifacts : [],
    );

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitForRunCheckboxes(3);

    await waitFor(
      () => {
        expect(getBodyText()).toContain('Scalar Metrics');
      },
      { timeout: 10000 },
    );

    fireEvent.click(screen.getByRole('button', { name: 'ROC Curve' }));

    // ROC Curve tab shows "ROC Curve: artifactName" when run 2 (with ROC artifact) is selected
    await waitFor(
      () => {
        expect(getBodyText()).toMatch(/ROC Curve:.*artifactName/);
      },
      { timeout: 10000 },
    );

    // De-select run 1; ROC curve (from run 2) should still be visible
    let runCheckboxes = await waitForRunCheckboxes(3);
    fireEvent.click(runCheckboxes[0]);
    await waitFor(
      () => {
        expect(getBodyText()).toMatch(/ROC Curve:.*artifactName/);
      },
      { timeout: 5000 },
    );
    // De-select run 2 (has ROC artifact); ROC curve should disappear
    runCheckboxes = await waitForRunCheckboxes(2);
    fireEvent.click(runCheckboxes[0]);
    await waitFor(
      () => {
        expect(getBodyText()).not.toMatch(/ROC Curve:.*artifactName/);
      },
      { timeout: 5000 },
    );
  }, 20000);

  it('Multiple ROC Curves shown on select', async () => {
    const getRunSpy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) =>
      Promise.resolve(runs.find((r) => r.run_id === id)),
    );

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 300),
    ];
    vi.spyOn(mlmdUtils, 'getKfpV2RunContext').mockImplementation((runID: string) =>
      Promise.resolve(contexts.find((c) => c.getName() === runID)),
    );

    // Use same pattern as Confusion Matrix test: shared artifacts/events, context-specific executions
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(300)]];
    vi.spyOn(mlmdUtils, 'getExecutionsFromContext').mockImplementation((context: Context) =>
      Promise.resolve(executions.find((e) => e[0].getId() === context.getId()) ?? []),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, false, true, 'firstArtifactName'),
      newMockArtifact(300, false, true, 'secondArtifactName'),
    ];
    vi.spyOn(mlmdUtils, 'getArtifactsFromContext').mockResolvedValue(artifacts);

    const events = [
      newMockEvent(1),
      newMockEvent(200, 'firstArtifactName'),
      newMockEvent(300, 'secondArtifactName'),
    ];
    vi.spyOn(mlmdUtils, 'getEventsByExecutions').mockResolvedValue(events);

    vi.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([]);

    vi.spyOn(mlmdUtils, 'filterLinkedArtifactsByType').mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.ClassificationMetrics' ? linkedArtifacts : [],
    );

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitForRunCheckboxes(3);

    await waitFor(
      () => {
        expect(getBodyText()).toContain('Scalar Metrics');
      },
      { timeout: 10000 },
    );

    fireEvent.click(screen.getByRole('button', { name: 'ROC Curve' }));
    // Wait for ROC curve data to load - either "multiple artifacts" (2+ selected) or artifact names
    await waitFor(
      () => {
        const text = getBodyText();
        expect(
          text.match(/ROC Curve:.*multiple artifacts/) ||
            (text.includes('firstArtifactName') && text.includes('secondArtifactName')),
        ).toBeTruthy();
      },
      { timeout: 10000 },
    );
    await screen.findByLabelText('Filter artifacts');
  });
});
