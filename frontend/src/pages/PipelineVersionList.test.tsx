/*
 * Copyright 2019 Google LLC
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
import TestUtils from '../TestUtils';
import { ApiPipelineVersion } from '../apis/pipeline';
import { Apis, ListRequest } from '../lib/Apis';
import { shallow, ReactWrapper, ShallowWrapper } from 'enzyme';
import { range } from 'lodash';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate HoC receive the t function as a prop
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: () => '' };
    return Component;
  },
}));

class PipelineVersionListTest extends PipelineVersionList {
  public _loadPipelineVersions(request: ListRequest): Promise<string> {
    return super._loadPipelineVersions(request);
  }
}

describe('PipelineVersionList', () => {
  let tree: ReactWrapper | ShallowWrapper;

  const listPipelineVersionsSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelineVersions');
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

  async function mountWithNPipelineVersions(n: number): Promise<ReactWrapper> {
    listPipelineVersionsSpy.mockImplementation((pipelineId: string) => ({
      versions: range(n).map(i => ({
        id: 'test-pipeline-version-id' + i,
        name: 'test pipeline version name' + i,
      })),
    }));
    tree = TestUtils.mountWithRouter(<PipelineVersionList {...generateProps()} />);
    await listPipelineVersionsSpy;
    await TestUtils.flushPromises();
    tree.update(); // Make sure the tree is updated before returning it
    return tree;
  }

  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    await tree.unmount();
    jest.resetAllMocks();
  });

  it('renders an empty list with empty state message', () => {
    tree = shallow(<PipelineVersionList {...generateProps()} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one pipeline version', async () => {
    tree = shallow(<PipelineVersionList {...generateProps()} />);
    tree.setState({
      pipelineVersions: [
        {
          created_at: new Date(2018, 8, 22, 11, 5, 48),
          name: 'pipelineversion1',
        } as ApiPipelineVersion,
      ],
    });
    await listPipelineVersionsSpy;
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one pipeline version without created date', async () => {
    tree = shallow(<PipelineVersionList {...generateProps()} />);
    tree.setState({
      pipelines: [
        {
          name: 'pipelineversion1',
        } as ApiPipelineVersion,
      ],
    });
    await listPipelineVersionsSpy;
    expect(tree).toMatchSnapshot();
  });

  it('renders a list of one pipeline version with error', async () => {
    tree = shallow(<PipelineVersionList {...generateProps()} />);
    tree.setState({
      pipelineVersions: [
        {
          created_at: new Date(2018, 8, 22, 11, 5, 48),
          error: 'oops! could not load pipeline',
          name: 'pipeline1',
          parameters: [],
        } as ApiPipelineVersion,
      ],
    });
    await listPipelineVersionsSpy;
    expect(tree).toMatchSnapshot();
  });

  it('calls Apis to list pipeline versions, sorted by creation time in descending order', async () => {
    tree = await mountWithNPipelineVersions(2);
    await (tree.instance() as PipelineVersionListTest)._loadPipelineVersions({
      pageSize: 10,
      pageToken: '',
      sortBy: 'created_at',
    } as ListRequest);
    expect(listPipelineVersionsSpy).toHaveBeenLastCalledWith(
      'PIPELINE',
      'pipeline',
      10,
      '',
      'created_at',
    );
    expect(tree).toMatchSnapshot();
  });

  it('calls Apis to list pipeline versions, sorted by pipeline version name in descending order', async () => {
    tree = await mountWithNPipelineVersions(3);
    await (tree.instance() as PipelineVersionListTest)._loadPipelineVersions({
      pageSize: 10,
      pageToken: '',
      sortBy: 'name',
    } as ListRequest);
    expect(listPipelineVersionsSpy).toHaveBeenLastCalledWith(
      'PIPELINE',
      'pipeline',
      10,
      '',
      'name',
    );
    expect(tree).toMatchSnapshot();
  });
});
