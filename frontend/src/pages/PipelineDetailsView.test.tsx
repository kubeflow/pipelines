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
import { ThemeProvider } from 'src/components/shell/ThemeProvider';
import { CommonTestWrapper } from 'src/TestWrapper';
import { mockResizeObserver, testBestPractices } from 'src/TestUtils';
import { V2Pipeline, V2PipelineVersion } from 'src/apisv2/pipeline';
import PipelineDetailsView from './PipelineDetailsView';
import v2YamlTemplateString from 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml?raw';
import { nestedArtifactSpec } from 'src/data/test/groupedFlow';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import { convertFlowElements } from 'src/lib/v2/StaticFlow';

testBestPractices();

beforeEach(() => {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  );
});

describe('PipelineDetailsV2', () => {
  let testV2Pipeline: V2Pipeline = {};
  let testV2PipelineVersion: V2PipelineVersion = {};
  let newTestV2PipelineVersion: V2PipelineVersion = {};
  let thirdPipelineVersion: V2PipelineVersion = {};

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
    localStorage.clear();
    mockResizeObserver();
  });

  it('renders the graph canvas without a top border', async () => {
    render(
      <CommonTestWrapper>
        <ThemeProvider>
          <PipelineDetailsView
            pipelineFlowElements={[]}
            pipeline={null}
            selectedVersion={undefined}
            versions={[]}
            handleVersionSelected={function (versionId: string): Promise<void> {
              return Promise.resolve();
            }}
          ></PipelineDetailsView>
        </ThemeProvider>
      </CommonTestWrapper>,
    );
    const canvas = screen.getByTestId('DagCanvas');
    expect(canvas).toBeInTheDocument();
    expect(canvas.parentElement!.style.borderTop).toBe('');
  });

  it('opens the Details tab', async () => {
    render(
      <CommonTestWrapper>
        <ThemeProvider>
          <PipelineDetailsView
            pipelineFlowElements={[]}
            pipeline={null}
            selectedVersion={undefined}
            versions={[]}
            handleVersionSelected={function (versionId: string): Promise<void> {
              return Promise.resolve();
            }}
          ></PipelineDetailsView>
        </ThemeProvider>
      </CommonTestWrapper>,
    );
    await userEvent.click(screen.getByRole('tab', { name: 'Details', exact: true }));
  });

  it('keeps the version selector out of the content area and offers a Versions tab', () => {
    render(
      <CommonTestWrapper>
        <ThemeProvider>
          <PipelineDetailsView
            pipelineFlowElements={[]}
            pipeline={testV2Pipeline}
            selectedVersion={testV2PipelineVersion}
            versions={[testV2PipelineVersion]}
            handleVersionSelected={async () => {}}
            manageVersionsHref='/pipelines/details/pipeline?tab=versions'
          />
        </ThemeProvider>
      </CommonTestWrapper>,
    );
    expect(screen.queryByRole('combobox', { name: 'Version' })).not.toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Versions' })).toBeVisible();
    expect(screen.queryByText('Manage versions')).not.toBeInTheDocument();
  });

  it('Render Execution node', async () => {
    render(
      <CommonTestWrapper>
        <ThemeProvider>
          <PipelineDetailsView
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
            pipeline={null}
            selectedVersion={undefined}
            versions={[]}
            handleVersionSelected={function (versionId: string): Promise<void> {
              return Promise.resolve();
            }}
          ></PipelineDetailsView>
        </ThemeProvider>
      </CommonTestWrapper>,
    );
    expect(screen.getByTestId('DagCanvas')).not.toBeNull();
    screen.getByText('flip-coin-op');
  });

  it('shows the nested component details without navigating into its sub-DAG', async () => {
    render(
      <CommonTestWrapper>
        <ThemeProvider>
          <PipelineDetailsView
            templateString={JSON.stringify(PipelineSpec.toJSON(nestedArtifactSpec))}
            pipelineFlowElements={convertFlowElements(nestedArtifactSpec)}
            pipeline={null}
            selectedVersion={undefined}
            versions={[]}
            handleVersionSelected={async () => {}}
          />
        </ThemeProvider>
      </CommonTestWrapper>,
    );
    fireEvent.click(screen.getByText('Train model'));
    await screen.findByText('Output Artifacts');
    expect(screen.getByRole('button', { name: 'Collapse Training pipeline' })).toBeInTheDocument();
    expect(screen.queryByText('Layers')).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('Training pipeline'));
    expect(screen.queryByRole('button', { name: 'Open Sub-DAG' })).not.toBeInTheDocument();
    expect(screen.getByText('Deploy model')).toBeInTheDocument();
  });

  it('retains the selected node when switching to the spec tab and back', async () => {
    render(
      <CommonTestWrapper>
        <ThemeProvider>
          <PipelineDetailsView
            templateString={JSON.stringify(PipelineSpec.toJSON(nestedArtifactSpec))}
            pipelineFlowElements={convertFlowElements(nestedArtifactSpec)}
            pipeline={null}
            selectedVersion={undefined}
            versions={[]}
            handleVersionSelected={async () => {}}
          />
        </ThemeProvider>
      </CommonTestWrapper>,
    );
    fireEvent.click(screen.getByText('Train model'));
    expect(screen.getByText('Output Artifacts')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('tab', { name: 'Pipeline spec' }));
    expect(screen.queryByTestId('DagCanvas')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('tab', { name: 'Graph', exact: true }));
    expect(screen.getByText('Output Artifacts')).toBeInTheDocument();
  });

  it('Show Side panel when select node', async () => {
    render(
      <CommonTestWrapper>
        <ThemeProvider>
          <PipelineDetailsView
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
            pipeline={null}
            selectedVersion={undefined}
            versions={[]}
            handleVersionSelected={function (versionId: string): Promise<void> {
              return Promise.resolve();
            }}
          ></PipelineDetailsView>
        </ThemeProvider>
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

it('shows inline definition metadata, source and falsy parameter defaults in Details', () => {
  mockResizeObserver();
  render(
    <CommonTestWrapper>
      <ThemeProvider>
        <PipelineDetailsView
          templateString={JSON.stringify({
            pipelineInfo: { name: 'inline-training', description: 'Train a model' },
            deploymentSpec: {},
            root: {
              dag: { tasks: {} },
              inputDefinitions: {
                parameters: {
                  retries: { parameterType: 'NUMBER_INTEGER', defaultValue: 0 },
                  enabled: { parameterType: 'BOOLEAN', defaultValue: false },
                  label: { parameterType: 'STRING', defaultValue: '' },
                  required: { parameterType: 'STRING' },
                },
              },
            },
          })}
          definitionSource={{
            name: 'Nightly training',
            href: '/recurringrun/details/schedule-1',
            inline: true,
          }}
          pipelineFlowElements={[]}
          pipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={async () => {}}
        />
      </ThemeProvider>
    </CommonTestWrapper>,
  );
  fireEvent.click(screen.getByRole('tab', { name: 'Details', exact: true }));
  expect(screen.getByText('inline-training')).toBeInTheDocument();
  expect(screen.getByText('Train a model')).toBeInTheDocument();
  expect(screen.getByText('Inline definition')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Nightly training' })).toHaveAttribute(
    'href',
    '/recurringrun/details/schedule-1',
  );
  for (const value of ['0', 'false', '""', 'No default'])
    expect(screen.getByText(value, { exact: true })).toBeInTheDocument();
  expect(screen.queryByText('Version ID')).not.toBeInTheDocument();
  expect(screen.queryByText('Uploaded on')).not.toBeInTheDocument();
});

it('shows a useful Details empty state when spec metadata is absent', () => {
  mockResizeObserver();
  render(
    <CommonTestWrapper>
      <ThemeProvider>
        <PipelineDetailsView
          pipelineFlowElements={[]}
          pipeline={null}
          selectedVersion={undefined}
          versions={[]}
          handleVersionSelected={async () => {}}
        />
      </ThemeProvider>
    </CommonTestWrapper>,
  );
  fireEvent.click(screen.getByRole('tab', { name: 'Details', exact: true }));
  expect(screen.getByText('Not specified')).toBeInTheDocument();
  expect(screen.getByText('No description')).toBeInTheDocument();
  expect(screen.getByText('No input parameters defined.')).toBeInTheDocument();
});
