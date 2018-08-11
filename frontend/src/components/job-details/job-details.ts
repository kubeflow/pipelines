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
import { Job, Trigger } from '../../api/job';
import { DialogResult } from '../../components/popup-dialog/popup-dialog';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { RunList } from '../run-list/run-list';

import './job-details.html';

@customElement('job-details')
export class JobDetails extends PageElement {

  @property({ type: Object })
  public job: Job | null = null;

  @property({ type: Number })
  public selectedTab = 0;

  @property({ type: Boolean })
  protected _disableCloneJobButton = true;

  @property({ type: Boolean })
  protected _busy = false;

  @property({
    computed: '_computeAllowJobEnable(job.enabled, job.trigger)',
    type: Boolean
  })
  protected _allowJobEnable = false;

  @property({
    computed: '_computeAllowJobDisable(job.enabled, job.trigger)',
    type: Boolean
  })
  protected _allowJobDisable = false;

  public get tabs(): PaperTabsElement {
    return this.$.tabs as PaperTabsElement;
  }

  public get cloneButton(): PaperButtonElement {
    return this.$.cloneBtn as PaperButtonElement;
  }

  public get refreshButton(): PaperButtonElement {
    return this.$.refreshBtn as PaperButtonElement;
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

  public async load(id: string): Promise<void> {
    if (!!id) {
      this.selectedTab = 0;

      this._loadJob(id);
    }
  }

  protected _refresh(): void {
    if (this.job) {
      this._loadJob(this.job.id);
    }
  }

  protected async _loadJob(id: string): Promise<void> {
    try {
      const job = await Apis.getJob(id);
      this.job = job;

      (this.$.runs as RunList).loadRuns(this.job.id);
      this._disableCloneJobButton = false;
    } catch (err) {
      this.showPageError(
          'There was an error while loading details for job ' + id, err.message);
      Utils.log.verbose('Error loading job:', err);
    }
  }

  protected _cloneJob(): void {
    if (this.job) {
      this.dispatchEvent(
          new RouteEvent(
            '/jobs/new',
            {
              parameters: this.job.parameters,
              pipelineId: this.job.pipeline_id,
            }));
    }
  }

  protected async _enableJob(): Promise<void> {
    if (this.job) {
      try {
        this._busy = true;
        await Apis.enableJob(this.job.id);
        this.job = await Apis.getJob(this.job.id);
        Utils.showNotification('Job enabled');
      } catch (err) {
        Utils.showDialog('Error enabling job: ' + err);
      } finally {
        this._busy = false;
      }
    }
  }

  protected async _disableJob(): Promise<void> {
    if (this.job) {
      try {
        this._busy = true;
        await Apis.disableJob(this.job.id);
        this.job = await Apis.getJob(this.job.id);
        Utils.showNotification('Job disabled');
      } catch (err) {
        Utils.showDialog('Error disabling job: ' + err);
      } finally {
        this._busy = false;
      }
    }
  }

  protected async _deleteJob(): Promise<void> {
    if (this.job) {
      const dialogResult = await Utils.showDialog(
          'Delete job?',
          'You are about to delete this job. Are you sure you want to proceed?',
          'Delete job',
          'Cancel');

      // BUTTON1 is Delete
      if (dialogResult !== DialogResult.BUTTON1) {
        return;
      }

      this._busy = true;
      try {
        await Apis.deleteJob(this.job.id);

        Utils.showNotification(`Successfully deleted Job: "${this.job.name}"`);

        // Navigate back to Job list page upon successful deletion.
        this.dispatchEvent(new RouteEvent('/jobs'));
      } catch (err) {
        Utils.showDialog('Failed to delete Job', err);
      } finally {
        this._busy = false;
      }
    }
  }

  protected _enabledDisplayString(trigger: Trigger, enabled: boolean): string {
    return Utils.enabledDisplayString(trigger, enabled);
  }

  protected _scheduleDisplayString(): string {
    if (this.job && this.job.trigger) {
      return this.job.trigger.toString();
    }
    return '-';
  }

  protected _formatDateString(date: string): string {
    return Utils.formatDateString(date);
  }

  // Job can only be enabled/disabled if there's a schedule/trigger
  protected _computeAllowJobEnable(enabled: boolean, trigger: Trigger|null): boolean {
    return !!trigger && !enabled;
  }

  // Job can only be enabled/disabled if there's a schedule/trigger
  protected _computeAllowJobDisable(enabled: boolean, trigger: Trigger|null): boolean {
    return !!trigger && enabled;
  }
}
