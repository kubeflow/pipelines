import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';
import '../data-plotter/data-plot';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

// @ts-ignore
import prettyJson from 'json-pretty-html';
import { customElement, property } from '../../decorators';
import { PageElement } from '../../lib/page_element';
import { parseTemplateOuputPaths } from '../../lib/template_parser';
import { Job, JobStatus } from '../../model/job';
import { PlotMetadata } from '../../model/output_metadata';

import './job-details.html';

const progressCssColors = {
  completed: '--success-color',
  errored: '--error-color',
  notStarted: '',
  running: '--progress-color',
};

@customElement('job-details')
export class JobDetails extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public outputPlots: PlotMetadata[] = [];

  @property({ type: Object })
  public job: Job | null = null;

  @property({ type: Number })
  public selectedTab = 0;

  private _pipelineId = -1;
  private _jobId = '';

  public async refresh(_: string, queryParams: { jobId?: string, pipelineId: number }) {
    if (queryParams.jobId !== undefined && queryParams.pipelineId > -1) {
      this._pipelineId = queryParams.pipelineId;
      this._jobId = queryParams.jobId;
      this.job = await Apis.getJob(this._pipelineId, this._jobId);

      const pipeline = await Apis.getPipeline(this._pipelineId);
      const templateYaml = await Apis.getPackageTemplate(pipeline.packageId);

      const baseOutputPath = pipeline
        .parameters
        .filter((p) => p.name === 'output')[0]
        .value.toString();

      const outputPaths = parseTemplateOuputPaths(templateYaml, baseOutputPath, this._jobId);

      // Clear outputPlots to keep from re-adding the same outputs over and over.
      this.set('outputPlots', [])

      outputPaths.forEach(async (path) => {
        const fileList = await Apis.listFiles(path);
        const metadataFile = fileList.filter((f) => f.endsWith('metadata.json'))[0];
        if (metadataFile) {
          const metadataJson = await Apis.readFile(metadataFile);
          this.push('outputPlots', JSON.parse(metadataJson) as PlotMetadata);
        }
      });

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
