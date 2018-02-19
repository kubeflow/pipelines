/// <reference path="../../../bower_components/polymer/types/lib/elements/dom-repeat.d.ts" />

import 'iron-icons/device-icons.html';
import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from '../../decorators';
import { RouteEvent, RunClickEvent } from '../../lib/events';
import { PageElement } from '../../lib/page_element';
import { Run } from '../../lib/run';

import './run-list.html';

const progressCssColors = {
  completed: '--success-color',
  errored: '--error-color',
  notStarted: '',
  running: '--progress-color',
};

interface RunsQueryParams {
  instanceId?: string;
}

@customElement
export class RunList extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public runs: Run[] = [];

  @property({ type: String })
  public pageTitle = 'Run list:';

  public async refresh(_: string, queryParams: RunsQueryParams) {
    let id;
    if (queryParams.instanceId) {
      id = Number.parseInt(queryParams.instanceId);
      if (isNaN(id)) {
        id = undefined;
      }
    }
    this.runs = await Apis.getRuns(id);
    if (id !== undefined) {
      this.pageTitle = `Run list for instance ${id}:`;
    }
    this._colorProgressBars();
  }

  protected _navigate(ev: RunClickEvent) {
    const index = ev.model.run.id;
    this.dispatchEvent(new RouteEvent(`/runs/details/${index}`));
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
    const runsRepeatTemplate = this.$.runsRepeatTemplate as Polymer.DomRepeat;
    (this.shadowRoot as ShadowRoot).querySelectorAll('.run').forEach((runEl) => {
      const model = runsRepeatTemplate.modelForElement(runEl as HTMLElement);
      let color = '';
      switch ((model as any).run.state) {
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
      (runEl.querySelector('paper-progress') as any).updateStyles({
        '--paper-progress-active-color': `var(${color})`,
      });
    });
  }
}
