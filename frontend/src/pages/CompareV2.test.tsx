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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { forwardRef, useImperativeHandle } from 'react';
import {
  ArtifactArtifactType,
  PipelineTaskTaskType,
  V2beta1PipelineTask,
  V2beta1Run,
} from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import { PageProps } from 'src/pages/Page';
import { CommonTestWrapper } from 'src/TestWrapper';
import { testBestPractices } from 'src/TestUtils';
import { buildParamsTableProps, buildScalarMetricsTableProps, CompareV2 } from './CompareV2';

vi.mock('src/pages/RunList', () => ({
  default: forwardRef(function MockRunList(_props, ref) {
    useImperativeHandle(ref, () => ({ refresh: vi.fn() }));
    return <div>Run list</div>;
  }),
}));

vi.mock('src/components/viewers/RuntimeMetricsVisualizations', () => ({
  RuntimeMetricsVisualizations: ({ artifacts }: { artifacts: Array<{ artifact_id?: string }> }) => (
    <div>Visualized {artifacts.map((artifact) => artifact.artifact_id).join(',')}</div>
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
  };

  function generateProps(): PageProps {
    return {
      history: { push: vi.fn(), replace: vi.fn() } as any,
      location: { pathname: '/compare', search: '?runlist=run-1,run-2' } as any,
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
      async (runId) => runs.find((run) => run.run_id === runId)!,
    );
    vi.spyOn(Apis.runServiceApiV2, 'tasks').mockImplementation(async (runId) => ({
      tasks: tasksByRun[runId] || [],
    }));
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

  it('shows classification artifacts grouped by run', async () => {
    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await screen.findByText('Train / accuracy');

    fireEvent.click(screen.getByText('Classification Metrics'));

    expect(await screen.findAllByText('First run')).toHaveLength(2);
    screen.getByText('Visualized classification-1');
  });

  it('shows an actionable banner when native task retrieval fails', async () => {
    vi.mocked(Apis.runServiceApiV2.tasks).mockRejectedValue(new Error('Task service unavailable'));
    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() =>
      expect(updateBannerSpy).toHaveBeenCalledWith({
        additionalInfo: 'Task service unavailable',
        message: 'Cannot get native task and artifact data for the selected runs.',
        mode: 'error',
      }),
    );
  });
});
