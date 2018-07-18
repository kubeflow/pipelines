import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { ListPipelinesRequest, PipelineSortKeys } from '../../api/list_pipelines_request';
import { Pipeline } from '../../api/pipeline';
import { DialogResult } from '../../components/popup-dialog/popup-dialog';
import {
  ItemDblClickEvent,
  ListFormatChangeEvent,
  NewListPageEvent,
  RouteEvent,
} from '../../model/events';
import { PageElement } from '../../model/page_element';
import {
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

  public get newButton(): PaperButtonElement {
    return this.$.newBtn as PaperButtonElement;
  }

  public get refreshButton(): PaperButtonElement {
    return this.$.refreshBtn as PaperButtonElement;
  }

  public get cloneButton(): PaperButtonElement {
    return this.$.cloneBtn as PaperButtonElement;
  }

  public get deleteButton(): PaperButtonElement {
    return this.$.deleteBtn as PaperButtonElement;
  }

  public get itemList(): ItemListElement {
    return this.$.pipelinesItemList as ItemListElement;
  }

  protected pipelineListRows: ItemListRow[] = [];

  protected pipelineListColumns: ItemListColumn[] = [
    new ItemListColumn('Name', ColumnTypeName.STRING, PipelineSortKeys.NAME),
    new ItemListColumn('Description', ColumnTypeName.STRING),
    new ItemListColumn('Package ID', ColumnTypeName.NUMBER, PipelineSortKeys.PACKAGE_ID),
    new ItemListColumn('Created at', ColumnTypeName.DATE, PipelineSortKeys.CREATED_AT),
    new ItemListColumn('Schedule', ColumnTypeName.STRING),
    new ItemListColumn('Enabled', ColumnTypeName.STRING),
  ];

  private _debouncer: Polymer.Debouncer;

  public ready(): void {
    super.ready();
    this.itemList.addEventListener(ListFormatChangeEvent.name, this._listFormatChanged.bind(this));
    this.itemList.addEventListener(NewListPageEvent.name, this._loadNewListPage.bind(this));
    this.itemList.addEventListener('selected-indices-changed',
        this._selectedItemsChanged.bind(this));
    this.itemList.addEventListener(ItemDblClickEvent.name, this._navigate.bind(this));
  }

  public load(_: string): void {
    this.itemList.reset();
    this._loadPipelines(new ListPipelinesRequest(this.itemList.selectedPageSize));
  }

  protected _navigate(ev: ItemDblClickEvent): void {
    const pipelineId = this.pipelines[ev.detail.index].id;
    this.dispatchEvent(new RouteEvent(`/pipelines/details/${pipelineId}`));
  }

  protected _refresh(): void {
    this.load('');
  }

  protected _clonePipeline(): void {
    // Clone Pipeline button is only enabled if there is one selected item.
    const selectedPipeline = this.pipelines[this.itemList.selectedIndices[0]];
    this.dispatchEvent(
        new RouteEvent(
          '/pipelines/new',
          {
            packageId: selectedPipeline.package_id,
            parameters: selectedPipeline.parameters
          }));
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
      Utils.showDialog(`Failed to delete ${unsuccessfulDeletes} Pipelines`, errorMessage);
    }

    this._busy = false;
  }

  protected _newPipeline(): void {
    this.dispatchEvent(new RouteEvent('/pipelines/new'));
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

  // TODO: figure out what to set the browser cache time for these queries to so
  // that we make use of the cache but don't use it so much that we miss updates
  // to the actual backing database.
  private _listFormatChanged(ev: ListFormatChangeEvent): void {
    // This function will wait 300ms after last time it is called before getPipelines() is called.
    this._debouncer = Polymer.Debouncer.debounce(
        this._debouncer,
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
      const getPipelinesResponse = await Apis.getPipelines(request);
      this.pipelines = getPipelinesResponse.pipelines || [];

      this.itemList.updateNextPageToken(getPipelinesResponse.next_page_token || '');
    } catch (err) {
      this.showPageError('There was an error while loading the pipeline list.');
      Utils.log.error('Error loading pipelines:', err);
    }

    this.pipelineListRows = this.pipelines.map((pipeline) => {
      // TODO: we should just call pipeline.trigger.toString() here, but the lack of types in
      // the mocked data prevents us from being able to use functions at the moment.
      let schedule = '-';
      if (pipeline && pipeline.trigger) {
        schedule = pipeline.trigger.toString();
      }
      const row = new ItemListRow({
        columns: [
          pipeline.name,
          pipeline.description,
          pipeline.package_id,
          Utils.formatDateString(pipeline.created_at),
          schedule,
          Utils.enabledDisplayString(pipeline.trigger, pipeline.enabled)
        ],
        selected: false,
      });
      return row;
    });
  }
}
