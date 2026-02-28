/**
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import TestUtils from 'src/TestUtils';
import { Apis } from 'src/lib/Apis';
import { V2beta1ListPipelinesResponse } from 'src/apisv2beta1/pipeline';
import { GettingStarted } from './GettingStarted';
import { PageProps } from './Page';

describe('GettingStarted page', () => {
  const updateBannerSpy = vi.fn();
  const updateToolbarSpy = vi.fn();
  const historyPushSpy = vi.fn();
  const pipelineListSpy = vi.spyOn(Apis.pipelineServiceApiV2, 'listPipelines');

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      GettingStarted,
      {} as any,
      {} as any,
      historyPushSpy,
      updateBannerSpy,
      null,
      updateToolbarSpy,
      null,
    );
  }

  beforeEach(() => {
    vi.resetAllMocks();
    const empty: V2beta1ListPipelinesResponse = {
      pipelines: [],
      total_size: 0,
    };
    pipelineListSpy.mockImplementation(() => Promise.resolve(empty));
  });

  it('initially renders documentation', () => {
    const { container } = render(<GettingStarted {...generateProps()} />);
    expect(container).toMatchSnapshot();
  });

  it('renders documentation with pipeline deep link after querying demo pipelines', async () => {
    let count = 0;
    pipelineListSpy.mockImplementation(() => {
      ++count;
      const response: V2beta1ListPipelinesResponse = {
        pipelines: [{ pipeline_id: `pipeline-id-${count}` }],
      };
      return Promise.resolve(response);
    });
    render(<GettingStarted {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(pipelineListSpy.mock.calls).toMatchSnapshot();
    expect(screen.getByRole('link', { name: 'Data passing in Python components' })).toHaveAttribute(
      'href',
      '#/pipelines/details/pipeline-id-1?',
    );
    expect(screen.getByRole('link', { name: 'DSL - Control structures' })).toHaveAttribute(
      'href',
      '#/pipelines/details/pipeline-id-2?',
    );
  });

  it('fallbacks to show pipeline list page if request failed', async () => {
    let count = 0;
    pipelineListSpy.mockImplementation(
      (): Promise<V2beta1ListPipelinesResponse> => {
        ++count;
        if (count === 1) {
          return Promise.reject(new Error('Mocked error'));
        }
        return Promise.resolve({
          pipelines: [{ pipeline_id: `pipeline-id-${count}` }],
          total_size: 1,
        });
      },
    );
    render(<GettingStarted {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(screen.getByRole('link', { name: 'Data passing in Python components' })).toHaveAttribute(
      'href',
      '#/pipelines',
    );
    expect(screen.getByRole('link', { name: 'DSL - Control structures' })).toHaveAttribute(
      'href',
      '#/pipelines/details/pipeline-id-2?',
    );
  });
});
