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
import { QueryClient, QueryClientProvider } from 'react-query';
import { BrowserRouter } from 'react-router-dom';
import { MetricsTab } from './MetricsTab';

describe('MetricsTab', () => {
  const mlmdUtils = require('src/lib/MlmdUtils');
  const OutputArtifactLoader = require('src/lib/OutputArtifactLoader');
  const execution = new Execution();
  const queryClient = new QueryClient();

  beforeEach(() => {
    const executionGetIdSpy = jest.spyOn(execution, 'getId');
    executionGetIdSpy.mockReturnValue(123);
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  it('shows ROC curve', async () => {
    const artifact = new Artifact();
    const artifactType = new ArtifactType();

    const confidenceMetrics = {
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
    };
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact
      .getCustomPropertiesMap()
      .set(
        'confidenceMetrics',
        new Value().setStructValue(Struct.fromJavaScript(confidenceMetrics)),
      );

    jest
      .spyOn(mlmdUtils, 'getOutputArtifactsInExecution')
      .mockReturnValue(Promise.resolve([artifact]));
    jest
      .spyOn(OutputArtifactLoader, 'getArtifactTypes')
      .mockReturnValue(Promise.resolve([artifactType]));
    jest.spyOn(OutputArtifactLoader, 'filterArtifactsByType').mockReturnValue([artifact]);

    const { getByText } = render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <MetricsTab execution={execution}></MetricsTab>
        </QueryClientProvider>
      </BrowserRouter>,
    );

    getByText('Metrics is loading.');
    // We should upgrade react-scripts for capability to use libraries normally:
    // https://github.com/testing-library/dom-testing-library/issues/477
    await waitFor(() => getByText('ROC Curve'));
  });

  it('has no metrics', async () => {
    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockReturnValue(Promise.resolve([]));
    jest.spyOn(OutputArtifactLoader, 'getArtifactTypes').mockReturnValue(Promise.resolve([]));
    jest.spyOn(OutputArtifactLoader, 'filterArtifactsByType').mockReturnValue([]);

    const { getByText } = render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <MetricsTab execution={execution}></MetricsTab>
        </QueryClientProvider>
      </BrowserRouter>,
    );

    await waitFor(() => getByText('There is no metrics artifact available in this step.'));
  });
});
