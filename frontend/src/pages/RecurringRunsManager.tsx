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
import BusyButton from '../atoms/BusyButton';
import CustomTable, { Column, Row } from '../components/CustomTable';
import Toolbar, { ToolbarActionConfig } from '../components/Toolbar';
import { ApiJob, ApiResourceType } from '../apis/job';
import { Apis, JobSortKeys, ListRequest } from '../lib/Apis';
import { DialogProps, RoutePage, RouteParams } from '../components/Router';
import { Link } from 'react-router-dom';
import { RouteComponentProps } from 'react-router';
import { SnackbarProps } from '@material-ui/core/Snackbar';
import { commonCss } from '../Css';
import { logger, formatDateString, errorToMessage } from '../lib/Utils';

interface RecurringRunListProps extends RouteComponentProps {
  experimentId: string;
  updateDialog: (dialogProps: DialogProps) => void;
  updateSnackbar: (snackbarProps: SnackbarProps) => void;
}

interface RecurringRunListState {
  busyIds: Set<string>;
  runs: ApiJob[];
  selectedIds: string[];
  sortBy: string;
  toolbarActions: ToolbarActionConfig[];
}

class RecurringRunsManager extends React.Component<RecurringRunListProps, RecurringRunListState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      busyIds: new Set(),
      runs: [],
      selectedIds: [],
      sortBy: JobSortKeys.CREATED_AT,
      toolbarActions: [],
    };
  }

  public render(): JSX.Element {
    const { runs, selectedIds, sortBy, toolbarActions } = this.state;

    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer.bind(this),
        flex: 2,
        label: 'Run name',
        sortKey: JobSortKeys.NAME,
      },
      { label: 'Created at', flex: 2, sortKey: JobSortKeys.CREATED_AT },
      { customRenderer: this._enabledCustomRenderer.bind(this), label: '', flex: 1 },
    ];

    const rows: Row[] = runs.map(r => {
      return {
        error: r.error,
        id: r.id!,
        otherFields: [
          r.name,
          formatDateString(r.created_at),
          r.enabled,
        ],
      };
    });

    return (<React.Fragment>
      <Toolbar actions={toolbarActions} breadcrumbs={[{ displayName: 'Recurring runs', href: '' }]} />
      <CustomTable columns={columns} rows={rows} ref={this._tableRef} selectedIds={selectedIds}
        updateSelection={ids => this.setState({ selectedIds: ids })} initialSortColumn={sortBy}
        reload={this._loadRuns.bind(this)} emptyMessage={'No recurring runs found in this experiment.'}
        disableSelection={true} />
    </React.Fragment>);
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      await this._tableRef.current.reload();
    }
  }

  private async _loadRuns(request: ListRequest): Promise<string> {
    let runs: ApiJob[] = [];
    let nextPageToken = '';
    try {
      const response = await Apis.jobServiceApi.listJobs(
        request.pageToken,
        request.pageSize,
        request.sortBy,
        ApiResourceType.EXPERIMENT.toString(),
        this.props.experimentId,
      );
      runs = response.jobs || [];
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

    this.setState({ runs, sortBy: request.sortBy! });
    return nextPageToken;
  }

  private _nameCustomRenderer(value: string, id: string): JSX.Element {
    return <Link className={commonCss.link}
      to={RoutePage.RECURRING_RUN.replace(':' + RouteParams.runId, id)}>{value}</Link>;
  }

  private async _setEnabledState(id: string, enabled: boolean): Promise<void> {
    try {
      await (enabled ? Apis.jobServiceApi.enableJob(id) : Apis.jobServiceApi.disableJob(id));
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

  private _enabledCustomRenderer(value: boolean | undefined, id: string): JSX.Element {
    const isBusy = this.state.busyIds.has(id);
    return <BusyButton outlined={value} title={value === true ? 'Enabled' : 'Disabled'}
      busy={isBusy} onClick={() => {
        let busyIds = this.state.busyIds;
        busyIds.add(id);
        this.setState({ busyIds }, async () => {
          await this._setEnabledState(id, !value);
          busyIds = this.state.busyIds;
          busyIds.delete(id);
          this.setState({ busyIds });
          await this.refresh();
        });
      }} />;
  }
}

export default RecurringRunsManager;
