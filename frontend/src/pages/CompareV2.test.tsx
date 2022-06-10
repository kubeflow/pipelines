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
import { QUERY_PARAMS } from 'src/components/Router';
import CompareV2 from './CompareV2';
import { PageProps } from './Page';

testBestPractices();
describe('CompareV2', () => {
  const MOCK_RUN_1_ID = 'mock-run-1-id';
  const MOCK_RUN_2_ID = 'mock-run-2-id';
  const MOCK_RUN_3_ID = 'mock-run-3-id';
  const updateBannerSpy = jest.fn();

  function generateProps(): PageProps {
    const pageProps: PageProps = {
      history: {} as any,
      location: {
        search: `?${QUERY_PARAMS.runlist}=${MOCK_RUN_1_ID},${MOCK_RUN_2_ID},${MOCK_RUN_3_ID}`,
      } as any,
      match: {} as any,
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: () => null,
      updateSnackbar: () => null,
      updateToolbar: () => null,
    };
    return pageProps;
  }

  it('Render Compare v2 page', async () => {
    render(
      <CommonTestWrapper>
        <CompareV2 {...generateProps()} />
      </CommonTestWrapper>,
    );
    screen.getByText('Parameter Section V2');
  });
});
