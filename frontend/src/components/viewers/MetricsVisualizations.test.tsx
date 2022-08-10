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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import * as React from 'react';
import { CommonTestWrapper } from 'src/TestWrapper';
import { testBestPractices } from 'src/TestUtils';
import { Artifact, Event } from 'src/third_party/mlmd';
import { LinkedArtifact } from 'src/mlmd/MlmdUtils';
import { Struct, Value } from 'google-protobuf/google/protobuf/struct_pb';
import {
  ConfidenceMetricsFilter,
  ConfidenceMetricsSection,
  ConfidenceMetricsSectionProps,
} from './MetricsVisualizations';
import { FullArtifactPath, FullArtifactPathMap, RocCurveColorMap } from 'src/lib/v2/CompareUtils';
import { lineColors } from 'src/components/viewers/ROCCurve';
import TestUtils from 'src/TestUtils';
import * as rocCurveHelper from './ROCCurveHelper';

testBestPractices();
describe('ConfidenceMetricsSection', () => {
  const setSelectedIdsSpy = jest.fn();
  const setSelectedIdColorMapSpy = jest.fn();
  const setLineColorsStackSpy = jest.fn();

  function newMockEvent(artifactId: number, executionId: number, displayName?: string): Event {
    const event = new Event();
    event.setArtifactId(artifactId);
    event.setExecutionId(executionId);
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
    const customPropertiesMap: Map<string, Value> = new Map();

    const confidenceMetrics: Value = new Value();
    confidenceMetrics.setStructValue(
      Struct.fromJavaScript({
        list: [
          {
            confidenceThreshold: 2,
            falsePositiveRate: 0,
            recall: 0,
          },
          {
            confidenceThreshold: 1,
            falsePositiveRate: 0,
            recall: 0.33962264150943394,
          },
          {
            confidenceThreshold: 0.9,
            falsePositiveRate: 0,
            recall: 0.6037735849056604,
          },
        ],
      }),
    );
    customPropertiesMap.set('confidenceMetrics', confidenceMetrics);

    if (displayName) {
      const displayNameValue = new Value();
      displayNameValue.setStringValue(displayName);
      customPropertiesMap.set('display_name', displayNameValue);
    }
    jest.spyOn(artifact, 'getCustomPropertiesMap').mockReturnValue(customPropertiesMap);
    return artifact;
  }

  function newMockLinkedArtifact(
    artifactId: number,
    executionId: number,
    displayName?: string,
  ): LinkedArtifact {
    return {
      artifact: newMockArtifact(artifactId, displayName),
      event: newMockEvent(artifactId, executionId, displayName),
    } as LinkedArtifact;
  }

  interface RocCurveData {
    linkedArtifacts: LinkedArtifact[];
    fullArtifactPathMap: FullArtifactPathMap;
    selectedIds: string[];
  }

  function generateRocCurveDataByCount(count: number): RocCurveData {
    const linkedArtifacts: LinkedArtifact[] = [];
    const fullArtifactPathMap: FullArtifactPathMap = {};
    const selectedIds: string[] = [];
    for (let i = 0; i < count; i++) {
      const rocCurveId: string = `${i}-${i}`;
      linkedArtifacts.push(newMockLinkedArtifact(i, i, `artifact${i}`));
      fullArtifactPathMap[`${i}-${i}`] = {
        run: { name: `run${i}`, id: `${i}` },
        execution: { name: `execution${i}`, id: `${i}` },
        artifact: { name: `artifact${i}`, id: `${i}` },
      };
      if (i < 10) {
        selectedIds.push(rocCurveId);
      }
    }
    return {
      linkedArtifacts,
      fullArtifactPathMap,
      selectedIds,
    } as RocCurveData;
  }

  function generateProps(
    selectedIds: string[],
    linkedArtifacts?: LinkedArtifact[],
    fullArtifactPathMap?: FullArtifactPathMap,
  ): ConfidenceMetricsSectionProps {
    const lineColorsStack = [...lineColors].reverse();
    const selectedIdColorMap: RocCurveColorMap = {};
    // Note: Only 11 colors are available on the line colors stack, as UI limits selection to 10.
    for (const selectedId of selectedIds) {
      selectedIdColorMap[selectedId] = lineColorsStack.pop()!;
    }
    const props: ConfidenceMetricsSectionProps = {
      linkedArtifacts: linkedArtifacts || [
        newMockLinkedArtifact(1, 1, 'artifact1'),
        newMockLinkedArtifact(2, 1, 'artifact2'),
        newMockLinkedArtifact(4, 2, 'artifact3'),
      ],
      filter: {
        selectedIds,
        setSelectedIds: setSelectedIdsSpy,
        fullArtifactPathMap: fullArtifactPathMap || {
          '1-1': {
            run: {
              name: 'run1',
              id: '1',
            },
            execution: {
              name: 'execution1',
              id: '1',
            },
            artifact: {
              name: 'artifact1',
              id: '1',
            },
          } as FullArtifactPath,
          '1-2': {
            run: {
              name: 'run1',
              id: '1',
            },
            execution: {
              name: 'Execution ID #1',
              id: '1',
            },
            artifact: {
              name: 'Artifact ID #2',
              id: '2',
            },
          } as FullArtifactPath,
          '2-4': {
            run: {
              name: 'Run ID #2',
              id: '2',
            },
            execution: {
              name: 'execution2',
              id: '2',
            },
            artifact: {
              name: 'artifact4',
              id: '4',
            },
          } as FullArtifactPath,
        },
        selectedIdColorMap,
        setSelectedIdColorMap: setSelectedIdColorMapSpy,
        lineColorsStack,
        setLineColorsStack: setLineColorsStackSpy,
      } as ConfidenceMetricsFilter,
    };
    return props;
  }

  it('Render Confidence Metrics section with no selected artifacts', async () => {
    render(
      <CommonTestWrapper>
        <ConfidenceMetricsSection {...generateProps([])} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();
    screen.getByText('ROC Curve: no artifacts');
  });

  it('Render Confidence Metrics section with one selected artifact', async () => {
    render(
      <CommonTestWrapper>
        <ConfidenceMetricsSection {...generateProps(['1-1'])} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();
    screen.getByText('ROC Curve: artifact1');
  });

  it('Render Confidence Metrics section with multiple artifacts', async () => {
    render(
      <CommonTestWrapper>
        <ConfidenceMetricsSection {...generateProps(['1-1', '1-2'])} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();
    screen.getByText('ROC Curve: multiple artifacts');
  });

  it('Error in confidenceMetrics data format', async () => {
    const validateConfidenceMetricsSpy = jest.spyOn(rocCurveHelper, 'validateConfidenceMetrics');
    validateConfidenceMetricsSpy.mockReturnValue({
      error: 'test error',
    });
    render(
      <CommonTestWrapper>
        <ConfidenceMetricsSection {...generateProps(['1-1', '1-2'])} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    expect(validateConfidenceMetricsSpy).toHaveBeenCalledTimes(1);
    screen.getByText(
      "Error in artifact1 (artifact ID #1) artifact's confidenceMetrics data format.",
    );
  });

  it('ROC Curve filter selection check all update', async () => {
    render(
      <CommonTestWrapper>
        <ConfidenceMetricsSection {...generateProps(['1-1', '1-2'])} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    // Only the selected items are checked.
    const selectedCheckboxes = screen
      .queryAllByRole('checkbox', { checked: true })
      .filter(r => r.nodeName === 'INPUT');
    expect(selectedCheckboxes).toHaveLength(2);

    // Check all checkboxes (since the top row starts out indeterminate).
    const checkboxes = screen.queryAllByRole('checkbox').filter(r => r.nodeName === 'INPUT');
    fireEvent.click(checkboxes[0]);
    expect(setSelectedIdsSpy).toHaveBeenLastCalledWith(['1-1', '1-2', '2-4']);
    expect(setSelectedIdColorMapSpy).toHaveBeenLastCalledWith({
      '1-1': '#4285f4',
      '1-2': '#2b9c1e',
      '2-4': '#e00000',
    });
  });

  it('ROC Curve filter selection uncheck single update', async () => {
    render(
      <CommonTestWrapper>
        <ConfidenceMetricsSection {...generateProps(['1-1', '1-2', '2-4'])} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    // Only the selected items are checked.
    const selectedCheckboxes = screen
      .queryAllByRole('checkbox', { checked: true })
      .filter(r => r.nodeName === 'INPUT');
    expect(selectedCheckboxes).toHaveLength(4);

    // Uncheck the first (non-"select all") checkbox.
    const checkboxes = screen.queryAllByRole('checkbox').filter(r => r.nodeName === 'INPUT');
    fireEvent.click(checkboxes[1]);
    expect(setSelectedIdsSpy).toHaveBeenLastCalledWith(['1-2', '2-4']);
    expect(setSelectedIdColorMapSpy).toHaveBeenLastCalledWith({
      '1-2': '#2b9c1e',
      '2-4': '#e00000',
    });
    expect(setLineColorsStackSpy).toBeCalledTimes(1);
  });

  it('ROC Curve show banner when selecting 10 runs', async () => {
    const rocCurveData = generateRocCurveDataByCount(15);
    render(
      <CommonTestWrapper>
        <ConfidenceMetricsSection
          {...generateProps(
            rocCurveData.selectedIds,
            rocCurveData.linkedArtifacts,
            rocCurveData.fullArtifactPathMap,
          )}
        />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    // Only the selected items are checked.
    let checkboxes = screen.queryAllByRole('checkbox').filter(r => r.nodeName === 'INPUT');
    let selectedCheckboxes = screen
      .queryAllByRole('checkbox', { checked: true })
      .filter(r => r.nodeName === 'INPUT');
    expect(checkboxes).toHaveLength(11);
    expect(selectedCheckboxes).toHaveLength(11);
    screen.getByText(
      'You have reached the maximum number of ROC Curves (10) you can select at once.',
    );

    // Get the next page button as the last button.
    const buttons = screen.queryAllByRole('button');
    const nextPage = buttons[buttons.length - 1];

    // Ensure none of the next page checkboxes are checked.
    fireEvent.click(nextPage);
    checkboxes = screen.queryAllByRole('checkbox').filter(r => r.nodeName === 'INPUT');
    selectedCheckboxes = screen
      .queryAllByRole('checkbox', { checked: true })
      .filter(r => r.nodeName === 'INPUT');
    expect(checkboxes).toHaveLength(6);
    expect(selectedCheckboxes).toHaveLength(0);

    // Selecting a disabled checkbox has no change.
    fireEvent.click(checkboxes[1]);
    expect(setSelectedIdsSpy).toHaveBeenLastCalledWith(rocCurveData.selectedIds);
  });

  it('Filter table is present with relevant rows', async () => {
    render(
      <CommonTestWrapper>
        <ConfidenceMetricsSection {...generateProps([])} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();

    // Test the header columns and some different rows
    screen.getByText('Execution name > Artifact name');
    screen.getByText('execution1 > artifact1');
    screen.getByText('Run name');
    screen.getByText('Run ID #2');
  });
});
