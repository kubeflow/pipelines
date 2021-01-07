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
import CustomTable, { Column, Row } from '../components/CustomTable';
import Toolbar, { ToolbarActionMap } from '../components/Toolbar';
import { ListRequest } from '../lib/Apis';
import { RouteComponentProps } from 'react-router-dom';
import { logger, errorToMessage, formatDateString } from '../lib/Utils';
import { DialogProps } from '../components/Router';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

interface BaseResponse {
  resources: BaseResource[];
  nextPageToken: string;
}

export interface BaseResource {
  id?: string;
  created_at?: Date;
  description?: string;
  name?: string;
  error?: string;
}

export interface ResourceSelectorProps extends RouteComponentProps {
  listApi: (...args: any[]) => Promise<BaseResponse>;
  columns: Column[];
  emptyMessage: string;
  filterLabel: string;
  initialSortColumn: any;
  selectionChanged: (resource: BaseResource) => void;
  title: string;
  toolbarActionMap?: ToolbarActionMap;
  updateDialog: (dialogProps: DialogProps) => void;
  t: TFunction;
}

interface ResourceSelectorState {
  resources: BaseResource[];
  rows: Row[];
  selectedIds: string[];
  toolbarActionMap: ToolbarActionMap;
}

class ResourceSelector extends React.Component<ResourceSelectorProps, ResourceSelectorState> {
  protected _isMounted = true;

  constructor(props: any) {
    super(props);

    this.state = {
      resources: [],
      rows: [],
      selectedIds: [],
      toolbarActionMap: (props && props.toolbarActionMap) || {},
    };
  }

  public render(): JSX.Element {
    const { rows, selectedIds, toolbarActionMap } = this.state;
    const { columns, title, filterLabel, emptyMessage, initialSortColumn, t } = this.props;

    return (
      <React.Fragment>
        <Toolbar actions={toolbarActionMap} breadcrumbs={[]} pageTitle={title} />
        <CustomTable
          columns={columns}
          rows={rows}
          selectedIds={selectedIds}
          useRadioButtons={true}
          updateSelection={this._selectionChanged.bind(this)}
          filterLabel={filterLabel}
          initialSortColumn={initialSortColumn}
          reload={this._load.bind(this)}
          emptyMessage={emptyMessage}
          t={t}
        />
      </React.Fragment>
    );
  }

  public componentWillUnmount(): void {
    this._isMounted = false;
  }

  protected setStateSafe(newState: Partial<ResourceSelectorState>, cb?: () => void): void {
    if (this._isMounted) {
      this.setState(newState as any, cb);
    }
  }

  protected _selectionChanged(selectedIds: string[]): void {
    const { t } = this.props;
    if (!Array.isArray(selectedIds) || selectedIds.length !== 1) {
      logger.error(`${selectedIds.length} ${t('resourcesSelected')}`, selectedIds);
      return;
    }
    const selected = this.state.resources.find(r => r.id === selectedIds[0]);
    if (selected) {
      this.props.selectionChanged(selected);
    } else {
      logger.error(`${t('noResourceFoundWithId')}: ${selectedIds[0]}`);
      return;
    }
    this.setStateSafe({ selectedIds });
  }

  protected async _load(request: ListRequest): Promise<string> {
    let nextPageToken = '';
    const { t } = this.props;
    try {
      const response = await this.props.listApi(
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
      );

      this.setStateSafe({
        resources: response.resources,
        rows: this._resourcesToRow(response.resources),
      });

      nextPageToken = response.nextPageToken;
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.props.updateDialog({
        buttons: [{ text: t('dismiss') }],
        content: `${t('listRequestFailed')}:\n` + errorMessage,
        title: t('errorRetrieveResources'),
      });
      logger.error(t('listResourcesFailed'), errorMessage);
    }
    return nextPageToken;
  }

  protected _resourcesToRow(resources: BaseResource[]): Row[] {
    return resources.map(
      r =>
        ({
          error: (r as any).error,
          id: r.id!,
          otherFields: [r.name, r.description, formatDateString(r.created_at)],
        } as Row),
    );
  }
}

export default withTranslation('common')(ResourceSelector);
