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

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { forwardRef, useImperativeHandle } from 'react';
import { BrowserRouter } from 'react-router-dom';
import {
  ArtifactArtifactType,
  PipelineTaskTaskType,
  V2beta1PipelineTask,
  V2beta1Run,
  V2beta1RuntimeState,
} from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import { PageProps } from 'src/pages/Page';
import { CommonTestWrapper } from 'src/TestWrapper';
import { testBestPractices } from 'src/TestUtils';
import {
  buildParamsTableProps,
  buildScalarMetricsTableProps,
  collectRuntimeComparisonArtifacts,
  CompareV2,
  ACTIVE_COMPARISON_REFRESH_INTERVAL,
} from './CompareV2';

vi.mock('src/pages/RunList', () => ({
  default: forwardRef(function MockRunList(_props, ref) {
    useImperativeHandle(ref, () => ({ refresh: vi.fn() }));
    return <div>Run list</div>;
  }),
}));

vi.mock('src/components/viewers/RuntimeArtifactComparison', () => ({
  createRuntimeArtifactComparisonSelectionState: () => ({
    panelSelections: { 'confusion matrix': ['', ''], html: ['', ''], markdown: ['', ''] },
    rocColorByKey: {},
  }),
  RuntimeArtifactComparison: ({
    artifacts,
    selectionState,
    setSelectionState,
  }: {
    artifacts: Array<{ artifact: { artifact_id?: string } }>;
    selectionState: {
      panelSelections: Record<string, [string, string]>;
    };
    setSelectionState: (updater: (current: any) => any) => void;
  }) => (
    <div>
      Compared {artifacts.map(({ artifact }) => artifact.artifact_id).join(',')}
      <button
        onClick={() =>
          setSelectionState((current) => ({
            ...current,
            panelSelections: {
              ...current.panelSelections,
              html: ['run-1:html-1', ''],
            },
          }))
        }
      >
        Select comparison artifact
      </button>
      <span data-testid='comparison-selection'>{selectionState.panelSelections.html[0]}</span>
    </div>
  ),
}));

testBestPractices();

