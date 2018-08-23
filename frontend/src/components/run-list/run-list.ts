// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import * as xss from 'xss';
import { ListRunsRequest, RunSortKeys } from '../../api/list_runs_request';
import { RunMetadata } from '../../api/run';
import { NodePhase } from '../../model/argo_template';
import {
  ItemDblClickEvent,
  ListFormatChangeEvent,
  NewListPageEvent,
  RouteEvent,
} from '../../model/events';
import {
  ColumnType,
  ColumnTypeName,
  ItemListColumn,
  ItemListElement,
  ItemListRow,
} from '../item-list/item-list';

import './run-list.html';

@customElement('run-list')
export class RunList extends Polymer.Element {

  @property({ type: Array })
  public runsMetadata: RunMetadata[] = [];

  public get itemList(): ItemListElement {
    return this.$.runsItemList as ItemListElement;
  }

  protected runListRows: ItemListRow[] = [];

  protected runListColumns: ItemListColumn[] = [
    new ItemListColumn('Run Name', ColumnTypeName.STRING, RunSortKeys.NAME),
    new ItemListColumn('Created at', ColumnTypeName.DATE, RunSortKeys.CREATED_AT),
    new ItemListColumn('Scheduled at', ColumnTypeName.DATE),
  ];

  private _debouncer: Polymer.Debouncer | undefined = undefined;

  private _jobId = '';

  public ready(): void {
    super.ready();
    this.itemList.addEventListener(ListFormatChangeEvent.name, this._listFormatChanged.bind(this));
    this.itemList.addEventListener(NewListPageEvent.name, this._loadNewListPage.bind(this));
    this.itemList.addEventListener(ItemDblClickEvent.name, this._navigate.bind(this));

    this.itemList.renderColumn = (value: ColumnType, colIndex: number, rowIndex: number) => {
      let text = '-';
      if (this.itemList.columns[colIndex] && value) {
        if (this.itemList.columns[colIndex].type === ColumnTypeName.DATE) {
          text = (value as Date).toLocaleString();
        } else {
          text = value.toString();
        }
      }
      text = xss(text);
      return colIndex ? `<span>${text}</span>` :
          `<a class="link"
              href="/jobRun?jobId=${this._jobId}&runId=${this.runsMetadata[rowIndex].id}">
            ${text}
          </a>`;
    };
  }

  public loadRuns(jobId: string): void {
    this._jobId = jobId;
    this.itemList.reset();
    this._loadRunsInternal(new ListRunsRequest(jobId, this.itemList.selectedPageSize));
  }

  protected _navigate(ev: ItemDblClickEvent): void {
    const runId = this.runsMetadata[ev.detail.index].id;
    this.dispatchEvent(
        new RouteEvent(`/jobRun?jobId=${this._jobId}&runId=${runId}`));
  }

  protected _getStatusIcon(status: NodePhase): string {
    return Utils.nodePhaseToIcon(status);
  }

  protected _getRuntime(start: string, end: string, status: NodePhase): string {
    if (!status) {
      return '-';
    }
    const startDate = new Date(start);
    const endDate = end ? new Date(end) : new Date();
    return Utils.dateDiffToString(endDate.valueOf() - startDate.valueOf());
  }

  private async _loadRunsInternal(request: ListRunsRequest): Promise<void> {
    try {
      const listRunsResponse = await Apis.listRuns(request);
      this.runsMetadata = listRunsResponse.runs || [];

      this.itemList.updateNextPageToken(listRunsResponse.next_page_token || '');
    } catch (err) {
      // TODO: This error should be bubbled up to job-details to be shown as a page error.
      Utils.showDialog('There was an error while loading the run list', err);
    }

    this.runListRows = this.runsMetadata.map((runMetadata) => {
      const row = new ItemListRow({
        columns: [
          runMetadata.name,
          Utils.formatDateString(runMetadata.created_at),
          Utils.formatDateString(runMetadata.scheduled_at),
        ],
        selected: false,
      });
      return row;
    });
  }

  private _loadNewListPage(ev: NewListPageEvent): void {
    const request = new ListRunsRequest(this._jobId, ev.detail.pageSize);
    request.filterBy = ev.detail.filterBy;
    request.pageToken = ev.detail.pageToken;
    request.sortBy = ev.detail.sortBy;
    this._loadRunsInternal(request);
  }

  private _listFormatChanged(ev: ListFormatChangeEvent): void {
    // This function will wait 300ms after last time it is called before listRuns() is called.
    this._debouncer = Polymer.Debouncer.debounce(
        this._debouncer || null,
        Polymer.Async.timeOut.after(300),
        async () => {
          const request = new ListRunsRequest(this._jobId, ev.detail.pageSize);
          request.filterBy = ev.detail.filterString;
          request.orderAscending = ev.detail.orderAscending;
          request.sortBy = ev.detail.sortColumn;
          this._loadRunsInternal(request);
        }
    );
    // Allows tests to use Polymer.flush to ensure debounce has completed.
    Polymer.enqueueDebouncer(this._debouncer);
  }
}
