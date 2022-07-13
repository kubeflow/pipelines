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
import { QUERY_PARAMS } from 'src/components/Router';
import * as mlmdUtils from 'src/mlmd/MlmdUtils';
import * as metricsVisualizations from 'src/components/viewers/MetricsVisualizations';
import * as Utils from 'src/lib/Utils';
import CompareV2 from './CompareV2';
import { PageProps } from './Page';
import { ApiRunDetail } from 'src/apis/run';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';
import { MetricsType, RunArtifact, SelectedArtifact } from 'src/pages/CompareV2';
import { LinkedArtifact } from 'src/mlmd/MlmdUtils';
import * as jspb from 'google-protobuf';
import MetricsDropdown from './MetricsDropdown';

/*

interface MetricsDropdownProps {
  filteredRunArtifacts: RunArtifact[];
  metricsTab: MetricsType;
  metricsTabText: string;
  selectedArtifacts: SelectedArtifact[];
  updateSelectedArtifacts: (selectedArtifacts: SelectedArtifact[]) => void;
}

function MetricsDropdown(props: MetricsDropdownProps) {

filteredRunArtifacts: [],
metricsTab: MetricsType.CONFUSION_MATRIX,
selectedArtifacts: [],
updateSelectedArtifacts: () => {},


*/

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

