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
import PipelineSelector, { PipelineSelectorProps } from './PipelineSelector';
import TestUtils from '../TestUtils';
import { ApiPipeline } from '../apis/pipeline';
import { ListRequest, Apis } from '../lib/Apis';
import { shallow } from 'enzyme';

describe('PipelineSelector', () => {
  class TestPipelineSelector extends PipelineSelector {
    public async _loadPipelines(request: ListRequest): Promise<string> {
      return super._loadPipelines(request);
    }
    public _pipelineSelectionChanged(selectedIds: string[]): void {
      return super._pipelineSelectionChanged(selectedIds);
    }
  }

  const updateDialogSpy = jest.fn();
  const pipelineSelectionChangedCbSpy = jest.fn();
  const listPipelinesSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');
  const PIPELINES: ApiPipeline[] = [{
    created_at: new Date(2018, 10, 9, 8, 7, 6),
    description: 'test pipeline description',
    name: 'test pipeline name',
  }];

  function generateProps(): PipelineSelectorProps {
    return {
      history: {} as any,
      location: '' as any,
      match: {} as any,
      pipelineSelectionChanged: pipelineSelectionChangedCbSpy,
      updateDialog: updateDialogSpy,
    };
  }

  beforeEach(() => {
    listPipelinesSpy.mockReset();
    listPipelinesSpy.mockImplementation(() => ({ pipelines: PIPELINES }));
    updateDialogSpy.mockReset();
    pipelineSelectionChangedCbSpy.mockReset();
  });

  it('calls API to load pipelines', async () => {
    const tree = shallow(<TestPipelineSelector {...generateProps()} />);
    await (tree.instance() as TestPipelineSelector)._loadPipelines({});
    expect(listPipelinesSpy).toHaveBeenCalledTimes(1);
    expect(listPipelinesSpy).toHaveBeenLastCalledWith(undefined, undefined, undefined);
    expect(tree.state('pipelines')).toEqual(PIPELINES);
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('shows error dialog if listing fails', async () => {
    TestUtils.makeErrorResponseOnce(listPipelinesSpy, 'woops!');
    jest.spyOn(console, 'error').mockImplementation();
    const tree = shallow(<TestPipelineSelector {...generateProps()} />);
    await (tree.instance() as TestPipelineSelector)._loadPipelines({});
    expect(listPipelinesSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      content: 'List pipelines request failed with:\nwoops!',
      title: 'Error retrieving pipelines',
    }));
    expect(tree.state('pipelines')).toEqual([]);
    tree.unmount();
  });

  it('calls selection callback when a pipeline is selected', async () => {
    const tree = shallow(<TestPipelineSelector {...generateProps()} />);
    await (tree.instance() as TestPipelineSelector)._loadPipelines({});
    expect(tree.state('selectedIds')).toEqual([]);
    (tree.instance() as TestPipelineSelector)._pipelineSelectionChanged(['pipeline-id']);
    expect(pipelineSelectionChangedCbSpy).toHaveBeenLastCalledWith('pipeline-id');
    expect(tree.state('selectedIds')).toEqual(['pipeline-id']);
    tree.unmount();
  });

  it('logs error if more than one pipeline is selected', async () => {
    const tree = shallow(<TestPipelineSelector {...generateProps()} />);
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    await (tree.instance() as TestPipelineSelector)._loadPipelines({});
    expect(tree.state('selectedIds')).toEqual([]);
    (tree.instance() as TestPipelineSelector)._pipelineSelectionChanged(['pipeline-id', 'pipeline2-id']);
    expect(pipelineSelectionChangedCbSpy).not.toHaveBeenCalled();
    expect(tree.state('selectedIds')).toEqual([]);
    expect(consoleSpy).toHaveBeenCalled();
    tree.unmount();
  });
});
