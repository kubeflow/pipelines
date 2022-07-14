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
import { Artifact, Context, Event, Execution, Value } from 'src/third_party/mlmd';
import { Apis } from 'src/lib/Apis';
import * as mlmdUtils from 'src/mlmd/MlmdUtils';
import * as metricsVisualizations from 'src/components/viewers/MetricsVisualizations';
import * as Utils from 'src/lib/Utils';
import CompareV2 from './CompareV2';
import { MetricsType, RunArtifact, SelectedArtifact } from 'src/pages/CompareV2';
import { LinkedArtifact } from 'src/mlmd/MlmdUtils';
import * as jspb from 'google-protobuf';
import MetricsDropdown from './MetricsDropdown';

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

const scalarMetricsArtifacts: RunArtifact[] = [
  {
    run: {
      run: {
        id: '1',
        name: 'run1',
      },
    },
    executionArtifacts: [
      {
        execution: newMockExecution(1, 'execution1'),
        linkedArtifacts: [
          newMockLinkedArtifact(1, 'artifact1'),
          newMockLinkedArtifact(2, 'artifact2'),
        ],
      },
      {
        execution: newMockExecution(2, 'execution2'),
        linkedArtifacts: [newMockLinkedArtifact(3, 'artifact3')],
      },
    ],
  },
  {
    run: {
      run: {
        id: '2',
        name: 'run2',
      },
    },
    executionArtifacts: [
      {
        execution: newMockExecution(3, 'execution1'),
        linkedArtifacts: [newMockLinkedArtifact(4, 'artifact1')],
      },
    ],
  },
];

