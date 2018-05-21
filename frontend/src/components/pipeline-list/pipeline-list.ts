import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import {
  FILTER_CHANGED_EVENT,
  FilterChangedEvent,
  ItemClickEvent,
  RouteEvent
} from '../../model/events';
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
  protected atLeastOneItemIsSelected = false;

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
    itemList.addEventListener(FILTER_CHANGED_EVENT, this._filterChanged.bind(this));
    itemList.addEventListener('selected-indices-changed', this._selectedItemsChanged.bind(this));
    itemList.addEventListener('itemDoubleClick', this._navigate.bind(this));
  }

  public load(_: string): void {
    this._loadPipelines();
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
      this._loadPipelines();
    }

    if (unsuccessfulDeletes > 0) {
      Utils.showDialog(`Failed to delete ${unsuccessfulDeletes} Pipelines`, errorMessage);
    }

    this._busy = false;
  }

  protected _newPipeline(): void {
    this.dispatchEvent(new RouteEvent('/pipelines/new'));
  }

  private _selectedItemsChanged(): void {
    const itemList = this.$.pipelinesItemList as ItemListElement;
    if (itemList.selectedIndices) {
      this.oneItemIsSelected = itemList.selectedIndices.length === 1;
      this.atLeastOneItemIsSelected = itemList.selectedIndices.length > 0;
    } else {
      this.oneItemIsSelected = false;
      this.atLeastOneItemIsSelected = false;
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
        async () => { this._loadPipelines(ev.detail.filterString); }
    );
    // Allows tests to use Polymer.flush to ensure debounce has completed.
    Polymer.enqueueDebouncer(this._keystrokeDebouncer);
  }

  private async _loadPipelines(filterString?: string): Promise<void> {
    try {
      this.pipelines = await Apis.getPipelines(filterString);
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
