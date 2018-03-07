import { customElement, property } from 'polymer-decorators/src/decorators';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import { ItemClickEvent, RouteEvent } from '../../lib/events';
import { PageElement } from '../../lib/page_element';
import { Pipeline } from '../../lib/pipeline';
import { ColumnTypeName, ItemListColumn, ItemListElement, ItemListRow } from '../item-list/item-list';

import './pipeline-list.html';

@customElement('pipeline-list')
export class PipelineList extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public pipelines: Pipeline[] = [];

  private itemListColumns: ItemListColumn[] = [
    { name: 'Name', type: ColumnTypeName.STRING },
    { name: 'Description', type: ColumnTypeName.STRING },
    { name: 'Package ID', type: ColumnTypeName.NUMBER },
    { name: 'Starts', type: ColumnTypeName.DATE },
    { name: 'Ends', type: ColumnTypeName.DATE },
    { name: 'Recurring', type: ColumnTypeName.STRING },
  ];

  public async refresh(_: string) {
    this.pipelines = (await Apis.getPipelines()).map((p) => {
      if (p.createAt) {
        p.createAt = new Date(p.createAt || '').toLocaleString();
      }
      return p;
    });
    this._drawPipelineList();
  }

  /**
   * Creates a new ItemListRow object for each entry in the file list, and sends
   * the created list to the item-list to render.
   */
  _drawPipelineList() {
    const itemList = this.$.pipelinesItemList as ItemListElement;
    itemList.addEventListener('itemDoubleClick', this._navigate.bind(this));
    itemList.columns = this.itemListColumns;
    itemList.rows = this.pipelines.map((pipeline) => {
      const row = new ItemListRow({
        columns: [
          pipeline.name,
          pipeline.description,
          pipeline.packageId,
          new Date(pipeline.starts),
          new Date(pipeline.ends),
          pipeline.recurring ? 'True' : 'False',
        ],
        selected: false,
      });
      return row;
    });
  }

  protected _navigate(ev: ItemClickEvent) {
    const pipelineId = this.pipelines[ev.detail.index].id;
    this.dispatchEvent(new RouteEvent(`/pipelines/details/${pipelineId}`));
  }

  protected _create() {
    this.dispatchEvent(new RouteEvent('/pipelines/new'));
  }
}
