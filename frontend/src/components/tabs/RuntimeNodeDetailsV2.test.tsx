/*
 * Copyright 2023 The Kubeflow Authors
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
import { dump, loadAll } from 'js-yaml';
import {
  ArtifactArtifactType,
  PipelineTaskTaskPodType,
  PipelineTaskTaskState,
  PipelineTaskTaskType,
  V2PipelineTask,
} from 'src/apisv2/run';
import {
  getDriverLogsInfo,
  getDriverLogUri,
  getTaskDetailsFields,
  getLogsInfo,
  LOGS_BANNER_ADDITIONAL_INFO,
  LOGS_BANNER_MESSAGE,
  LIFECYCLE_ENDED_MESSAGE,
  LIFECYCLE_FAILURE_MESSAGE,
  LIFECYCLE_WARNING_MESSAGE,
  LOGS_DETAILS,
  RuntimeNodeDetailsV2,
} from 'src/components/tabs/RuntimeNodeDetailsV2';
import v2PvcYamlString from 'src/data/test/create_mount_delete_dynamic_pvc.yaml?raw';
import { Apis } from 'src/lib/Apis';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { PlotType } from 'src/components/viewers/Viewer';

const V2_PVC_TEMPLATE_STRING = dump({
  pipeline_spec: loadAll(v2PvcYamlString)[0],
  platform_spec: loadAll(v2PvcYamlString)[1],
});

testBestPractices();

describe('RuntimeNodeDetailsV2', () => {
  const TEST_RUN_ID = 'test-run-id';
  const TEST_TASK_ID = 'test-task-id';
  const TEST_POD_NAME = 'test-pod-name';
  const TEST_NAMESPACE = 'kubeflow';
  const TEST_LOG_VIEW_ID = 'logs-view-window';

  afterEach(() => vi.useRealTimers());

  const executionElement = {
    data: { label: 'preprocess' },
    id: 'task.preprocess',
    position: { x: 100, y: 100 },
    type: 'EXECUTION',
  } as const;

  function createTask(overrides: Partial<V2PipelineTask> = {}): V2PipelineTask {
    return {
      task_id: TEST_TASK_ID,
      run_id: TEST_RUN_ID,
      name: 'preprocess',
      display_name: 'Preprocess',
      type: PipelineTaskTaskType.RUNTIME,
      state: PipelineTaskTaskState.SUCCEEDED,
      create_time: new Date('2026-08-11T12:00:00Z'),
      pods: [{ name: TEST_POD_NAME, type: PipelineTaskTaskPodType.EXECUTOR }],
      ...overrides,
    };
  }

  function taskView(task: V2PipelineTask, extraProps: Record<string, unknown> = {}) {
    return (
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          runId={TEST_RUN_ID}
          element={executionElement}
          elementRuntimeInfo={{ task }}
          namespace={TEST_NAMESPACE}
          {...extraProps}
        />
      </CommonTestWrapper>
    );
  }

  function renderTask(task: V2PipelineTask, extraProps: Record<string, unknown> = {}) {
    return render(taskView(task, extraProps));
  }

  describe('System Logs', () => {
    const driverLogUri = 's3://pipeline-root/driver-logs.txt';

    beforeEach(() => {
      // Give the real virtualized LogViewer a viewport in jsdom.
      vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(600);
      vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(800);
    });

    function createSystemLogsTask(
      uri: unknown = driverLogUri,
      overrides: Partial<V2PipelineTask> = {},
    ): V2PipelineTask {
      return createTask({
        status_metadata: { custom_properties: { driver_logs_uri: uri as object } },
        ...overrides,
      });
    }

    it('derives the driver log URI only from a non-empty string property', () => {
      expect(getDriverLogUri(createSystemLogsTask())).toBe(driverLogUri);
      expect(getDriverLogUri(createSystemLogsTask(''))).toBeUndefined();
      expect(getDriverLogUri(createSystemLogsTask(42))).toBeUndefined();
      expect(getDriverLogUri(createTask())).toBeUndefined();
      expect(getDriverLogUri(undefined)).toBeUndefined();
    });

    it('reports system logs with the same keys as pod logs', async () => {
      const readFileSpy = vi
        .spyOn(Apis, 'readFile')
        .mockResolvedValueOnce('driver logs')
        .mockRejectedValueOnce(new Error('storage temporarily unavailable'));

      expect(await getDriverLogsInfo(driverLogUri, TEST_NAMESPACE)).toEqual(
        new Map([[LOGS_DETAILS, 'driver logs']]),
      );
      expect(await getDriverLogsInfo(driverLogUri, TEST_NAMESPACE)).toEqual(
        new Map([
          [LOGS_BANNER_MESSAGE, 'Failed to retrieve system logs.'],
          [LOGS_BANNER_ADDITIONAL_INFO, 'Error response: storage temporarily unavailable'],
        ]),
      );
      expect(readFileSpy).toHaveBeenCalledTimes(2);
    });

    it.each([PipelineTaskTaskState.SUCCEEDED, PipelineTaskTaskState.FAILED])(
      'retrieves driver logs for a %s task only after opening the tab',
      async (state) => {
        const readFileSpy = vi
          .spyOn(Apis, 'readFile')
          .mockResolvedValue('driver initialization\nfinished');
        const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('executor output');
        renderTask(
          createSystemLogsTask(driverLogUri, {
            state,
            status_metadata: {
              message: state === PipelineTaskTaskState.FAILED ? 'executor failed' : undefined,
              custom_properties: { driver_logs_uri: driverLogUri as unknown as object },
            },
          }),
        );

        expect(readFileSpy).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole('button', { name: 'Task Details', exact: true }));
        expect(readFileSpy).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole('button', { name: 'Logs', exact: true }));
        expect(await screen.findByText('executor output')).toBeVisible();
        expect(readFileSpy).not.toHaveBeenCalled();

        fireEvent.click(screen.getByRole('button', { name: 'System Logs', exact: true }));

        expect(await screen.findByText('driver initialization')).toBeVisible();
        expect(screen.getByText('finished')).toBeVisible();
        expect(screen.queryByText('executor output')).not.toBeInTheDocument();
        expect(readFileSpy).toHaveBeenCalledExactlyOnceWith({
          path: {
            bucket: 'pipeline-root',
            key: 'driver-logs.txt',
            keyEncoding: 'storage',
            source: 's3',
          },
          namespace: TEST_NAMESPACE,
        });
        expect(getPodLogsSpy).toHaveBeenCalledTimes(1);
      },
    );

    it.each([
      ['missing', undefined],
      ['null', null],
      ['number', 42],
      ['boolean', false],
      ['object', { uri: driverLogUri }],
      ['array', [driverLogUri]],
      ['empty', ''],
    ])('shows an unavailable state without requesting logs for a %s URI', async (_name, uri) => {
      const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('unused');
      const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('unused');
      renderTask(uri === undefined ? createTask() : createSystemLogsTask(uri));

      fireEvent.click(screen.getByRole('button', { name: 'System Logs', exact: true }));

      expect(await screen.findByText('System logs are not available yet.')).toBeVisible();
      expect(screen.queryByTestId(TEST_LOG_VIEW_ID)).not.toBeInTheDocument();
      expect(readFileSpy).not.toHaveBeenCalled();
      expect(getPodLogsSpy).not.toHaveBeenCalled();
    });

    it('shows artifact-fetch error details and recovers on Refresh without leaving System Logs', async () => {
      const readFileSpy = vi
        .spyOn(Apis, 'readFile')
        .mockRejectedValueOnce(new Error('storage temporarily unavailable'))
        .mockResolvedValueOnce('recovered system logs');
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      renderTask(createSystemLogsTask());

      fireEvent.click(screen.getByRole('button', { name: 'System Logs', exact: true }));

      expect(await screen.findByText('Failed to retrieve system logs.')).toBeVisible();
      expect(screen.queryByTestId(TEST_LOG_VIEW_ID)).not.toBeInTheDocument();
      fireEvent.click(screen.getByRole('button', { name: 'Details', exact: true }));
      const dialog = await screen.findByRole('dialog');
      expect(
        within(dialog).getByText('Error response: storage temporarily unavailable'),
      ).toBeVisible();
      fireEvent.click(within(dialog).getByRole('button', { name: 'Dismiss' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Refresh' }));

      expect(await screen.findByText('recovered system logs')).toBeVisible();
      expect(screen.queryByText('Failed to retrieve system logs.')).not.toBeInTheDocument();
      expect(readFileSpy).toHaveBeenCalledTimes(2);
      expect(consoleErrorSpy).not.toHaveBeenCalled();
    });

    it('refetches transient artifact errors while running and stops polling when the tab closes', async () => {
      vi.useFakeTimers();
      const readFileSpy = vi
        .spyOn(Apis, 'readFile')
        .mockRejectedValueOnce(new Error('artifact upload in progress'))
        .mockResolvedValue('live system logs');
      renderTask(createSystemLogsTask(driverLogUri, { state: PipelineTaskTaskState.RUNNING }));

      await act(async () => vi.advanceTimersByTimeAsync(20_000));
      expect(readFileSpy).not.toHaveBeenCalled();
      fireEvent.click(screen.getByRole('button', { name: 'System Logs', exact: true }));
      await act(async () => vi.advanceTimersByTimeAsync(1));
      expect(screen.getByText('Failed to retrieve system logs.')).toBeVisible();

      await act(async () => vi.advanceTimersByTimeAsync(10_000));
      expect(screen.getByText('live system logs')).toBeVisible();
      expect(screen.queryByText('Failed to retrieve system logs.')).not.toBeInTheDocument();
      expect(readFileSpy).toHaveBeenCalledTimes(2);

      fireEvent.click(screen.getByRole('button', { name: 'Task Details', exact: true }));
      await act(async () => vi.advanceTimersByTimeAsync(20_000));
      expect(readFileSpy).toHaveBeenCalledTimes(2);
    });

    it.each(['task', 'URI', 'namespace'])(
      'does not retain stale system logs when the %s changes',
      async (changedSource) => {
        let resolveNextLogs!: (logs: string) => void;
        const nextLogs = new Promise<string>((resolve) => {
          resolveNextLogs = resolve;
        });
        const readFileSpy = vi
          .spyOn(Apis, 'readFile')
          .mockResolvedValueOnce('previous system logs')
          .mockReturnValueOnce(nextLogs);
        const view = renderTask(createSystemLogsTask());
        fireEvent.click(screen.getByRole('button', { name: 'System Logs', exact: true }));
        expect(await screen.findByText('previous system logs')).toBeVisible();

        view.rerender(
          taskView(
            createSystemLogsTask(
              changedSource === 'URI' ? 's3://pipeline-root/retry-driver-logs.txt' : driverLogUri,
              changedSource === 'task' ? { task_id: 'second-task-id' } : {},
            ),
            { namespace: changedSource === 'namespace' ? 'second-namespace' : TEST_NAMESPACE },
          ),
        );

        await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(2));
        expect(screen.queryByText('previous system logs')).not.toBeInTheDocument();
        expect(screen.queryByTestId(TEST_LOG_VIEW_ID)).not.toBeInTheDocument();
        await act(async () => resolveNextLogs('current system logs'));
        expect(await screen.findByText('current system logs')).toBeVisible();
      },
    );

    it('ignores a late artifact response from a previously selected task', async () => {
      let resolvePreviousLogs!: (logs: string) => void;
      const previousLogs = new Promise<string>((resolve) => {
        resolvePreviousLogs = resolve;
      });
      const readFileSpy = vi
        .spyOn(Apis, 'readFile')
        .mockReturnValueOnce(previousLogs)
        .mockResolvedValueOnce('current task system logs');
      const view = renderTask(createSystemLogsTask());
      fireEvent.click(screen.getByRole('button', { name: 'System Logs', exact: true }));
      await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(1));

      view.rerender(taskView(createSystemLogsTask(driverLogUri, { task_id: 'second-task-id' })));
      expect(await screen.findByText('current task system logs')).toBeVisible();
      await act(async () => resolvePreviousLogs('previous task system logs'));

      expect(screen.getByText('current task system logs')).toBeVisible();
      expect(screen.queryByText('previous task system logs')).not.toBeInTheDocument();
    });

    it('retrieves a newly available URI on task refresh without resetting the selected tab', async () => {
      const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('new system logs');
      const view = renderTask(createTask());
      fireEvent.click(screen.getByRole('button', { name: 'System Logs', exact: true }));
      expect(await screen.findByText('System logs are not available yet.')).toBeVisible();
      expect(readFileSpy).not.toHaveBeenCalled();

      view.rerender(taskView(createSystemLogsTask()));

      expect(await screen.findByText('new system logs')).toBeVisible();
      expect(screen.queryByText('System logs are not available yet.')).not.toBeInTheDocument();
      expect(readFileSpy).toHaveBeenCalledTimes(1);
    });

    it('fetches final system logs when the parent run finishes and stops polling', async () => {
      vi.useFakeTimers();
      const readFileSpy = vi
        .spyOn(Apis, 'readFile')
        .mockResolvedValueOnce('live system logs')
        .mockResolvedValue('final system logs');
      const task = createSystemLogsTask(driverLogUri, { state: PipelineTaskTaskState.RUNNING });
      const view = renderTask(task, { sourceFinished: false });
      fireEvent.click(screen.getByRole('button', { name: 'System Logs', exact: true }));
      await act(async () => vi.advanceTimersByTimeAsync(1));
      expect(screen.getByText('live system logs')).toBeVisible();

      view.rerender(taskView(task, { sourceFinished: true }));
      await act(async () => vi.advanceTimersByTimeAsync(1));
      expect(screen.getByText('final system logs')).toBeVisible();
      expect(readFileSpy).toHaveBeenCalledTimes(2);
      await act(async () => vi.advanceTimersByTimeAsync(20_000));
      expect(readFileSpy).toHaveBeenCalledTimes(2);
    });
  });

  it('shows an error when pod logs and the native artifact fallback are unavailable', async () => {
    const getPodLogsSpy = vi
      .spyOn(Apis, 'getPodLogs')
      .mockRejectedValue(new Error('Failed to retrieve pod logs'));
    renderTask(createTask());

    fireEvent.click(await screen.findByText('Logs'));

    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalled());
    await screen.findByText('Failed to retrieve pod logs.');
  });

  it('displays pod logs on the execution side panel', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('test-logs-details');
    renderTask(createTask());

    fireEvent.click(await screen.findByText('Logs'));

    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalled());
    await screen.findByTestId(TEST_LOG_VIEW_ID);
  });

  it('retrieves pod logs without an experiment namespace', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('test-logs-details');

    const logsInfo = await getLogsInfo(createTask(), TEST_RUN_ID);

    expect(logsInfo.get(LOGS_DETAILS)).toBe('test-logs-details');
    expect(getPodLogsSpy).toHaveBeenCalledWith(TEST_RUN_ID, TEST_POD_NAME, '', '2026-08-11');
  });

  it('falls back to explicitly labeled driver logs when executor output is unavailable', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('driver diagnostics');

    const logsInfo = await getLogsInfo(
      createTask({ pods: [{ name: 'driver-pod', type: PipelineTaskTaskPodType.DRIVER }] }),
      TEST_RUN_ID,
      TEST_NAMESPACE,
    );

    expect(getPodLogsSpy).toHaveBeenCalledWith(
      TEST_RUN_ID,
      'driver-pod',
      TEST_NAMESPACE,
      '2026-08-11',
    );
    expect(logsInfo.get(LOGS_DETAILS)).toBe('driver diagnostics');
    expect(logsInfo.get(LOGS_BANNER_MESSAGE)).toBe(
      'Showing driver initialization logs. These are not component executor output logs.',
    );
  });

  it('shows the driver-log label alongside driver diagnostics', async () => {
    vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('driver diagnostics');
    renderTask(
      createTask({ pods: [{ name: 'driver-pod', type: PipelineTaskTaskPodType.DRIVER }] }),
    );

    fireEvent.click(await screen.findByText('Logs'));

    await screen.findByText(
      'Showing driver initialization logs. These are not component executor output logs.',
    );
    screen.getByTestId(TEST_LOG_VIEW_ID);
  });

  it('prefers executor pod logs when both executor and driver pods are available', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('executor output');

    const logsInfo = await getLogsInfo(
      createTask({
        pods: [
          { name: 'driver-pod', type: PipelineTaskTaskPodType.DRIVER },
          { name: TEST_POD_NAME, type: PipelineTaskTaskPodType.EXECUTOR },
        ],
      }),
      TEST_RUN_ID,
      TEST_NAMESPACE,
    );

    expect(getPodLogsSpy).toHaveBeenCalledTimes(1);
    expect(getPodLogsSpy).toHaveBeenCalledWith(
      TEST_RUN_ID,
      TEST_POD_NAME,
      TEST_NAMESPACE,
      '2026-08-11',
    );
    expect(logsInfo.get(LOGS_DETAILS)).toBe('executor output');
    expect(logsInfo.has(LOGS_BANNER_MESSAGE)).toBe(false);
  });

  it('uses the latest executor pod appended for a retried task', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('latest output');

    await getLogsInfo(
      createTask({
        pods: [
          { name: 'first-attempt-executor', type: PipelineTaskTaskPodType.EXECUTOR },
          { name: 'retry-executor', type: PipelineTaskTaskPodType.EXECUTOR },
        ],
      }),
      TEST_RUN_ID,
      TEST_NAMESPACE,
    );

    expect(getPodLogsSpy).toHaveBeenCalledWith(
      TEST_RUN_ID,
      'retry-executor',
      TEST_NAMESPACE,
      '2026-08-11',
    );
  });

  it('keeps prior logs visible while a newly discovered source is loading', async () => {
    let resolveRetryLogs: ((logs: string) => void) | undefined;
    const retryLogs = new Promise<string>((resolve) => {
      resolveRetryLogs = resolve;
    });
    const getPodLogsSpy = vi
      .spyOn(Apis, 'getPodLogs')
      .mockResolvedValueOnce('first attempt logs')
      .mockReturnValueOnce(retryLogs);
    const initialTask = createTask({ state: PipelineTaskTaskState.RUNNING });
    const view = renderTask(initialTask);

    fireEvent.click(await screen.findByText('Logs'));
    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalledTimes(1));
    expect(await screen.findByTestId(TEST_LOG_VIEW_ID)).toBeVisible();

    view.rerender(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          runId={TEST_RUN_ID}
          element={executionElement}
          elementRuntimeInfo={{
            task: createTask({
              state: PipelineTaskTaskState.RUNNING,
              pods: [
                { name: TEST_POD_NAME, type: PipelineTaskTaskPodType.EXECUTOR },
                { name: 'retry-pod', type: PipelineTaskTaskPodType.EXECUTOR },
              ],
            }),
          }}
          namespace={TEST_NAMESPACE}
        />
      </CommonTestWrapper>,
    );

    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalledTimes(2));
    expect(screen.getByTestId(TEST_LOG_VIEW_ID)).toBeVisible();
    await act(async () => resolveRetryLogs?.('retry logs'));
  });

  it('does not show placeholder logs from a previously selected task', async () => {
    let resolveSecondTaskLogs: ((logs: string) => void) | undefined;
    const secondTaskLogs = new Promise<string>((resolve) => {
      resolveSecondTaskLogs = resolve;
    });
    const getPodLogsSpy = vi
      .spyOn(Apis, 'getPodLogs')
      .mockResolvedValueOnce('first task logs')
      .mockReturnValueOnce(secondTaskLogs);
    const view = renderTask(createTask());

    fireEvent.click(await screen.findByText('Logs'));
    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalledTimes(1));
    expect(await screen.findByTestId(TEST_LOG_VIEW_ID)).toBeVisible();

    view.rerender(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          runId={TEST_RUN_ID}
          element={{ ...executionElement, id: 'task.train', data: { label: 'train' } }}
          elementRuntimeInfo={{
            task: createTask({
              task_id: 'second-task-id',
              name: 'train',
              pods: [{ name: 'second-pod', type: PipelineTaskTaskPodType.EXECUTOR }],
            }),
          }}
          namespace={TEST_NAMESPACE}
        />
      </CommonTestWrapper>,
    );

    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalledTimes(2));
    expect(screen.queryByTestId(TEST_LOG_VIEW_ID)).not.toBeInTheDocument();

    await act(async () => resolveSecondTaskLogs?.('second task logs'));
    expect(await screen.findByTestId(TEST_LOG_VIEW_ID)).toBeVisible();
  });

  it('stops log polling when the parent run is terminal', async () => {
    vi.useFakeTimers();
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('final output');
    renderTask(createTask({ state: PipelineTaskTaskState.RUNNING }), { sourceFinished: true });

    fireEvent.click(screen.getByText('Logs'));
    await act(async () => vi.advanceTimersByTimeAsync(0));
    expect(getPodLogsSpy).toHaveBeenCalledTimes(1);

    await act(async () => vi.advanceTimersByTimeAsync(20_000));
    expect(getPodLogsSpy).toHaveBeenCalledTimes(1);
    vi.useRealTimers();
  });

  it('falls back to the native executor-logs artifact when pod logs fail', async () => {
    vi.spyOn(Apis, 'getPodLogs').mockRejectedValue(new Error('Pod logs unavailable'));
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('artifact-log-details');
    renderTask(
      createTask({
        outputs: {
          artifacts: [
            {
              artifact_key: 'executor-logs',
              artifacts: [
                {
                  artifact_id: 'logs-artifact',
                  name: 'executor-logs',
                  type: ArtifactArtifactType.Artifact,
                  uri: 's3://pipeline-root/logs.txt',
                  namespace: TEST_NAMESPACE,
                },
              ],
            },
          ],
        },
      }),
    );

    fireEvent.click(await screen.findByText('Logs'));

    await waitFor(() => expect(readFileSpy).toHaveBeenCalled());
    expect(await screen.findByTestId(TEST_LOG_VIEW_ID)).toBeVisible();
  });

  it('reports both pod and executor-artifact errors when both log sources fail', async () => {
    vi.spyOn(Apis, 'getPodLogs').mockRejectedValue(new Error('pod was garbage collected'));
    vi.spyOn(Apis, 'readFile').mockRejectedValue(new Error('storage credentials expired'));
    const logsInfo = await getLogsInfo(
      createTask({
        outputs: {
          artifacts: [
            {
              artifact_key: 'executor-logs',
              artifacts: [
                {
                  name: 'executor-logs',
                  uri: 's3://pipeline-root/logs.txt',
                  metadata: { store_session_info: 'stale-session' } as any,
                },
              ],
            },
          ],
        },
      }),
      TEST_RUN_ID,
      TEST_NAMESPACE,
    );

    expect(logsInfo.get(LOGS_BANNER_MESSAGE)).toBe('Failed to retrieve task logs.');
    expect(logsInfo.get(LOGS_BANNER_ADDITIONAL_INFO)).toContain(
      'Pod logs error: pod was garbage collected',
    );
    expect(logsInfo.get(LOGS_BANNER_ADDITIONAL_INFO)).toContain(
      'Executor logs artifact error: storage credentials expired',
    );
    expect(Apis.readFile).toHaveBeenCalledWith({
      path: {
        bucket: 'pipeline-root',
        key: 'logs.txt',
        keyEncoding: 'storage',
        source: 's3',
      },
      namespace: TEST_NAMESPACE,
    });
  });

  it('falls back to the current date when the task creation timestamp is invalid', async () => {
    vi.useFakeTimers();
    try {
      vi.setSystemTime(new Date('2026-08-22T12:00:00Z'));
      const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('executor logs');

      const logsInfo = await getLogsInfo(
        createTask({
          create_time: new Date('invalid'),
          pods: [{ name: 'executor-pod', type: PipelineTaskTaskPodType.EXECUTOR }],
        }),
        TEST_RUN_ID,
        TEST_NAMESPACE,
      );

      expect(logsInfo.get(LOGS_DETAILS)).toBe('executor logs');
      expect(getPodLogsSpy).toHaveBeenCalledWith(
        TEST_RUN_ID,
        'executor-pod',
        TEST_NAMESPACE,
        '2026-08-22',
      );
    } finally {
      vi.useRealTimers();
    }
  });

  it('returns cached text without retrieving pod logs', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('unused');
    const logsInfo = await getLogsInfo(
      createTask({ state: PipelineTaskTaskState.CACHED }),
      TEST_RUN_ID,
      TEST_NAMESPACE,
    );

    expect(logsInfo.get(LOGS_DETAILS)).toBe('This step output is taken from cache.');
    expect(getPodLogsSpy).not.toHaveBeenCalled();
  });

  it('shows a warning banner with the raw lifecycle message behind Details', () => {
    renderTask(
      createTask({
        state: PipelineTaskTaskState.RUNNING,
        lifecycle_message: 'Back-off pulling image "ghcr.io/example/does-not-exist:v1"',
      }),
    );

    expect(screen.getByText(LIFECYCLE_WARNING_MESSAGE)).toBeInTheDocument();
    expect(
      screen.queryByText('Back-off pulling image "ghcr.io/example/does-not-exist:v1"'),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Details' }));
    expect(
      screen.getByText('Back-off pulling image "ghcr.io/example/does-not-exist:v1"'),
    ).toBeInTheDocument();
  });

  it('does not say the task may recover once the run has finished', () => {
    renderTask(
      createTask({
        state: PipelineTaskTaskState.RUNNING,
        lifecycle_message: 'OOMKilled (exit code 137)',
      }),
      { sourceFinished: true },
    );

    expect(screen.getByText(LIFECYCLE_ENDED_MESSAGE)).toBeInTheDocument();
    expect(screen.queryByText(LIFECYCLE_WARNING_MESSAGE)).not.toBeInTheDocument();
  });

  it('says the task failed, not that it has not failed yet, once a task with a lifecycle message fails', () => {
    renderTask(
      createTask({
        state: PipelineTaskTaskState.FAILED,
        lifecycle_message: 'OOMKilled (exit code 137)',
      }),
    );

    expect(screen.getByText(LIFECYCLE_FAILURE_MESSAGE)).toBeInTheDocument();
    expect(screen.queryByText(LIFECYCLE_WARNING_MESSAGE)).not.toBeInTheDocument();
  });

  it('shows the lifecycle warning banner on a sub-DAG node', () => {
    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          runId={TEST_RUN_ID}
          element={{ ...executionElement, type: 'SUB_DAG' } as any}
          elementRuntimeInfo={{
            task: createTask({
              type: PipelineTaskTaskType.DAG,
              state: PipelineTaskTaskState.RUNNING,
              lifecycle_message: 'OOMKilled',
            }),
          }}
          namespace={TEST_NAMESPACE}
        />
      </CommonTestWrapper>,
    );

    expect(screen.getByText(LIFECYCLE_WARNING_MESSAGE)).toBeInTheDocument();
  });

  it('uses the finished-run wording on a sub-DAG node too', () => {
    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          runId={TEST_RUN_ID}
          element={{ ...executionElement, type: 'SUB_DAG' } as any}
          elementRuntimeInfo={{
            task: createTask({
              type: PipelineTaskTaskType.DAG,
              state: PipelineTaskTaskState.RUNNING,
              lifecycle_message: 'OOMKilled',
            }),
          }}
          namespace={TEST_NAMESPACE}
          sourceFinished={true}
        />
      </CommonTestWrapper>,
    );

    expect(screen.getByText(LIFECYCLE_ENDED_MESSAGE)).toBeInTheDocument();
    expect(screen.queryByText(LIFECYCLE_WARNING_MESSAGE)).not.toBeInTheDocument();
  });

  it('shows no lifecycle banner when the task has no lifecycle message', () => {
    renderTask(createTask({ state: PipelineTaskTaskState.RUNNING }));
    expect(screen.queryByText(LIFECYCLE_WARNING_MESSAGE)).not.toBeInTheDocument();
  });

  it('shows no lifecycle banner for a task that already succeeded', () => {
    renderTask(
      createTask({
        state: PipelineTaskTaskState.SUCCEEDED,
        lifecycle_message: 'ImagePullBackOff',
      }),
    );
    expect(screen.queryByText(LIFECYCLE_WARNING_MESSAGE)).not.toBeInTheDocument();
  });

  it('displays volume mounts in the task details tab', async () => {
    renderTask(createTask(), {
      pipelineJobString: V2_PVC_TEMPLATE_STRING,
      element: {
        ...executionElement,
        data: { label: 'producer' },
        id: 'task.producer',
      },
    });

    fireEvent.click(await screen.findByText('Task Details'));

    screen.getByText('/data');
    screen.getByText('createpvc');
  });

  it('does not reparse the pipeline job when task polling rerenders the details tab', async () => {
    const convertPipelineSpy = vi.spyOn(WorkflowUtils, 'convertYamlToV2PipelineSpec');
    const convertPlatformSpy = vi.spyOn(WorkflowUtils, 'convertYamlToPlatformSpec');
    const layers = ['root'];
    const producerElement = {
      ...executionElement,
      data: { label: 'producer' },
      id: 'task.producer',
    };
    const view = (task: V2PipelineTask) => (
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          element={producerElement}
          elementRuntimeInfo={{ task }}
          layers={layers}
          namespace={TEST_NAMESPACE}
          pipelineJobString={V2_PVC_TEMPLATE_STRING}
          runId={TEST_RUN_ID}
        />
      </CommonTestWrapper>
    );

    const { rerender } = render(view(createTask({ state: PipelineTaskTaskState.RUNNING })));
    fireEvent.click(await screen.findByText('Task Details'));
    const initialPipelineParses = convertPipelineSpy.mock.calls.length;
    const initialPlatformParses = convertPlatformSpy.mock.calls.length;
    expect(initialPipelineParses).toBeGreaterThan(0);
    expect(initialPlatformParses).toBeGreaterThan(0);

    rerender(view(createTask({ state: PipelineTaskTaskState.SUCCEEDED })));
    expect(convertPipelineSpy).toHaveBeenCalledTimes(initialPipelineParses);
    expect(convertPlatformSpy).toHaveBeenCalledTimes(initialPlatformParses);

    convertPipelineSpy.mockRestore();
    convertPlatformSpy.mockRestore();
  });

  it('formats task timestamps consistently with other details pages', () => {
    const createdAt = new Date('2026-08-11T12:00:00Z');
    const finishedAt = new Date('2026-08-11T12:05:00Z');

    expect(
      getTaskDetailsFields(
        executionElement,
        createTask({ create_time: createdAt, end_time: finishedAt }),
      ),
    ).toEqual(
      expect.arrayContaining([
        ['Created At', createdAt.toLocaleString()],
        ['Finished At', finishedAt.toLocaleString()],
      ]),
    );
  });

  it('shows native task identity, scope, pod roles, cache identity, and state history', () => {
    const updatedAt = new Date('2026-08-11T12:01:00Z');
    const fields = getTaskDetailsFields(
      executionElement,
      createTask({
        cache_fingerprint: 'cache-fingerprint',
        parent_task_id: 'parent-task',
        pods: [
          {
            name: 'executor-pod',
            type: PipelineTaskTaskPodType.EXECUTOR,
            uid: 'executor-uid',
          },
        ],
        scope_path: 'root.preprocess',
        state_history: [
          {
            error: { message: 'image pull delayed' },
            state: PipelineTaskTaskState.RUNNING,
            update_time: updatedAt,
          },
        ],
        type_attributes: { iteration_index: '2' },
      }),
    );

    expect(fields).toEqual(
      expect.arrayContaining([
        ['Task type', PipelineTaskTaskType.RUNTIME],
        ['Parent task ID', 'parent-task'],
        ['Scope path', 'root.preprocess'],
        ['Cache fingerprint', 'cache-fingerprint'],
        ['Type attributes', '{"iteration_index":"2"}'],
        ['State history', `Running · ${updatedAt.toLocaleString()} · image pull delayed`],
      ]),
    );
  });

  it('keeps pod identities behind a collapsed section in their recorded order', async () => {
    renderTask(
      createTask({
        pods: [
          { name: 'driver-first', uid: 'driver-uid', type: PipelineTaskTaskPodType.DRIVER },
          { name: 'executor-first', uid: 'first-uid', type: PipelineTaskTaskPodType.EXECUTOR },
          { name: 'executor-retry', uid: 'retry-uid', type: PipelineTaskTaskPodType.EXECUTOR },
        ],
      }),
    );
    fireEvent.click(await screen.findByText('Task Details'));
    const section = screen.getByText('Pods (3)').closest('details');
    expect(section).not.toHaveAttribute('open');
    expect(section).toHaveTextContent('Pod 1 · DRIVER');
    expect(section).toHaveTextContent('driver-first');
    expect(section).toHaveTextContent('Pod 3 · EXECUTOR');
    expect(section).toHaveTextContent('executor-retry');
    expect(section).toHaveTextContent('retry-uid');
    fireEvent.click(screen.getByText('Pods (3)'));
    expect(section).toHaveAttribute('open');
  });

  it('retains a failed retry error in history without displaying it as the successful task message', () => {
    const task = createTask({
      status_metadata: { message: 'failed to execute component: exit status 1' },
      state_history: [
        { state: PipelineTaskTaskState.FAILED, error: { message: 'attempt failed' } },
        { state: PipelineTaskTaskState.SUCCEEDED },
      ],
    });
    const fields = getTaskDetailsFields(executionElement, task);
    expect(fields.some(([name]) => name === 'Message')).toBe(false);
    expect(fields.find(([name]) => name === 'State history')?.[1]).toContain('attempt failed');
    expect(
      getTaskDetailsFields(executionElement, { ...task, state: PipelineTaskTaskState.FAILED }),
    ).toContainEqual(['Message', 'failed to execute component: exit status 1']);
  });

  it('renders each state transition in its own row with a separate timestamp column', async () => {
    const startedAt = new Date('2026-08-11T12:00:00Z');
    const failedAt = new Date('2026-08-11T12:01:00Z');
    const finishedAt = new Date('2026-08-11T12:02:00Z');
    renderTask(
      createTask({
        state_history: [
          { state: PipelineTaskTaskState.RUNNING, update_time: startedAt },
          {
            state: PipelineTaskTaskState.FAILED,
            update_time: failedAt,
            error: { message: 'first attempt failed' },
          },
          { state: PipelineTaskTaskState.SUCCEEDED, update_time: finishedAt },
        ],
      }),
    );
    fireEvent.click(await screen.findByText('Task Details'));

    const history = screen.getByRole('table', { name: 'State history' });
    const rows = within(history).getAllByRole('row');
    expect(rows).toHaveLength(4);
    expect(
      within(rows[1])
        .getAllByRole('cell')
        .map((cell) => cell.textContent),
    ).toEqual(['Running', startedAt.toLocaleString()]);
    expect(within(rows[2]).getAllByRole('cell')[0]).toHaveTextContent('first attempt failed');
    expect(
      within(rows[3])
        .getAllByRole('cell')
        .map((cell) => cell.textContent),
    ).toEqual(['Succeeded', finishedAt.toLocaleString()]);
  });

  it('formats artifact timestamps consistently with other details pages', () => {
    const createdAt = new Date('2026-08-11T12:00:00Z');
    const artifactElement = {
      data: { label: 'model' },
      id: 'artifact.preprocess.model',
      position: { x: 100, y: 100 },
      type: 'ARTIFACT',
    } as const;

    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          element={artifactElement}
          elementRuntimeInfo={{
            task: createTask(),
            artifactGroup: {
              artifact_key: 'model',
              artifacts: [{ name: 'model', created_at: createdAt }],
            },
          }}
          namespace={TEST_NAMESPACE}
        />
      </CommonTestWrapper>,
    );

    screen.getByText(createdAt.toLocaleString());
  });

  it('uses the native output key to restore legacy UI metadata visualizations', async () => {
    const loadSpy = vi.spyOn(OutputArtifactLoader, 'loadResult').mockResolvedValue({
      configs: [{ data: [['restored']], labels: ['value'], type: PlotType.TABLE }],
      errors: [],
    });
    const artifactElement = {
      data: { label: 'legacy-output' },
      id: 'artifact.preprocess.mlpipeline-ui-metadata',
      position: { x: 100, y: 100 },
      type: 'ARTIFACT',
    } as const;

    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          element={artifactElement}
          elementRuntimeInfo={{
            task: createTask(),
            artifactGroup: {
              artifact_key: 'mlpipeline-ui-metadata',
              artifacts: [
                {
                  artifact_id: 'legacy-metadata-1',
                  name: 'legacy-output',
                  uri: 's3://reports/metadata.json',
                },
              ],
            },
          }}
          namespace={TEST_NAMESPACE}
        />
      </CommonTestWrapper>,
    );

    fireEvent.click(screen.getByText('Visualization'));

    expect(await screen.findByText('restored')).toBeVisible();
    expect(loadSpy).toHaveBeenCalledTimes(1);
  });

  it('refreshes an artifact visualization when the producing task finishes', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('<h1>Report</h1>');
    const artifactElement = {
      data: { label: 'report' },
      id: 'artifact.preprocess.report',
      position: { x: 100, y: 100 },
      type: 'ARTIFACT',
    } as const;
    const view = (state: PipelineTaskTaskState) => (
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          element={artifactElement}
          elementRuntimeInfo={{
            task: createTask({ state }),
            artifactGroup: {
              artifact_key: 'report',
              artifacts: [
                {
                  artifact_id: 'live-report',
                  name: 'report',
                  type: ArtifactArtifactType.HTML,
                  uri: 's3://reports/output.html',
                },
              ],
            },
          }}
          namespace={TEST_NAMESPACE}
        />
      </CommonTestWrapper>
    );

    const { rerender } = render(view(PipelineTaskTaskState.RUNNING));
    fireEvent.click(screen.getByText('Visualization'));
    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(1));

    rerender(view(PipelineTaskTaskState.SUCCEEDED));
    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(2));
  });

  it('finalizes an artifact visualization when the parent run finishes first', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('<h1>Report</h1>');
    const artifactElement = {
      data: { label: 'report' },
      id: 'artifact.preprocess.report',
      position: { x: 100, y: 100 },
      type: 'ARTIFACT',
    } as const;
    const view = (sourceFinished: boolean) => (
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          element={artifactElement}
          elementRuntimeInfo={{
            artifactGroup: {
              artifact_key: 'report',
              artifacts: [
                {
                  artifact_id: 'live-report',
                  name: 'report',
                  type: ArtifactArtifactType.HTML,
                  uri: 's3://reports/output.html',
                },
              ],
            },
            task: createTask({ state: PipelineTaskTaskState.RUNNING }),
          }}
          layers={['root']}
          namespace={TEST_NAMESPACE}
          sourceFinished={sourceFinished}
        />
      </CommonTestWrapper>
    );

    const { rerender } = render(view(false));
    fireEvent.click(screen.getByText('Visualization'));
    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(1));

    rerender(view(true));
    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(2));
  });
});
