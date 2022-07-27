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

import { render, screen } from '@testing-library/react';
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
import { FullArtifactPath } from 'src/lib/v2/CompareUtils';

testBestPractices();
describe('ConfidenceMetricsSection', () => {
  const setSelectedIdsSpy = jest.fn();
  const setSelectedIdColorMapSpy = jest.fn();

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

  function generateProps(selectedIds: string[]): ConfidenceMetricsSectionProps {
    const props: ConfidenceMetricsSectionProps = {
      linkedArtifacts: [
        newMockLinkedArtifact(1, 1, 'artifact1'),
        newMockLinkedArtifact(2, 1, 'artifact2'),
        newMockLinkedArtifact(4, 2, 'artifact3'),
      ],
      filter: {
        selectedIds,
        setSelectedIds: setSelectedIdsSpy,
        fullArtifactPathMap: {
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
              id: '1',
            },
            artifact: {
              id: '2',
            },
          } as FullArtifactPath,
          '2-4': {
            run: {
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
        selectedIdColorMap: {
          '1-1': '#ffffff',
          '1-2': '#eeeeee',
          '2-4': '#dddddd',
        },
        setSelectedIdColorMap: setSelectedIdColorMapSpy,
      } as ConfidenceMetricsFilter,
    };
    return props;
  }

  it('Render Confidence Metrics section', async () => {
    render(
      <CommonTestWrapper>
        <ConfidenceMetricsSection {...generateProps(['1-1', '1-2'])} />
      </CommonTestWrapper>,
    );
    screen.getByText('ROC Curve: multiple artifacts');
  });
});
