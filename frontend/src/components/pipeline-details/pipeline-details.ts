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
import { Pipeline } from '../../api/pipeline';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { JobList } from '../job-list/job-list';

import './pipeline-details.html';

@customElement('pipeline-details')
export class PipelineDetails extends PageElement {

  @property({ type: Object })
  public pipeline: Pipeline | null = null;

  @property({ type: Number })
  public selectedTab = 0;

  @property({ type: Boolean })
  disableClonePipelineButton = true;

  @property({ type: Boolean })
  protected _busy = false;

  @property({
    computed: '_computeAllowPipelineEnable(pipeline.enabled, pipeline.schedule)',
    type: Boolean
  })
  protected _allowPipelineEnable = false;

  @property({
    computed: '_computeAllowPipelineDisable(pipeline.enabled, pipeline.schedule)',
    type: Boolean
  })
  protected _allowPipelineDisable = false;

  public get cloneButton(): PaperButtonElement {
    return this.$.cloneBtn as PaperButtonElement;
  }

  public get deleteButton(): PaperButtonElement {
    return this.$.deleteBtn as PaperButtonElement;
  }

  public get enableButton(): PaperButtonElement {
    return this.$.enableBtn as PaperButtonElement;
  }

  public get disableButton(): PaperButtonElement {
    return this.$.disableBtn as PaperButtonElement;
  }

  public async load(path: string): Promise<void> {
    if (path !== '') {
      this.selectedTab = 0;
      const id = Number.parseInt(path);
      if (isNaN(id)) {
        Utils.log.error(`Bad pipeline path: ${id}`);
        return;
      }

      try {
        const pipeline = await Apis.getPipeline(id);
        this.pipeline = pipeline;

        (this.$.jobs as JobList).loadJobs(this.pipeline.id);
        this.disableClonePipelineButton = false;
      } catch (err) {
        this.showPageError('There was an error while loading details for pipeline ' + id);
        Utils.log.error('Error loading pipeline:', err);
      }
    }
  }

  protected _clonePipeline(): void {
    if (this.pipeline) {
      this.dispatchEvent(
          new RouteEvent(
            '/pipelines/new',
            {
              packageId: this.pipeline.package_id,
              parameters: this.pipeline.parameters
            }));
    }
  }

  protected async _enablePipeline(): Promise<void> {
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

  protected async _disablePipeline(): Promise<void> {
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

  protected async _deletePipeline(): Promise<void> {
    if (this.pipeline) {
      this._busy = true;
      try {
        await Apis.deletePipeline(this.pipeline.id);

        Utils.showNotification(`Successfully deleted Pipeline: "${this.pipeline.name}"`);

        // Navigate back to Pipeline list page upon successful deletion.
        this.dispatchEvent(new RouteEvent('/pipelines'));
      } catch (err) {
        Utils.showDialog('Failed to delete Pipeline', err);
      } finally {
        this._busy = false;
      }
    }
  }

  protected _enabledDisplayString(schedule: string, enabled: boolean): string {
    return Utils.enabledDisplayString(schedule, enabled);
  }

  protected _formatDateString(date: string): string {
    return Utils.formatDateString(date);
  }

  // Pipeline can only be enabled/disabled if there's a schedule
  protected _computeAllowPipelineEnable(enabled: boolean, schedule: string): boolean {
    return !!schedule && !enabled;
  }

  // Pipeline can only be enabled/disabled if there's a schedule
  protected _computeAllowPipelineDisable(enabled: boolean, schedule: string): boolean {
    return !!schedule && enabled;
  }
}
