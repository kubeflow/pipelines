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

import {
  Api,
  Execution,
  ExecutionType,
  GetExecutionsRequest,
  GetExecutionTypesRequest,
  ListRequest,
} from '@kubeflow/frontend';
import * as React from 'react';
import { Link } from 'react-router-dom';
import { classes } from 'typestyle';
import CustomTable, {
  Column,
  Row,
  ExpandState,
  CustomRendererProps,
} from '../components/CustomTable';
import { Page } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { commonCss, padding } from '../Css';
import {
  rowCompareFn,
  rowFilterFn,
  groupRows,
  getExpandedRow,
  CollapsedAndExpandedRows,
  serviceErrorToString,
} from '../lib/Utils';
import { RoutePageFactory } from '../components/Router';
import { ExecutionHelpers } from 'src/lib/MlmdUtils';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

interface ExecutionListState {
  executions: Execution[];
  rows: Row[];
  expandedRows: Map<number, Row[]>;
  columns: Column[];
}

class ExecutionList extends Page<{ t: TFunction }, ExecutionListState> {
  private tableRef = React.createRef<CustomTable>();
  private api = Api.getInstance();
  private executionTypesMap: Map<number, ExecutionType>;

  constructor(props: any) {
    super(props);
    const { t } = this.props;
    this.state = {
      columns: [
        {
          customRenderer: this.nameCustomRenderer,
          flex: 2,
          label: t('runWorkspacePipeline'),
          sortKey: 'workspace',
        },
        {
          customRenderer: this.nameCustomRenderer,
          flex: 1,
          label: t('common:name'),
          sortKey: 'name',
        },
        { label: t('common:state'), flex: 1, sortKey: 'state' },
        { label: t('common:id'), flex: 1, sortKey: 'id' },
        { label: t('common:type'), flex: 2, sortKey: 'type' },
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
    const { t } = this.props;
    return {
      actions: {},
      breadcrumbs: [],
      pageTitle: t('common:executions'),
      t,
    };
  }

  public render(): JSX.Element {
    const { rows, columns } = this.state;
    const { t } = this.props;
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
          emptyMessage={t('noExecutionsFound')}
          t={t}
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
    const { t } = this.props;
    try {
      const response = await this.api.metadataStoreService.getExecutions(
        new GetExecutionsRequest(),
      );
      return response.getExecutionsList();
    } catch (err) {
      // Code === 5 means no record found in backend. This is a temporary workaround.
      // TODO: remove err.code !== 5 check when backend is fixed.
      if (err.code !== 5) {
        err.message = `${t('getExecutionsFailed')}: ` + err.message;
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

export default withTranslation(['executions', 'common'])(ExecutionList);
