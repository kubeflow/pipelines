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

import { Artifact, ArtifactType, Execution, Value } from '@kubeflow/frontend';
import { render, waitFor } from '@testing-library/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import React from 'react';
import * as mlmdUtils from 'src/lib/MlmdUtils';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { MetricsTab } from './MetricsTab';

testBestPractices();

describe('MetricsTab', () => {
  it('shows ROC curve', async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifactType = buildBasicArtifactType();
    const artifact = buildBasicArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set(
      'confidenceMetrics',
      new Value().setStructValue(
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
      ),
    );

    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockResolvedValueOnce([artifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValueOnce([artifactType]);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution}></MetricsTab>
      </CommonTestWrapper>,
    );

    getByText('Metrics is loading.');
    // We should upgrade react-scripts for capability to use libraries normally:
    // https://github.com/testing-library/dom-testing-library/issues/477
    await waitFor(() => getByText('ROC Curve: metrics'));
  });

  it('shows error banner when confidenceMetric type is wrong', async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifactType = buildBasicArtifactType();
    const artifact = buildBasicArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set(
      'confidenceMetrics',
      new Value().setStructValue(
        Struct.fromJavaScript({
          list: [
            {
              invalidType: 2,
              falsePositiveRate: 0,
              recall: 0,
            },
            {
              confidenceThreshold: 1,
              falsePositiveRate: 0,
              recall: 0.33962264150943394,
            },
          ],
        }),
      ),
    );

    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockResolvedValue([artifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([artifactType]);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution}></MetricsTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText("Error in metrics artifact's confidenceMetrics data format."));
  });

  it('shows error banner when confidenceMetric is not array', async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifactType = buildBasicArtifactType();
    const artifact = buildBasicArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set(
      'confidenceMetrics',
      new Value().setStructValue(
        Struct.fromJavaScript({
          variable: 'i am not a list',
        }),
      ),
    );

    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockResolvedValue([artifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([artifactType]);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution}></MetricsTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText("Error in metrics artifact's confidenceMetrics data format."));
  });

  it('has no metrics', async () => {
    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockReturnValue(Promise.resolve([]));
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue(Promise.resolve([]));
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution}></MetricsTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText('There is no metrics artifact available in this step.'));
  });

  it('shows info banner when execution is in UNKNOWN', async () => {
    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockReturnValue(Promise.resolve([]));
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue(Promise.resolve([]));

    const execution = buildBasicExecution().setLastKnownState(Execution.State.UNKNOWN);
    const { getByText, queryByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution}></MetricsTab>
      </CommonTestWrapper>,
    );
    await waitFor(() => getByText('Task is in unknown state.'));
    await waitFor(() => queryByText('Task has not completed.') === null);
  });

  it('shows info banner when execution is in NEW', async () => {
    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockReturnValue(Promise.resolve([]));
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue(Promise.resolve([]));

    const execution = buildBasicExecution().setLastKnownState(Execution.State.NEW);
    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution}></MetricsTab>
      </CommonTestWrapper>,
    );
    await waitFor(() => getByText('Task has not completed.'));
  });

  it('shows info banner when execution is in RUNNING', async () => {
    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockReturnValue(Promise.resolve([]));
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue(Promise.resolve([]));

    const execution = buildBasicExecution().setLastKnownState(Execution.State.RUNNING);
    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution}></MetricsTab>
      </CommonTestWrapper>,
    );
    await waitFor(() => getByText('Task has not completed.'));
  });
});

function buildBasicExecution() {
  const execution = new Execution();
  execution.setId(123);
  return execution;
}
function buildBasicArtifactType() {
  const artifactType = new ArtifactType();
  artifactType.setName('system.ClassificationMetrics');
  artifactType.setId(1);
  return artifactType;
}
function buildBasicArtifact() {
  const artifact = new Artifact();
  artifact.setTypeId(1);
  return artifact;
}
