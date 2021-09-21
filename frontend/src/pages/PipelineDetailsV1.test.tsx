/*
 * Copyright 2018 The Kubeflow Authors
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

import * as JsYaml from 'js-yaml';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { graphlib } from 'dagre';
import * as React from 'react';
import { testBestPractices } from 'src/TestUtils';
import PipelineDetailsV1, { PipelineDetailsV1Props } from './PipelineDetailsV1';
import { color } from 'src/Css';
import { Constants } from 'src/lib/Constants';
import { SelectedNodeInfo } from 'src/lib/StaticGraphParser';

testBestPractices();
describe('PipelineDetailsV1', () => {
  const testPipeline = {
    created_at: new Date(2018, 8, 5, 4, 3, 2),
    description: 'test pipeline description',
    id: 'test-pipeline-id',
    name: 'test pipeline',
    parameters: [{ name: 'param1', value: 'value1' }],
    default_version: {
      id: 'test-pipeline-version-id',
      name: 'test-pipeline-version',
    },
  };
  const testPipelineVersion = {
    id: 'test-pipeline-version-id',
    name: 'test-pipeline-version',
  };
  const pipelineSpecTemplate = `
  apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: entry-point-test-
spec:
  arguments:
    parameters: []
  entrypoint: entry-point-test
  templates:
  - dag:
      tasks:
      - name: recurse-1
        template: recurse-1
      - name: leaf-1
        template: leaf-1
    name: start
  - dag:
      tasks:
      - name: start
        template: start
      - name: recurse-2
        template: recurse-2
    name: recurse-1
  - dag:
      tasks:
      - name: start
        template: start
      - name: leaf-2
        template: leaf-2
      - name: recurse-3
        template: recurse-3
    name: recurse-2
  - dag:
      tasks:
      - name: start
        template: start
      - name: recurse-1
        template: recurse-1
      - name: recurse-2
        template: recurse-2
    name: recurse-3
  - dag:
      tasks:
      - name: start
        template: start
    name: entry-point-test
  - container:
    name: leaf-1
  - container:
    name: leaf-2
`;

  function generateProps(
    graph: graphlib.Graph | null,
    reducedGraph: graphlib.Graph | null,
  ): PipelineDetailsV1Props {
    const props: PipelineDetailsV1Props = {
      pipeline: testPipeline,
      selectedVersion: testPipelineVersion,
      versions: [testPipelineVersion],
      graph: graph,
      reducedGraph: reducedGraph,
      templateString: JSON.stringify({ template: JsYaml.safeDump(pipelineSpecTemplate) }),
      updateBanner: bannerProps => {},
      handleVersionSelected: async versionId => {},
    };
    return props;
  }

  beforeEach(() => {});

  it('shows correct versions in version selector', async () => {
    render(<PipelineDetailsV1 {...generateProps(new graphlib.Graph(), new graphlib.Graph())} />);

    expect(screen.getByText('test-pipeline-version'));
    expect(screen.getByTestId('version_selector').childElementCount).toEqual(1);
  });

  it('shows clicked node info in the side panel if it is in the graph', async () => {
    // Arrange
    const graph = createSimpleGraph();
    const reducedGraph = createSimpleGraph();

    // Act
    render(<PipelineDetailsV1 {...generateProps(graph, reducedGraph)} />);
    fireEvent(
      screen.getByText('start'),
      new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
      }),
    );

    // Assert
    screen.getByText('/start');
  });

  it('closes side panel when close button is clicked', async () => {
    // Arrange
    const graph = createSimpleGraph();
    const reducedGraph = createSimpleGraph();

    // Act
    render(<PipelineDetailsV1 {...generateProps(graph, reducedGraph)} />);
    fireEvent(
      screen.getByText('start'),
      new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
      }),
    );
    fireEvent(
      screen.getByLabelText('close'),
      new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
      }),
    );

    // Assert
    expect(screen.queryByText('/start')).toBeNull();
  });

  it('shows pipeline source code when config tab is clicked', async () => {
    render(<PipelineDetailsV1 {...generateProps(new graphlib.Graph(), new graphlib.Graph())} />);

    fireEvent(
      screen.getByText('YAML'),
      new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
      }),
    );
    screen.getByTestId('spec-yaml');
  });

  it('shows the summary card when clicking Show button', async () => {
    const graph = createSimpleGraph();
    const reducedGraph = createSimpleGraph();
    render(<PipelineDetailsV1 {...generateProps(graph, reducedGraph)} />);

    screen.getByText('Hide');
    fireEvent(
      screen.getByText('Hide'),
      new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
      }),
    );
    fireEvent(
      screen.getByText('Show summary'),
      new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
      }),
    );
    screen.getByText('Hide');
  });

  it('shows empty pipeline details with empty graph', async () => {
    render(<PipelineDetailsV1 {...generateProps(null, null)} />);

    screen.getByText('No graph to show');
  });
});

function createSimpleGraph() {
  const graph = new graphlib.Graph();
  graph.setGraph({ width: 1000, height: 700 });
  graph.setNode('/start', {
    bgColor: undefined,
    height: Constants.NODE_HEIGHT,
    info: new SelectedNodeInfo(),
    label: 'start',
    width: Constants.NODE_WIDTH,
  });
  return graph;
}
