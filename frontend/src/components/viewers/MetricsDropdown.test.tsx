/*
 * Copyright 2022 The Kubeflow Authors
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

import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import * as React from 'react';
import { CommonTestWrapper } from 'src/TestWrapper';
import TestUtils, { testBestPractices } from 'src/TestUtils';
import { Artifact, Event, Execution, Value } from 'src/third_party/mlmd';
import * as metricsVisualizations from 'src/components/viewers/MetricsVisualizations';
import * as Utils from 'src/lib/Utils';
import { SelectedArtifact } from 'src/pages/CompareV2';
import { LinkedArtifact } from 'src/mlmd/MlmdUtils';
import * as jspb from 'google-protobuf';
import MetricsDropdown from './MetricsDropdown';
import { MetricsType, RunArtifact } from 'src/lib/v2/CompareUtils';

function newMockExecution(id: number, displayName?: string): Execution {
  const execution = new Execution();
  execution.setId(id);
  if (displayName) {
    const customPropertiesMap: Map<string, Value> = new Map();
    const displayNameValue = new Value();
    displayNameValue.setStringValue(displayName);
    customPropertiesMap.set('display_name', displayNameValue);
    jest.spyOn(execution, 'getCustomPropertiesMap').mockReturnValue(customPropertiesMap);
  }
  return execution;
}

function newMockEvent(id: number, displayName?: string): Event {
  const event = new Event();
  event.setArtifactId(id);
  event.setExecutionId(id);
  event.setType(Event.Type.OUTPUT);
  if (displayName) {
    const path = new Event.Path();
    const step = new Event.Path.Step();
    step.setKey(displayName);
    path.addSteps(step);
    event.setPath(path);
  }
  return event;
}

function newMockArtifact(id: number, displayName?: string): Artifact {
  const artifact = new Artifact();
  artifact.setId(id);

  const customPropertiesMap: jspb.Map<string, Value> = jspb.Map.fromObject([], null, null);
  if (displayName) {
    const displayNameValue = new Value();
    displayNameValue.setStringValue(displayName);
    customPropertiesMap.set('display_name', displayNameValue);
  }

  jest.spyOn(artifact, 'getCustomPropertiesMap').mockReturnValue(customPropertiesMap);
  return artifact;
}

function newMockLinkedArtifact(id: number, displayName?: string): LinkedArtifact {
  return {
    artifact: newMockArtifact(id, displayName),
    event: newMockEvent(id, displayName),
  } as LinkedArtifact;
}

testBestPractices();
describe('MetricsDropdown', () => {
  const updateSelectedArtifactsSpy = jest.fn();
  let emptySelectedArtifacts: SelectedArtifact[];
  let firstLinkedArtifact: LinkedArtifact;
  let secondLinkedArtifact: LinkedArtifact;
  let scalarMetricsArtifacts: RunArtifact[];

  beforeEach(() => {
    emptySelectedArtifacts = [
      {
        selectedItem: { itemName: '', subItemName: '' },
      },
      {
        selectedItem: { itemName: '', subItemName: '' },
      },
    ];
    firstLinkedArtifact = newMockLinkedArtifact(1, 'artifact1');
    secondLinkedArtifact = newMockLinkedArtifact(2, 'artifact2');
    scalarMetricsArtifacts = [
      {
        run: {
          run: {
            id: '1',
            name: 'run1',
          },
          pipeline_runtime: {
            workflow_manifest: '',
          },
        },
        executionArtifacts: [
          {
            execution: newMockExecution(1, 'execution1'),
            linkedArtifacts: [firstLinkedArtifact, secondLinkedArtifact],
          },
        ],
      },
    ];
  });

  it('Metrics dropdown has no dropdown loaded as Confusion Matrix is not present', async () => {
    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={[]}
          metricsTab={MetricsType.CONFUSION_MATRIX}
          selectedArtifacts={emptySelectedArtifacts}
          updateSelectedArtifacts={updateSelectedArtifactsSpy}
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();
    screen.getByText('There are no Confusion Matrix artifacts available on the selected runs.');
  });

  it('Metrics dropdown has no dropdown loaded as HTML is not present', async () => {
    render(
      <MetricsDropdown
        filteredRunArtifacts={[]}
        metricsTab={MetricsType.HTML}
        selectedArtifacts={emptySelectedArtifacts}
        updateSelectedArtifacts={updateSelectedArtifactsSpy}
      />,
    );
    await TestUtils.flushPromises();
    screen.getByText('There are no HTML artifacts available on the selected runs.');
  });

  it('Metrics dropdown has no dropdown loaded as Markdown is not present', async () => {
    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={[]}
          metricsTab={MetricsType.MARKDOWN}
          selectedArtifacts={emptySelectedArtifacts}
          updateSelectedArtifacts={updateSelectedArtifactsSpy}
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();
    screen.getByText('There are no Markdown artifacts available on the selected runs.');
  });

  it('Dropdown loaded when content is present', async () => {
    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifacts}
          metricsTab={MetricsType.CONFUSION_MATRIX}
          selectedArtifacts={emptySelectedArtifacts}
          updateSelectedArtifacts={updateSelectedArtifactsSpy}
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();
    screen.getByText('Choose a first Confusion Matrix artifact');
  });

  it('Log warning when specified run does not have a name', async () => {
    const scalarMetricsArtifactsNoRunName: RunArtifact[] = [
      {
        run: {
          run: {
            id: '1',
          },
        },
        executionArtifacts: [
          {
            execution: newMockExecution(1, 'execution1'),
            linkedArtifacts: [newMockLinkedArtifact(1, 'artifact1')],
          },
        ],
      },
    ];
    const warnSpy = jest.spyOn(Utils.logger, 'warn');

    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifactsNoRunName}
          metricsTab={MetricsType.CONFUSION_MATRIX}
          selectedArtifacts={emptySelectedArtifacts}
          updateSelectedArtifacts={updateSelectedArtifactsSpy}
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => {
      expect(warnSpy).toHaveBeenLastCalledWith(
        'Failed to fetch the display name of the run with the following ID: 1',
      );
    });

    // Ensure that the dropdown is empty if run name is not provided.
    screen.getByText('There are no Confusion Matrix artifacts available on the selected runs.');
  });

  it('Selected artifacts updated with user selection', async () => {
    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifacts}
          metricsTab={MetricsType.CONFUSION_MATRIX}
          selectedArtifacts={emptySelectedArtifacts}
          updateSelectedArtifacts={updateSelectedArtifactsSpy}
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('Choose a first Confusion Matrix artifact'));
    fireEvent.mouseEnter(screen.getByText('run1'));
    fireEvent.click(screen.getByTitle('execution1 > artifact1'));

    const newSelectedArtifacts: SelectedArtifact[] = [
      {
        linkedArtifact: firstLinkedArtifact,
        selectedItem: {
          itemName: 'run1',
          subItemName: 'execution1',
          subItemSecondaryName: 'artifact1',
        },
      },
      {
        selectedItem: {
          itemName: '',
          subItemName: '',
        },
      },
    ];
    expect(updateSelectedArtifactsSpy).toHaveBeenLastCalledWith(newSelectedArtifacts);
  });

  it('HTML files read only on initial select', async () => {
    const getHtmlViewerConfigSpy = jest.spyOn(metricsVisualizations, 'getHtmlViewerConfig');
    getHtmlViewerConfigSpy.mockResolvedValue([]);

    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifacts}
          metricsTab={MetricsType.HTML}
          selectedArtifacts={emptySelectedArtifacts}
          updateSelectedArtifacts={updateSelectedArtifactsSpy}
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    // Choose the first HTML element.
    fireEvent.click(screen.getByText('Choose a first HTML artifact'));
    fireEvent.mouseEnter(screen.getByText('run1'));
    fireEvent.click(screen.getByTitle('execution1 > artifact1'));

    await waitFor(() => {
      expect(getHtmlViewerConfigSpy).toHaveBeenLastCalledWith([firstLinkedArtifact], undefined);
    });

    // Choose another HTML element.
    fireEvent.click(screen.getByTitle('run1 > execution1 > artifact1'));
    fireEvent.mouseEnter(screen.getByText('run1'));
    fireEvent.click(screen.getByTitle('execution1 > artifact2'));

    await waitFor(() => {
      expect(getHtmlViewerConfigSpy).toHaveBeenLastCalledWith([secondLinkedArtifact], undefined);
    });

    // Return and re-select the first HTML element.
    fireEvent.click(screen.getByTitle('run1 > execution1 > artifact2'));
    fireEvent.mouseEnter(screen.getByText('run1'));
    fireEvent.click(screen.getByTitle('execution1 > artifact1'));

    // File is not re-read if that artifact has already been selected.
    await waitFor(() => {
      expect(getHtmlViewerConfigSpy).toBeCalledTimes(2);
    });
  });

  it('Markdown files read only on initial select', async () => {
    const getMarkdownViewerConfigSpy = jest.spyOn(metricsVisualizations, 'getMarkdownViewerConfig');
    getMarkdownViewerConfigSpy.mockResolvedValue([]);

    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifacts}
          metricsTab={MetricsType.MARKDOWN}
          selectedArtifacts={emptySelectedArtifacts}
          updateSelectedArtifacts={updateSelectedArtifactsSpy}
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    // Choose the first Markdown element.
    fireEvent.click(screen.getByText('Choose a first Markdown artifact'));
    fireEvent.mouseEnter(screen.getByText('run1'));
    fireEvent.click(screen.getByTitle('execution1 > artifact1'));

    await waitFor(() => {
      expect(getMarkdownViewerConfigSpy).toHaveBeenLastCalledWith([firstLinkedArtifact], undefined);
    });

    // Choose another Markdown element.
    fireEvent.click(screen.getByTitle('run1 > execution1 > artifact1'));
    fireEvent.mouseEnter(screen.getByText('run1'));
    fireEvent.click(screen.getByTitle('execution1 > artifact2'));

    await waitFor(() => {
      expect(getMarkdownViewerConfigSpy).toHaveBeenLastCalledWith(
        [secondLinkedArtifact],
        undefined,
      );
    });

    // Return and re-select the first Markdown element.
    fireEvent.click(screen.getByTitle('run1 > execution1 > artifact2'));
    fireEvent.mouseEnter(screen.getByText('run1'));
    fireEvent.click(screen.getByTitle('execution1 > artifact1'));

    // File is not re-read if that artifact has already been selected.
    await waitFor(() => {
      expect(getMarkdownViewerConfigSpy).toBeCalledTimes(2);
    });
  });

  it('HTML file loading and error display with namespace input', async () => {
    const getHtmlViewerConfigSpy = jest.spyOn(metricsVisualizations, 'getHtmlViewerConfig');
    getHtmlViewerConfigSpy.mockRejectedValue(new Error('HTML file not found.'));

    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifacts}
          metricsTab={MetricsType.HTML}
          selectedArtifacts={emptySelectedArtifacts}
          updateSelectedArtifacts={updateSelectedArtifactsSpy}
          namespace='namespaceInput'
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('Choose a first HTML artifact'));
    fireEvent.mouseEnter(screen.getByText('run1'));
    fireEvent.click(screen.getByTitle('execution1 > artifact1'));

    screen.getByRole('circularprogress');
    await waitFor(() => {
      expect(getHtmlViewerConfigSpy).toHaveBeenLastCalledWith(
        [firstLinkedArtifact],
        'namespaceInput',
      );
      screen.getByText('Error: failed loading HTML file. Click Details for more information.');
    });
  });

  it('Dropdown initially loaded with selected artifact', async () => {
    const newSelectedArtifacts: SelectedArtifact[] = [
      {
        selectedItem: {
          itemName: '',
          subItemName: '',
        },
      },
      {
        linkedArtifact: firstLinkedArtifact,
        selectedItem: {
          itemName: 'run1',
          subItemName: 'execution1',
          subItemSecondaryName: 'artifact1',
        },
      },
    ];

    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifacts}
          metricsTab={MetricsType.CONFUSION_MATRIX}
          selectedArtifacts={newSelectedArtifacts}
          updateSelectedArtifacts={updateSelectedArtifactsSpy}
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    screen.getByText('Choose a first Confusion Matrix artifact');
    screen.getByTitle('run1 > execution1 > artifact1');
  });
});