testBestPractices();
describe('MetricsDropdown', () => {
  const updateSelectedArtifactsSpy = jest.fn();
  let emptySelectedArtifacts: SelectedArtifact[];

  // Prevent state from previous renders from leaking into subsequent tests.
  beforeEach(() => {
    emptySelectedArtifacts = [
      {
        selectedItem: { itemName: '', subItemName: '' },
      },
      {
        selectedItem: { itemName: '', subItemName: '' },
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

  it('Log warning when specified run does not have a ', async () => {
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

  /*it('Selected artifacts updated with user selection', async () => {
    const linkedArtifact: LinkedArtifact = newMockLinkedArtifact(1, 'artifact1');
    const scalarMetricsArtifactsOneLinked: RunArtifact[] = [
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
            linkedArtifacts: [linkedArtifact],
          },
        ],
      },
    ];

    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifactsOneLinked}
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
        linkedArtifact: linkedArtifact,
        selectedItem: {
          itemName: 'run1',
          subItemName: 'execution1',
          subItemSecondaryName: 'artifact1',
        }
      },
      {
        selectedItem: {
          itemName: '',
          subItemName: '',
        },
      }
    ];
    expect(updateSelectedArtifactsSpy).toHaveBeenLastCalledWith(newSelectedArtifacts);
  });*/

  it('HTML files read only on initial select', async () => {
    const firstLinkedArtifact: LinkedArtifact = newMockLinkedArtifact(1, 'artifact1');
    const secondLinkedArtifact: LinkedArtifact = newMockLinkedArtifact(2, 'artifact2');
    const scalarMetricsArtifactsTwoLinked: RunArtifact[] = [
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

    const getHtmlViewerConfigSpy = jest.spyOn(metricsVisualizations, 'getHtmlViewerConfig');
    getHtmlViewerConfigSpy.mockResolvedValue([]);

    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifactsTwoLinked}
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
      expect(getHtmlViewerConfigSpy).toHaveBeenLastCalledWith(
        [firstLinkedArtifact],
        undefined,
      );
    });

    // Choose another HTML element.
    fireEvent.click(screen.getByTitle('run1 > execution1 > artifact1'));
    fireEvent.mouseEnter(screen.getByText('run1'));
    fireEvent.click(screen.getByTitle('execution1 > artifact2'));

    await waitFor(() => {
      expect(getHtmlViewerConfigSpy).toHaveBeenLastCalledWith(
        [secondLinkedArtifact],
        undefined,
      );
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
    const firstLinkedArtifact: LinkedArtifact = newMockLinkedArtifact(1, 'artifact1');
    const secondLinkedArtifact: LinkedArtifact = newMockLinkedArtifact(2, 'artifact2');
    const scalarMetricsArtifactsTwoLinked: RunArtifact[] = [
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

    const getMarkdownViewerConfigSpy = jest.spyOn(metricsVisualizations, 'getMarkdownViewerConfig');
    getMarkdownViewerConfigSpy.mockResolvedValue([]);

    render(
      <CommonTestWrapper>
        <MetricsDropdown
          filteredRunArtifacts={scalarMetricsArtifactsTwoLinked}
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
      expect(getMarkdownViewerConfigSpy).toHaveBeenLastCalledWith(
        [firstLinkedArtifact],
        undefined,
      );
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
/*
  it('Markdown files read only on initial select', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = jest.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [
      [newMockExecution(1)],
      [newMockExecution(200)],
      [newMockExecution(3, 'executionName')],
    ];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, false, 'firstArtifactName'),
      newMockArtifact(3, false, 'secondArtifactName'),
    ];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [
      newMockEvent(1),
      newMockEvent(200, 'firstArtifactName'),
      newMockEvent(3, 'secondArtifactName'),
    ];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    // Simulate all artifacts as type Markdown.
    const filterLinkedArtifactsByTypeSpy = jest.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
    filterLinkedArtifactsByTypeSpy.mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.Markdown' ? linkedArtifacts : [],
    );

    const getMarkdownViewerConfigSpy = jest.spyOn(metricsVisualizations, 'getMarkdownViewerConfig');
    getMarkdownViewerConfigSpy.mockReturnValue(Promise.resolve([]));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('Markdown'));

    // Get the second element that has run text: first will be the run list.
    fireEvent.click(screen.getByText('Choose a first Markdown artifact'));
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    fireEvent.click(screen.getByText(/firstArtifactName/));

    const firstSelectedLinkedArtifact: mlmdUtils.LinkedArtifact = {
      event: events[1],
      artifact: artifacts[1],
    };
    await waitFor(() => {
      expect(getMarkdownViewerConfigSpy).toHaveBeenLastCalledWith(
        [firstSelectedLinkedArtifact],
        undefined,
      );
    });

    // Choose another Markdown element.
    fireEvent.click(screen.getByText(/firstArtifactName/));
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_3_ID}`)[1]);
    fireEvent.click(screen.getByText(/secondArtifactName/));

    const secondSelectedLinkedArtifact: mlmdUtils.LinkedArtifact = {
      event: events[2],
      artifact: artifacts[2],
    };
    await waitFor(() => {
      expect(getMarkdownViewerConfigSpy).toHaveBeenLastCalledWith(
        [secondSelectedLinkedArtifact],
        undefined,
      );
    });

    // Return and re-select the first Markdown element.
    fireEvent.click(screen.getByText(/secondArtifactName/));
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    fireEvent.click(screen.getByText(/firstArtifactName/));

    // File is not re-read if that artifact has already been selected.
    await waitFor(() => {
      expect(getMarkdownViewerConfigSpy).toBeCalledTimes(2);
    });
  });

  it('HTML file loading and error display', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = jest.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    // No execution name is provided to ensure that it can be selected by ID.
    const executions = [[newMockExecution(1)], [newMockExecution(200)], [newMockExecution(3)]];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [
      newMockArtifact(1),
      newMockArtifact(200, false, 'firstArtifactName'),
      newMockArtifact(3),
    ];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [newMockEvent(1), newMockEvent(200, 'firstArtifactName'), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    // Simulate all artifacts as type HTML.
    const filterLinkedArtifactsByTypeSpy = jest.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
    filterLinkedArtifactsByTypeSpy.mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.HTML' ? linkedArtifacts : [],
    );

    const getHtmlViewerConfigSpy = jest.spyOn(metricsVisualizations, 'getHtmlViewerConfig');
    getHtmlViewerConfigSpy.mockRejectedValue(new Error('HTML file not found.'));

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('HTML'));

    // Get the second element that has run text: first will be the run list.
    fireEvent.click(screen.getByText('Choose a first HTML artifact'));
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    fireEvent.click(screen.getByText(/firstArtifactName/));

    screen.getByRole('circularprogress');
    await waitFor(() => {
      screen.getByText('Error: failed loading HTML file. Click Details for more information.');
    });
  });*/
});
