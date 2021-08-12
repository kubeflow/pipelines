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
import { Link } from 'react-router-dom';
import { ListRequest } from 'src/lib/Apis';
import { ExecutionHelpers } from 'src/mlmd/MlmdUtils';
import { Api } from 'src/mlmd/library';
import {
  Execution,
  ExecutionType,
  GetExecutionsRequest,
  GetExecutionTypesRequest,
} from 'src/third_party/mlmd';
import { classes } from 'typestyle';
import CustomTable, {
  Column,
  CustomRendererProps,
  ExpandState,
  Row,
} from '../components/CustomTable';
import { RoutePageFactory } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { commonCss, padding } from '../Css';
import {
  CollapsedAndExpandedRows,
  getExpandedRow,
  groupRows,
  rowCompareFn,
  rowFilterFn,
  serviceErrorToString,
} from '../lib/Utils';
import { Page } from './Page';

interface ExecutionListState {
  executions: Execution[];
  rows: Row[];
  expandedRows: Map<number, Row[]>;
  columns: Column[];
}

class ExecutionList extends Page<{}, ExecutionListState> {
  private tableRef = React.createRef<CustomTable>();
  private api = Api.getInstance();
  private executionTypesMap: Map<number, ExecutionType>;

  constructor(props: any) {
    super(props);
    this.state = {
      columns: [
        {
          customRenderer: this.nameCustomRenderer,
          flex: 2,
          label: 'Run ID/Workspace/Pipeline',
          sortKey: 'workspace',
        },
        {
          customRenderer: this.nameCustomRenderer,
          flex: 1,
          label: 'Name',
          sortKey: 'name',
        },
        { label: 'State', flex: 1, sortKey: 'state' },
        { label: 'ID', flex: 1, sortKey: 'id' },
        { label: 'Type', flex: 2, sortKey: 'type' },
      ],
      executions: [],
      expandedRows: new Map(),
      rows: [],
    };
    this.reload = this.reload.bind(this);
    this.toggleRowExpand = this.toggleRowExpand.bind(this);
    this.getExpandedExecutionsRow = this.getExpandedExecutionsRow.bind(this);
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [],
      pageTitle: 'Executions',
    };
  }

  public render(): JSX.Element {
    const { rows, columns } = this.state;
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <CustomTable
          ref={this.tableRef}
          columns={columns}
          rows={rows}
          disablePaging={true}
          disableSelection={true}
          reload={this.reload}
          initialSortColumn='pipelineName'
          initialSortOrder='asc'
          getExpandComponent={this.getExpandedExecutionsRow}
          toggleExpansion={this.toggleRowExpand}
          emptyMessage='No executions found.'
        />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this.tableRef.current) {
      await this.tableRef.current.reload();
    }
  }

  private async reload(request: ListRequest): Promise<string> {
    try {
      // TODO: Consider making an Api method for returning and caching types
      if (!this.executionTypesMap || !this.executionTypesMap.size) {
        this.executionTypesMap = await this.getExecutionTypes();
      }
      if (!this.state.executions.length) {
        const executions = await this.getExecutions();
        this.setState({ executions });
        this.clearBanner();
        const collapsedAndExpandedRows = this.getRowsFromExecutions(request, executions);
        this.setState({
          expandedRows: collapsedAndExpandedRows.expandedRows,
          rows: collapsedAndExpandedRows.collapsedRows,
        });
      }
    } catch (err) {
      this.showPageError(serviceErrorToString(err));
    }
    return '';
  }

  private async getExecutions(): Promise<Execution[]> {
    try {
      const response = await this.api.metadataStoreService.getExecutions(
        new GetExecutionsRequest(),
      );
      return response.getExecutionsList();
    } catch (err) {
      // Code === 5 means no record found in backend. This is a temporary workaround.
      // TODO: remove err.code !== 5 check when backend is fixed.
      if (err.code !== 5) {
        err.message = 'Failed getting executions: ' + err.message;
        throw err;
      }
    }
    return [];
  }

  private async getExecutionTypes(): Promise<Map<number, ExecutionType>> {
    try {
      const response = await this.api.metadataStoreService.getExecutionTypes(
        new GetExecutionTypesRequest(),
      );

      const executionTypesMap = new Map<number, ExecutionType>();

      response.getExecutionTypesList().forEach(executionType => {
        executionTypesMap.set(executionType.getId(), executionType);
      });

      return executionTypesMap;
    } catch (err) {
      this.showPageError(serviceErrorToString(err));
    }
    return new Map();
  }

  private nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    return (
      <Link
        onClick={e => e.stopPropagation()}
        className={commonCss.link}
        to={RoutePageFactory.executionDetails(Number(props.id))}
      >
        {props.value}
      </Link>
    );
  };

  /**
   * Temporary solution to apply sorting, filtering, and pagination to the
   * local list of executions until server-side handling is available
   * TODO: Replace once https://github.com/kubeflow/metadata/issues/73 is done.
   * @param request
   * @param executions
   */
  private getRowsFromExecutions(
    request: ListRequest,
    executions: Execution[],
  ): CollapsedAndExpandedRows {
    return groupRows(
      executions
        .map(execution => {
          // Flattens
          const executionType = this.executionTypesMap!.get(execution.getTypeId());
          const type = executionType ? executionType.getName() : execution.getTypeId();
          return {
            id: `${execution.getId()}`,
            otherFields: [
              ExecutionHelpers.getWorkspace(execution),
              ExecutionHelpers.getName(execution),
              ExecutionHelpers.getState(execution),
              execution.getId(),
              type,
            ],
          };
        })
        .filter(rowFilterFn(request))
        .sort(rowCompareFn(request, this.state.columns)),
    );
  }

  /**
   * Toggles the expansion state of a row
   * @param index
   */
  private toggleRowExpand(index: number): void {
    const { rows } = this.state;
    if (!rows[index]) {
      return;
    }
    rows[index].expandState =
      rows[index].expandState === ExpandState.EXPANDED
        ? ExpandState.COLLAPSED
        : ExpandState.EXPANDED;
    this.setState({ rows });
  }

  private getExpandedExecutionsRow(index: number): React.ReactNode {
    return getExpandedRow(this.state.expandedRows, this.state.columns)(index);
  }
}

export default ExecutionList;
