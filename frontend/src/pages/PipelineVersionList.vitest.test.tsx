/*
 * Copyright 2019 The Kubeflow Authors
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

import * as React from 'react';
import { render, waitFor } from '@testing-library/react';
import { range } from 'lodash';
import { V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { Apis, ListRequest } from 'src/lib/Apis';
import TestUtils from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import PipelineVersionList, { PipelineVersionListProps } from './PipelineVersionList';
import { vi } from 'vitest';

describe('PipelineVersionList', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let pipelineVersionListRef: React.RefObject<PipelineVersionList> | null = null;

  let listPipelineVersionsSpy: ReturnType<typeof vi.spyOn>;
  let onErrorSpy: ReturnType<typeof vi.fn>;

  function spyInit() {
    onErrorSpy = vi.fn();
    listPipelineVersionsSpy = vi
      .spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions')
      .mockResolvedValue({ pipeline_versions: [] });
  }

  function generateProps(): PipelineVersionListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
      pipelineId: 'pipeline',
    };
  }

  async function waitForVersionsLoad(): Promise<void> {
    await waitFor(() => {
      expect(listPipelineVersionsSpy).toHaveBeenCalled();
    });
    await TestUtils.flushPromises();
  }

  async function renderPipelineVersionList(
    customProps?: Partial<PipelineVersionListProps>,
  ): Promise<void> {
    pipelineVersionListRef = React.createRef<PipelineVersionList>();
    const props = { ...generateProps(), ...customProps } as PipelineVersionListProps;
    renderResult = render(
      <CommonTestWrapper>
        <PipelineVersionList ref={pipelineVersionListRef} {...props} />
      </CommonTestWrapper>,
    );
    await waitForVersionsLoad();
  }

  async function mountWithNPipelineVersions(n: number): Promise<void> {
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: range(n).map(i => ({
        pipeline_version_id: 'test-pipeline-version-id' + i,
        display_name: 'test pipeline version name' + i,
      })),
    });
    await renderPipelineVersionList();
  }

  beforeEach(() => {
    spyInit();
  });

  afterEach(() => {
    if (renderResult) {
      renderResult.unmount();
      renderResult = null;
    }
    pipelineVersionListRef = null;
    vi.resetAllMocks();
  });

  it('renders an empty list with empty state message', async () => {
    await renderPipelineVersionList();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('renders a list of one pipeline version', async () => {
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [
        {
          created_at: new Date(2018, 8, 22, 11, 5, 48),
          display_name: 'pipelineversion1',
          name: 'pipelineversion1',
        } as V2beta1PipelineVersion,
      ],
    });
    await renderPipelineVersionList();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('renders a list of one pipeline version with description', async () => {
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [
        {
          created_at: new Date(2018, 8, 22, 11, 5, 48),
          display_name: 'pipelineversion1',
          name: 'pipelineversion1',
          description: 'pipelineversion1 description',
        } as V2beta1PipelineVersion,
      ],
    });
    await renderPipelineVersionList();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('renders a list of one pipeline version without created date', async () => {
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [
        {
          display_name: 'pipelineversion1',
          name: 'pipelineversion1',
        } as V2beta1PipelineVersion,
      ],
    });
    await renderPipelineVersionList();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('renders a list of one pipeline version with error', async () => {
    listPipelineVersionsSpy.mockResolvedValue({
      pipeline_versions: [
        {
          created_at: new Date(2018, 8, 22, 11, 5, 48),
          error: 'oops! could not load pipeline',
          display_name: 'pipeline1',
          name: 'pipeline1',
        } as V2beta1PipelineVersion,
      ],
    });
    await renderPipelineVersionList();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('calls Apis to list pipeline versions, sorted by creation time in descending order', async () => {
    await mountWithNPipelineVersions(2);
    await (pipelineVersionListRef?.current as any)._loadPipelineVersions({
      pageSize: 10,
      pageToken: '',
      sortBy: 'created_at',
    } as ListRequest);
    expect(listPipelineVersionsSpy).toHaveBeenLastCalledWith(
      'pipeline',
      '',
      10,
      'created_at',
      undefined,
    );
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('calls Apis to list pipeline versions, sorted by pipeline version name in descending order', async () => {
    await mountWithNPipelineVersions(3);
    await (pipelineVersionListRef?.current as any)._loadPipelineVersions({
      pageSize: 10,
      pageToken: '',
      sortBy: 'name',
    } as ListRequest);
    expect(listPipelineVersionsSpy).toHaveBeenLastCalledWith('pipeline', '', 10, 'name', undefined);
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });
});
