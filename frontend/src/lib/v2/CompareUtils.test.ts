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

import { testBestPractices } from 'src/TestUtils';
import { getCompareTableProps, getValidRocCurveLinkedArtifacts, RunArtifact } from './CompareUtils';
import { Artifact, Event, Execution, Value } from 'src/third_party/mlmd';
import { LinkedArtifact } from 'src/mlmd/MlmdUtils';
import * as jspb from 'google-protobuf';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';

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

function newMockArtifact(
  id: number,
  scalarMetricValues?: number[],
  confidenceMetricsStruct?: any,
  displayName?: string,
): Artifact {
  const artifact = new Artifact();
  artifact.setId(id);

  const customPropertiesMap: jspb.Map<string, Value> = jspb.Map.fromObject([], null, null);
  if (displayName) {
    const displayNameValue = new Value();
    displayNameValue.setStringValue(displayName);
    customPropertiesMap.set('display_name', displayNameValue);
  }

  if (scalarMetricValues) {
    scalarMetricValues.forEach((scalarMetricValue, index) => {
      const value = new Value();
      value.setDoubleValue(scalarMetricValue);
      customPropertiesMap.set(`scalarMetric${index}`, value);
    });
  }

  if (confidenceMetricsStruct) {
    const confidenceMetrics: Value = new Value();
    confidenceMetrics.setStructValue(confidenceMetricsStruct);
    customPropertiesMap.set('confidenceMetrics', confidenceMetrics);
  }

  jest.spyOn(artifact, 'getCustomPropertiesMap').mockReturnValue(customPropertiesMap);
  return artifact;
}

function newMockLinkedArtifact(
  id: number,
  executionId: number,
  scalarMetricValues?: number[],
  confidenceMetricsStruct?: any,
  displayName?: string,
): LinkedArtifact {
  return {
    artifact: newMockArtifact(id, scalarMetricValues, confidenceMetricsStruct, displayName),
    event: newMockEvent(id, executionId, displayName),
  } as LinkedArtifact;
}

testBestPractices();
describe('CompareUtils', () => {
  it('Empty scalar metrics artifacts results in empty table data', () => {
    expect(getCompareTableProps([], 0)).toMatchObject({
      xLabels: [],
      yLabels: [],
      xParentLabels: [],
      rows: [],
    });
  });

  it('Scalar metrics artifacts with all data and names populated', () => {
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
              newMockLinkedArtifact(1, 1, [1, 2], undefined, 'artifact1'),
              newMockLinkedArtifact(2, 1, [1], undefined, 'artifact2'),
            ],
          },
          {
            execution: newMockExecution(2, 'execution2'),
            linkedArtifacts: [newMockLinkedArtifact(3, 2, [3], undefined, 'artifact3')],
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
            linkedArtifacts: [newMockLinkedArtifact(4, 3, [4], undefined, 'artifact1')],
          },
        ],
      },
    ];
    const artifactCount: number = 4;

    expect(getCompareTableProps(scalarMetricsArtifacts, artifactCount)).toMatchObject({
      xLabels: [
        'execution1 > artifact1',
        'execution1 > artifact2',
        'execution2 > artifact3',
        'execution1 > artifact1',
      ],
      yLabels: ['scalarMetric0', 'scalarMetric1'],
      xParentLabels: [
        { colSpan: 3, label: 'run1' },
        { colSpan: 1, label: 'run2' },
      ],
      rows: [
        ['1', '1', '3', '4'],
        ['2', '', '', ''],
      ],
    });
  });

  it('Scalar metrics artifacts with data populated and no names', () => {
    const scalarMetricsArtifacts: RunArtifact[] = [
      {
        run: {
          run: {
            id: '1',
          },
        },
        executionArtifacts: [
          {
            execution: newMockExecution(1),
            linkedArtifacts: [
              newMockLinkedArtifact(1, 1, [1, 2]),
              newMockLinkedArtifact(2, 1, [1]),
            ],
          },
          {
            execution: newMockExecution(2),
            linkedArtifacts: [newMockLinkedArtifact(3, 2, [3])],
          },
        ],
      },
      {
        run: {
          run: {
            id: '2',
          },
        },
        executionArtifacts: [
          {
            execution: newMockExecution(3),
            linkedArtifacts: [newMockLinkedArtifact(4, 3, [4])],
          },
        ],
      },
    ];
    const artifactCount: number = 4;

    expect(getCompareTableProps(scalarMetricsArtifacts, artifactCount)).toMatchObject({
      xLabels: ['- > -', '- > -', '- > -', '- > -'],
      yLabels: ['scalarMetric0', 'scalarMetric1'],
      xParentLabels: [
        { colSpan: 3, label: '-' },
        { colSpan: 1, label: '-' },
      ],
      rows: [
        ['1', '1', '3', '4'],
        ['2', '', '', ''],
      ],
    });
  });

  it('Validate the ROC Curve linked artifacts', () => {
    const confidenceMetricsStruct = Struct.fromJavaScript({
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
    });

    const validLinkedArtifacts: LinkedArtifact[] = [
      newMockLinkedArtifact(1, 1, undefined, confidenceMetricsStruct, 'artifact1'),
      newMockLinkedArtifact(2, 1, undefined, confidenceMetricsStruct, 'artifact2'),
      newMockLinkedArtifact(4, 3, undefined, confidenceMetricsStruct),
    ];
    const rocCurveRunArtifacts: RunArtifact[] = [
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
            linkedArtifacts: [validLinkedArtifacts[0], validLinkedArtifacts[1]],
          },
          {
            execution: newMockExecution(2, 'execution2'),
            linkedArtifacts: [newMockLinkedArtifact(3, 2, undefined, undefined, 'artifact3')],
          },
        ],
      },
      {
        run: {
          run: {
            id: '2',
          },
        },
        executionArtifacts: [
          {
            execution: newMockExecution(3, 'execution1'),
            linkedArtifacts: [validLinkedArtifacts[2]],
          },
        ],
      },
    ];

    const rocCurveArtifactData = getValidRocCurveLinkedArtifacts(rocCurveRunArtifacts);
    expect(rocCurveArtifactData.validLinkedArtifacts).toMatchObject(validLinkedArtifacts);

    const fullArtifactPathMap = rocCurveArtifactData.fullArtifactPathMap;
    expect(Object.keys(fullArtifactPathMap)).toHaveLength(3);
    expect(fullArtifactPathMap['1-1']).toMatchObject({
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
    });
    expect(fullArtifactPathMap['3-4']).toMatchObject({
      run: {
        name: 'Run ID #2',
        id: '2',
      },
      execution: {
        name: 'execution1',
        id: '3',
      },
      artifact: {
        name: 'Artifact ID #4',
        id: '4',
      },
    });
  });
});
