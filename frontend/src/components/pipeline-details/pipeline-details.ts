import 'iron-icons/iron-icons.html';
import 'iron-icons/maps-icons.html';
import 'paper-button/paper-button.html';
import 'polymer/polymer.html';

import { customElement, property } from '../../decorators';
import * as Apis from '../../lib/apis';
import { RouteEvent } from '../../lib/events';
import { PageElement } from '../../lib/page_element';
import * as Utils from '../../lib/utils';
import { Pipeline } from '../../model/pipeline';

import './pipeline-details.html';

@customElement('pipeline-details')
export class PipelineDetails extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public pipeline: Pipeline | null = null;

  @property({ type: Number })
  public selectedTab = 0;

  public async refresh(path: string) {
    if (path !== '') {
      const id = Number.parseInt(path);
      if (isNaN(id)) {
        Utils.log.error(`Bad pipeline path: ${id}`);
        return;
      }
      const pipeline = await Apis.getPipeline(id);
      if (pipeline.createdAt) {
        pipeline.createdAt = new Date(pipeline.createdAt).toLocaleString();
      }

      (this.$.jobs as any).loadJobs(pipeline.id);

      this.pipeline = pipeline;
    }
  }

  protected async _runOnce() {
    if (this.pipeline && this.pipeline.id !== undefined) {
      await Apis.newJob(this.pipeline.id);
      this.dispatchEvent(new RouteEvent(`/jobs?pipelineId=${this.pipeline.id}`));
    }
  }
}
