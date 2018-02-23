/// <reference path="../../../bower_components/polymer/types/lib/elements/dom-repeat.d.ts" />

import 'iron-icons/device-icons.html';
import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from '../../decorators';
import { JobClickEvent, RouteEvent } from '../../lib/events';
import { Job } from '../../lib/job';
import { PageElement } from '../../lib/page_element';

import './job-list.html';

const progressCssColors = {
  completed: '--success-color',
  errored: '--error-color',
  notStarted: '',
  running: '--progress-color',
};

interface JobsQueryParams {
  instanceId?: string;
}

<<<<<<< HEAD:frontend/src/components/job-list/job-list.ts
@customElement('job-list')
export class JobList extends Polymer.Element implements PageElement {
=======
@customElement('run-list')
export class RunList extends Polymer.Element implements PageElement {
>>>>>>> add webpack prod config:frontend/src/components/run-list/run-list.ts

  @property({ type: Array })
  public jobs: Job[] = [];

  @property({ type: String })
  public pageTitle = 'Job list:';

  public async refresh(_: string, queryParams: JobsQueryParams) {
    let id;
    if (queryParams.instanceId) {
      id = Number.parseInt(queryParams.instanceId);
      if (isNaN(id)) {
        id = undefined;
      }
    }
    this.jobs = await Apis.getJobs(id);
    if (id !== undefined) {
      this.pageTitle = `Job list for instance ${id}:`;
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

  protected _dateToString(date: number) {
    return date === -1 ? '-' : new Date(date).toLocaleString();
  }

  protected _getState(state: string) {
    return state[0].toUpperCase() + state.substr(1);
  }

  protected _getStateIcon(state: string) {
    switch (state) {
      case 'running': return 'device:access-time';
      case 'completed': return 'check';
      case 'not started': return 'sort';
      default: return 'error-outline';
    }
  }

  protected _getRuntime(start: number, end: number, state: string) {
    if (state === 'not started') {
      return '-';
    }
    if (end === -1) {
      end = Date.now();
    }
    return Utils.dateDiffToString(end - start);
  }

  private _colorProgressBars() {
    // Make sure the dom-repeat element is flushed, because we iterate
    // on its elements here.
    (Polymer.dom as any).flush();
    const jobsRepeatTemplate = this.$.jobsRepeatTemplate as Polymer.DomRepeat;
    (this.shadowRoot as ShadowRoot).querySelectorAll('.job').forEach((jobEl) => {
      const model = jobsRepeatTemplate.modelForElement(jobEl as HTMLElement);
      let color = '';
      switch ((model as any).job.state) {
        case 'running':
          color = progressCssColors.running;
          break;
        case 'completed':
          color = progressCssColors.completed;
          break;
        case 'not started':
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
}
