import 'iron-icons/av-icons.html';
import 'iron-icons/iron-icons.html';
import 'iron-icons/maps-icons.html';
import 'paper-button/paper-button.html';
import 'paper-progress/paper-progress.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
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

  @property({ type: Boolean })
  protected _busy = false;

  @property({ computed: '_computeAllowPipelineEnable(pipeline.enabled, pipeline.schedule)',
              type: Boolean })
  protected _allowPipelineEnable = false;

  @property({ computed: '_computeAllowPipelineDisable(pipeline.enabled, pipeline.schedule)',
              type: Boolean })
  protected _allowPipelineDisable = false;

  public async load(path: string) {
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

  protected async _enablePipeline() {
    if (this.pipeline) {
      try {
        this._busy = true;
        await Apis.enablePipeline(this.pipeline.id);
        this.pipeline = await Apis.getPipeline(this.pipeline.id);
        Utils.showNotification('Pipeline enabled');
      } catch (err) {
        Utils.showDialog('Error enabling pipeline: ' + err);
      } finally {
        this._busy = false;
      }
    }
  }

  protected async _disablePipeline() {
    if (this.pipeline) {
      try {
        this._busy = true;
        await Apis.disablePipeline(this.pipeline.id);
        this.pipeline = await Apis.getPipeline(this.pipeline.id);
        Utils.showNotification('Pipeline disabled');
      } catch (err) {
        Utils.showDialog('Error disabling pipeline: ' + err);
      } finally {
        this._busy = false;
      }
    }
  }

  protected _formatDateString(date: string) {
    return Utils.formatDateString(date);
  }

  // Pipeline can only be enabled/disabled if there's a schedule
  protected _computeAllowPipelineEnable(enabled: boolean, schedule: string) {
    return !!schedule && !enabled;
  }

  // Pipeline can only be enabled/disabled if there's a schedule
  protected _computeAllowPipelineDisable(enabled: boolean, schedule: string) {
    return !!schedule && enabled;
  }
}
