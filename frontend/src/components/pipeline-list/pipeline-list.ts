import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import {
  EventName,
  FilterChangedEvent,
  ItemClickEvent,
  NewListPageEvent,
  RouteEvent
} from '../../model/events';
import { ListPipelinesRequest } from '../../model/list_pipelines_request';
import { PageElement } from '../../model/page_element';
import { Pipeline } from '../../model/pipeline';
import {
  ColumnTypeName,
  ItemListColumn,
  ItemListElement,
  ItemListRow
} from '../item-list/item-list';

import './pipeline-list.html';

@customElement('pipeline-list')
export class PipelineList extends PageElement {

  @property({ type: Array })
  public pipelines: Pipeline[] = [];

  @property({ type: Boolean })
  protected _busy = false;

  @property({ type: Boolean })
  protected oneItemIsSelected = false;

  @property({ type: Boolean })
  protected _atLeastOneItemIsSelected = false;

  @property({ type: Number })
  protected _pageSize = 20;

  protected pipelineListRows: ItemListRow[] = [];

  protected pipelineListColumns: ItemListColumn[] = [
    { name: 'Name', type: ColumnTypeName.STRING },
    { name: 'Description', type: ColumnTypeName.STRING },
    { name: 'Package ID', type: ColumnTypeName.NUMBER },
    { name: 'Created at', type: ColumnTypeName.DATE },
    { name: 'Schedule', type: ColumnTypeName.STRING },
    { name: 'Enabled', type: ColumnTypeName.STRING },
  ];

  private _keystrokeDebouncer: Polymer.Debouncer;

  ready(): void {
    super.ready();
    const itemList = this.$.pipelinesItemList as ItemListElement;
    itemList.addEventListener(EventName.FILTER_CHANGED, this._filterChanged.bind(this));
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

  protected _clonePipeline(): void {
    const itemList = this.$.pipelinesItemList as ItemListElement;
    // Clone Pipeline button is only enabled if there is one selected item.
    const selectedPipeline = this.pipelines[itemList.selectedIndices[0]];
    this.dispatchEvent(
        new RouteEvent(
          '/pipelines/new',
          {
            packageId: selectedPipeline.packageId,
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
      this.oneItemIsSelected = itemList.selectedIndices.length === 1;
      this._atLeastOneItemIsSelected = itemList.selectedIndices.length > 0;
    } else {
      this.oneItemIsSelected = false;
      this._atLeastOneItemIsSelected = false;
    }
  }

  // TODO: figure out what to set the browser cache time for these queries to so
  // that we make use of the cache but don't use it so much that we miss updates
  // to the actual backing database.
  private _filterChanged(ev: FilterChangedEvent): void {
    // This function will wait 300ms after last time it is called (last
    // keystroke in filter box) before getPipelines() is called.
    this._keystrokeDebouncer = Polymer.Debouncer.debounce(
        this._keystrokeDebouncer,
        Polymer.Async.timeOut.after(300),
        async () => {
          const request = new ListPipelinesRequest(this._pageSize);
          request.filterBy = ev.detail.filterString;
          this._loadPipelines(request);
        }
    );
    // Allows tests to use Polymer.flush to ensure debounce has completed.
    Polymer.enqueueDebouncer(this._keystrokeDebouncer);
  }

  private async _loadPipelines(request: ListPipelinesRequest): Promise<void> {
    try {
      const getPipelinesResponse = await Apis.getPipelines(request);
      this.pipelines = getPipelinesResponse.pipelines;

      const itemList = this.$.pipelinesItemList as ItemListElement;
      itemList.updateNextPageToken(getPipelinesResponse.nextPageToken);
    } catch (err) {
      this.showPageError('There was an error while loading the pipeline list.');
      Utils.log.error('Error loading pipelines:', err);
    }

    this.pipelineListRows = this.pipelines.map((pipeline) => {
      const row = new ItemListRow({
        columns: [
          pipeline.name,
          pipeline.description,
          pipeline.packageId,
          Utils.formatDateInSeconds(pipeline.createdAt),
          pipeline.schedule,
          Utils.enabledDisplayString(pipeline.schedule, pipeline.enabled)
        ],
        selected: false,
      });
      return row;
    });
  }
}
