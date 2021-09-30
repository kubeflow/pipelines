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

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import * as React from 'react';
import * as v2PipelineSpec from 'src/data/test/mock_lightweight_python_functions_v2_pipeline.json';
import { CommonTestWrapper } from 'src/TestWrapper';
import { mockResizeObserver, testBestPractices } from '../TestUtils';
import PipelineDetailsV2 from './PipelineDetailsV2';

testBestPractices();
describe('PipelineDetailsV2', () => {
  beforeEach(() => {
    mockResizeObserver();
  });

  it('Render detail page with reactflow', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          pipelineFlowElements={[]}
          setSubDagLayers={function(layers: string[]): void {
            return;
          }}
          apiPipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={function(versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );
    expect(screen.getByTestId('StaticCanvas')).not.toBeNull();
  });

  it('Render summary card', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          pipelineFlowElements={[]}
          setSubDagLayers={function(layers: string[]): void {
            return;
          }}
          apiPipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={function(versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );
    userEvent.click(screen.getByText('Show Summary'));
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
          setSubDagLayers={layers => {}}
          apiPipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={function(versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );
    expect(screen.getByTestId('StaticCanvas')).not.toBeNull();
    screen.getByText('flip-coin-op');
  });

  it('Show Side panel when select node', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2
          templateString={JSON.stringify(v2PipelineSpec)}
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
          setSubDagLayers={layers => {}}
          apiPipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={function(versionId: string): Promise<void> {
            return Promise.resolve();
          }}
        ></PipelineDetailsV2>
      </CommonTestWrapper>,
    );

    userEvent.click(screen.getByText('preprocess'));
    await screen.findByText('Input Artifacts');
    await screen.findByText('Input Parameters');
    await screen.findByText('Output Artifacts');
    await screen.findByText('Output Parameters');
  });
});
