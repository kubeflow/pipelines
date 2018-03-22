import { customElement, property } from 'polymer-decorators/src/decorators';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import { PageElement } from '../../lib/page_element';
import { ItemClickEvent, RouteEvent } from '../../model/events';
import { Pipeline } from '../../model/pipeline';
import { ColumnTypeName, ItemListColumn, ItemListElement, ItemListRow } from '../item-list/item-list';

import './pipeline-list.html';

@customElement('pipeline-list')
export class PipelineList extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public pipelines: Pipeline[] = [];

  protected pipelineListRows: ItemListRow[] = [];

  protected pipelineListColumns: ItemListColumn[] = [
    { name: 'Name', type: ColumnTypeName.STRING },
    { name: 'Description', type: ColumnTypeName.STRING },
    { name: 'Package ID', type: ColumnTypeName.NUMBER },
    { name: 'Created', type: ColumnTypeName.DATE },
    { name: 'Recurring', type: ColumnTypeName.STRING },
  ];

  ready() {
    super.ready();
    const itemList = this.$.pipelinesItemList as ItemListElement;
    itemList.addEventListener('itemDoubleClick', this._navigate.bind(this));
  }

  public async refresh(_: string) {
    this.pipelines = (await Apis.getPipelines()).map((p) => {
      if (p.createdAt) {
        p.createdAt = new Date(p.createdAt || '').toLocaleString();
      }
      return p;
    });

    this.pipelineListRows = this.pipelines.map((pipeline) => {
      const row = new ItemListRow({
        columns: [
          pipeline.name,
          pipeline.description,
          pipeline.packageId,
          pipeline.createdAt ? new Date(pipeline.createdAt) : '-',
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

  protected _newPipeline() {
    this.dispatchEvent(new RouteEvent('/pipelines/new'));
  }
}
