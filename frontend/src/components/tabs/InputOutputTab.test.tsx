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

import { Artifact, Execution, Value } from '@kubeflow/frontend';
import { render, waitFor } from '@testing-library/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import React from 'react';
import * as mlmdUtils from 'src/lib/MlmdUtils';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import InputOutputTab from './InputOutputTab';

const executionName = 'fake-execution';
const artifactName = 'artifactName';
const artifactUri = 'gs://test';

testBestPractices();
describe('InoutOutputTab', () => {
  it('shows execution title', () => {
    const { getByText } = render(
      <CommonTestWrapper>
        <InputOutputTab execution={buildBasicExecution()}></InputOutputTab>
      </CommonTestWrapper>,
    );
    getByText(executionName, { selector: 'a', exact: false });
  });

  it('shows Input/Output artifacts and parameters title', () => {
    const { getAllByText, getByText } = render(
      <CommonTestWrapper>
        <InputOutputTab execution={buildBasicExecution()}></InputOutputTab>
      </CommonTestWrapper>,
    );
    getByText('Input');
    getByText('Output');
    expect(getAllByText('Parameters').length).toEqual(2);
    expect(getAllByText('Artifacts').length).toEqual(2);
  });

  it('shows Input parameters with various types', async () => {
    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockResolvedValueOnce([]);
    jest.spyOn(mlmdUtils, 'getInputArtifactsInExecution').mockResolvedValueOnce([]);

    const execution = buildBasicExecution();
    execution
      .getCustomPropertiesMap()
      .set('thisKeyIsNotInput', new Value().setStringValue("value shouldn't show"));
    execution
      .getCustomPropertiesMap()
      .set('input:stringkey', new Value().setStringValue('string input'));
    execution.getCustomPropertiesMap().set('input:intkey', new Value().setIntValue(42));
    execution.getCustomPropertiesMap().set('input:doublekey', new Value().setDoubleValue(1.99));
    execution
      .getCustomPropertiesMap()
      .set(
        'input:structkey',
        new Value().setStructValue(Struct.fromJavaScript({ struct: { key: 'value', num: 42 } })),
      );
    execution
      .getCustomPropertiesMap()
      .set(
        'input:arraykey',
        new Value().setStructValue(Struct.fromJavaScript({ list: ['a', 'b', 'c'] })),
      );
    const { queryByText, getByText } = render(
      <CommonTestWrapper>
        <InputOutputTab execution={execution}></InputOutputTab>
      </CommonTestWrapper>,
    );

    getByText('stringkey');
    getByText('string input');
    getByText('intkey');
    getByText('42');
    getByText('doublekey');
    getByText('1.99');
    getByText('structkey');
    getByText('arraykey');
    expect(queryByText('thisKeyIsNotInput')).toBeNull();
  });

  it('shows Output parameters with various types', async () => {
    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockResolvedValueOnce([]);
    jest.spyOn(mlmdUtils, 'getInputArtifactsInExecution').mockResolvedValueOnce([]);

    const execution = buildBasicExecution();
    execution
      .getCustomPropertiesMap()
      .set('thisKeyIsNotOutput', new Value().setStringValue("value shouldn't show"));
    execution
      .getCustomPropertiesMap()
      .set('output:stringkey', new Value().setStringValue('string output'));
    execution.getCustomPropertiesMap().set('output:intkey', new Value().setIntValue(42));
    execution.getCustomPropertiesMap().set('output:doublekey', new Value().setDoubleValue(1.99));
    execution
      .getCustomPropertiesMap()
      .set(
        'output:structkey',
        new Value().setStructValue(Struct.fromJavaScript({ struct: { key: 'value', num: 42 } })),
      );
    execution
      .getCustomPropertiesMap()
      .set(
        'output:arraykey',
        new Value().setStructValue(Struct.fromJavaScript({ list: ['a', 'b', 'c'] })),
      );
    const { queryByText, getByText } = render(
      <CommonTestWrapper>
        <InputOutputTab execution={execution}></InputOutputTab>
      </CommonTestWrapper>,
    );

    getByText('stringkey');
    getByText('string output');
    getByText('intkey');
    getByText('42');
    getByText('doublekey');
    getByText('1.99');
    getByText('structkey');
    getByText('arraykey');
    expect(queryByText('thisKeyIsNotOutput')).toBeNull();
  });

  it('shows Input artifacts', async () => {
    const artifact = buildArtifact();
    jest.spyOn(mlmdUtils, 'getInputArtifactsInExecution').mockResolvedValueOnce([artifact]);
    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockResolvedValueOnce([]);
    const { getByText } = render(
      <CommonTestWrapper>
        <InputOutputTab execution={buildBasicExecution()}></InputOutputTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText(artifactName));
    await waitFor(() => getByText(artifactUri));
  });

  it('shows Output artifacts', async () => {
    const artifact = buildArtifact();
    jest.spyOn(mlmdUtils, 'getInputArtifactsInExecution').mockResolvedValueOnce([]);
    jest.spyOn(mlmdUtils, 'getOutputArtifactsInExecution').mockResolvedValueOnce([artifact]);
    const { getByText } = render(
      <CommonTestWrapper>
        <InputOutputTab execution={buildBasicExecution()}></InputOutputTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText(artifactName));
    await waitFor(() => getByText(artifactUri));
  });
});

function buildBasicExecution() {
  const execution = new Execution();
  const executionId = 123;

  execution.setId(executionId);
  execution.getCustomPropertiesMap().set('task_name', new Value().setStringValue(executionName));

  return execution;
}

function buildArtifact() {
  const artifact = new Artifact();
  artifact.getCustomPropertiesMap().set('name', new Value().setStringValue(artifactName));
  artifact.setUri(artifactUri);
  return artifact;
}
