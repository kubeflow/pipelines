import 'iron-icons/iron-icons.html';
import 'iron-icons/maps-icons.html';
import 'paper-button/paper-button.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';

import { customElement, property } from '../../decorators';
import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { Pipeline } from '../../model/pipeline';

import './pipeline-details.html';

@customElement('pipeline-details')
export class PipelineDetails extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public pipeline: Pipeline | null = null;

  @property({ type: Number })
  public selectedTab = 0;

  @property({ type: Boolean })
  disableClonePipelineButton = true;

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
      if (this.pipeline) {
        this.disableClonePipelineButton = false;
      }
    }
  }

  protected _clonePipeline() {
    if (this.pipeline) {
      this.dispatchEvent(
        new RouteEvent(
          '/pipelines/new',
          {
            packageId: this.pipeline.packageId,
            parameters: this.pipeline.parameters
          }));
    }
  }

  protected _formatDateString(date: string) {
    return Utils.formatDateString(date);
  }
}
