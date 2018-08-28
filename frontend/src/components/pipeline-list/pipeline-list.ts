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

import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import * as xss from 'xss';
import { ListPipelinesRequest, PipelineSortKeys } from '../../api/list_pipelines_request';
import { Pipeline } from '../../api/pipeline';
import { DialogResult } from '../../components/popup-dialog/popup-dialog';
import {
  ItemDblClickEvent,
  ListFormatChangeEvent,
  NewListPageEvent,
} from '../../model/events';
import { PageElement } from '../../model/page_element';
import {
  ColumnType,
  ColumnTypeName,
  ItemListColumn,
  ItemListElement,
  ItemListRow,
} from '../item-list/item-list';

import './pipeline-list.html';

@customElement('pipeline-list')
export class PipelineList extends PageElement {

  @property({ type: Array })
  public pipelines: Pipeline[] = [];

  @property({ type: Boolean })
  protected _busy = false;

  @property({ type: Boolean })
  protected _oneItemIsSelected = false;

  @property({ type: Boolean })
  protected _atLeastOneItemIsSelected = false;

  public get uploadButton(): PaperButtonElement {
    return this.$.newBtn as PaperButtonElement;
  }

  public get refreshButton(): PaperButtonElement {
    return this.$.refreshBtn as PaperButtonElement;
  }

  public get deleteButton(): PaperButtonElement {
    return this.$.deleteBtn as PaperButtonElement;
  }

  public get itemList(): ItemListElement {
    return this.$.pipelinesItemList as ItemListElement;
  }

  protected pipelineListRows: ItemListRow[] = [];

  protected pipelineListColumns: ItemListColumn[] = [
    new ItemListColumn('Name', ColumnTypeName.STRING, PipelineSortKeys.NAME, 1.5),
    new ItemListColumn('Description', ColumnTypeName.STRING, undefined, 1.5),
    new ItemListColumn('Uploaded on', ColumnTypeName.DATE, PipelineSortKeys.CREATED_AT),
  ];

  private _debouncer: Polymer.Debouncer | undefined = undefined;

  public ready(): void {
    super.ready();
    this.itemList.addEventListener(ListFormatChangeEvent.name, this._listFormatChanged.bind(this));
    this.itemList.addEventListener(NewListPageEvent.name, this._loadNewListPage.bind(this));
    this.itemList.addEventListener('selected-indices-changed',
        this._selectedItemsChanged.bind(this));
    this.itemList.addEventListener(ItemDblClickEvent.name, this._navigate.bind(this));

    this.itemList.renderColumn = (value: ColumnType, colIndex: number, rowIndex: number) => {
      if (value === undefined) {
        return '-';
      }
      let text = xss(value.toString());
      if (colIndex === 0) {
        // TODO: Make this a link once Pipeline details page is ready.
        return text;
      } else {
        if (this.itemList.columns[colIndex] && value &&
            this.itemList.columns[colIndex].type === ColumnTypeName.DATE) {
          text = (value as Date).toLocaleString();
        }
        return text;
      }
    };
  }

  public load(): void {
    this.itemList.reset();
    this._loadPipelines(new ListPipelinesRequest(this.itemList.selectedPageSize));
  }

  protected _navigate(ev: ItemDblClickEvent): void {
    // TODO: add navigation once details page is ready
  }

  protected _refresh(): void {
    this.load();
  }

