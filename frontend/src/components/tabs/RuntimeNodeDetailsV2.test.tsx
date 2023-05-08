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
import React from 'react';
import { Apis } from 'src/lib/Apis';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { NodeMlmdInfo } from 'src/pages/RunDetailsV2';
import { RuntimeNodeDetailsV2 } from 'src/components/tabs/RuntimeNodeDetailsV2';
import { Execution, Value } from 'src/third_party/mlmd';
import TestUtils from 'src/TestUtils';
import fs from 'fs';
import jsyaml from 'js-yaml';

const V2_PVC_PIPELINESPEC_PATH = 'src/data/test/create_mount_delete_dynamic_pvc.yaml';
const V2_PVC_YAML_STRING = fs.readFileSync(V2_PVC_PIPELINESPEC_PATH, 'utf8');
// The templateStr used in RuntimeNodeDetailsV2 is not directly from yaml file.
// Instead, it is from BE (already been processed).
const V2_PVC_TEMPLATE_STRING_OBJ = {
  pipeline_spec: jsyaml.safeLoadAll(V2_PVC_YAML_STRING)[0],
  platform_spec: jsyaml.safeLoadAll(V2_PVC_YAML_STRING)[1],
};
const V2_PVC_TEMPLATE_STRING = jsyaml.safeDump(V2_PVC_TEMPLATE_STRING_OBJ);

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
    const getPodLogsSpy = jest.spyOn(Apis, 'getPodLogs');
    TestUtils.makeErrorResponseOnce(getPodLogsSpy, 'Failed to retrieve pod logs');
    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          onLayerChange={layers => {}}
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

    screen.getByText('Failed to retrieve pod logs.');
  });

  it('displays logs details on side panel of execution node', async () => {
    const getPodLogsSpy = jest.spyOn(Apis, 'getPodLogs');
    getPodLogsSpy.mockImplementation(() => 'test-logs-details');
    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          onLayerChange={layers => {}}
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
    const getPodLogsSpy = jest.spyOn(Apis, 'getPodLogs');
    getPodLogsSpy.mockImplementation(() => 'test-logs-details');

    render(
      <CommonTestWrapper>
        <RuntimeNodeDetailsV2
          layers={['root']}
          onLayerChange={layers => {}}
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
          onLayerChange={layers => {}}
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
});
