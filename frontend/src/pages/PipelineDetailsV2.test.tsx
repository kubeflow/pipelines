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

import { render, screen } from '@testing-library/react';
import * as React from 'react';
import { CommonTestWrapper } from 'src/TestWrapper';
import { mockResizeObserver, testBestPractices } from '../TestUtils';
import PipelineDetailsV2 from './PipelineDetailsV2';

testBestPractices();
describe('PipelineDetailsV2', () => {
  beforeEach(() => {
    mockResizeObserver();
  });

  it('Render detail page with reactflow', async () => {
    render(
      <CommonTestWrapper>
        <PipelineDetailsV2 pipelineFlowElements={[]}></PipelineDetailsV2>
      </CommonTestWrapper>,
    );
    expect(screen.getByTestId('StaticCanvas')).not.toBeNull();
  });
});