  protected async _deletePipeline(): Promise<void> {
    const deletedItemsLen = this.itemList.selectedIndices.length;
    const pluralS = deletedItemsLen > 1 ? 's' : '';
    const dialogResult = await Utils.showDialog(
        `Delete ${deletedItemsLen} pipeline${pluralS}?`,
        `You are about to delete ${deletedItemsLen} pipeline${pluralS}.
         Are you sure you want to proceed?`,
        `Delete ${deletedItemsLen} pipeline${pluralS}`,
        'Cancel');

    // BUTTON1 is Delete
    if (dialogResult !== DialogResult.BUTTON1) {
      return;
    }

    this._busy = true;
    let unsuccessfulDeletes = 0;
    let errorMessage = '';

    await Promise.all(this.itemList.selectedIndices.map(async (i) => {
      try {
        await Apis.deletePipeline(this.pipelines[i].id);
      } catch (err) {
        errorMessage = `Deleting Pipeline: "${this.pipelines[i].name}" failed with error: "${err}"`;
        unsuccessfulDeletes++;
      }
    }));

    const successfulDeletes = this.itemList.selectedIndices.length - unsuccessfulDeletes;
    if (successfulDeletes > 0) {
      Utils.showNotification(`Successfully deleted ${successfulDeletes} Pipelines!`);
      this.itemList.reset();
      this._loadPipelines(new ListPipelinesRequest(this.itemList.selectedPageSize));
    }

    if (unsuccessfulDeletes > 0) {
      Utils.showDialog(
          `Failed to delete ${unsuccessfulDeletes} Pipelines`, errorMessage, 'Dismiss');
    }

    this._busy = false;
  }

  protected _altUpload(): void {
    (this.$.altFileUpload as HTMLInputElement).click();
  }

  protected async _upload(): Promise<void> {
    const files = (this.$.altFileUpload as HTMLInputElement).files;

    if (!files) {
      return;
    }

    const file = files[0];
    this._busy = true;
    try {
      await Apis.uploadPipeline(file);
      // Refresh list after uploading
      this._refresh();
    } catch (err) {
      Utils.showDialog('There was an error uploading the pipeline.', err);
    } finally {
      (this.$.altFileUpload as HTMLInputElement).value = '';
      this._busy = false;
    }
  }

  private _loadNewListPage(ev: NewListPageEvent): void {
    const request = new ListPipelinesRequest(ev.detail.pageSize);
    request.filterBy = ev.detail.filterBy;
    request.pageToken = ev.detail.pageToken;
    request.sortBy = ev.detail.sortBy;

    this._loadPipelines(request);
  }

  private _selectedItemsChanged(): void {
    if (this.itemList.selectedIndices) {
      this._oneItemIsSelected = this.itemList.selectedIndices.length === 1;
      this._atLeastOneItemIsSelected = this.itemList.selectedIndices.length > 0;
    } else {
      this._oneItemIsSelected = false;
      this._atLeastOneItemIsSelected = false;
    }
  }

  private _listFormatChanged(ev: ListFormatChangeEvent): void {
    // This function will wait 300ms after last time it is called before listPipelines() is called.
    this._debouncer = Polymer.Debouncer.debounce(
        this._debouncer || null,
        Polymer.Async.timeOut.after(300),
        async () => {
          const request = new ListPipelinesRequest(ev.detail.pageSize);
          request.filterBy = ev.detail.filterString;
          request.orderAscending = ev.detail.orderAscending;
          request.sortBy = ev.detail.sortColumn;
          this._loadPipelines(request);
        }
    );
    // Allows tests to use Polymer.flush to ensure debounce has completed.
    Polymer.enqueueDebouncer(this._debouncer);
  }

  private async _loadPipelines(request: ListPipelinesRequest): Promise<void> {
    try {
      const listPipelinesResponse = await Apis.listPipelines(request);
      this.pipelines = listPipelinesResponse.pipelines || [];

      this.itemList.updateNextPageToken(listPipelinesResponse.next_page_token || '');
    } catch (err) {
      this.showPageError('There was an error while loading the pipeline list', err.message);
      Utils.log.verbose('Error loading pipelines:', err);
    }

    this.pipelineListRows = this.pipelines.map((pipeline) => {
      const row = new ItemListRow({
        columns: [
          pipeline.name,
          pipeline.description,
          // TODO: should this be "uploaded_on" or is that different?
          Utils.formatDateString(pipeline.created_at),
        ],
        selected: false,
      });
      return row;
    });
  }
}
