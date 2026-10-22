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
  getTaskDriverLogsInfo,
  getDriverLogUri,
  getTaskDetailsFields,
  getLogsInfo,
  getTaskLogSources,
  LOGS_BANNER_ADDITIONAL_INFO,
  LOGS_BANNER_MESSAGE,
  LIFECYCLE_ENDED_MESSAGE,
  LIFECYCLE_FAILURE_MESSAGE,
  LIFECYCLE_WARNING_MESSAGE,
  LOGS_DETAILS,
  RuntimeNodeDetails,
} from 'src/components/tabs/RuntimeNodeDetails';
import v2PvcYamlString from 'src/data/test/create_mount_delete_dynamic_pvc.yaml?raw';
import { Apis } from 'src/lib/Apis';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper as BaseTestWrapper } from 'src/TestWrapper';
import type { PropsWithChildren } from 'react';
import { ThemeProvider } from '../shell/ThemeProvider';
import { PlotType } from 'src/components/viewers/Viewer';

function CommonTestWrapper({ children }: PropsWithChildren) {
  return (
    <BaseTestWrapper>
      <ThemeProvider>{children}</ThemeProvider>
    </BaseTestWrapper>
  );
}

const V2_PVC_TEMPLATE_STRING = dump({
  pipeline_spec: loadAll(v2PvcYamlString)[0],
  platform_spec: loadAll(v2PvcYamlString)[1],
});

testBestPractices();

