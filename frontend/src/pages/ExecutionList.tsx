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
import CustomTable, {
  Column,
  Row,
  ExpandState,
  CustomRendererProps,
} from '../components/CustomTable';
import { Page } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import {
  getResourceProperty,
  rowCompareFn,
  rowFilterFn,
  groupRows,
  getExpandedRow,
  serviceErrorToString,
} from '../lib/Utils';
import { Link } from 'react-router-dom';
import { RoutePage, RouteParams } from '../components/Router';
import { Execution, ExecutionType } from '../generated/src/apis/metadata/metadata_store_pb';
import { ListRequest, ExecutionProperties, ExecutionCustomProperties, Apis } from '../lib/Apis';
import {
  GetExecutionsRequest,
  GetExecutionTypesRequest,
} from '../generated/src/apis/metadata/metadata_store_service_pb';

interface ExecutionListState {
  executions: Execution[];
  rows: Row[];
  expandedRows: Map<number, Row[]>;
  columns: Column[];
}

class ExecutionList extends Page<{}, ExecutionListState> {
  private tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);
    this.state = {
      columns: [
        {
          customRenderer: this.nameCustomRenderer,
          flex: 2,
          label: 'Pipeline/Workspace',
          sortKey: 'pipelineName',
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
    const {
      error: err,
      response: res,
    } = await Apis.getMetadataServicePromiseClient().getExecutions(new GetExecutionsRequest());

    if (err) {
      // Code === 5 means no record found in backend. This is a temporary workaround.
      // TODO: remove err.code !== 5 check when backend is fixed.
      if (err.code !== 5) {
        this.showPageError(serviceErrorToString(err));
      }
      return '';
    }

    const executions = (res && res.getExecutionsList()) || [];
    await this.getRowsFromExecutions(request, executions);
    return '';
  }

  private nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    const [executionType, executionId] = props.id.split(':');
    const link = RoutePage.EXECUTION_DETAILS.replace(
      `:${RouteParams.EXECUTION_TYPE}+`,
      executionType,
    ).replace(`:${RouteParams.ID}`, executionId);
    return (
      <Link onClick={e => e.stopPropagation()} className={commonCss.link} to={link}>
        {props.value}
      </Link>
    );
  };

  /**
   * Temporary solution to apply sorting, filtering, and pagination to the
   * local list of executions until server-side handling is available
   * TODO: Replace once https://github.com/kubeflow/metadata/issues/73 is done.
   * @param request
   */
  private async getRowsFromExecutions(
    request: ListRequest,
    executions: Execution[],
  ): Promise<void> {
    const executionTypesMap = new Map<number, ExecutionType>();
    // TODO: Consider making an Api method for returning and caching types
    const {
      error: err,
      response: res,
    } = await Apis.getMetadataServicePromiseClient().getExecutionTypes(
      new GetExecutionTypesRequest(),
    );

    if (err) {
      this.showPageError(serviceErrorToString(err));
      return;
    }

    ((res && res.getExecutionTypesList()) || []).forEach(executionType => {
      executionTypesMap.set(executionType.getId()!, executionType);
    });

    const collapsedAndExpandedRows = groupRows(
      executions
        .map(execution => {
          // Flattens
          const typeId = execution.getTypeId();
          const type =
            typeId && executionTypesMap && executionTypesMap.get(typeId)
              ? executionTypesMap.get(typeId)!.getName()
              : typeId;
          return {
            id: `${type}:${execution.getId()}`, // Join with colon so we can build the link
            otherFields: [
              getResourceProperty(execution, ExecutionProperties.PIPELINE_NAME) ||
                getResourceProperty(execution, ExecutionCustomProperties.WORKSPACE, true),
              getResourceProperty(execution, ExecutionProperties.COMPONENT_ID),
              getResourceProperty(execution, ExecutionProperties.STATE),
              execution.getId(),
              type,
            ],
          };
        })
        .filter(rowFilterFn(request))
        .sort(rowCompareFn(request, this.state.columns)),
    );

    this.setState({
      executions,
      expandedRows: collapsedAndExpandedRows.expandedRows,
      rows: collapsedAndExpandedRows.collapsedRows,
    });
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
