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
import { testBestPractices } from 'src/TestUtils';
import CompareV2 from './CompareV2';

testBestPractices();
describe('CompareV2', () => {
  it('Render Compare v2 page', async () => {
    render(
      <CompareV2 />,
    );
    screen.getByText('This is the V2 Run Comparison page.');
  });
});
