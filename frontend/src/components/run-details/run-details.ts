import 'polymer/polymer.html';
import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-dialog/paper-dialog.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';

import '../file-browser/file-browser';
import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';
import PageElement from '../../lib/page_element';
// @ts-ignore
import prettyJson from 'json-pretty-html';
import { Run } from '../../lib/run';
import { customElement, property } from '../../decorators';
import { select as d3select, csvParseRows } from 'd3';

import './run-details.html';
import { TabSelectedEvent } from 'src/lib/events';
import FileBrowser from '../file-browser/file-browser';
import Template from '../../lib/template';

const progressCssColors = {
  completed: '--success-color',
  errored: '--error-color',
  notStarted: '',
  running: '--progress-color',
};

@customElement
export default class RunDetails extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public template: Template | null = null;

  @property({ type: Object })
  public run: Run | null = null;

  public async refresh(path: string) {
    if (path !== '') {
      const id = Number.parseInt(path);
      if (isNaN(id)) {
        Utils.log.error(`Bad run path: ${id}`);
        return;
      }
      this.run = await Apis.getRun(id);
      this.template = await Apis.getTemplate(this.run.templateId);

      (this.$.stepTabs as any).selected = 0;
      this._colorProgressBar();
    }
  }

  protected _stepSelected(e: TabSelectedEvent) {
    if (!this.run) {
      return;
    }
    const step = this.run.steps[e.detail.value];
    if (!step) {
      Utils.log.error('Could not retrieve selected step #' + e.detail.value);
      return;
    }
    (this.$.fileBrowser as FileBrowser).path = step.outputs;
  }

  protected _getParameterArray(params: {}) {
    return Utils.objectToArray(params);
  }

  protected _dateToString(date: number) {
    return date === -1 ? '-' : new Date(date).toLocaleString();
  }

  protected _getState(state: string) {
    if (!state) {
      return '';
    }
    return state[0].toUpperCase() + state.substr(1);
  }

  protected _getStateIcon(state: string) {
    switch (state) {
      case 'running': return 'device:access-time';
      case 'completed': return 'check';
      case 'not started': return 'sort';
      case 'errored': return 'error-outline';
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

  protected async _preview() {
    const browser = this.$.fileBrowser as FileBrowser;
    const selectedFiles = browser.files.filter(f => f.selected);
    if (selectedFiles.length !== 1) {
      return;
    }
    const fileName = selectedFiles[0].name;
    const path = browser.path + '/' + fileName;
    const data = await Apis.readFile(path);

    (this.$.dialogTitle as any).innerText = 'Preview for file: ' + path;
    if (fileName.endsWith('.json')) {
      this.$.preview.innerHTML = prettyJson(JSON.parse(data));
    } else if (fileName.endsWith('.txt')) {
      (this.$.preview as HTMLElement).innerText = data;
    } else if (fileName.endsWith('.csv')) {
      var parsedCSV = csvParseRows(data, (r, i) => {
        if (i < 10) {
          return r;
        }
      });

      this.$.preview.innerHTML = '';
      await d3select(this.$.preview)
        .append("table")

        .selectAll("tr")
        .data(parsedCSV).enter()
        .append("tr")

        .selectAll("td")
        .data(function (d) { return d; }).enter()
        .append("td")
        .text(function (d) { return d; });
    } else {
      Utils.log.error('No render method can be inferred from file name: ' + fileName);
      return;
    }

    (this.$.previewDialog as any).open();
    (this.$.previewDialog as any).center();
  }

  private _colorProgressBar() {
    if (!this.run) {
      return;
    }
    let color = '';
    switch (this.run.state) {
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
    (this.$.progress as any).updateStyles({
      '--paper-progress-active-color': `var(${color})`,
    });
  }
}
