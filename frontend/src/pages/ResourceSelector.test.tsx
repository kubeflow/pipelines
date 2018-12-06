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
import ResourceSelector, { ResourceSelectorProps } from './ResourceSelector';
import TestUtils from '../TestUtils';
import { ApiPipeline } from '../apis/pipeline';
import { ListRequest, Apis, PipelineSortKeys, ExperimentSortKeys } from '../lib/Apis';
import { shallow, ReactWrapper, ShallowWrapper } from 'enzyme';
import { ApiExperiment } from '../apis/experiment';
import { formatDateString } from '../lib/Utils';
import { Row } from '../components/CustomTable';

class TestResourceSelector extends ResourceSelector {
  public async _load(request: ListRequest): Promise<string> {
    return super._load(request);
  }
  public _selectionChanged(selectedIds: string[]): void {
    return super._selectionChanged(selectedIds);
  }
}

describe('ResourceSelector', () => {
  let tree: ReactWrapper | ShallowWrapper;

  const updateDialogSpy = jest.fn();
  const selectionChangedCbSpy = jest.fn();
  const listExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'listExperiment');
  const EXPERIMENTS: ApiExperiment[] = [
    {
      created_at: new Date(2018, 1, 2, 3, 4, 5),
      description: 'test experiment-1 description',
      id: 'some-experiment-1-id',
      name: 'test experiment-1 name',
    }, {
      created_at: new Date(2018, 10, 9, 8, 7, 6),
      description: 'test experiment-2 description',
      id: 'some-experiment-2-id',
      name: 'test experiment-2 name',
    }
  ];
  const listPipelinesSpy = jest.spyOn(Apis.pipelineServiceApi, 'listPipelines');
  const PIPELINES: ApiPipeline[] = [
    {
      created_at: new Date(2018, 1, 2, 3, 4, 5),
      description: 'test pipeline-1 description',
      id: 'some-pipeline-id-1',
      name: 'test pipeline-1 name',
    }, {
      created_at: new Date(2018, 10, 9, 8, 7, 6),
      description: 'test pipeline-2 description',
      id: 'some-pipeline-2-id',
      name: 'test pipeline-2 name',
    }
  ];

  const pipelineSelectorColumns = [
    { label: 'Pipeline name', flex: 1, sortKey: PipelineSortKeys.NAME },
    { label: 'Description', flex: 1.5 },
    { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
  ];

  const experimentSelectorColumns = [
    { label: 'Experiment name', flex: 1, sortKey: ExperimentSortKeys.NAME },
    { label: 'Description', flex: 1.5 },
    { label: 'Created at', flex: 1, sortKey: ExperimentSortKeys.CREATED_AT },
  ];

  const testEmptyMessage = 'Test - Sorry, no pipelines.';
  const testTitle = 'A test selector';

  const resourceToRow = (r: ApiExperiment | ApiPipeline) => {
    return {
      // error does not exist (yet) on ApiExperiment
      error: (r as any).error,
      id: r.id!,
      otherFields: [
        r.name,
        r.description,
        formatDateString(r.created_at),
      ],
    } as Row;
  };

  function generateProps(): ResourceSelectorProps {
    return {
      columns: pipelineSelectorColumns,
      emptyMessage: testEmptyMessage,
      history: {} as any,
      initialSortColumn: PipelineSortKeys.CREATED_AT,
      listApi: listPipelinesSpy as any,
      location: '' as any,
      match: {} as any,
      resourceToRow,
      selectionChanged: selectionChangedCbSpy,
      title: testTitle,
      updateDialog: updateDialogSpy,
    };
  }

  beforeEach(() => {
    listExperimentSpy.mockReset();
    listExperimentSpy.mockImplementation(() => ({ experiments: EXPERIMENTS }));
    listPipelinesSpy.mockReset();
    listPipelinesSpy.mockImplementation(() => ({ pipelines: PIPELINES }));
    updateDialogSpy.mockReset();
    selectionChangedCbSpy.mockReset();
  });

  afterEach(() => {
    tree.unmount();
  });

  it('displays pipeline selector UI when configured for pipelines', async () => {
    const props = generateProps();
    props.columns = pipelineSelectorColumns;
    props.listApi = listPipelinesSpy as any;
    props.initialSortColumn = PipelineSortKeys.CREATED_AT;

    tree = shallow(<TestResourceSelector {...props} />);
    await (tree.instance() as TestResourceSelector)._load({});

    expect(listPipelinesSpy).toHaveBeenCalledTimes(1);
    expect(listPipelinesSpy).toHaveBeenLastCalledWith(undefined, undefined, undefined);
    expect(tree.state('resources')).toEqual(PIPELINES);
    expect(tree).toMatchSnapshot();
  });

  it('calls the provided helper function for converting a resource into a table row', async () => {
    const props = generateProps();
    props.columns = pipelineSelectorColumns;
    props.initialSortColumn = PipelineSortKeys.CREATED_AT;
    const pipelines: ApiPipeline[] = [
      {
        created_at: new Date(2018, 1, 2, 3, 4, 5),
        description: 'a description',
        id: 'an-id',
        name: 'a name',
      }
    ];
    listPipelinesSpy.mockImplementationOnce(() => ({ pipelines }));
    props.listApi = listPipelinesSpy as any;

    props.resourceToRow = (r: ApiPipeline) => {
      return {
        id: r.id! + ' - test',
        otherFields: [
          r.name + ' - test',
          r.description + ' - test',
          formatDateString(r.created_at),
        ],
      } as Row;
    };

    tree = shallow(<TestResourceSelector {...props} />);
    await (tree.instance() as TestResourceSelector)._load({});

    expect(tree.state('rows')).toEqual([{
      id: 'an-id - test',
      otherFields: [
        'a name - test',
        'a description - test',
        '2/2/2018, 3:04:05 AM',
      ],
    }]);
  });

  it('displays experiment selector UI when configured for experiments', async () => {
    const props = generateProps();
    props.columns = experimentSelectorColumns;
    props.listApi = listExperimentSpy as any;
    props.initialSortColumn = ExperimentSortKeys.CREATED_AT;

    tree = shallow(<TestResourceSelector {...props} />);
    await (tree.instance() as TestResourceSelector)._load({});

    expect(listExperimentSpy).toHaveBeenCalledTimes(1);
    expect(listExperimentSpy).toHaveBeenLastCalledWith(undefined, undefined, undefined);
    expect(tree.state('resources')).toEqual(EXPERIMENTS);
    expect(tree).toMatchSnapshot();
  });

  it('shows error dialog if listing fails', async () => {
    TestUtils.makeErrorResponseOnce(listPipelinesSpy, 'woops!');
    jest.spyOn(console, 'error').mockImplementation();

    tree = shallow(<TestResourceSelector {...generateProps()} />);
    await (tree.instance() as TestResourceSelector)._load({});

    expect(listPipelinesSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      content: 'List request failed with:\nwoops!',
      title: 'Error retrieving resources',
    }));
    expect(tree.state('resources')).toEqual([]);
  });

  it('calls selection callback when a resource is selected', async () => {
    tree = shallow(<TestResourceSelector {...generateProps()} />);
    await (tree.instance() as TestResourceSelector)._load({});

    expect(tree.state('selectedIds')).toEqual([]);
    (tree.instance() as TestResourceSelector)._selectionChanged([PIPELINES[1].id!]);
    expect(selectionChangedCbSpy).toHaveBeenLastCalledWith(PIPELINES[1]);
    expect(tree.state('selectedIds')).toEqual([PIPELINES[1].id]);
  });

  it('logs error if more than one resource is selected', async () => {
    tree = shallow(<TestResourceSelector {...generateProps()} />);
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    await (tree.instance() as TestResourceSelector)._load({});

    expect(tree.state('selectedIds')).toEqual([]);

    (tree.instance() as TestResourceSelector)._selectionChanged([PIPELINES[0].id!, PIPELINES[1].id!]);

    expect(selectionChangedCbSpy).not.toHaveBeenCalled();
    expect(tree.state('selectedIds')).toEqual([]);
    expect(consoleSpy).toHaveBeenLastCalledWith(
      '2 resources were selected somehow', [PIPELINES[0].id, PIPELINES[1].id]);
  });

  it('logs error if selected resource ID is not found in list', async () => {
    tree = shallow(<TestResourceSelector {...generateProps()} />);
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    await (tree.instance() as TestResourceSelector)._load({});

    expect(tree.state('selectedIds')).toEqual([]);

    (tree.instance() as TestResourceSelector)._selectionChanged(['id-not-in-list']);

    expect(selectionChangedCbSpy).not.toHaveBeenCalled();
    expect(tree.state('selectedIds')).toEqual([]);
    expect(consoleSpy).toHaveBeenLastCalledWith(
      'Somehow no resource was found with ID: id-not-in-list');
  });

  it('logs error if list response contains neither experiments nor pipelines', async () => {
    listPipelinesSpy.mockImplementationOnce(() => ({}));

    tree = shallow(<TestResourceSelector {...generateProps()} />);
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    await (tree.instance() as TestResourceSelector)._load({});

    expect(listPipelinesSpy).toHaveBeenCalledTimes(1);
    expect(consoleSpy).toHaveBeenLastCalledWith(
      'Somehow response contained neither experiments nor pipelines'
    );
    expect(tree.state('resources')).toEqual([]);
  });
});
