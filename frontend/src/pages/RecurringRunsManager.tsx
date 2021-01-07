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
import CustomTable, { Column, Row, CustomRendererProps } from '../components/CustomTable';
import Toolbar, { ToolbarActionMap } from '../components/Toolbar';
import { ApiJob } from '../apis/job';
import { Apis, JobSortKeys, ListRequest } from '../lib/Apis';
import { DialogProps, RoutePage, RouteParams } from '../components/Router';
import { Link } from 'react-router-dom';
import { RouteComponentProps } from 'react-router';
import { SnackbarProps } from '@material-ui/core/Snackbar';
import { commonCss } from '../Css';
import { logger, formatDateString, errorToMessage } from '../lib/Utils';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

export interface RecurringRunListProps extends RouteComponentProps {
  experimentId: string;
  updateDialog: (dialogProps: DialogProps) => void;
  updateSnackbar: (snackbarProps: SnackbarProps) => void;
  t: TFunction;
}

interface RecurringRunListState {
  busyIds: Set<string>;
  runs: ApiJob[];
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
    const { t } = this.props;

    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 2,
        label: t('experiments:runName'),
        sortKey: JobSortKeys.NAME,
      },
      { label: t('common:createdAt'), flex: 2, sortKey: JobSortKeys.CREATED_AT },
      { customRenderer: this._enabledCustomRenderer, label: '', flex: 1 },
    ];

    const rows: Row[] = runs.map(r => {
      return {
        error: r.error,
        id: r.id!,
        otherFields: [r.name, formatDateString(r.created_at), r.enabled],
      };
    });

    return (
      <React.Fragment>
        <Toolbar actions={toolbarActions} breadcrumbs={[]} pageTitle={t('recurringRuns')} />
        <CustomTable
          columns={columns}
          rows={rows}
          ref={this._tableRef}
          selectedIds={selectedIds}
          updateSelection={ids => this.setState({ selectedIds: ids })}
          initialSortColumn={JobSortKeys.CREATED_AT}
          reload={this._loadRuns.bind(this)}
          filterLabel={t('filterRecurRuns')}
          disableSelection={true}
          emptyMessage={t('noRecurRunsFound')}
          t={t}
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
        to={RoutePage.RECURRING_RUN.replace(':' + RouteParams.runId, props.id)}
      >
        {props.value}
      </Link>
    );
  };

  public _enabledCustomRenderer: React.FC<CustomRendererProps<boolean>> = (
    props: CustomRendererProps<boolean>,
  ) => {
    const isBusy = this.state.busyIds.has(props.id);
    const { t } = this.props;
    return (
      <BusyButton
        outlined={props.value}
        title={props.value === true ? t('common:enabled') : t('common:disabled')}
        busy={isBusy}
        onClick={() => {
          let busyIds = this.state.busyIds;
          busyIds.add(props.id);
          this.setState({ busyIds }, async () => {
            await this._setEnabledState(props.id, !props.value);
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
    let runs: ApiJob[] = [];
    let nextPageToken = '';
    const { t } = this.props;
    try {
      const response = await Apis.jobServiceApi.listJobs(
        request.pageToken,
        request.pageSize,
        request.sortBy,
        'EXPERIMENT',
        this.props.experimentId,
        request.filter,
      );
      runs = response.jobs || [];
      nextPageToken = response.next_page_token || '';
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.props.updateDialog({
        buttons: [{ text: t('common:dismiss') }],
        content: `${t('listRecurRunConfigFailed')}:\n` + errorMessage,
        title: t('errorRetrieveRecurrRunConfig'),
      });
      logger.error('Could not get list of recurring runs', errorMessage);
    }

    this.setState({ runs });
    return nextPageToken;
  }

  protected async _setEnabledState(id: string, enabled: boolean): Promise<void> {
    const { t } = this.props;
    try {
      await (enabled ? Apis.jobServiceApi.enableJob(id) : Apis.jobServiceApi.disableJob(id));
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.props.updateDialog({
        buttons: [{ text: t('common:dismiss') }],
        content: `${t('errorChangeStateRecurRun')}:\n` + errorMessage,
        title: t('common:error'),
      });
      logger.error('Error changing enabled state of recurring run', errorMessage);
    }
  }
}

export default withTranslation(['experiments', 'common'])(RecurringRunsManager);
