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
    screen.getByTestId(TEST_LOG_VIEW_ID);
  });

  it('retrieves pod logs without an experiment namespace', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs').mockResolvedValue('test-logs-details');

    const logsInfo = await getLogsInfo(createTask(), TEST_RUN_ID);

    expect(logsInfo.get(LOGS_DETAILS)).toBe('test-logs-details');
    expect(getPodLogsSpy).toHaveBeenCalledWith(TEST_RUN_ID, TEST_POD_NAME, '', '2026-08-11');
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
    const loadSpy = vi
      .spyOn(OutputArtifactLoader, 'load')
      .mockResolvedValue([{ data: [['restored']], labels: ['value'], type: PlotType.TABLE }]);
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
});
