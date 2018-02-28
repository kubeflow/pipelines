import 'polymer/polymer.html';

import { customElement, property } from '../../decorators';
import * as Apis from '../../lib/apis';
import { PipelineClickEvent, RouteEvent } from '../../lib/events';
import { PageElement } from '../../lib/page_element';
import { Pipeline } from '../../lib/pipeline';

import './pipeline-list.html';

@customElement('pipeline-list')
export class PipelineList extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public pipelines: Pipeline[] = [];

  public async refresh(_: string) {
    this.pipelines = await Apis.getPipelines();
  }

  protected _navigate(ev: PipelineClickEvent) {
    const index = ev.model.pipeline.id;
    this.dispatchEvent(new RouteEvent(`/pipelines/details/${index}`));
  }

  protected _create() {
    this.dispatchEvent(new RouteEvent('/pipelines/new'));
  }
}
