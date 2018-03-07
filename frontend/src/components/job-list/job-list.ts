import 'iron-icons/device-icons.html';
import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import { customElement, property } from 'polymer-decorators/src/decorators';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { ItemClickEvent, RouteEvent } from '../../lib/events';
import { Job, JobStatus } from '../../lib/job';

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

  private jobListColumns: ItemListColumn[] = [
    { name: 'Run', type: ColumnTypeName.NUMBER },
    { name: 'Start time', type: ColumnTypeName.DATE },
    { name: 'End time', type: ColumnTypeName.DATE },
  ];

  // TODO: should these jobs be cached?
  public async loadJobs(pipelineId: number) {
    this.jobs = await Apis.getJobs(pipelineId);

    this._drawJobList();
  }

  protected _navigate(ev: ItemClickEvent) {
    const jobId = this.jobs[ev.detail.index].id;
    this.dispatchEvent(new RouteEvent(`/jobs/details/${jobId}`));
  }

  protected _paramsToArray(paramsObject: {}) {
    return Utils.objectToArray(paramsObject);
  }

  protected _getStatusIcon(status: JobStatus) {
    switch (status) {
      case 'Running': return 'device:access-time';
      case 'Succeeded': return 'check';
      case 'Not started': return 'sort';
      default: return 'error-outline';
    }
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
        case 'Not started':
          color = progressCssColors.notStarted;
        default:
          color = progressCssColors.errored;
          break;
      }
      (jobEl.querySelector('paper-progress') as any).updateStyles({
        '--paper-progress-active-color': `var(${color})`,
      });
    });
  }

  /**
   * Creates a new ItemListRow object for each entry in the file list, and sends
   * the created list to the item-list to render.
   */
  private _drawJobList() {
    const itemList = this.$.jobsItemList as ItemListElement;
    itemList.addEventListener('itemDoubleClick', this._navigate.bind(this));
    itemList.columns = this.jobListColumns;
    itemList.rows = this.jobs.map((job) => {
      const row = new ItemListRow({
        columns: [
          job.id,
          new Date(job.startedAt),
          new Date(job.endedAt),
        ],
        selected: false,
      });
      return row;
    });
    this._colorProgressBars();
  }
}
