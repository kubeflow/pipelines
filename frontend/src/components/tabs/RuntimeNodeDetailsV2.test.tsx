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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { dump, loadAll } from 'js-yaml';
import {
  ArtifactArtifactType,
  PipelineTaskTaskPodType,
  PipelineTaskTaskState,
  PipelineTaskTaskType,
  V2beta1PipelineTask,
} from 'src/apisv2beta1/run';
import {
  getTaskDetailsFields,
  getLogsInfo,
  LOGS_BANNER_ADDITIONAL_INFO,
  LOGS_BANNER_MESSAGE,
  LOGS_DETAILS,
  RuntimeNodeDetailsV2,
} from 'src/components/tabs/RuntimeNodeDetailsV2';
import v2PvcYamlString from 'src/data/test/create_mount_delete_dynamic_pvc.yaml?raw';
import { Apis } from 'src/lib/Apis';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
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

  const executionElement = {
    data: { label: 'preprocess' },
    id: 'task.preprocess',
    position: { x: 100, y: 100 },
    type: 'EXECUTION',
  } as const;

  function createTask(overrides: Partial<V2beta1PipelineTask> = {}): V2beta1PipelineTask {
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

  function renderTask(task: V2beta1PipelineTask, extraProps: Record<string, unknown> = {}) {
    return render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          onLayerChange={() => {}}
          runId={TEST_RUN_ID}
          element={executionElement}
          elementRuntimeInfo={{ task }}
          namespace={TEST_NAMESPACE}
          {...extraProps}
        />
      </CommonTestWrapper>,
    );
  }

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
    screen.getByTestId(TEST_LOG_VIEW_ID);
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
      path: { bucket: 'pipeline-root', key: 'logs.txt', source: 's3' },
      namespace: TEST_NAMESPACE,
    });
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
        ['Pods', 'EXECUTOR · executor-pod · UID executor-uid'],
        ['State history', `Running · ${updatedAt.toLocaleString()} · image pull delayed`],
      ]),
    );
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
          onLayerChange={() => {}}
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
          onLayerChange={() => {}}
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
          onLayerChange={() => {}}
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
});