function newMockLinkedArtifact(
  id: number,
  displayName?: string,
): LinkedArtifact {
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

const emptySelectedArtifacts: SelectedArtifact[] = [
  {
    selectedItem: { itemName: '', subItemName: '' },
  },
  {
    selectedItem: { itemName: '', subItemName: '' },
  },
];

testBestPractices();
describe('MetricsDropdown', () => {
  const updateSelectedArtifactsSpy = jest.fn();

  it('Metrics dropdown has no dropdown loaded as content is not present', async () => {
    render(
      <MetricsDropdown
        filteredRunArtifacts={[]}
        metricsTab={MetricsType.CONFUSION_MATRIX}
        selectedArtifacts={emptySelectedArtifacts}
        updateSelectedArtifacts={updateSelectedArtifactsSpy}
      />,
    );
    await TestUtils.flushPromises();
    screen.getByText('There are no Confusion Matrix artifacts available on the selected runs.');
  });
/*
  it('Only confusion matrix tab has dropdown loaded with content', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 1),
      newMockContext(MOCK_RUN_2_ID, 2),
      newMockContext(MOCK_RUN_3_ID, 3),
    ];
    const getContextSpy = jest.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    const executions = [
      [newMockExecution(1)],
      [newMockExecution(2, 'executionName')],
      [newMockExecution(3)],
    ];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [newMockArtifact(1), newMockArtifact(2, true), newMockArtifact(3)];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [newMockEvent(1), newMockEvent(2, 'artifactName'), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    // Simulate all artifacts as type "ClassificationMetrics" (Confusion Matrix or ROC Curve).
    const filterLinkedArtifactsByTypeSpy = jest.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
    filterLinkedArtifactsByTypeSpy.mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.ClassificationMetrics' ? linkedArtifacts : [],
    );

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    await waitFor(() => expect(filterLinkedArtifactsByTypeSpy).toHaveBeenCalledTimes(9));

    fireEvent.click(screen.getByText('Confusion Matrix'));
    screen.getByText('Choose a first Confusion Matrix artifact');

    fireEvent.click(screen.getByText('HTML'));
    screen.getByText('There are no HTML artifacts available on the selected runs.');

    fireEvent.click(screen.getByText('Markdown'));
    screen.getByText('There are no Markdown artifacts available on the selected runs.');
  });

  it('Log warnings when specified run, execution, or artifact does not have a name', async () => {
    const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
    runs = [newMockRun(MOCK_RUN_1_ID, true), newMockRun(MOCK_RUN_2_ID), newMockRun(MOCK_RUN_3_ID)];
    getRunSpy.mockImplementation((id: string) => runs.find(r => r.run!.id === id));

    const contexts = [
      newMockContext(MOCK_RUN_1_ID, 100),
      newMockContext(MOCK_RUN_2_ID, 200),
      newMockContext(MOCK_RUN_3_ID, 300),
    ];
    const getContextSpy = jest.spyOn(mlmdUtils, 'getKfpV2RunContext');
    getContextSpy.mockImplementation((runID: string) =>
      Promise.resolve(contexts.find(c => c.getName() === runID)),
    );

    const executions = [
      [newMockExecution(100)],
      [newMockExecution(200)],
      [newMockExecution(300, 'executionName')],
    ];
    const getExecutionsSpy = jest.spyOn(mlmdUtils, 'getExecutionsFromContext');
    getExecutionsSpy.mockImplementation((context: Context) =>
      Promise.resolve(executions.find(e => e[0].getId() === context.getId())),
    );

    const artifacts = [newMockArtifact(100), newMockArtifact(200), newMockArtifact(300)];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [newMockEvent(100), newMockEvent(200), newMockEvent(300)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    const filterLinkedArtifactsByTypeSpy = jest.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
    filterLinkedArtifactsByTypeSpy.mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.HTML' ? linkedArtifacts : [],
    );

    const warnSpy = jest.spyOn(Utils.logger, 'warn');

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    fireEvent.click(screen.getByText('HTML'));
    await waitFor(() => {
      expect(warnSpy).toHaveBeenNthCalledWith(
        1,
        `Failed to fetch the display name of the run with the following ID: ${MOCK_RUN_1_ID}`,
      );
      expect(warnSpy).toHaveBeenNthCalledWith(
        2,
        'Failed to fetch the display name of the execution with the following ID: 200',
      );
      expect(warnSpy).toHaveBeenLastCalledWith(
        'Failed to fetch the display name of the artifact with the following ID: 300',
      );
    });

    // Ensure that the dropdown appropriately replaces display names with IDs.
    fireEvent.click(screen.getByText('Choose a first HTML artifact'));
    expect(screen.queryByText(`test run ${MOCK_RUN_1_ID}`)).toBeNull();

    // Get second element: first is run list. Execution and Artifact ID = 200, both in same item.
    fireEvent.mouseOver(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    screen.getByText(/200/);

    // Execution name = 'executionName', Artifact ID = 300
    fireEvent.mouseOver(screen.queryAllByText(`test run ${MOCK_RUN_3_ID}`)[1]);
    screen.getByText(/executionName/);
    screen.getByText(/300/);
  });

  it('Confusion matrix shown on select, stays after tab change or section collapse', async () => {
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
      newMockArtifact(200, true, 'artifactName'),
      newMockArtifact(3),
    ];
    const getArtifactsSpy = jest.spyOn(mlmdUtils, 'getArtifactsFromContext');
    getArtifactsSpy.mockReturnValue(Promise.resolve(artifacts));

    const events = [newMockEvent(1), newMockEvent(200, 'artifactName'), newMockEvent(3)];
    const getEventsSpy = jest.spyOn(mlmdUtils, 'getEventsByExecutions');
    getEventsSpy.mockReturnValue(Promise.resolve(events));

    const getArtifactTypesSpy = jest.spyOn(mlmdUtils, 'getArtifactTypes');
    getArtifactTypesSpy.mockReturnValue([]);

    // Simulate all artifacts as type "ClassificationMetrics" (Confusion Matrix or ROC Curve).
    const filterLinkedArtifactsByTypeSpy = jest.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
    filterLinkedArtifactsByTypeSpy.mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.ClassificationMetrics' ? linkedArtifacts : [],
    );

    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    expect(screen.queryByText(/Confusion matrix: artifactName/)).toBeNull();

    fireEvent.click(screen.getByText('Confusion Matrix'));
    fireEvent.click(screen.getByText('Choose a first Confusion Matrix artifact'));

    // Get the second element that has run text: first will be the run list.
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    fireEvent.click(screen.getByText(/artifactName/));
    screen.getByText(/Confusion Matrix: artifactName/);
    screen.getByText(/200/);

    // Change the tab and return, ensure that the confusion matrix and selected item are present.
    fireEvent.click(screen.getByText('HTML'));
    fireEvent.click(screen.getByText('Confusion Matrix'));
    screen.getByText(/Confusion Matrix: artifactName/);
    screen.getByText(/200/);

    // Collapse and expand Metrics, ensure that the confusion matrix and selected item are present.
    fireEvent.click(screen.getByText('Metrics'));
    fireEvent.click(screen.getByText('Metrics'));
    screen.getByText(/Confusion Matrix: artifactName/);
    screen.getByText(/200/);
  });

  it('HTML files read only on initial select', async () => {
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

    // Simulate all artifacts as type HTML.
    const filterLinkedArtifactsByTypeSpy = jest.spyOn(mlmdUtils, 'filterLinkedArtifactsByType');
    filterLinkedArtifactsByTypeSpy.mockImplementation(
      (metricsFilter: string, _: ArtifactType[], linkedArtifacts: LinkedArtifact[]) =>
        metricsFilter === 'system.HTML' ? linkedArtifacts : [],
    );

    const getHtmlViewerConfigSpy = jest.spyOn(metricsVisualizations, 'getHtmlViewerConfig');
    getHtmlViewerConfigSpy.mockReturnValue(Promise.resolve([]));

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

    const firstSelectedLinkedArtifact: mlmdUtils.LinkedArtifact = {
      event: events[1],
      artifact: artifacts[1],
    };
    await waitFor(() => {
      expect(getHtmlViewerConfigSpy).toHaveBeenLastCalledWith(
        [firstSelectedLinkedArtifact],
        undefined,
      );
    });

    // Choose another HTML element.
    fireEvent.click(screen.getByText(/firstArtifactName/));
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_3_ID}`)[1]);
    fireEvent.click(screen.getByText(/secondArtifactName/));

    const secondSelectedLinkedArtifact: mlmdUtils.LinkedArtifact = {
      event: events[2],
      artifact: artifacts[2],
    };
    await waitFor(() => {
      expect(getHtmlViewerConfigSpy).toHaveBeenLastCalledWith(
        [secondSelectedLinkedArtifact],
        undefined,
      );
    });

    // Return and re-select the first HTML element.
    fireEvent.click(screen.getByText(/secondArtifactName/));
    fireEvent.mouseEnter(screen.queryAllByText(`test run ${MOCK_RUN_2_ID}`)[1]);
    fireEvent.click(screen.getByText(/firstArtifactName/));

    // File is not re-read if that artifact has already been selected.
    await waitFor(() => {
      expect(getHtmlViewerConfigSpy).toBeCalledTimes(2);
    });
  });

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
