import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-dialog/paper-dialog.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { csvParseRows, select as d3select } from 'd3';
// @ts-ignore
import prettyJson from 'json-pretty-html';
import { customElement, property } from '../../decorators';
import { TabSelectedEvent } from '../../lib/events';
import { Job } from '../../lib/job';
import { PageElement } from '../../lib/page_element';
import { Pipeline } from '../../lib/pipeline';
import { DataPlotter } from '../data-plotter/data-plotter';
import { FileBrowser } from '../file-browser/file-browser';

import '../file-browser/file-browser';
import './job-details.html';

const progressCssColors = {
  completed: '--success-color',
  errored: '--error-color',
  notStarted: '',
  running: '--progress-color',
};

const PREVIEW_LINES_COUNT = 10;

@customElement('job-details')
export class JobDetails extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public pipeline: Pipeline | null = null;

  @property({ type: Object })
  public job: Job | null = null;

  public async refresh(path: string) {
    if (path !== '') {
      const id = Number.parseInt(path);
      if (isNaN(id)) {
        Utils.log.error(`Bad job path: ${id}`);
        return;
      }
      this.job = await Apis.getJob(id);
      this.pipeline = await Apis.getPipeline(this.job.pipelineId);

      (this.$.stepTabs as any).selected = 0;
      this._colorProgressBar();

      if (this.job.steps) {
        this._switchToStep(0);
      }
    }
  }

  protected _stepSelected(e: TabSelectedEvent) {
    this._switchToStep(e.detail.value);
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

  protected async _preview() {
    const browser = this.$.fileBrowser as FileBrowser;
    const selectedFiles = browser.files.filter((f) => f.selected);
    if (selectedFiles.length !== 1) {
      return;
    }
    const fileName = selectedFiles[0].name;
    const path = browser.path + '/' + fileName;
    const data = await Apis.readFile(path);
    this.$.preview.innerHTML = '';

    (this.$.previewTitle as any).innerText = 'Preview for file: ' + path;
    if (fileName.endsWith('.json')) {
      this.$.preview.innerHTML = prettyJson(JSON.parse(data));
    } else if (fileName.endsWith('.txt')) {
      (this.$.preview as HTMLElement).innerText = data;
    } else if (fileName.endsWith('.csv')) {
      const parsedCSV = csvParseRows(data);
      (this.$.previewTitle as any).innerText +=
        `. Showing ${PREVIEW_LINES_COUNT} out of ${parsedCSV.length} lines`;
      parsedCSV.splice(PREVIEW_LINES_COUNT);

      await d3select(this.$.preview)
        .append('table')

        .selectAll('tr')
        .data(parsedCSV).enter()
        .append('tr')

        .selectAll('td')
        .data((d) => d).enter()
        .append('td')
        .text((d) => d);
    } else {
      Utils.log.error('No render method can be inferred from file name: ' + fileName);
      return;
    }

    (this.$.previewDialog as any).open();
    (this.$.previewDialog as any).center();
  }

  protected async _plot() {
    const browser = this.$.fileBrowser as FileBrowser;
    const selectedFiles = browser.files.filter((f) => f.selected);
    if (selectedFiles.length !== 1) {
      return;
    }
    const fileName = selectedFiles[0].name;
    const path = browser.path + '/' + fileName;
    this.$.plot.innerHTML = '';
    const lineColor = getComputedStyle(this).getPropertyValue('--accent-color');

    // TODO(yebrahim): use a better way to get the output type
    if (fileName === 'confusion_matrix.json') {
      const data = JSON.parse(await Apis.readFile(path));

      const matrix = data.matrix as number[][];
      const headers = data.headers as string[];

      // Validate inputs
      if (!headers || !matrix) {
        Utils.log.error(
          'Confusion matrix JSON file should contain both "matrix" and "headers" fields');
        return;
      }
      if (!Array.isArray(headers) || !Array.isArray(matrix) || !Array.isArray(matrix[0])) {
        Utils.log.error('Both matrix and headers fields should be arrays');
        return;
      }
      if (typeof matrix[0][0] !== 'number' || typeof headers[0] !== 'string') {
        Utils.log.error('Matrix data should be numeric, headers should be strings');
        return;
      }

      const d = new DataPlotter(this.$.plot as HTMLElement);
      await d.plotConfusionMatrix(data.matrix, data.headers, lineColor);

      (this.$.plotTitle as any).innerText = 'Confusion matrix plot from file: ' + path;
      (this.$.plotDialog as any).open();
      (this.$.plotDialog as any).center();

    } else if (fileName === 'roc.csv') {
      const data = csvParseRows(await Apis.readFile(path));
      const d = new DataPlotter(this.$.plot as HTMLElement);
      await d.plotRocCurve(data, lineColor);
      (this.$.plotTitle as any).innerText = 'ROC curve from file: ' + path;
    } else {
      Utils.log.error('No plot method can be inferred from file name: ' + fileName);
      return;
    }

    (this.$.plotDialog as any).open();
    (this.$.plotDialog as any).center();
  }

  private _colorProgressBar() {
    if (!this.job) {
      return;
    }
    let color = '';
    switch (this.job.state) {
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

  private _switchToStep(index: number) {
    if (!this.job || !this.job.steps) {
      return;
    }
    const step = this.job.steps[index];
    if (!step) {
      Utils.log.error('Could not retrieve selected step #' + index);
      return;
    }
    (this.$.fileBrowser as FileBrowser).path = step.outputs;
  }
}
