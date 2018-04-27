import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import 'paper-spinner/paper-spinner.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

// @ts-ignore
import prettyJson from 'json-pretty-html';
import { customElement, property } from 'polymer-decorators/src/decorators';
import { parseTemplateOuputPaths } from '../../lib/template_parser';
import { NodePhase, Workflow } from '../../model/argo_template';
import { OutputMetadata, PlotMetadata } from '../../model/output_metadata';
import { PageElement } from '../../model/page_element';
import { Pipeline } from '../../model/pipeline';
import { JobGraph } from '../job-graph/job-graph';

import '../data-plotter/data-plot';
import '../job-graph/job-graph';
import './job-details.html';

@customElement('job-details')
export class JobDetails extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public outputPlots: PlotMetadata[] = [];

  @property({ type: Object })
  public jobDetail: Workflow;

  @property({ type: Object })
  public pipeline: Pipeline;

  @property({ type: Number })
  public selectedTab = 0;

  @property({ type: Boolean })
  protected _loadingJob = false;

  @property({ type: Boolean })
  protected _loadingOutputs = false;

  @property({ type: Number })
  protected _pipelineId = -1;

  private _jobId = '';

  public async load(_: string, queryParams: { jobId?: string, pipelineId: number }): Promise<void> {
    // Reset the selected tab each time the user navigates to this page.
    this.selectedTab = 0;

    if (queryParams.jobId !== undefined && queryParams.pipelineId > -1) {
      this._pipelineId = queryParams.pipelineId;
      this._jobId = queryParams.jobId;

      let templateYaml = '';
      this._loadingJob = true;
      try {
        this.jobDetail = (await Apis.getJob(this._pipelineId, this._jobId)).jobDetail;
        this.pipeline = await Apis.getPipeline(this._pipelineId);
        templateYaml = await Apis.getPackageTemplate(this.pipeline.packageId);
      } finally {
        // TODO: Handle errors.
        this._loadingJob = false;
      }

      // Render the job graph
      try {
        (this.$.jobGraph as JobGraph).refresh(this.jobDetail);
      } catch (err) {
        Utils.log.error('Could not draw job graph from object:', this.jobDetail);
      }

      const baseOutputPathValue = this.pipeline
        .parameters
        .filter((p) => p.name === 'output')[0]
        .value;
      const baseOutputPath = baseOutputPathValue ? baseOutputPathValue.toString() : '';

      // TODO: Catch and show template parsing errors here
      const outputPaths = parseTemplateOuputPaths(templateYaml, baseOutputPath, this._jobId);

      // Clear outputPlots to keep from re-adding the same outputs over and over.
      this.set('outputPlots', []);

      this._loadingOutputs = true;
      try {
        outputPaths.forEach(async (path) => {
          const fileList = await Apis.listFiles(path);
          const metadataFile = fileList.filter((f) => f.endsWith('/metadata.json'))[0];
          if (metadataFile) {
            const metadataJson = await Apis.readFile(metadataFile);
            const metadata = JSON.parse(metadataJson) as OutputMetadata;
            this.outputPlots = this.outputPlots.concat(metadata.outputs);
          }
        });
      } finally {
        // TODO: Handle errors.
        this._loadingOutputs = false;
      }
    }
  }

  protected _formatDateString(date: string): string {
    return Utils.formatDateString(date);
  }

  protected _getStatusIcon(status: NodePhase): string {
    return Utils.nodePhaseToIcon(status);
  }

  protected _getRuntime(start: string, end: string, status: NodePhase): string {
    if (!status) {
      return '-';
    }
    const parsedStart = start ? new Date(start).getTime() : 0;
    const parsedEnd = end ? new Date(end).getTime() : Date.now();

    return (parsedStart && parsedEnd) ?
      Utils.dateDiffToString(parsedEnd - parsedStart) : '-';
  }

  protected _getProgressColor(status: NodePhase): string {
    return Utils.nodePhaseToColor(status);
  }
}
