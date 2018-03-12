import 'iron-icons/iron-icons.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

// @ts-ignore
import prettyJson from 'json-pretty-html';
import { customElement, property } from '../../decorators';
import { PageElement } from '../../lib/page_element';
import { Job, JobStatus } from '../../model/job';

import './job-details.html';

const progressCssColors = {
  completed: '--success-color',
  errored: '--error-color',
  notStarted: '',
  running: '--progress-color',
};

@customElement('job-details')
export class JobDetails extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public job: Job | null = null;

  public async refresh(_: string, queryParams: { jobId?: string }) {
    if (queryParams.jobId !== undefined) {
      this.job = await Apis.getJob(queryParams.jobId);

      this._colorProgressBar();
    }
  }

  protected _dateToString(date: number) {
    return Utils.dateToString(date);
  }

  protected _getStatusIcon(status: JobStatus) {
    return Utils.jobStatusToIcon(status);
  }

  protected _getRuntime(start: number, end: number, status: JobStatus) {
    if (status === 'Not started') {
      return '-';
    }
    if (end === -1) {
      end = Date.now();
    }
    return Utils.dateDiffToString(end - start);
  }

  private _colorProgressBar() {
    if (!this.job) {
      return;
    }
    let color = '';
    switch (this.job.status) {
      case 'Running':
        color = progressCssColors.running;
        break;
      case 'Succeeded':
        color = progressCssColors.completed;
        break;
      case 'Errored':
        color = progressCssColors.errored;
      default:
        color = progressCssColors.notStarted;
        break;
    }
    (this.$.progress as any).updateStyles({
      '--paper-progress-active-color': `var(${color})`,
    });
  }
}
