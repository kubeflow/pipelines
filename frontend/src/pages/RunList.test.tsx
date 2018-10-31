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
import RunList, { RunListProps } from './RunList';
import { Apis } from '../lib/Apis';
import { shallow } from 'enzyme';
import { ApiRun, ApiRunDetail, ApiResourceType } from '../apis/run';
jest.mock('../lib/Apis');

describe('RunList', () => {
  const onErrorSpy = jest.fn();
  function generateProps(): RunListProps {
    return {
      history: {} as any,
      location: '' as any,
      match: '' as any,
      onError: onErrorSpy,
    };
  }

  beforeEach(() => {
    onErrorSpy.mockClear();
  });

  it('renders the empty experience', () => {
    expect(shallow(<RunList {...generateProps()} />)).toMatchSnapshot();
  });

  it('loads all runs', async () => {
    (Apis as any).runServiceApi = {
      getRun: () => Promise.resolve({
        pipeline_runtime: { workflow_manifest: '' },
        run: {},
      } as ApiRunDetail),
      listRuns: () => Promise.resolve({
        runs: [{
          id: 'testrun1',
          name: 'test run1',
        } as ApiRun],
      }),
    };
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('calls error callback when loading runs fails', async () => {
    (Apis as any).runServiceApi = {
      getRun: jest.fn(),
      listRuns: () => Promise.reject('bad stuff'),
    };
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    expect(props.onError).toHaveBeenLastCalledWith('Error: failed to fetch runs.', 'bad stuff');
  });

  // TODO: Add tests for displaying associated Pipeline.

  it('displays error in run row if it failed to parse', async () => {
    (Apis as any).runServiceApi = {
      getRun: jest.fn(() => Promise.resolve({
        pipeline_runtime: { workflow_manifest: 'bad json' },
        run: {},
      })),
      listRuns: () => Promise.resolve({
        runs: [{
          id: 'testrun1',
          name: 'test run1',
        } as ApiRun],
      }),
    };
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    expect(tree).toMatchSnapshot();
  });

  it('displays error in run row if it failed to parse (run list mask)', async () => {
    const listRunsSpy = jest.fn();
    const getRunSpy = jest.fn(() => Promise.resolve({
      pipeline_runtime: { workflow_manifest: '' },
      run: {},
    } as ApiRunDetail));
    (Apis as any).runServiceApi = {
      getRun: getRunSpy,
      listRuns: listRunsSpy,
    };
    const props = generateProps();
    props.runIdListMask = ['run1', 'run2'];
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    expect(tree).toMatchSnapshot();
  });

  it('shows run time for each run', async () => {
    (Apis as any).runServiceApi = {
      getRun: () => Promise.resolve({
        pipeline_runtime: {
          workflow_manifest: JSON.stringify({
            status: {
              finishedAt: new Date(2018, 10, 10, 11, 11, 11),
              phase: 'Succeeded',
              startedAt: new Date(2018, 10, 10, 10, 10, 10),
            }
          }),
        },
        run: {},
      } as ApiRunDetail),
      listRuns: () => Promise.resolve({
        runs: [{
          id: 'testrun1',
          name: 'test run1',
        } as ApiRun],
      }),
    };
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('loads runs for a given experiment id', async () => {
    const listRunsSpy = jest.fn(() => Promise.resolve({
      runs: [{ id: 'testRun1' }],
    }));
    (Apis as any).runServiceApi = {
      getRun: () => Promise.resolve({
        pipeline_runtime: { workflow_manifest: '' },
        run: {},
      } as ApiRunDetail),
      listRuns: listRunsSpy,
    };
    const props = generateProps();
    props.experimentIdMask = 'experiment1';
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    expect(props.onError).not.toHaveBeenCalled();
    expect(listRunsSpy).toHaveBeenLastCalledWith(
      '', 10, 'created_at desc', ApiResourceType.EXPERIMENT.toString(), 'experiment1');
    expect(tree).toMatchSnapshot();
  });

  it('loads given list of runs only', async () => {
    const listRunsSpy = jest.fn();
    const getRunSpy = jest.fn(() => Promise.resolve({
      pipeline_runtime: { workflow_manifest: '' },
      run: {},
    } as ApiRunDetail));
    (Apis as any).runServiceApi = {
      getRun: getRunSpy,
      listRuns: listRunsSpy,
    };
    const props = generateProps();
    props.runIdListMask = ['run1', 'run2'];
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    expect(props.onError).not.toHaveBeenCalled();
    expect(listRunsSpy).not.toHaveBeenCalled();
    expect(getRunSpy).toHaveBeenCalledWith('run1');
    expect(getRunSpy).toHaveBeenCalledWith('run2');
  });

});
