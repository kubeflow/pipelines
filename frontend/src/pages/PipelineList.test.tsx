/*
 * Copyright 2018 Google LLC
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
import PipelineList from './PipelineList';
import { PageProps } from './Page';
import { shallow } from 'enzyme';
import { Apis } from '../lib/Apis';

describe('PipelineList', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const listPipelinesSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');

  function generateProps(): PageProps {
    return {
      history: {} as any,
      location: '' as any,
      match: '' as any,
      toolbarProps: { actions: [], breadcrumbs: [] },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  it('renders an empty list with empty state message', () => {
    expect(shallow(<PipelineList {...generateProps()} />)).toMatchSnapshot();
  });

  it('calls Apis to list pipelines, sorted by creation time in descending order', () => {
    const tree = shallow(<PipelineList {...generateProps()} />);
    (tree.instance() as PipelineList).load();
    expect(listPipelinesSpy).toHaveBeenLastCalledWith('', 10, 'created_at desc');
  });

});
