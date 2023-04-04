/*
 * Copyright 2018 The Kubeflow Authors
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
import BusyButton from 'src/atoms/BusyButton';
import CustomTable, { Column, Row, CustomRendererProps } from 'src/components/CustomTable';
import Toolbar, { ToolbarActionMap } from 'src/components/Toolbar';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { Apis, JobSortKeys, ListRequest } from 'src/lib/Apis';
import { DialogProps, RoutePage, RouteParams } from 'src/components/Router';
import { Link } from 'react-router-dom';
import { RouteComponentProps } from 'react-router';
import { SnackbarProps } from '@material-ui/core/Snackbar';
import { commonCss } from 'src/Css';
import { logger, formatDateString, errorToMessage } from 'src/lib/Utils';

export interface RecurringRunListProps extends RouteComponentProps {
  experimentId: string;
  updateDialog: (dialogProps: DialogProps) => void;
  updateSnackbar: (snackbarProps: SnackbarProps) => void;
}

interface RecurringRunListState {
  busyIds: Set<string>;
  runs: V2beta1RecurringRun[];
  selectedIds: string[];
  toolbarActionMap: ToolbarActionMap;
}

class RecurringRunsManager extends React.Component<RecurringRunListProps, RecurringRunListState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      busyIds: new Set(),
      runs: [],
      selectedIds: [],
      toolbarActionMap: {},
    };
  }

  public render(): JSX.Element {
    const { runs, selectedIds, toolbarActionMap: toolbarActions } = this.state;

    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 2,
        label: 'Run name',
        sortKey: JobSortKeys.NAME,
      },
      { label: 'Created at', flex: 2, sortKey: JobSortKeys.CREATED_AT },
      { customRenderer: this._enabledCustomRenderer, label: '', flex: 1 },
    ];

    const rows: Row[] = runs.map(r => {
      return {
        error: r.error?.toString(),
        id: r.recurring_run_id!,
        otherFields: [r.display_name, formatDateString(r.created_at), r.status],
      };
    });

    return (
      <React.Fragment>
        <Toolbar actions={toolbarActions} breadcrumbs={[]} pageTitle='Recurring runs' />
        <CustomTable
          columns={columns}
          rows={rows}
          ref={this._tableRef}
          selectedIds={selectedIds}
          updateSelection={ids => this.setState({ selectedIds: ids })}
          initialSortColumn={JobSortKeys.CREATED_AT}
          reload={this._loadRuns.bind(this)}
          filterLabel='Filter recurring runs'
          disableSelection={true}
          emptyMessage={'No recurring runs found in this experiment.'}
        />
      </React.Fragment>
    );
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      await this._tableRef.current.reload();
    }
  }

  public _nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    return (
      <Link
        className={commonCss.link}
        to={RoutePage.RECURRING_RUN_DETAILS.replace(':' + RouteParams.recurringRunId, props.id)}
      >
        {props.value}
      </Link>
    );
  };

  public _enabledCustomRenderer: React.FC<CustomRendererProps<V2beta1RecurringRunStatus>> = (
    props: CustomRendererProps<V2beta1RecurringRunStatus>,
  ) => {
    const isBusy = this.state.busyIds.has(props.id);
    return (
      <BusyButton
        outlined={props.value === V2beta1RecurringRunStatus.ENABLED}
        title={props.value === V2beta1RecurringRunStatus.ENABLED ? 'Enabled' : 'Disabled'}
        busy={isBusy}
        onClick={() => {
          let busyIds = this.state.busyIds;
          busyIds.add(props.id);
          this.setState({ busyIds }, async () => {
            props.value === V2beta1RecurringRunStatus.ENABLED
              ? await this._setEnabledState(props.id, false)
              : await this._setEnabledState(props.id, true);
            busyIds = this.state.busyIds;
            busyIds.delete(props.id);
            this.setState({ busyIds });
            await this.refresh();
          });
        }}
      />
    );
  };

  protected async _loadRuns(request: ListRequest): Promise<string> {
    let runs: V2beta1RecurringRun[] = [];
    let nextPageToken = '';
    try {
      const response = await Apis.recurringRunServiceApi.listRecurringRuns(
        request.pageToken,
        request.pageSize,
        request.sortBy,
        undefined,
        request.filter,
        this.props.experimentId,
      );
      runs = response.recurringRuns || [];
      nextPageToken = response.next_page_token || '';
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.props.updateDialog({
        buttons: [{ text: 'Dismiss' }],
        content: 'List recurring run configs request failed with:\n' + errorMessage,
        title: 'Error retrieving recurring run configs',
      });
      logger.error('Could not get list of recurring runs', errorMessage);
    }

    this.setState({ runs });
    return nextPageToken;
  }

  protected async _setEnabledState(id: string, enabled: boolean): Promise<void> {
    try {
      await (enabled
        ? Apis.recurringRunServiceApi.enableRecurringRun(id)
        : Apis.recurringRunServiceApi.disableRecurringRun(id));
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.props.updateDialog({
        buttons: [{ text: 'Dismiss' }],
        content: 'Error changing enabled state of recurring run:\n' + errorMessage,
        title: 'Error',
      });
      logger.error('Error changing enabled state of recurring run', errorMessage);
    }
  }
}

export default RecurringRunsManager;
