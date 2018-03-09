import 'iron-icons/device-icons.html';
import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import { customElement, property } from 'polymer-decorators/src/decorators';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { ItemClickEvent, RouteEvent } from '../../lib/events';
import { Job, JobStatus } from '../../model/job';

import { ColumnTypeName, ItemListColumn, ItemListElement, ItemListRow } from '../item-list/item-list';
import './job-list.html';

const progressCssColors = {
  completed: '--success-color',
  errored: '--error-color',
  notStarted: '',
  running: '--progress-color',
};

@customElement('job-list')
export class JobList extends Polymer.Element {

  @property({ type: Array })
  public jobs: Job[] = [];

  protected jobListRows: ItemListRow[] = [];

  protected jobListColumns: ItemListColumn[] = [
    { name: 'Job Name', type: ColumnTypeName.STRING },
    { name: 'Create time', type: ColumnTypeName.DATE },
    { name: 'Start time', type: ColumnTypeName.DATE },
    { name: 'Finish time', type: ColumnTypeName.DATE },
  ];

  ready() {
    super.ready();
    const itemList = this.$.jobsItemList as ItemListElement;
    itemList.addEventListener('itemDoubleClick', this._navigate.bind(this));
  }

  // TODO: should these jobs be cached?
  public async loadJobs(pipelineId: number) {
    this.jobs = await Apis.getJobs(pipelineId);

    this.jobListRows = this.jobs.map((job) => {
      const row = new ItemListRow({
        columns: [
          job.name,
          new Date(job.createdAt),
          new Date(job.startedAt),
          new Date(job.finishedAt),
        ],
        icon: this._getStatusIcon(job.status),
        selected: false,
      });
      return row;
    });

    this._colorProgressBars();
  }

  protected _navigate(ev: ItemClickEvent) {
    const jobId = this.jobs[ev.detail.index].name;
    this.dispatchEvent(new RouteEvent(`/jobs/details?jobId=${jobId}`));
  }

  protected _paramsToArray(paramsObject: {}) {
    return Utils.objectToArray(paramsObject);
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
