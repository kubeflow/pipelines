import 'iron-icons/device-icons.html';
import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import { customElement, property } from 'polymer-decorators/src/decorators';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { JobClickEvent, RouteEvent } from '../../lib/events';
import { Job, JobStatus } from '../../lib/job';
import { PageElement } from '../../lib/page_element';

import './job-list.html';

const progressCssColors = {
  completed: '--success-color',
  errored: '--error-color',
  notStarted: '',
  running: '--progress-color',
};

interface JobsQueryParams {
  pipelineId?: string;
}

@customElement('job-list')
export class JobList extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public jobs: Job[] = [];

  @property({ type: String })
  public pageTitle = 'Job list:';

  public async refresh(_: string, queryParams: JobsQueryParams) {
    const id = Number.parseInt(queryParams.pipelineId || '');
    if (!queryParams.pipelineId || isNaN(id)) {
      Utils.log.error('No valid pipeline id specified.');
      return;
    }
    this.jobs = await Apis.getJobs(id);
    if (id !== undefined) {
      this.pageTitle = `Job list for pipeline ${id}:`;
    } else {
      this.pageTitle = 'Job list:';
    }
    this._colorProgressBars();
  }

  protected _navigate(ev: JobClickEvent) {
    const index = ev.model.job.id;
    this.dispatchEvent(new RouteEvent(`/jobs/details/${index}`));
  }

  protected _paramsToArray(paramsObject: {}) {
    return Utils.objectToArray(paramsObject);
  }

  protected _dateToString(date: string) {
    return date ? new Date(date).toLocaleString() : '-';
  }

  protected _getStatusIcon(status: JobStatus) {
    return Utils.jobStatusToIcon(status);
  }

  protected _getRuntime(start: string, end: string, status: JobStatus) {
    if (status === 'Not started') {
      return '-';
    }
    const startDate = new Date(start);
    const endDate = end ? new Date(end) : new Date();
    return Utils.dateDiffToString(endDate.valueOf() - startDate.valueOf());
  }

  private _colorProgressBars() {
    // Make sure the dom-repeat element is flushed, because we iterate
    // on its elements here.
    (Polymer.dom as any).flush();
    const jobsRepeatTemplate = this.$.jobsRepeatTemplate as Polymer.DomRepeat;
    (this.shadowRoot as ShadowRoot).querySelectorAll('.job').forEach((jobEl) => {
      const model = jobsRepeatTemplate.modelForElement(jobEl as HTMLElement);
      let color = '';
      switch ((model as any).job.status as JobStatus) {
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
      (jobEl.querySelector('paper-progress') as any).updateStyles({
        '--paper-progress-active-color': `var(${color})`,
      });
    });
  }
}
