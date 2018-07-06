import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { ListPipelinesRequest, PipelineSortKeys } from '../../api/list_pipelines_request';
import { Pipeline } from '../../api/pipeline';
import {
  EventName,
  ItemClickEvent,
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

  @property({ type: Number })
  protected _pageSize = 20;

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
    const itemList = this.$.pipelinesItemList as ItemListElement;
    itemList.addEventListener(EventName.LIST_FORMAT_CHANGE, this._listFormatChanged.bind(this));
    itemList.addEventListener(EventName.NEW_LIST_PAGE, this._loadNewListPage.bind(this));
    itemList.addEventListener('selected-indices-changed', this._selectedItemsChanged.bind(this));
    itemList.addEventListener('itemDoubleClick', this._navigate.bind(this));
  }

  public load(_: string): void {
    const itemList = this.$.pipelinesItemList as ItemListElement;
    itemList.reset();
    this._loadPipelines(new ListPipelinesRequest(this._pageSize));
  }

  protected _navigate(ev: ItemClickEvent): void {
    const pipelineId = this.pipelines[ev.detail.index].id;
    this.dispatchEvent(new RouteEvent(`/pipelines/details/${pipelineId}`));
  }

  protected _refresh(): void {
    this.load('');
  }

  protected _clonePipeline(): void {
    const itemList = this.$.pipelinesItemList as ItemListElement;
    // Clone Pipeline button is only enabled if there is one selected item.
    const selectedPipeline = this.pipelines[itemList.selectedIndices[0]];
    this.dispatchEvent(
        new RouteEvent(
          '/pipelines/new',
          {
            packageId: selectedPipeline.package_id,
            parameters: selectedPipeline.parameters
          }));
  }

  protected async _deletePipeline(): Promise<void> {
    const itemList = this.$.pipelinesItemList as ItemListElement;
    this._busy = true;
    let unsuccessfulDeletes = 0;
    let errorMessage = '';

    await Promise.all(itemList.selectedIndices.map(async (i) => {
      try {
        await Apis.deletePipeline(this.pipelines[i].id);
      } catch (err) {
        errorMessage = `Deleting Pipeline: "${this.pipelines[i].name}" failed with error: "${err}"`;
        unsuccessfulDeletes++;
      }
    }));

    const successfulDeletes = itemList.selectedIndices.length - unsuccessfulDeletes;
    if (successfulDeletes > 0) {
      Utils.showNotification(`Successfully deleted ${successfulDeletes} Pipelines!`);
      itemList.reset();
      this._loadPipelines(new ListPipelinesRequest(this._pageSize));
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
    const request = new ListPipelinesRequest(this._pageSize);
    request.filterBy = ev.detail.filterBy;
    request.pageToken = ev.detail.pageToken;
    request.sortBy = ev.detail.sortBy;

    this._loadPipelines(request);
  }

  private _selectedItemsChanged(): void {
    const itemList = this.$.pipelinesItemList as ItemListElement;
    if (itemList.selectedIndices) {
      this._oneItemIsSelected = itemList.selectedIndices.length === 1;
      this._atLeastOneItemIsSelected = itemList.selectedIndices.length > 0;
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
          const request = new ListPipelinesRequest(this._pageSize);
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

      const itemList = this.$.pipelinesItemList as ItemListElement;
      itemList.updateNextPageToken(getPipelinesResponse.nextPageToken || '');
    } catch (err) {
      this.showPageError('There was an error while loading the pipeline list.');
      Utils.log.error('Error loading pipelines:', err);
    }

    this.pipelineListRows = this.pipelines.map((pipeline) => {
      const row = new ItemListRow({
        columns: [
          pipeline.name,
          pipeline.description,
          pipeline.package_id,
          Utils.formatDateString(pipeline.created_at),
          pipeline.schedule,
          Utils.enabledDisplayString(pipeline.schedule, pipeline.enabled)
        ],
        selected: false,
      });
      return row;
    });
  }
}
