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

import { Execution, Value } from '@kubeflow/frontend';
import { render, screen } from '@testing-library/react';
import React from 'react';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { ExecutionTitle } from './ExecutionTitle';

testBestPractices();
describe('ExecutionTitle', () => {
  const execution = new Execution();
  const executionName = 'fake-execution';
  const executionId = 123;
  beforeEach(() => {
    execution.setId(executionId);
    execution.getCustomPropertiesMap().set('task_name', new Value().setStringValue(executionName));
  });

  it('Shows execution name', () => {
    render(
      <CommonTestWrapper>
        <ExecutionTitle execution={execution}></ExecutionTitle>
      </CommonTestWrapper>,
    );
    screen.getByText(executionName, { selector: 'a', exact: false });
  });

  it('Shows execution description', () => {
    render(
      <CommonTestWrapper>
        <ExecutionTitle execution={execution}></ExecutionTitle>
      </CommonTestWrapper>,
    );
    screen.getByText('This step corresponds to execution');
  });
});
