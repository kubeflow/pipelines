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
import { Apis } from 'src/lib/Apis';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import { getNodeRuntimeInfo } from 'src/lib/v2/DynamicFlow';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { PlotType } from 'src/components/viewers/Viewer';
import datasetPreview from '../../../mock-backend/data/v2/artifact-previews/dataset.json';

const V2_PVC_YAML_STRING = v2PvcYamlString;
// The templateStr used in RuntimeNodeDetailsV2 is not directly from yaml file.
// Instead, it is from BE (already been processed).
const V2_PVC_TEMPLATE_STRING_OBJ = {
  pipeline_spec: loadAll(V2_PVC_YAML_STRING)[0],
  platform_spec: loadAll(V2_PVC_YAML_STRING)[1],
};
const V2_PVC_TEMPLATE_STRING = dump(V2_PVC_TEMPLATE_STRING_OBJ);

testBestPractices();

describe('RuntimeNodeDetailsV2', () => {
  const TEST_RUN_ID = 'test-run-id';
  const TEST_EXECUTION = new Execution();
  const TEST_EXECUTION_NAME = 'test-execution-name';
  const TEST_EXECUTION_ID = 123;
  const TEST_POD_NAME = 'test-pod-name';
  const TEST_NAMESPACE = 'kubeflow';
  const TSET_MLMD_INFO: NodeMlmdInfo = {
    execution: TEST_EXECUTION,
  };
  const TEST_LOG_VIEW_ID = 'logs-view-window';

  beforeEach(() => {
    vi.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([]);
    vi.spyOn(mlmdUtils, 'getLinkedArtifactsByExecution').mockResolvedValue([]);
    TEST_EXECUTION.setId(TEST_EXECUTION_ID);
    TEST_EXECUTION.getCustomPropertiesMap().set(
      'task_name',
      new Value().setStringValue(TEST_EXECUTION_NAME),
    );
    TEST_EXECUTION.getCustomPropertiesMap().set(
      'pod_name',
      new Value().setStringValue(TEST_POD_NAME),
    );
    TEST_EXECUTION.getCustomPropertiesMap().set(
      'namespace',
      new Value().setStringValue(TEST_NAMESPACE),
    );
  });

  it('shows error when failing to get logs details', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs');
    TestUtils.makeErrorResponseOnce(getPodLogsSpy, 'Failed to retrieve pod logs');
    // Also mock MLMD artifact retrieval to fail since it's used as fallback
    const getLinkedArtifactsSpy = vi.spyOn(mlmdUtils, 'getLinkedArtifactsByExecution');
    getLinkedArtifactsSpy.mockRejectedValueOnce(new Error('MLMD unavailable'));
    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          runId={TEST_RUN_ID}
          element={{
            data: {
              label: 'preprocess',
            },
            id: 'task.preprocess',
            position: { x: 100, y: 100 },
            type: 'EXECUTION',
          }}
          elementMlmdInfo={TSET_MLMD_INFO}
          namespace={undefined}
        ></RuntimeNodeDetailsV2>
      </CommonTestWrapper>,
    );

    const logsTab = await screen.findByText('Logs');
    fireEvent.click(logsTab); // Switch logs tab

    await waitFor(() => {
      expect(getPodLogsSpy).toHaveBeenCalled();
    });

    await waitFor(() => {
      screen.getByText('Failed to retrieve pod logs.');
    });
  });

  it('displays logs details on side panel of execution node', async () => {
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs');
    getPodLogsSpy.mockImplementation(() => 'test-logs-details');
    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          runId={TEST_RUN_ID}
          element={{
            data: {
              label: 'preprocess',
            },
            id: 'task.preprocess',
            position: { x: 100, y: 100 },
            type: 'EXECUTION',
          }}
          elementMlmdInfo={TSET_MLMD_INFO}
          namespace={undefined}
        ></RuntimeNodeDetailsV2>
      </CommonTestWrapper>,
    );

    const logsTab = await screen.findByText('Logs');
    fireEvent.click(logsTab); // Switch logs tab

    await waitFor(() => {
      expect(getPodLogsSpy).toHaveBeenCalled();
    });

    screen.getByTestId(TEST_LOG_VIEW_ID);
  });

  it('shows cached text if the execution is cached', async () => {
    TEST_EXECUTION.getCustomPropertiesMap().set(
      'cached_execution_id',
      new Value().setStringValue('135'),
    );
    const getPodLogsSpy = vi.spyOn(Apis, 'getPodLogs');
    getPodLogsSpy.mockImplementation(() => 'test-logs-details');

    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          runId={TEST_RUN_ID}
          element={{
            data: {
              label: 'preprocess',
            },
            id: 'task.preprocess',
            position: { x: 100, y: 100 },
            type: 'EXECUTION',
          }}
          elementMlmdInfo={TSET_MLMD_INFO}
          namespace={undefined}
        ></RuntimeNodeDetailsV2>
      </CommonTestWrapper>,
    );

    const logsTab = await screen.findByText('Logs');
    fireEvent.click(logsTab); // Switch logs tab

    await waitFor(() => {
      // getPodLogs() won't be called if it's cached execution
      expect(getPodLogsSpy).toHaveBeenCalledTimes(0);
    });

    screen.getByTestId(TEST_LOG_VIEW_ID); // Still can load log view window
  });

  it('displays volume mounts in details tab on side panel of execution node', async () => {
    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          pipelineJobString={V2_PVC_TEMPLATE_STRING}
          runId={TEST_RUN_ID}
          element={{
            data: {
              label: 'producer',
            },
            id: 'task.producer',
            position: { x: 100, y: 100 },
            type: 'EXECUTION',
          }}
          elementMlmdInfo={TSET_MLMD_INFO}
          namespace={undefined}
        ></RuntimeNodeDetailsV2>
      </CommonTestWrapper>,
    );

    const detailsTab = await screen.findByText('Task Details');
    fireEvent.click(detailsTab); // Switch details tab

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

  it('lazy-loads a Parquet Dataset preview only in its Visualization tab', async () => {
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockResolvedValue(JSON.stringify(datasetPreview));
    const rawRead = vi.spyOn(Apis, 'readFile');
    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          element={{
            data: { label: 'dataset' },
            id: 'artifact.preprocess.dataset',
            position: { x: 100, y: 100 },
            type: 'ARTIFACT',
          }}
          elementRuntimeInfo={{
            task: createTask(),
            artifactGroup: {
              artifact_key: 'dataset',
              artifacts: [
                {
                  artifact_id: 'dataset-1',
                  type: ArtifactArtifactType.Dataset,
                  uri: 's3://bucket/dataset',
                  metadata: { _kfp_preview: { version: 1, path: 'preview.json' } },
                },
              ],
            },
          }}
          namespace={TEST_NAMESPACE}
        />
      </CommonTestWrapper>,
    );
    expect(read).not.toHaveBeenCalled();
    fireEvent.click(screen.getByText('Visualization'));
    expect(await screen.findByRole('table', { name: 'Dataset sample' })).toBeVisible();
    expect(screen.queryByText('There is no metrics artifact available in this step.')).toBeNull();
    expect(read.mock.calls.filter(([, , signal]) => !signal?.aborted)).toEqual([
      ['dataset-1', TEST_NAMESPACE, expect.objectContaining({ aborted: false })],
    ]);
    expect(rawRead).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Artifact Info' }));
    expect(screen.queryByRole('table', { name: 'Dataset sample' })).toBeNull();
  });

  it('offers every collected member without fetching a preview before selection', async () => {
    const read = vi
      .spyOn(Apis, 'getArtifactContentPreview')
      .mockResolvedValue(JSON.stringify(datasetPreview));
    const root: V2PipelineTask = {
      task_id: 'root-task',
      name: 'root',
      type: PipelineTaskTaskType.ROOT,
    };
    const task = createTask({
      parent_task_id: root.task_id,
      outputs: {
        artifacts: [1, 2].map((index) => ({
          artifact_key: 'datasets',
          artifacts: [
            {
              artifact_id: `dataset-${index}`,
              name: 'dataset',
              type: ArtifactArtifactType.Dataset,
              uri: `s3://bucket/dataset-${index}`,
              metadata: { _kfp_preview: { version: 1, path: 'preview.json' } },
            },
          ],
        })),
      },
    });
    const element = {
      data: { label: 'datasets' },
      id: 'artifact.preprocess.datasets',
      position: { x: 100, y: 100 },
      type: 'ARTIFACT',
    } as const;
    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          element={element}
          elementRuntimeInfo={getNodeRuntimeInfo(element, [root, task], ['root'])}
          namespace={TEST_NAMESPACE}
        />
      </CommonTestWrapper>,
    );
    expect(screen.getByText('Artifact Count')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: 'Visualization' }));
    expect(read).not.toHaveBeenCalled();
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Artifact preview' }));
    fireEvent.click(screen.getByRole('option', { name: 'dataset (2)', exact: true }));
    await screen.findByRole('table', { name: 'Dataset sample' });
    expect(read.mock.calls.filter(([, , signal]) => !signal?.aborted)).toEqual([
      ['dataset-2', TEST_NAMESPACE, expect.objectContaining({ aborted: false })],
    ]);
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