describe('CompareV2', () => {
  const updateBannerSpy = vi.fn();
  const updateToolbarSpy = vi.fn();
  const runs: V2beta1Run[] = [
    {
      run_id: 'run-1',
      display_name: 'First run',
      runtime_config: { parameters: { epochs: 5, optimizer: 'adam' } },
    },
    {
      run_id: 'run-2',
      display_name: 'Second run',
      runtime_config: { parameters: { epochs: 10 } },
    },
  ];
  const thirdRun: V2beta1Run = {
    run_id: 'run-3',
    display_name: 'Third run',
    runtime_config: { parameters: { epochs: 15 } },
  };
  const tasksByRun: Record<string, V2beta1PipelineTask[]> = {
    'run-1': [
      {
        task_id: 'task-1',
        name: 'train',
        display_name: 'Train',
        type: PipelineTaskTaskType.RUNTIME,
        outputs: {
          artifacts: [
            {
              artifact_key: 'accuracy',
              artifacts: [
                {
                  artifact_id: 'metric-1',
                  name: 'accuracy',
                  type: ArtifactArtifactType.Metric,
                  number_value: 0.91,
                },
              ],
            },
            {
              artifact_key: 'classification',
              artifacts: [
                {
                  artifact_id: 'classification-1',
                  name: 'evaluation',
                  type: ArtifactArtifactType.ClassificationMetric,
                  metadata: {
                    confusionMatrix: { categories: ['cat', 'dog'], matrix: [1, 0, 0, 1] },
                  },
                },
              ],
            },
          ],
        },
      },
    ],
    'run-2': [
      {
        task_id: 'task-2',
        name: 'train',
        display_name: 'Train',
        type: PipelineTaskTaskType.RUNTIME,
        outputs: {
          artifacts: [
            {
              artifact_key: 'accuracy',
              artifacts: [
                {
                  artifact_id: 'metric-2',
                  name: 'accuracy',
                  type: ArtifactArtifactType.Metric,
                  number_value: 0.95,
                },
              ],
            },
          ],
        },
      },
    ],
    'run-3': [
      {
        task_id: 'task-3',
        name: 'train',
        display_name: 'Train',
        type: PipelineTaskTaskType.RUNTIME,
        outputs: {
          artifacts: [
            {
              artifact_key: 'accuracy',
              artifacts: [
                {
                  artifact_id: 'metric-3',
                  name: 'accuracy',
                  type: ArtifactArtifactType.Metric,
                  number_value: 0.99,
                },
              ],
            },
          ],
        },
      },
    ],
  };

  function generateProps(runIds = ['run-1', 'run-2']): PageProps {
    return {
      history: { push: vi.fn(), replace: vi.fn() } as any,
      location: { pathname: '/compare', search: `?runlist=${runIds.join(',')}` } as any,
      match: { params: {}, isExact: true, path: '/compare', url: '/compare' } as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: vi.fn(),
      updateSnackbar: vi.fn(),
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    vi.spyOn(Apis.runServiceApiV2, 'getRun').mockImplementation(
      async (runId) => [...runs, thirdRun].find((run) => run.run_id === runId)!,
    );
    vi.spyOn(Apis.runServiceApiV2, 'tasks').mockImplementation(async (runId) => ({
      tasks: tasksByRun[runId] || [],
    }));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('builds a parameter comparison from native runs', () => {
    expect(
      buildParamsTableProps(runs.map((run) => ({ run, tasks: tasksByRun[run.run_id!] }))),
    ).toEqual({
      xLabels: ['First run', 'Second run'],
      yLabels: ['epochs', 'optimizer'],
      rows: [
        ['5', '10'],
        ['adam', ''],
      ],
    });
  });

  it('builds scalar metric comparison from hydrated task artifacts', () => {
    expect(
      buildScalarMetricsTableProps(runs.map((run) => ({ run, tasks: tasksByRun[run.run_id!] }))),
    ).toEqual({
      xLabels: ['First run', 'Second run'],
      yLabels: ['Train / accuracy'],
      rows: [['0.91', '0.95']],
    });
  });

  it('uses the named metadata value and a dash fallback for scalar metrics', () => {
    const tasks: V2beta1PipelineTask[] = [
      {
        name: 'evaluate',
        outputs: {
          artifacts: [
            {
              artifact_key: 'accuracy',
              artifacts: [
                {
                  name: 'accuracy',
                  type: ArtifactArtifactType.Metric,
                  metadata: { accuracy: 0.88, ignored: 1 },
                },
              ],
            },
            {
              artifact_key: 'loss',
              artifacts: [
                {
                  name: 'loss',
                  type: ArtifactArtifactType.Metric,
                  metadata: { loss: null as any },
                },
              ],
            },
          ],
        },
      },
    ];

    expect(buildScalarMetricsTableProps([{ run: runs[0], tasks }])?.rows).toEqual([
      ['0.88'],
      ['-'],
    ]);
  });

  it('keeps metrics from separate loop iterations distinct', () => {
    const iterationTasks: V2beta1PipelineTask[] = [0, 1].map((iteration) => ({
      task_id: `task-${iteration}`,
      name: 'train',
      display_name: 'Train',
      scope_path: 'root.loop.train',
      type_attributes: { iteration_index: String(iteration) },
      outputs: {
        artifacts: [
          {
            artifact_key: 'metrics',
            artifacts: [
              {
                artifact_id: `metric-${iteration}`,
                name: 'accuracy',
                type: ArtifactArtifactType.Metric,
                number_value: 0.9 + iteration / 100,
              },
            ],
          },
        ],
      },
    }));

    expect(buildScalarMetricsTableProps([{ run: runs[0], tasks: iterationTasks }])).toEqual({
      xLabels: ['First run'],
      yLabels: ['loop.train [iteration 0] / accuracy', 'loop.train [iteration 1] / accuracy'],
      rows: [['0.9'], ['0.91']],
    });
  });

  it('keeps same-named metrics from different artifact keys distinct across runs', () => {
    const comparisonData = runs.map((run, runIndex) => ({
      run,
      tasks: [
        {
          name: 'evaluate',
          outputs: {
            artifacts: ['train-metrics', 'validation-metrics'].map((artifactKey, metricIndex) => ({
              artifact_key: artifactKey,
              artifacts: [
                {
                  name: 'accuracy',
                  type: ArtifactArtifactType.Metric,
                  number_value: 0.8 + runIndex / 10 + metricIndex / 100,
                },
              ],
            })),
          },
        },
      ],
    }));

    expect(buildScalarMetricsTableProps(comparisonData)).toEqual({
      xLabels: ['First run', 'Second run'],
      yLabels: ['evaluate / train-metrics / accuracy', 'evaluate / validation-metrics / accuracy'],
      rows: [
        ['0.8', '0.9'],
        ['0.81', '0.91'],
      ],
    });
  });

  it('keeps duplicate same-named metrics within one artifact group', () => {
    const tasks: V2beta1PipelineTask[] = [
      {
        name: 'evaluate',
        outputs: {
          artifacts: [
            {
              artifact_key: 'metrics',
              artifacts: [0.8, 0.9].map((numberValue) => ({
                name: 'accuracy',
                type: ArtifactArtifactType.Metric,
                number_value: numberValue,
              })),
            },
          ],
        },
      },
    ];

    expect(buildScalarMetricsTableProps([{ run: runs[0], tasks }])).toEqual({
      xLabels: ['First run'],
      yLabels: ['evaluate / accuracy', 'evaluate / accuracy (2)'],
      rows: [['0.8'], ['0.9']],
    });
  });

  it('builds stable native comparison labels with run, task, and artifact provenance', () => {
    expect(
      collectRuntimeComparisonArtifacts(
        runs.map((run) => ({ run, tasks: tasksByRun[run.run_id!] })),
        'team-a',
      ),
    ).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          key: 'run-1:task-1:classification:0:classification-1',
          label: 'First run / Train / evaluation',
          namespace: 'team-a',
        }),
      ]),
    );
  });

  it('loads runs and tasks in parallel and displays native comparisons', async () => {
    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );

    await screen.findByText('Train / accuracy');
    screen.getByText('0.91');
    screen.getByText('0.95');
    screen.getByText('epochs');
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledTimes(2);
    expect(Apis.runServiceApiV2.tasks).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('polls active comparisons and stops after observing a terminal run state', async () => {
    vi.useFakeTimers();
    const runningRun = { ...runs[0], state: V2beta1RuntimeState.RUNNING };
    const succeededRun = { ...runs[0], state: V2beta1RuntimeState.SUCCEEDED };
    vi.mocked(Apis.runServiceApiV2.getRun)
      .mockResolvedValueOnce(runningRun)
      .mockResolvedValue(succeededRun);

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps(['run-1'])} />
      </CommonTestWrapper>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(ACTIVE_COMPARISON_REFRESH_INTERVAL);
      await Promise.resolve();
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledTimes(2);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(ACTIVE_COMPARISON_REFRESH_INTERVAL * 2);
    });
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledTimes(2);
  });

  it('preserves cached task metrics when a background task refresh fails', async () => {
    vi.useFakeTimers();
    vi.mocked(Apis.runServiceApiV2.getRun).mockResolvedValue({
      ...runs[0],
      state: V2beta1RuntimeState.RUNNING,
    });
    vi.mocked(Apis.runServiceApiV2.tasks)
      .mockResolvedValueOnce({ tasks: tasksByRun['run-1'] })
      .mockRejectedValueOnce(new Error('Task service unavailable'));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps(['run-1'])} />
      </CommonTestWrapper>,
    );
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(screen.getByText('0.91')).toBeVisible();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(ACTIVE_COMPARISON_REFRESH_INTERVAL);
      await Promise.resolve();
      await vi.advanceTimersByTimeAsync(1);
    });

    expect(screen.getByText('0.91')).toBeVisible();
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({ additionalInfo: 'run-1 tasks: Task service unavailable' }),
    );
  });

  it('passes artifacts from all selected runs to the native comparison surface', async () => {
    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await screen.findByText('Train / accuracy');

    fireEvent.click(screen.getByText('Classification Metrics'));

    screen.getByText(/Compared .*classification-1/);
  });

  it('preserves artifact selections after visiting Scalar Metrics', async () => {
    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await screen.findByText('Train / accuracy');

    fireEvent.click(screen.getByText('HTML'));
    fireEvent.click(screen.getByRole('button', { name: 'Select comparison artifact' }));
    expect(screen.getByTestId('comparison-selection')).toHaveTextContent('run-1:html-1');

    fireEvent.click(screen.getByText('Scalar Metrics'));
    expect(screen.queryByTestId('comparison-selection')).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('HTML'));

    expect(screen.getByTestId('comparison-selection')).toHaveTextContent('run-1:html-1');
  });

  it('keeps available runs visible when one comparison query fails', async () => {
    vi.mocked(Apis.runServiceApiV2.getRun).mockImplementation(async (runId) => {
      if (runId === 'run-2') {
        throw new Error('Permission denied');
      }
      return [...runs, thirdRun].find((run) => run.run_id === runId)!;
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: 3, retryDelay: 0 } },
    });
    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <CompareV2 {...generateProps(['run-1', 'run-2', 'run-3'])} />
        </QueryClientProvider>
      </BrowserRouter>,
    );

    await screen.findByText('0.91');
    screen.getByText('0.99');
    expect(screen.queryByText('0.95')).not.toBeInTheDocument();
    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith({
        additionalInfo: 'run-2: Permission denied',
        message:
          'Cannot get comparison data for 1 selected run. Available runs are still shown. Refresh the page to try again.',
        mode: 'warning',
      }),
    );
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledTimes(3);
    expect(Apis.runServiceApiV2.tasks).toHaveBeenCalledTimes(3);
  });

  it('reuses cached comparison data when the selected run list changes', async () => {
    const { rerender } = render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await screen.findByText('0.95');

    rerender(
      <CommonTestWrapper>
        <CompareV2 {...generateProps(['run-1', 'run-3'])} />
      </CommonTestWrapper>,
    );

    await screen.findByText('0.99');
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledTimes(3);
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledWith('run-1');
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledWith('run-2');
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledWith('run-3');
    expect(Apis.runServiceApiV2.tasks).toHaveBeenCalledTimes(3);
  });

  it('keeps run parameters visible when task hydration fails', async () => {
    vi.mocked(Apis.runServiceApiV2.tasks).mockRejectedValue(new Error('Task service unavailable'));
    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith({
        additionalInfo:
          'run-1 tasks: Task service unavailable\nrun-2 tasks: Task service unavailable',
        message:
          'Cannot get comparison data for 2 selected runs. Available runs are still shown. Refresh the page to try again.',
        mode: 'warning',
      }),
    );
    screen.getByText('epochs');
    screen.getByText('optimizer');
    expect(Apis.runServiceApiV2.getRun).toHaveBeenCalledTimes(2);
    expect(Apis.runServiceApiV2.tasks).toHaveBeenCalledTimes(2);
  });
});
