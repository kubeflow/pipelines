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
import { GettingStarted } from './GettingStarted';
import TestUtils, { diffHTML } from 'src/TestUtils';
import { render } from '@testing-library/react';
import { PageProps } from './Page';
import { Apis } from 'src/lib/Apis';
import { V2beta1ListPipelinesResponse } from 'src/apisv2beta1/pipeline';

const PATH_BACKEND_CONFIG = '../../../backend/src/apiserver/config/sample_config.json';
const PATH_FRONTEND_CONFIG = 'src/config/sample_config_from_backend.json';
describe(`${PATH_FRONTEND_CONFIG}`, () => {
  it(`should be in sync with ${PATH_BACKEND_CONFIG}, if not please run "npm run sync-backend-sample-config" to update.`, () => {
    const configBackend = require(PATH_BACKEND_CONFIG);
    const configFrontend = require(PATH_FRONTEND_CONFIG);
    expect(configFrontend).toEqual(
      configBackend.pipelines.map((sample: any) => sample.displayName),
    );
  });
});

describe('GettingStarted page', () => {
  const updateBannerSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const pipelineListSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelines');

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
    jest.resetAllMocks();
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
    const { container } = render(<GettingStarted {...generateProps()} />);
    const base = container.innerHTML;
    await TestUtils.flushPromises();
    expect(pipelineListSpy.mock.calls).toMatchSnapshot();
    expect(diffHTML({ base, update: container.innerHTML })).toMatchInlineSnapshot(`
      Snapshot Diff:
      Compared values have no visual difference.
    `);
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
    const { container } = render(<GettingStarted {...generateProps()} />);
    const base = container.innerHTML;
    await TestUtils.flushPromises();
    expect(diffHTML({ base, update: container.innerHTML })).toMatchInlineSnapshot(`
      Snapshot Diff:
      Compared values have no visual difference.
    `);
  });
});