describe('RuntimeNodeDetails', () => {
  const TEST_RUN_ID = 'test-run-id';
  const TEST_TASK_ID = 'test-task-id';
  const TEST_POD_NAME = 'test-pod-name';
  const TEST_NAMESPACE = 'kubeflow';
  const TEST_LOG_VIEW_ID = 'logs-view-window';

  beforeEach(() => {
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      })),
    );
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

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
        <RuntimeNodeDetails
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

  function openDriverLogs() {
    fireEvent.click(screen.getByRole('tab', { name: 'Logs', exact: true }));
    const toggle = screen.getByRole('button', { name: 'Driver', exact: true });
    if (toggle.getAttribute('aria-expanded') === 'false') fireEvent.click(toggle);
  }

  describe('Driver logs', () => {
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

    it('reports driver logs with the same keys as pod logs', async () => {
      const readFileSpy = vi
        .spyOn(Apis, 'readFile')
        .mockResolvedValueOnce('driver logs')
        .mockRejectedValueOnce(new Error('storage temporarily unavailable'));

      expect(await getDriverLogsInfo(driverLogUri, TEST_NAMESPACE)).toEqual(
        new Map([[LOGS_DETAILS, 'driver logs']]),
      );
      expect(await getDriverLogsInfo(driverLogUri, TEST_NAMESPACE)).toEqual(
        new Map([
          [LOGS_BANNER_MESSAGE, 'Failed to retrieve driver logs.'],
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
        fireEvent.click(screen.getByRole('tab', { name: 'Task Details', exact: true }));
        expect(readFileSpy).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole('tab', { name: 'Logs', exact: true }));
        expect(await screen.findByText('executor output')).toBeVisible();
        expect(readFileSpy).not.toHaveBeenCalled();

        openDriverLogs();
        expect(screen.getByText('Pipeline orchestration logs from the KFP driver.')).toBeVisible();

        expect(await screen.findByText('driver initialization')).toBeVisible();
        expect(screen.getByText('finished')).toBeVisible();
        expect(screen.getByText('executor output')).toBeVisible();
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

      openDriverLogs();

      expect(await screen.findByText('Driver logs are not available yet.')).toBeVisible();
      expect(screen.queryByTestId('driver-logs-view-window')).not.toBeInTheDocument();
      expect(readFileSpy).not.toHaveBeenCalled();
      expect(getPodLogsSpy).toHaveBeenCalledTimes(1);
    });

    it('shows loading until the driver log artifact arrives', async () => {
      let resolveLogs!: (value: string) => void;
      vi.spyOn(Apis, 'readFile').mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveLogs = resolve;
          }),
      );
      renderTask(createSystemLogsTask());
      openDriverLogs();
      expect(await screen.findByText('Loading driver logs…')).toBeVisible();
      await act(async () => resolveLogs('Creating task train-model in pod kubeflow'));
      expect(await screen.findByText('Creating task train-model in pod kubeflow')).toBeVisible();
      expect(screen.queryByText('Loading driver logs…')).not.toBeInTheDocument();
    });

    it.each(['', '  \n'])('shows an unavailable state for an empty artifact', async (contents) => {
      vi.spyOn(Apis, 'readFile').mockResolvedValue(contents);
      renderTask(createSystemLogsTask());
      openDriverLogs();
      expect(await screen.findByText('Driver logs are not available yet.')).toBeVisible();
      expect(screen.queryByTestId('driver-logs-view-window')).not.toBeInTheDocument();
    });

    it('shows artifact-fetch error details and recovers on Refresh without leaving Driver logs', async () => {
      const readFileSpy = vi
        .spyOn(Apis, 'readFile')
        .mockRejectedValueOnce(new Error('storage temporarily unavailable'))
        .mockResolvedValueOnce('recovered driver logs');
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      renderTask(createSystemLogsTask());

      openDriverLogs();

      expect(await screen.findByText('Failed to retrieve driver logs.')).toBeVisible();
      expect(screen.queryByTestId('driver-logs-view-window')).not.toBeInTheDocument();
      fireEvent.click(
        within(screen.getByRole('region', { name: 'Driver logs' })).getByRole('button', {
          name: 'Details',
          exact: true,
        }),
      );
      const dialog = await screen.findByRole('dialog');
      expect(
        within(dialog).getByText('Error response: storage temporarily unavailable'),
      ).toBeVisible();
      fireEvent.click(within(dialog).getByRole('button', { name: 'Dismiss' }));
      fireEvent.click(await screen.findByRole('button', { name: 'Refresh' }));

      expect(await screen.findByText('recovered driver logs')).toBeVisible();
      expect(screen.queryByText('Failed to retrieve driver logs.')).not.toBeInTheDocument();
      expect(readFileSpy).toHaveBeenCalledTimes(2);
      expect(consoleErrorSpy).not.toHaveBeenCalled();
    });

    it('refetches transient artifact errors while running and stops polling when the tab closes', async () => {
      vi.useFakeTimers();
      const readFileSpy = vi
        .spyOn(Apis, 'readFile')
        .mockRejectedValueOnce(new Error('artifact upload in progress'))
        .mockResolvedValue('live driver logs');
      renderTask(createSystemLogsTask(driverLogUri, { state: PipelineTaskTaskState.RUNNING }));

      await act(async () => vi.advanceTimersByTimeAsync(20_000));
      expect(readFileSpy).not.toHaveBeenCalled();
      openDriverLogs();
      await act(async () => vi.advanceTimersByTimeAsync(1));
      expect(screen.getByText('Failed to retrieve driver logs.')).toBeVisible();

      await act(async () => vi.advanceTimersByTimeAsync(10_000));
      expect(screen.getByText('live driver logs')).toBeVisible();
      expect(screen.queryByText('Failed to retrieve driver logs.')).not.toBeInTheDocument();
      expect(readFileSpy).toHaveBeenCalledTimes(2);

      fireEvent.click(screen.getByRole('tab', { name: 'Task Details', exact: true }));
      await act(async () => vi.advanceTimersByTimeAsync(20_000));
      expect(readFileSpy).toHaveBeenCalledTimes(2);
    });

    it.each(['task', 'URI', 'namespace'])(
      'does not retain stale driver logs when the %s changes',
      async (changedSource) => {
        let resolveNextLogs!: (logs: string) => void;
        const nextLogs = new Promise<string>((resolve) => {
          resolveNextLogs = resolve;
        });
        const readFileSpy = vi
          .spyOn(Apis, 'readFile')
          .mockResolvedValueOnce('previous driver logs')
          .mockReturnValueOnce(nextLogs);
        const view = renderTask(createSystemLogsTask());
        openDriverLogs();
        expect(await screen.findByText('previous driver logs')).toBeVisible();

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
        expect(screen.queryByText('previous driver logs')).not.toBeInTheDocument();
        expect(screen.queryByTestId('driver-logs-view-window')).not.toBeInTheDocument();
        await act(async () => resolveNextLogs('current driver logs'));
        expect(await screen.findByText('current driver logs')).toBeVisible();
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
        .mockResolvedValueOnce('current task driver logs');
      const view = renderTask(createSystemLogsTask());
      openDriverLogs();
      await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(1));

      view.rerender(taskView(createSystemLogsTask(driverLogUri, { task_id: 'second-task-id' })));
      expect(await screen.findByText('current task driver logs')).toBeVisible();
      await act(async () => resolvePreviousLogs('previous task driver logs'));

      expect(screen.getByText('current task driver logs')).toBeVisible();
      expect(screen.queryByText('previous task driver logs')).not.toBeInTheDocument();
    });

    it('retrieves a newly available URI on task refresh without resetting the selected tab', async () => {
      const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('new driver logs');
      const view = renderTask(createTask());
      openDriverLogs();
      expect(await screen.findByText('Driver logs are not available yet.')).toBeVisible();
      expect(readFileSpy).not.toHaveBeenCalled();

      view.rerender(taskView(createSystemLogsTask()));

      expect(await screen.findByText('new driver logs')).toBeVisible();
      expect(screen.queryByText('Driver logs are not available yet.')).not.toBeInTheDocument();
      expect(readFileSpy).toHaveBeenCalledTimes(1);
    });

    it('fetches final driver logs when the parent run finishes and stops polling', async () => {
      vi.useFakeTimers();
      const readFileSpy = vi
        .spyOn(Apis, 'readFile')
        .mockResolvedValueOnce('live driver logs')
        .mockResolvedValue('final driver logs');
      const task = createSystemLogsTask(driverLogUri, { state: PipelineTaskTaskState.RUNNING });
      const view = renderTask(task, { sourceFinished: false });
      openDriverLogs();
      await act(async () => vi.advanceTimersByTimeAsync(1));
      expect(screen.getByText('live driver logs')).toBeVisible();

      view.rerender(taskView(task, { sourceFinished: true }));
      await act(async () => vi.advanceTimersByTimeAsync(1));
      expect(screen.getByText('final driver logs')).toBeVisible();
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

    fireEvent.click(await screen.findByRole('tab', { name: 'Logs' }));

    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalled());
    await screen.findByText('Failed to retrieve pod logs.');
  });

  it('displays pod logs on the execution side panel', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('test-logs-details');
    renderTask(createTask());

    fireEvent.click(await screen.findByRole('tab', { name: 'Logs' }));

    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalled());
    await screen.findByTestId(TEST_LOG_VIEW_ID);
  });

  it('defers both pod and artifact fallback reads while the namespace is pending', async () => {
    const getPodLogsSpy = vi
      .spyOn(Apis, 'getPodLogs')
      .mockRejectedValue(new Error('Pod collected'));
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('artifact logs');
    const task = createTask({
      outputs: {
        artifacts: [
          {
            artifact_key: 'executor-logs',
            artifacts: [{ name: 'executor-logs', uri: 's3://pipeline-root/logs.txt' }],
          },
        ],
      },
    });
    const content = (namespacePending: boolean) => (
      <CommonTestWrapper>
        <RuntimeNodeDetails
          layers={['root']}
          onLayerChange={() => {}}
          runId={TEST_RUN_ID}
          element={executionElement}
          elementRuntimeInfo={{ task }}
          namespace={namespacePending ? undefined : TEST_NAMESPACE}
          namespacePending={namespacePending}
          selectedTaskTab={2}
        />
      </CommonTestWrapper>
    );
    const view = render(content(true));
    expect(await screen.findByText('Loading experiment namespace…')).toBeVisible();
    expect(getPodLogsSpy).not.toHaveBeenCalled();
    expect(readFileSpy).not.toHaveBeenCalled();
    view.rerender(content(false));
    expect(await screen.findByTestId(TEST_LOG_VIEW_ID)).toBeVisible();
    expect(getPodLogsSpy).toHaveBeenCalledExactlyOnceWith(
      TEST_RUN_ID,
      TEST_POD_NAME,
      TEST_NAMESPACE,
      '2026-08-11',
    );
    expect(readFileSpy).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ namespace: TEST_NAMESPACE }),
    );
    expect(screen.queryByText('Loading experiment namespace…')).not.toBeInTheDocument();
  });

  it.each(['EXECUTION', 'SUB_DAG'] as const)(
    'defers %s input/output artifact reads and downloads until namespace resolution',
    async (type) => {
      const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('scoped artifact preview');
      const task = createTask({
        outputs: {
          artifacts: [
            {
              artifact_key: 'dataset',
              artifacts: [
                { artifact_id: 'dataset-a', name: 'dataset', uri: 's3://pipeline-root/data.txt' },
              ],
            },
          ],
        },
      });
      const content = (namespacePending: boolean, refreshedTask = task) => (
        <CommonTestWrapper>
          <RuntimeNodeDetails
            layers={['root']}
            onLayerChange={() => {}}
            runId={TEST_RUN_ID}
            element={{ ...executionElement, type }}
            elementRuntimeInfo={{ task: refreshedTask }}
            namespace={namespacePending ? undefined : TEST_NAMESPACE}
            namespacePending={namespacePending}
            selectedTaskTab={0}
          />
        </CommonTestWrapper>
      );
      const view = render(content(true));
      await act(async () => {});
      expect(readFileSpy).not.toHaveBeenCalled();
      expect(screen.getByText('Loading experiment namespace…')).toBeVisible();
      expect(screen.getByRole('link', { name: 'dataset' })).toHaveAttribute(
        'href',
        '/artifacts/dataset-a',
      );
      expect(document.querySelector('a[download]')).not.toBeInTheDocument();
      view.rerender(content(false));
      expect(await screen.findByText('scoped artifact preview')).toBeVisible();
      expect(readFileSpy).toHaveBeenCalledExactlyOnceWith(
        expect.objectContaining({ namespace: TEST_NAMESPACE }),
      );
      expect(document.querySelector('a[download]')).toHaveAttribute(
        'href',
        expect.stringContaining('namespace=kubeflow'),
      );
      expect(screen.queryByText('Loading experiment namespace…')).not.toBeInTheDocument();
      view.rerender(content(false, { ...task }));
      await act(async () => {});
      expect(screen.getByText('scoped artifact preview')).toBeVisible();
      expect(readFileSpy).toHaveBeenCalledTimes(1);
    },
  );

  it('retrieves pod logs without an experiment namespace', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('test-logs-details');

    const logsInfo = await getLogsInfo(createTask(), TEST_RUN_ID);

    expect(logsInfo.get(LOGS_DETAILS)).toBe('test-logs-details');
    expect(getPodLogsSpy).toHaveBeenCalledWith(TEST_RUN_ID, TEST_POD_NAME, '', '2026-08-11');
  });

  it('collapses component logs independently and preserves that choice on refresh', async () => {
    vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('task output');
    const task = createTask();
    const view = renderTask(task);
    fireEvent.click(screen.getByRole('tab', { name: 'Logs', exact: true }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Copy component logs' })).toBeEnabled(),
    );
    const toggle = screen.getByRole('button', { name: 'Component', exact: true });
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    fireEvent.click(toggle);
    expect(screen.queryByTestId(TEST_LOG_VIEW_ID)).not.toBeInTheDocument();
    view.rerender(taskView({ ...task }));
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    fireEvent.click(toggle);
    expect(screen.getByTestId(TEST_LOG_VIEW_ID)).toBeInTheDocument();
    view.rerender(taskView(createTask({ task_id: 'another-task' })));
    expect(screen.getByRole('button', { name: 'Component', exact: true })).toHaveAttribute(
      'aria-expanded',
      'true',
    );
  });

  function retryTask() {
    return createTask({
      pods: [
        { name: 'earlier-pod', type: PipelineTaskTaskPodType.EXECUTOR },
        { name: TEST_POD_NAME, type: PipelineTaskTaskPodType.EXECUTOR },
      ],
      outputs: {
        artifacts: [
          {
            artifact_key: 'executor-logs',
            artifacts: [
              { artifact_id: 'second', uri: 's3://pipeline-root/executor-logs-1' },
              { artifact_id: 'first', uri: 's3://pipeline-root/executor-logs-0' },
            ],
          },
        ],
      },
    });
  }

  it.each([false, true])(
    'cancels pending archived copy when the task changes (ClipboardItem: %s)',
    async (modernClipboard) => {
      let resolveLogs!: (value: string) => void;
      const pending = new Promise<string>((resolve) => {
        resolveLogs = resolve;
      });
      const writeText = vi.fn().mockResolvedValue(undefined);
      const consume = vi.fn();
      const write = vi.fn(async (items: Array<{ content: Record<string, Promise<Blob>> }>) => {
        consume(await items[0].content['text/plain']);
      });
      vi.stubGlobal(
        'ClipboardItem',
        modernClipboard
          ? class {
              constructor(public content: Record<string, Promise<Blob>>) {}
            }
          : undefined,
      );
      vi.stubGlobal('navigator', {
        ...navigator,
        clipboard: { writeText, ...(modernClipboard ? { write } : {}) },
      });
      vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('latest output');
      vi.spyOn(Apis, 'readFile').mockImplementation((request) =>
        request.peek ? Promise.resolve('preview') : pending,
      );
      const view = renderTask({ ...retryTask(), task_id: `pending-copy-${modernClipboard}` });
      fireEvent.click(screen.getByRole('tab', { name: 'Logs', exact: true }));
      fireEvent.click(screen.getByRole('button', { name: 'Copy component · attempt 1 logs' }));
      if (modernClipboard) expect(write).toHaveBeenCalledTimes(1);
      view.rerender(taskView(createTask({ task_id: 'different-task' })));
      await act(async () => {
        resolveLogs('old attempt output');
        await pending;
      });
      expect(writeText).not.toHaveBeenCalled();
      expect(consume).not.toHaveBeenCalled();
    },
  );

  it('starts ClipboardItem write during the click and supplies an archived log as a Blob', async () => {
    let resolveLogs!: (value: string) => void;
    const pending = new Promise<string>((resolve) => {
      resolveLogs = resolve;
    });
    const consume = vi.fn();
    const write = vi.fn(async (items: Array<{ content: Record<string, Promise<Blob>> }>) => {
      consume(await items[0].content['text/plain']);
    });
    vi.stubGlobal(
      'ClipboardItem',
      class {
        constructor(public content: Record<string, Promise<Blob>>) {}
      },
    );
    vi.stubGlobal('navigator', { ...navigator, clipboard: { write } });
    vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('latest output');
    vi.spyOn(Apis, 'readFile').mockImplementation((request) =>
      request.peek ? Promise.resolve('preview') : pending,
    );
    renderTask({ ...retryTask(), task_id: 'activation-copy' });
    fireEvent.click(screen.getByRole('tab', { name: 'Logs', exact: true }));
    fireEvent.click(screen.getByRole('button', { name: 'Copy component · attempt 1 logs' }));
    expect(write).toHaveBeenCalledTimes(1);
    expect(consume).not.toHaveBeenCalled();
    await act(async () => {
      resolveLogs('selected attempt output');
      await pending;
    });
    await waitFor(() => expect(consume).toHaveBeenCalledWith(expect.any(Blob)));
    const blob = consume.mock.calls[0][0] as Blob;
    expect(blob.type).toBe('text/plain');
    expect(blob.size).toBe('selected attempt output'.length);
  });

  it('loads and copies archived component logs while their section stays collapsed', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } });
    vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('latest output');
    const readFile = vi.spyOn(Apis, 'readFile').mockResolvedValue('first attempt output');
    renderTask({ ...retryTask(), task_id: 'collapsed-copy-task' });
    fireEvent.click(screen.getByRole('tab', { name: 'Logs', exact: true }));
    expect(readFile.mock.calls.filter(([request]) => !request.peek)).toHaveLength(0);
    readFile.mockClear();
    fireEvent.click(screen.getByRole('button', { name: 'Copy component · attempt 1 logs' }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith('first attempt output'));
    expect(JSON.stringify(readFile.mock.calls)).toContain('executor-logs-0');
    expect(
      screen.getByRole('button', { name: 'Component · Attempt 1', exact: true }),
    ).toHaveAttribute('aria-expanded', 'false');
  });

  it('opens latest by default, lazily expands independent history, and preserves expansion on refresh', async () => {
    vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('latest pod output');
    const archiveRead = vi.spyOn(Apis, 'readFile').mockResolvedValue('first attempt failed');
    const task = { ...retryTask(), state: PipelineTaskTaskState.FAILED };
    const view = renderTask(task);
    fireEvent.click(screen.getByRole('tab', { name: 'Logs', exact: true }));
    const latest = screen.getByRole('button', {
      name: 'Component · Attempt 2 · Latest',
      exact: true,
    });
    const older = screen.getByRole('button', { name: 'Component · Attempt 1', exact: true });
    expect(latest).toHaveAttribute('aria-expanded', 'true');
    expect(older).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Pod · earlier-pod/ })).not.toBeInTheDocument();
    expect(archiveRead.mock.calls.filter(([request]) => !request.peek)).toHaveLength(0);
    archiveRead.mockClear();
    fireEvent.click(older);
    await waitFor(() => expect(screen.getAllByTestId(TEST_LOG_VIEW_ID)).toHaveLength(2));
    expect(latest).toHaveAttribute('aria-expanded', 'true');
    expect(JSON.stringify(archiveRead.mock.calls)).toContain('executor-logs-0');
    expect(JSON.stringify(archiveRead.mock.calls)).not.toContain('executor-logs-1');
    view.rerender(taskView({ ...task }));
    expect(older).toHaveAttribute('aria-expanded', 'true');
    fireEvent.click(latest);
    expect(screen.getAllByTestId(TEST_LOG_VIEW_ID)).toHaveLength(1);
    expect(older).toHaveAttribute('aria-expanded', 'true');
  });

  it('retains all archived attempts while a newer live pod runs', () => {
    vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('live attempt');
    renderTask({ ...retryTask(), state: PipelineTaskTaskState.RUNNING });
    fireEvent.click(screen.getByRole('tab', { name: 'Logs', exact: true }));
    expect(screen.getByRole('button', { name: 'Component · Latest', exact: true })).toHaveAttribute(
      'aria-expanded',
      'true',
    );
    expect(
      screen.getByRole('button', { name: 'Component · Attempt 2', exact: true }),
    ).toHaveAttribute('aria-expanded', 'false');
    expect(
      screen.getByRole('button', { name: 'Component · Attempt 1', exact: true }),
    ).toHaveAttribute('aria-expanded', 'false');
  });

  it('reports a selected archive failure without falling back to a different attempt', async () => {
    const task = retryTask();
    const podRead = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('wrong latest output');
    vi.spyOn(Apis, 'readFile').mockRejectedValue(new Error('Archived object removed'));
    const selected = getTaskLogSources(task).find((source) => source.key === 'artifact:first')!;
    const result = await getLogsInfo(task, TEST_RUN_ID, TEST_NAMESPACE, selected);
    expect(result.get(LOGS_DETAILS)).toBeUndefined();
    expect(result.get(LOGS_BANNER_ADDITIONAL_INFO)).toContain('Archived object removed');
    expect(podRead).not.toHaveBeenCalled();
  });

  it('deduplicates archives by URI and does not invent retry indices for named pods or driver archives', () => {
    const task = retryTask();
    task.outputs!.artifacts!.push(
      {
        artifact_key: 'executor-logs',
        artifacts: [{ artifact_id: 'duplicate', uri: 's3://pipeline-root/executor-logs-0' }],
      },
      {
        artifact_key: 'driver-logs',
        artifacts: [
          {
            artifact_id: 'driver-history',
            uri: 's3://pipeline-root/driver-logs',
            created_at: new Date('2026-08-11T12:00:00Z'),
          },
        ],
      },
    );
    expect(getTaskLogSources(task)).toHaveLength(3);
    const drivers = getTaskLogSources(task, PipelineTaskTaskPodType.DRIVER);
    expect(drivers).toHaveLength(1);
    expect(drivers[0].label).not.toContain('Attempt');
  });

  it('uses an available driver archive when status metadata has no current URI', async () => {
    const task = createTask({
      outputs: {
        artifacts: [
          {
            artifact_key: 'driver-logs',
            artifacts: [
              {
                artifact_id: 'older',
                uri: 's3://pipeline-root/older-driver',
                created_at: new Date('2026-08-10'),
              },
              {
                artifact_id: 'newer',
                uri: 's3://pipeline-root/newer-driver',
                created_at: new Date('2026-08-11'),
              },
            ],
          },
        ],
      },
    });
    const readFile = vi.spyOn(Apis, 'readFile').mockResolvedValue('available driver output');
    const result = await getTaskDriverLogsInfo(task, TEST_RUN_ID, TEST_NAMESPACE);
    expect(result.get(LOGS_DETAILS)).toBe('available driver output');
    expect(JSON.stringify(readFile.mock.calls)).toContain('newer-driver');
    expect(JSON.stringify(readFile.mock.calls)).not.toContain('older-driver');
  });

  it('reads selected archived driver logs independently of the current driver URI', async () => {
    const task = createTask({
      status_metadata: {
        custom_properties: {
          driver_logs_uri: 's3://pipeline-root/current-driver' as unknown as object,
        },
      },
      outputs: {
        artifacts: [
          {
            artifact_key: 'driver-logs',
            artifacts: [{ artifact_id: 'driver-history', uri: 's3://pipeline-root/older-driver' }],
          },
        ],
      },
    });
    const readFile = vi.spyOn(Apis, 'readFile').mockResolvedValue('older driver output');
    const selected = getTaskLogSources(task, PipelineTaskTaskPodType.DRIVER)[0];
    const result = await getTaskDriverLogsInfo(task, TEST_RUN_ID, TEST_NAMESPACE, selected);
    expect(result.get(LOGS_DETAILS)).toBe('older driver output');
    expect(JSON.stringify(readFile.mock.calls)).toContain('older-driver');
    expect(JSON.stringify(readFile.mock.calls)).not.toContain('current-driver');
  });

  it('copies each complete raw log source and reports clipboard failures', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } });
    const taskText = 'task first line\nsecond line\n';
    const driverText = 'driver first line\nsetup complete\n';
    vi.spyOn(Apis, 'getPodLogs').mockResolvedValue(taskText);
    vi.spyOn(Apis, 'readFile').mockResolvedValue(driverText);
    renderTask(
      createTask({
        status_metadata: {
          custom_properties: {
            driver_logs_uri: 's3://pipeline-root/driver-logs.txt' as unknown as object,
          },
        },
      }),
    );
    fireEvent.click(screen.getByRole('tab', { name: 'Logs', exact: true }));
    expect(screen.getByRole('button', { name: 'Copy driver logs' })).toBeEnabled();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Copy component logs' })).toBeEnabled(),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Copy component logs' }));
    await waitFor(() => expect(writeText).toHaveBeenLastCalledWith(taskText));
    expect(await screen.findByText('Component logs copied.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Driver', exact: true })).toHaveAttribute(
      'aria-expanded',
      'false',
    );
    fireEvent.click(screen.getByRole('button', { name: 'Copy driver logs' }));
    await waitFor(() => expect(writeText).toHaveBeenLastCalledWith(driverText));
    expect(await screen.findByText('Driver logs copied.')).toBeInTheDocument();
    writeText.mockRejectedValueOnce(new Error('Clipboard denied'));
    fireEvent.click(screen.getByRole('button', { name: 'Copy component logs' }));
    expect(
      await screen.findByText('Could not copy. Select the log text and copy it manually.'),
    ).toBeInTheDocument();
  });

  it('keeps legacy driver pod output out of component logs and retrieves it as driver logs', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('driver diagnostics');
    const task = createTask({
      pods: [{ name: 'driver-pod', type: PipelineTaskTaskPodType.DRIVER }],
    });
    const taskLogs = await getLogsInfo(task, TEST_RUN_ID, TEST_NAMESPACE);
    expect(taskLogs.get(LOGS_BANNER_MESSAGE)).toBe('Component logs are not available yet.');
    expect(getPodLogsSpy).not.toHaveBeenCalled();
    const driverLogs = await getTaskDriverLogsInfo(task, TEST_RUN_ID, TEST_NAMESPACE);
    expect(driverLogs.get(LOGS_DETAILS)).toBe('driver diagnostics');
    expect(getPodLogsSpy).toHaveBeenCalledExactlyOnceWith(
      TEST_RUN_ID,
      'driver-pod',
      TEST_NAMESPACE,
      '2026-08-11',
    );
  });

  it('automatically expands driver diagnostics without duplicating them as component logs', async () => {
    vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('driver diagnostics');
    const task = createTask({
      pods: [{ name: 'driver-pod', type: PipelineTaskTaskPodType.DRIVER }],
    });
    const view = renderTask(task);
    fireEvent.click(screen.getByRole('tab', { name: 'Logs' }));
    expect(screen.getByRole('button', { name: 'Driver' })).toHaveAttribute('aria-expanded', 'true');
    await screen.findByText('Component logs are not available yet.');
    expect(screen.getByTestId('driver-logs-view-window')).toBeInTheDocument();
    expect(screen.queryByTestId(TEST_LOG_VIEW_ID)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Driver' }));
    view.rerender(taskView({ ...task }));
    expect(screen.getByRole('button', { name: 'Driver' })).toHaveAttribute(
      'aria-expanded',
      'false',
    );
  });

  it('falls back to the legacy driver pod when the driver artifact is unavailable', async () => {
    vi.spyOn(Apis, 'readFile').mockRejectedValue(new Error('artifact missing'));
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('legacy driver output');
    const task = createTask({
      status_metadata: {
        custom_properties: { driver_logs_uri: 's3://root/driver.log' as unknown as object },
      },
      pods: [
        { name: 'driver-old', type: PipelineTaskTaskPodType.DRIVER },
        { name: 'driver-new', type: PipelineTaskTaskPodType.DRIVER },
      ],
    });
    expect((await getTaskDriverLogsInfo(task, TEST_RUN_ID, TEST_NAMESPACE)).get(LOGS_DETAILS)).toBe(
      'legacy driver output',
    );
    expect(getPodLogsSpy).toHaveBeenCalledExactlyOnceWith(
      TEST_RUN_ID,
      'driver-new',
      TEST_NAMESPACE,
      '2026-08-11',
    );
  });

  it('auto-expands driver logs after executor retrieval fails, preserving a manual collapse on refresh', async () => {
    vi.spyOn(Apis, 'getPodLogs').mockRejectedValue(new Error('executor unavailable'));
    vi.spyOn(Apis, 'readFile').mockResolvedValue('driver setup diagnostics');
    const task = createTask({
      status_metadata: {
        custom_properties: { driver_logs_uri: 's3://root/driver.log' as unknown as object },
      },
    });
    const view = renderTask(task);
    fireEvent.click(screen.getByRole('tab', { name: 'Logs' }));
    await screen.findByText('Failed to retrieve pod logs.');
    expect(screen.getByRole('button', { name: 'Driver' })).toHaveAttribute('aria-expanded', 'true');
    await screen.findByTestId('driver-logs-view-window');
    fireEvent.click(screen.getByRole('button', { name: 'Driver' }));
    view.rerender(taskView({ ...task, state: PipelineTaskTaskState.RUNNING }));
    expect(screen.getByRole('button', { name: 'Driver' })).toHaveAttribute(
      'aria-expanded',
      'false',
    );
  });

  it('normalizes a legacy driver-tab selection to the combined Logs tab', async () => {
    vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('executor output');
    renderTask(createTask(), { selectedTaskTab: 3 });
    expect(screen.getByRole('tab', { name: 'Logs' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.queryByRole('tab', { name: 'Driver logs' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Driver' })).toHaveAttribute('aria-expanded', 'true');
    expect(await screen.findByTestId(TEST_LOG_VIEW_ID)).toBeInTheDocument();
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

  it('does not show previous attempt output while a newly discovered pod is loading', async () => {
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

    fireEvent.click(await screen.findByRole('tab', { name: 'Logs' }));
    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalledTimes(1));
    expect(await screen.findByTestId(TEST_LOG_VIEW_ID)).toBeVisible();

    view.rerender(
      <CommonTestWrapper>
        <RuntimeNodeDetails
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
    expect(screen.queryByTestId(TEST_LOG_VIEW_ID)).not.toBeInTheDocument();
    await act(async () => resolveRetryLogs?.('retry logs'));
  });

  it('does not show placeholder logs from a previously selected task', async () => {
    let resolveSecondTaskLogs: ((logs: string) => void) | undefined;
    const secondTaskLogs = new Promise<string>((resolve) => {
      resolveSecondTaskLogs = resolve;
    });
    const getPodLogsSpy = vi
      .spyOn(Apis, 'getPodLogs')
      .mockResolvedValueOnce('first component logs')
      .mockReturnValueOnce(secondTaskLogs);
    const view = renderTask(createTask());

    fireEvent.click(await screen.findByRole('tab', { name: 'Logs' }));
    await waitFor(() => expect(getPodLogsSpy).toHaveBeenCalledTimes(1));
    expect(await screen.findByTestId(TEST_LOG_VIEW_ID)).toBeVisible();

    view.rerender(
      <CommonTestWrapper>
        <RuntimeNodeDetails
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

    await act(async () => resolveSecondTaskLogs?.('second component logs'));
    expect(await screen.findByTestId(TEST_LOG_VIEW_ID)).toBeVisible();
  });

  it('stops log polling when the parent run is terminal', async () => {
    vi.useFakeTimers();
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('final output');
    renderTask(createTask({ state: PipelineTaskTaskState.RUNNING }), { sourceFinished: true });

    fireEvent.click(screen.getByRole('tab', { name: 'Logs' }));
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

    fireEvent.click(await screen.findByRole('tab', { name: 'Logs' }));

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

    expect(logsInfo.get(LOGS_BANNER_MESSAGE)).toBe('Failed to retrieve component logs.');
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
        <RuntimeNodeDetails
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
        <RuntimeNodeDetails
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
        <RuntimeNodeDetails
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

  it.each(['Artifact Info', 'Visualization'] as const)(
    'defers artifact-node %s storage access until namespace resolution without resetting its tab',
    async (tab) => {
      const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('<h1>Scoped report</h1>');
      const task = createTask();
      const artifactGroup = {
        artifact_key: 'report',
        artifacts: [
          {
            artifact_id: 'report-a',
            name: 'report',
            type: ArtifactArtifactType.HTML,
            uri: 's3://reports/output.html',
          },
        ],
      };
      const content = (namespacePending: boolean, refreshedTask = task) => (
        <CommonTestWrapper>
          <RuntimeNodeDetails
            layers={['root']}
            onLayerChange={() => {}}
            element={{
              data: { label: 'report' },
              id: 'artifact.preprocess.report',
              position: { x: 100, y: 100 },
              type: 'ARTIFACT',
            }}
            elementRuntimeInfo={{ task: refreshedTask, artifactGroup }}
            namespace={namespacePending ? undefined : TEST_NAMESPACE}
            namespacePending={namespacePending}
          />
        </CommonTestWrapper>
      );
      const view = render(content(true));
      expect(
        within(screen.getByRole('region', { name: 'Artifact Info' })).getByText('report'),
      ).toBeVisible();
      if (tab === 'Visualization') fireEvent.click(screen.getByRole('tab', { name: tab }));
      await act(async () => {});
      expect(readFileSpy).not.toHaveBeenCalled();
      expect(document.querySelector('a[download]')).not.toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: 'Preview file contents' }),
      ).not.toBeInTheDocument();
      expect(
        screen
          .getAllByText('Loading experiment namespace…')
          .some((element) => !element.closest('[hidden]')),
      ).toBe(true);
      view.rerender(content(false));
      expect(screen.getByRole('tab', { name: tab })).toHaveAttribute('aria-selected', 'true');
      if (tab === 'Artifact Info')
        fireEvent.click(screen.getByRole('button', { name: 'Preview file contents' }));
      await waitFor(() =>
        expect(readFileSpy).toHaveBeenCalledExactlyOnceWith(
          expect.objectContaining({ namespace: TEST_NAMESPACE }),
        ),
      );
      expect(document.querySelector('a[download]')).toHaveAttribute(
        'href',
        expect.stringContaining('namespace=kubeflow'),
      );
      view.rerender(content(false, { ...task }));
      await act(async () => {});
      expect(screen.getByRole('tab', { name: tab })).toHaveAttribute('aria-selected', 'true');
      expect(readFileSpy).toHaveBeenCalledTimes(1);
      if (tab === 'Artifact Info') expect(screen.getByText('<h1>Scoped report</h1>')).toBeVisible();
    },
  );

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
        <RuntimeNodeDetails
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
        <RuntimeNodeDetails
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
        <RuntimeNodeDetails
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
        <RuntimeNodeDetails
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
