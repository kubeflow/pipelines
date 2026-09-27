/*
 * Copyright 2021 The Kubeflow Authors
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

import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CommonTestWrapper } from 'src/TestWrapper';
import { mockResizeObserver, testBestPractices } from 'src/TestUtils';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import PipelineDetailsV2 from './PipelineDetailsV2';
import v2YamlTemplateString from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';

testBestPractices();
describe('PipelineDetailsV2', () => {
  let testV2Pipeline: V2beta1Pipeline = {};
  let testV2PipelineVersion: V2beta1PipelineVersion = {};
  let newTestV2PipelineVersion: V2beta1PipelineVersion = {};
  let thirdPipelineVersion: V2beta1PipelineVersion = {};

  testV2Pipeline = {
    created_at: new Date(2018, 8, 5, 4, 3, 2),
    description: 'test pipeline description',
    pipeline_id: 'test-pipeline-id',
    display_name: 'test pipeline',
  };

  testV2PipelineVersion = {
    display_name: 'test-pipeline-version-v2',
    pipeline_id: 'test-pipeline-id',
    pipeline_version_id: 'test-pipeline-version-v2-id',
  };

  newTestV2PipelineVersion = {
    display_name: 'new-test-pipeline-version-v2',
    pipeline_id: 'test-pipeline-id',
    pipeline_version_id: 'new-test-pipeline-version-v2-id',
  };

  thirdPipelineVersion = {
    display_name: 'test-pipeline-version-third',
    pipeline_version_id: 'test-pipeline-version-third-id',
    pipeline_spec: {
      apiVersion: 'argoproj.io/v1alpha1',
      kind: 'Workflow',
      spec: { arguments: { parameters: [{ name: 'msg', value: 'param' }] } },
    },
  };

  beforeEach(() => {
    mockResizeObserver();
  });

  it('Render detail page with reactflow', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          pipelineFlowElements={[]}
          setSubDagLayers={function (layers: string[]): void {
            return;
          }}
          pipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={function (versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );
    expect(screen.getByTestId('DagCanvas')).not.toBeNull();
  });

  it('Render summary card', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          pipelineFlowElements={[]}
          setSubDagLayers={function (layers: string[]): void {
            return;
          }}
          pipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={function (versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );
    await userEvent.click(screen.getByText('Show Summary'));
  });

  it('shows selected version in summary card', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          pipelineFlowElements={[]}
          setSubDagLayers={function (layers: string[]): void {
            return;
          }}
          pipeline={testV2Pipeline}
          selectedVersion={testV2PipelineVersion}
          versions={[testV2PipelineVersion, newTestV2PipelineVersion, thirdPipelineVersion]}
          handleVersionSelected={function (versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );
    await userEvent.click(screen.getByText('Show Summary'));
    screen.getByText('test-pipeline-version-v2');
  });

  it('shows updated selected version in summary card after switching to another v2 version', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          pipelineFlowElements={[]}
          setSubDagLayers={function (layers: string[]): void {
            return;
          }}
          pipeline={testV2Pipeline}
          selectedVersion={testV2PipelineVersion}
          versions={[testV2PipelineVersion, newTestV2PipelineVersion, thirdPipelineVersion]}
          handleVersionSelected={function (versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );

    await userEvent.click(screen.getByText('Show Summary'));
    const selectedVersion = screen.getByText('test-pipeline-version-v2');
    await userEvent.click(selectedVersion); // Open dropdown list
    const anotherVersion = screen.getByText('new-test-pipeline-version-v2');
    await userEvent.click(anotherVersion); // Selected another version
    screen.getByText('new-test-pipeline-version-v2'); // Selected version change to another version
  });

  it('shows updated selected version in summary card after switching to third version', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          pipelineFlowElements={[]}
          setSubDagLayers={function (layers: string[]): void {
            return;
          }}
          pipeline={testV2Pipeline}
          selectedVersion={testV2PipelineVersion}
          versions={[testV2PipelineVersion, newTestV2PipelineVersion, thirdPipelineVersion]}
          handleVersionSelected={function (versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );

    await userEvent.click(screen.getByText('Show Summary'));
    const selectedVersion = screen.getByText('test-pipeline-version-v2');
    await userEvent.click(selectedVersion); // Open dropdown list
    const thirdVersion = screen.getByText('test-pipeline-version-third');
    await userEvent.click(thirdVersion); // Selected third version
    screen.getByText('test-pipeline-version-third'); // Selected version change to third version
  });

  it('Render Execution node', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          pipelineFlowElements={[
            {
              data: {
                label: 'flip-coin-op',
              },
              id: 'task.flip-coin-op',
              position: { x: 100, y: 100 },
              type: 'EXECUTION',
            },
          ]}
          setSubDagLayers={(layers) => {}}
          pipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={function (versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );
    expect(screen.getByTestId('DagCanvas')).not.toBeNull();
    screen.getByText('flip-coin-op');
  });

  it('Show Side panel when select node', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          templateString={v2YamlTemplateString}
          pipelineFlowElements={[
            {
              data: {
                label: 'preprocess',
              },
              id: 'task.preprocess',
              position: { x: 100, y: 100 },
              type: 'EXECUTION',
            },
          ]}
          setSubDagLayers={(layers) => {}}
          pipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={function (versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );

    // Use fireEvent: user-event v14 creates events with non-configurable view, which breaks
    // d3-drag (@xyflow/react) when event.view is null in jsdom.
    fireEvent.click(screen.getByText('preprocess'));
    await screen.findByText('Input Artifacts');
    await screen.findByText('Input Parameters');
    await screen.findByText('Output Artifacts');
    await screen.findByText('Output Parameters');
  });
});
