/*
 * Copyright 2018 The Kubeflow Authors
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

import * as React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import EnhancedCompareV1, { TEST_ONLY, TaggedViewerConfig } from './CompareV1';
import TestUtils from 'src/TestUtils';
import * as Utils from 'src/lib/Utils';
import { logger } from 'src/lib/Utils';
import { Apis } from 'src/lib/Apis';
import { PageProps } from './Page';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';
import { ApiRunDetail } from 'src/apis/run';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { PlotType } from 'src/components/viewers/Viewer';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import { Workflow } from 'src/third_party/mlmd/argo_template';
import { ButtonKeys } from 'src/lib/Buttons';
import { MemoryRouter, Router } from 'react-router-dom';
import { createMemoryHistory } from 'history';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';
import { stableMuiSnapshotFragment } from 'src/testUtils/muiSnapshot';

const CompareV1 = TEST_ONLY.CompareV1;

class TestCompare extends CompareV1 {
  public _selectionChanged(selectedIds: string[]): void {
    return super._selectionChanged(selectedIds);
  }
}

describe('CompareV1', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let compareRef: React.RefObject<TestCompare> | null = null;

  let updateToolbarSpy: ReturnType<typeof vi.fn>;
  let updateBannerSpy: ReturnType<typeof vi.fn>;
  let updateDialogSpy: ReturnType<typeof vi.fn>;
  let updateSnackbarSpy: ReturnType<typeof vi.fn>;
  let historyPushSpy: ReturnType<typeof vi.fn>;

  const getRunSpy = vi.spyOn(Apis.runServiceApi, 'getRun');
  const outputArtifactLoaderSpy = vi.spyOn(OutputArtifactLoader, 'load');
  const getRunV2Spy = vi.spyOn(Apis.runServiceApiV2, 'getRun');
  const listExperimentsSpy = vi.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
  const getPipelineVersionSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion');
  const formatDateStringSpy = vi.spyOn(Utils, 'formatDateString');

  const MOCK_RUN_1_ID = 'mock-run-1-id';
  const MOCK_RUN_2_ID = 'mock-run-2-id';
  const MOCK_RUN_3_ID = 'mock-run-3-id';

  let runs: ApiRunDetail[] = [];

  function generateProps(): PageProps {
    const location = {
      pathname: RoutePage.COMPARE,
      search: `?${QUERY_PARAMS.runlist}=${MOCK_RUN_1_ID},${MOCK_RUN_2_ID},${MOCK_RUN_3_ID}`,
    } as any;
    return TestUtils.generatePageProps(
      CompareV1,
      location,
      {} as any,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  function getInstance(): TestCompare {
    if (!compareRef?.current) {
      throw new Error('CompareV1 instance not available');
    }
    return compareRef.current;
  }

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

  function newMockRunV2(id: string) {
    return {
      run_id: id,
      display_name: 'test run ' + id,
      pipeline_spec: { pipeline_manifest: '' },
      created_at: '2020-01-01T00:00:00Z',
      state: V2beta1RuntimeState.SUCCEEDED,
    } as any;
  }

  async function renderCompare(props?: PageProps): Promise<PageProps> {
    compareRef = React.createRef<TestCompare>();
    const propsToUse = props || generateProps();
    renderResult = render(
      <MemoryRouter>
        <TestCompare ref={compareRef} {...propsToUse} />
      </MemoryRouter>,
    );
    await TestUtils.flushPromises();
    return propsToUse;
  }

  async function waitForRunRows(expectedCount: number): Promise<void> {
    await waitFor(() => {
      expect(screen.getAllByTestId('table-row')).toHaveLength(expectedCount);
    });
  }

  async function waitForViewersMap(expectedSize: number): Promise<void> {
    await waitFor(() => {
      expect(getInstance().state.viewersMap.size).toBe(expectedSize);
    });
  }

  function getCollapseButton(sectionName: string): HTMLElement {
    const label = screen.getByText(sectionName);
    const button = label.closest('button');
    if (!button) {
      throw new Error(`Collapse button not found for section ${sectionName}`);
    }
    return button;
  }

  async function setUpViewersAndRender(): Promise<void> {
    outputArtifactLoaderSpy.mockImplementation(() => [
      { type: PlotType.TENSORBOARD, url: 'gs://path' },
      { data: [[]], labels: ['col1, col2'], type: PlotType.TABLE },
    ]);

    const workflow = {
      status: {
        nodes: {
          node1: {
            outputs: {
              artifacts: [
                {
                  name: 'mlpipeline-ui-metadata',
                  s3: { s3Bucket: { bucket: 'test bucket' }, key: 'test key' },
                },
              ],
            },
          },
        },
      },
    };
    const run1 = newMockRun('run-with-workflow-1');
    run1.pipeline_runtime!.workflow_manifest = JSON.stringify(workflow);
    const run2 = newMockRun('run-with-workflow-2');
    run2.pipeline_runtime!.workflow_manifest = JSON.stringify(workflow);
    runs = [run1, run2];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=run-with-workflow-1,run-with-workflow-2`;
    await renderCompare(props);
    await waitForViewersMap(2);
  }

  beforeEach(() => {
    updateToolbarSpy = vi.fn();
    updateBannerSpy = vi.fn();
    updateDialogSpy = vi.fn();
    updateSnackbarSpy = vi.fn();
    historyPushSpy = vi.fn();

    getRunSpy.mockReset();
    outputArtifactLoaderSpy.mockReset();
    getRunV2Spy.mockReset();
    listExperimentsSpy.mockReset();
    getPipelineVersionSpy.mockReset();
    formatDateStringSpy.mockReset();

    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );
    outputArtifactLoaderSpy.mockResolvedValue([]);
    getRunV2Spy.mockImplementation((id: string) => Promise.resolve(newMockRunV2(id)));
    listExperimentsSpy.mockResolvedValue({ experiments: [] } as any);
    getPipelineVersionSpy.mockResolvedValue({
      display_name: 'pipeline version',
      pipeline_id: 'pipeline-id',
      pipeline_version_id: 'pipeline-version-id',
    } as any);
    formatDateStringSpy.mockImplementation(() => '1/2/2019, 12:34:56 PM');
  });

  afterEach(() => {
    renderResult?.unmount();
    renderResult = null;
    compareRef = null;
  });

  it('clears banner upon initial load', async () => {
    await renderCompare();
    await waitFor(() => expect(updateBannerSpy).toHaveBeenCalled());
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('renders a page with no runs', async () => {
    const props = generateProps();
    props.location.search = '';
    await renderCompare(props);

    await waitFor(() => expect(updateBannerSpy).toHaveBeenCalled());
    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it('renders a page with multiple runs', async () => {
    await renderCompare();
    await waitFor(() => expect(getRunSpy).toHaveBeenCalledTimes(3));
    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it('fetches a run for each ID in query params', async () => {
    runs = [newMockRun('run-1'), newMockRun('run-2'), newMockRun('run-3')];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=run-1,run-2,run-3`;

    await renderCompare(props);
    await waitFor(() => expect(getRunSpy).toHaveBeenCalledTimes(3));

    expect(getRunSpy).toHaveBeenCalledWith('run-1');
    expect(getRunSpy).toHaveBeenCalledWith('run-2');
    expect(getRunSpy).toHaveBeenCalledWith('run-3');
  });

  it('shows an error banner if fetching any run fails', async () => {
    TestUtils.makeErrorResponseOnce(getRunSpy as any, 'test error');
    const loggerErrorSpy = vi.spyOn(logger, 'error').mockImplementation(() => undefined);

    await renderCompare();
    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'test error',
          message: 'Error: failed loading 1 runs. Click Details for more information.',
          mode: 'error',
        }),
      ),
    );
    expect(loggerErrorSpy).toHaveBeenCalled();
    loggerErrorSpy.mockRestore();
  });

  it('shows an info banner if all runs are v2', async () => {
    runs = [
      newMockRun(MOCK_RUN_1_ID, true),
      newMockRun(MOCK_RUN_2_ID, true),
      newMockRun(MOCK_RUN_3_ID, true),
    ];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );

    await renderCompare();
    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith({
        additionalInfo:
          'The selected runs are all V2, but the V2_ALPHA feature flag is disabled.' +
          ' The V1 page will not show any useful information for these runs.',
        message:
          'Info: enable the V2_ALPHA feature flag in order to view the updated Run Comparison page.',
        mode: 'info',
      }),
    );
  });

  it('shows an error banner indicating the number of getRun calls that failed', async () => {
    const loggerErrorSpy = vi.spyOn(logger, 'error').mockImplementation(() => undefined);
    getRunSpy.mockImplementation(() => {
      throw {
        text: () => Promise.resolve('test error'),
      };
    });

    await renderCompare();
    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'test error',
          message: `Error: failed loading ${runs.length} runs. Click Details for more information.`,
          mode: 'error',
        }),
      ),
    );
    expect(loggerErrorSpy).toHaveBeenCalled();
    loggerErrorSpy.mockRestore();
  });

  it('clears the error banner on refresh', async () => {
    const loggerErrorSpy = vi.spyOn(logger, 'error').mockImplementation(() => undefined);
    TestUtils.makeErrorResponseOnce(getRunSpy as any, 'test error');

    await renderCompare();
    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({ mode: 'error' })),
    );

    await act(async () => {
      await getInstance().refresh();
    });

    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
    loggerErrorSpy.mockRestore();
  });

  it("displays run's parameters if the run has any", async () => {
    const workflow = {
      spec: {
        arguments: {
          parameters: [
            { name: 'param1', value: 'value1' },
            { name: 'param2', value: 'value2' },
          ],
        },
      },
    } as Workflow;

    const run = newMockRun('run-with-parameters');
    run.pipeline_runtime!.workflow_manifest = JSON.stringify(workflow);
    runs = [run];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=run-with-parameters`;

    await renderCompare(props);
    await waitFor(() =>
      expect(getInstance().state.paramsCompareProps).toEqual({
        rows: [['value1'], ['value2']],
        xLabels: ['test run run-with-parameters'],
        yLabels: ['param1', 'param2'],
      }),
    );
    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it('displays parameters from multiple runs', async () => {
    const run1Workflow = {
      spec: {
        arguments: {
          parameters: [
            { name: 'r1-unique-param', value: 'r1-unique-val1' },
            { name: 'shared-param', value: 'r1-shared-val2' },
          ],
        },
      },
    } as Workflow;
    const run2Workflow = {
      spec: {
        arguments: {
          parameters: [
            { name: 'r2-unique-param1', value: 'r2-unique-val1' },
            { name: 'shared-param', value: 'r2-shared-val2' },
          ],
        },
      },
    } as Workflow;

    const run1 = newMockRun('run1');
    run1.pipeline_runtime!.workflow_manifest = JSON.stringify(run1Workflow);
    const run2 = newMockRun('run2');
    run2.pipeline_runtime!.workflow_manifest = JSON.stringify(run2Workflow);
    runs = [run1, run2];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=run1,run2`;

    await renderCompare(props);
    await waitFor(() => expect(getRunSpy).toHaveBeenCalledTimes(2));
    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it("displays a run's metrics if the run has any", async () => {
    const run = newMockRun('run-with-metrics');
    run.run!.metrics = [
      { name: 'some-metric', number_value: 0.33 },
      { name: 'another-metric', number_value: 0.554 },
    ];
    runs = [run];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=run-with-metrics`;

    await renderCompare(props);
    await waitFor(() =>
      expect(getInstance().state.metricsCompareProps).toEqual({
        rows: [['0.330'], ['0.554']],
        xLabels: ['test run run-with-metrics'],
        yLabels: ['some-metric', 'another-metric'],
      }),
    );
    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it('displays metrics from multiple runs', async () => {
    const run1 = newMockRun('run1');
    run1.run!.metrics = [
      { name: 'some-metric', number_value: 0.33 },
      { name: 'another-metric', number_value: 0.554 },
    ];
    const run2 = newMockRun('run2');
    run2.run!.metrics = [{ name: 'some-metric', number_value: 0.67 }];
    runs = [run1, run2];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=run1,run2`;

    await renderCompare(props);
    await waitFor(() => expect(getRunSpy).toHaveBeenCalledTimes(2));
    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it('creates a map of viewers', async () => {
    outputArtifactLoaderSpy.mockImplementationOnce(() => [
      { type: PlotType.TENSORBOARD, url: 'gs://path' },
      { data: [['test']], labels: ['col1, col2'], type: PlotType.TABLE },
    ]);

    const workflow = {
      status: {
        nodes: {
          node1: {
            outputs: {
              artifacts: [
                {
                  name: 'mlpipeline-ui-metadata',
                  s3: { s3Bucket: { bucket: 'test bucket' }, key: 'test key' },
                },
              ],
            },
          },
        },
      },
    };
    const run = newMockRun('run-with-workflow');
    run.pipeline_runtime!.workflow_manifest = JSON.stringify(workflow);
    runs = [run];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=run-with-workflow`;

    await renderCompare(props);
    await waitForViewersMap(2);

    const expectedViewerMap = new Map([
      [
        PlotType.TABLE,
        [
          {
            config: { data: [['test']], labels: ['col1, col2'], type: PlotType.TABLE },
            runId: run.run!.id,
            runName: run.run!.name,
          } as TaggedViewerConfig,
        ],
      ],
      [
        PlotType.TENSORBOARD,
        [
          {
            config: { type: PlotType.TENSORBOARD, url: 'gs://path' },
            runId: run.run!.id,
            runName: run.run!.name,
          } as TaggedViewerConfig,
        ],
      ],
    ]);

    expect(getInstance().state.viewersMap as Map<PlotType, TaggedViewerConfig>).toEqual(
      expectedViewerMap,
    );
    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it('collapses all sections', async () => {
    await setUpViewersAndRender();
    const collapseBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.COLLAPSE];

    expect(getInstance().state.collapseSections).toEqual({});

    await act(async () => {
      collapseBtn!.action();
    });

    expect(getInstance().state.collapseSections).toEqual({
      [METRICS_SECTION_NAME]: true,
      [PARAMS_SECTION_NAME]: true,
      [OVERVIEW_SECTION_NAME]: true,
      Table: true,
      Tensorboard: true,
    });

    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it('expands all sections if they were collapsed', async () => {
    await setUpViewersAndRender();
    const collapseBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.COLLAPSE];
    const expandBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.EXPAND];

    expect(getInstance().state.collapseSections).toEqual({});

    await act(async () => {
      collapseBtn!.action();
    });

    expect(getInstance().state.collapseSections).toEqual({
      [METRICS_SECTION_NAME]: true,
      [PARAMS_SECTION_NAME]: true,
      [OVERVIEW_SECTION_NAME]: true,
      Table: true,
      Tensorboard: true,
    });

    await act(async () => {
      expandBtn!.action();
    });

    expect(getInstance().state.collapseSections).toEqual({});
    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it('allows individual viewers to be collapsed and expanded', async () => {
    await renderCompare();

    expect(getInstance().state.collapseSections).toEqual({});

    fireEvent.click(getCollapseButton(OVERVIEW_SECTION_NAME));
    await waitFor(() =>
      expect(getInstance().state.collapseSections).toEqual({
        [OVERVIEW_SECTION_NAME]: true,
      }),
    );

    fireEvent.click(getCollapseButton(PARAMS_SECTION_NAME));
    await waitFor(() =>
      expect(getInstance().state.collapseSections).toEqual({
        [PARAMS_SECTION_NAME]: true,
        [OVERVIEW_SECTION_NAME]: true,
      }),
    );

    fireEvent.click(getCollapseButton(OVERVIEW_SECTION_NAME));
    fireEvent.click(getCollapseButton(PARAMS_SECTION_NAME));

    await waitFor(() =>
      expect(getInstance().state.collapseSections).toEqual({
        [PARAMS_SECTION_NAME]: false,
        [OVERVIEW_SECTION_NAME]: false,
      }),
    );
  }, 20000);

  it('allows individual runs to be selected and deselected', async () => {
    await renderCompare();
    await waitForRunRows(3);

    await waitFor(() =>
      expect(getInstance().state.selectedIds).toEqual([
        MOCK_RUN_1_ID,
        MOCK_RUN_2_ID,
        MOCK_RUN_3_ID,
      ]),
    );

    const rows = screen.getAllByTestId('table-row');
    fireEvent.click(rows[0]);
    fireEvent.click(rows[2]);

    await waitFor(() => expect(getInstance().state.selectedIds).toEqual([MOCK_RUN_2_ID]));

    fireEvent.click(rows[0]);

    await waitFor(() =>
      expect(getInstance().state.selectedIds).toEqual([MOCK_RUN_2_ID, MOCK_RUN_1_ID]),
    );
  });

  it('does not show viewers for deselected runs', async () => {
    await setUpViewersAndRender();

    await act(async () => {
      getInstance()._selectionChanged([]);
    });

    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  it('creates an extra aggregation plot for compatible viewers', async () => {
    outputArtifactLoaderSpy.mockImplementation(() => [
      { type: PlotType.TENSORBOARD, url: 'gs://path' },
      { data: [], type: PlotType.ROC },
    ]);

    const workflow = {
      status: {
        nodes: {
          node1: {
            outputs: {
              artifacts: [
                {
                  name: 'mlpipeline-ui-metadata',
                  s3: { s3Bucket: { bucket: 'test bucket' }, key: 'test key' },
                },
              ],
            },
          },
        },
      },
    };
    const run1 = newMockRun('run1-id');
    run1.pipeline_runtime!.workflow_manifest = JSON.stringify(workflow);
    const run2 = newMockRun('run2-id');
    run2.pipeline_runtime!.workflow_manifest = JSON.stringify(workflow);
    runs = [run1, run2];
    getRunSpy.mockImplementation(
      (id: string) => runs.find(r => r.run!.id === id) || newMockRun(id),
    );

    const props = generateProps();
    props.location.search = `?${QUERY_PARAMS.runlist}=run1-id,run2-id`;

    await renderCompare(props);
    await waitForViewersMap(2);

    await waitFor(() => {
      expect(document.querySelectorAll('.plotCard')).toHaveLength(6);
    });

    expect(stableMuiSnapshotFragment(renderResult!.asFragment())).toMatchSnapshot();
  });

  describe('EnhancedCompareV1', () => {
    it('redirects to experiments page when namespace changes', () => {
      const history = createMemoryHistory({
        initialEntries: ['/does-not-matter'],
      });
      const { rerender } = render(
        <Router history={history}>
          <NamespaceContext.Provider value='ns1'>
            <EnhancedCompareV1 {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).not.toEqual('/experiments');
      rerender(
        <Router history={history}>
          <NamespaceContext.Provider value='ns2'>
            <EnhancedCompareV1 {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/experiments');
    });

    it('does not redirect when namespace stays the same', () => {
      const history = createMemoryHistory({
        initialEntries: ['/initial-path'],
      });
      const { rerender } = render(
        <Router history={history}>
          <NamespaceContext.Provider value='ns1'>
            <EnhancedCompareV1 {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/initial-path');
      rerender(
        <Router history={history}>
          <NamespaceContext.Provider value='ns1'>
            <EnhancedCompareV1 {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/initial-path');
    });

    it('does not redirect when namespace initializes', () => {
      const history = createMemoryHistory({
        initialEntries: ['/initial-path'],
      });
      const { rerender } = render(
        <Router history={history}>
          <NamespaceContext.Provider value={undefined}>
            <EnhancedCompareV1 {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/initial-path');
      rerender(
        <Router history={history}>
          <NamespaceContext.Provider value='ns1'>
            <EnhancedCompareV1 {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual('/initial-path');
    });
  });
});
