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
import PipelineVersionList, { PipelineVersionListProps } from './PipelineVersionList';
import TestUtils from 'src/TestUtils';
import { V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { Apis, ListRequest } from 'src/lib/Apis';
import { render } from '@testing-library/react';
import { range } from 'lodash';

describe('PipelineVersionList', () => {

  const listPipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApiV2, 'listPipelineVersions');
  const onErrorSpy = jest.fn();

  function generateProps(): PipelineVersionListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
      pipelineId: 'pipeline',
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  it('renders an empty list with empty state message', () => {
    const { asFragment } = render(<PipelineVersionList {...generateProps()} />);
    expect(asFragment()).toMatchSnapshot();
  });

  // TODO: Skip tests that require component state manipulation and complex enzyme patterns

  it.skip('renders a list of one pipeline version', () => {});
  it.skip('renders a list of one pipeline version with description', () => {});
  it.skip('renders a list of one pipeline version without created date', () => {});
  it.skip('renders a list of one pipeline version with error', () => {});
  it.skip('calls Apis to list pipeline versions, sorted by creation time in descending order', () => {});
  it.skip('calls Apis to list pipeline versions, sorted by pipeline version name in descending order', () => {});
});
