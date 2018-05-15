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
export class JobDetails extends PageElement {

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
    // Clear outputPlots to keep from re-adding the same outputs over and over.
    this.set('outputPlots', []);

    if (queryParams.jobId !== undefined && queryParams.pipelineId > -1) {
      this._pipelineId = queryParams.pipelineId;
      this._jobId = queryParams.jobId;

      let templateYaml = '';
      this._loadingJob = true;
      try {
        this.jobDetail = (await Apis.getJob(this._pipelineId, this._jobId)).jobDetail;
        this.pipeline = await Apis.getPipeline(this._pipelineId);
        templateYaml = await Apis.getPackageTemplate(this.pipeline.packageId);
      } catch (err) {
        this.showPageError('There was an error while loading details for job: ' + this._jobId);
        Utils.log.error('Error loading job details:', err);
      } finally {
        this._loadingJob = false;
      }

      // Render the job graph
      try {
        (this.$.jobGraph as JobGraph).refresh(this.jobDetail);
      } catch (err) {
        this.showPageError('There was an error while loading the job graph');
        Utils.log.error('Could not draw job graph from object:', this.jobDetail);
      }

      // If pipeline params include output, retrieve them so they can be rendered by the data-plot
      // component.
      const baseOutputPath = this._getBaseOutputPath();
      if (baseOutputPath) {
        this._loadJobOutputs(baseOutputPath, templateYaml);
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

  private _getBaseOutputPath(): string {
    if (this.pipeline.parameters) {
      const output = this.pipeline.parameters.find((p) => p.name === 'output');

      const baseOutputPathValue = output ? output.value : '';

      return baseOutputPathValue ? baseOutputPathValue.toString() : '';
    }
    return '';
  }

  private async _loadJobOutputs(baseOutputPath: string, templateYaml: string): Promise<void> {
    let outputPaths = [];
    try {
      outputPaths = parseTemplateOuputPaths(templateYaml, baseOutputPath, this._jobId);
    } catch (err) {
      // TODO: Determine how to display additional error details to user.
      this.showPageError('There was an error while parsing this job\'s YAML template');
      Utils.log.error('Error parsing job YAML:', err);
      return;
    }

    this._loadingOutputs = true;
    try {
      await Promise.all(outputPaths.map(async (path) => {
        const fileList = await Apis.listFiles(path);
        const metadataFile = fileList.filter((f) => f.endsWith('/metadata.json'))[0];
        if (metadataFile) {
          const metadataJson = await Apis.readFile(metadataFile);
          const metadata = JSON.parse(metadataJson) as OutputMetadata;
          this.outputPlots = this.outputPlots.concat(metadata.outputs);
        }
      }));
    } catch (err) {
      this.showPageError('There was an error while loading details for this job');
      Utils.log.error('Error loading job details:', err);
    } finally {
      this._loadingOutputs = false;
    }
  }

}
