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
import { Pipeline } from '../../api/pipeline';
import { PackageTemplate } from '../../api/pipeline_package';
import { OutputInfo, parseTemplateOuputPaths } from '../../lib/template_parser';
import { NodePhase, Workflow } from '../../model/argo_template';
import { RouteEvent } from '../../model/events';
import { OutputMetadata, PlotMetadata } from '../../model/output_metadata';
import { PageElement } from '../../model/page_element';
import { JobGraph } from '../job-graph/job-graph';

import '../data-plotter/data-plot';
import '../job-graph/job-graph';
import './job-details.html';

@customElement('job-details')
export class JobDetails extends PageElement {

  @property({ type: Array })
  public outputPlots: PlotMetadata[] = [];

  @property({ type: Object })
  public workflow: Workflow;

  @property({ type: Object })
  public pipeline: Pipeline;

  @property({ type: Object })
  public packageTemplate: PackageTemplate;

  @property({ type: Number })
  public selectedTab = 0;

  @property({ type: Boolean })
  protected _loadingJob = false;

  @property({ type: Boolean })
  protected _loadingOutputs = false;

  @property({ type: Number })
  protected _pipelineId = -1;

  private _jobId = '';

  public get refreshButton(): PaperButtonElement {
    return this.$.refreshButton as PaperButtonElement;
  }

  public get cloneButton(): PaperButtonElement {
    return this.$.cloneButton as PaperButtonElement;
  }

  public async load(_: string, queryParams: { jobId?: string, pipelineId: number }): Promise<void> {
    this._reset();

    if (queryParams.jobId !== undefined && queryParams.pipelineId > -1) {
      this._pipelineId = queryParams.pipelineId;
      this._jobId = queryParams.jobId;

      this._loadJob();
    }
  }

  protected async _loadJob(): Promise<void> {
    this._loadingJob = true;
    try {
      const response = await Apis.getJob(this._pipelineId, this._jobId);
      this.workflow = JSON.parse(response.workflow);
      this.pipeline = await Apis.getPipeline(this._pipelineId);
      this.packageTemplate = await Apis.getPackageTemplate(this.pipeline.package_id);
    } catch (err) {
      this.showPageError('There was an error while loading details for job: ' + this._jobId);
      Utils.log.error('Error loading job details:', err);
    } finally {
      this._loadingJob = false;
    }

    // Render the job graph
    try {
      (this.$.jobGraph as JobGraph).refresh(this.workflow);
    } catch (err) {
      this.showPageError('There was an error while loading the job graph');
      Utils.log.error('Could not draw job graph from object:', this.workflow);
    }

    // If pipeline params include output, retrieve them so they can be rendered by the data-plot
    // component.
    const baseOutputPath = this._getBaseOutputPath();
    if (baseOutputPath) {
      this._loadJobOutputs(baseOutputPath, this.packageTemplate);
    }
  }

  protected _refresh(): void {
    this._loadJob();
  }

  protected _clone(): void {
    this.dispatchEvent(
        new RouteEvent('/pipelines/new',
          {
            packageId: this.pipeline.package_id,
            parameters: this.pipeline.parameters,
          }));
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

  private _reset(): void {
    // Clear any preexisting page error.
    this._pageError = '';
    // Reset the selected tab each time the user navigates to this page.
    this.selectedTab = 0;
    // Clear outputPlots to keep from re-adding the same outputs over and over.
    this.set('outputPlots', []);
  }

  private async _loadJobOutputs(
      baseOutputPath: string, packageTemplate: PackageTemplate): Promise<void> {
    let outputPaths: OutputInfo[] = [];
    try {
      outputPaths = parseTemplateOuputPaths(packageTemplate, baseOutputPath, this._jobId);
    } catch (err) {
      // TODO: Determine how to display additional error details to user.
      this.showPageError('There was an error while parsing this job\'s YAML template');
      Utils.log.error('Error parsing job YAML:', err);
      return;
    }

    this._loadingOutputs = true;
    try {
      // Build a map of the list of PlotMetadata to their corresponding output
      // details. This map will help us keep outputs sorted by their index first
      // (which is essentially the order of the steps), then by the output path
      // to keep a reproducible order.
      const outputsMap = new Map<PlotMetadata[], OutputInfo>();
      await Promise.all(outputPaths.map(async (outputInfo, outputIndex) => {
        outputInfo.index = outputIndex;
        const fileList = await Apis.listFiles(outputInfo.path);
        const metadataFile = fileList.filter((f) => f.endsWith('/metadata.json'))[0];
        if (metadataFile) {
          const metadataJson = await Apis.readFile(metadataFile);
          const metadata = JSON.parse(metadataJson) as OutputMetadata;
          outputsMap.set(metadata.outputs, outputInfo);
        }
      }));

      this.outputPlots = Array.from(outputsMap.keys()).sort((metadata1, metadata2) => {
        const outputInfo1 = outputsMap.get(metadata1) as OutputInfo;
        const outputInfo2 = outputsMap.get(metadata2) as OutputInfo;
        const index1 = outputInfo1.index as number;
        const index2 = outputInfo2.index as number;
        if (index1 === index2) {
          return outputInfo1.path < outputInfo2.path ? -1 : 1;
        }
        return index1 < index2 ? -1 : 1;
      }).reduce((flattenedOutputs, currentOutputs) => flattenedOutputs.concat(currentOutputs));
    } catch (err) {
      this.showPageError('There was an error while loading details for this job');
      Utils.log.error('Error loading job details:', err);
    } finally {
      this._loadingOutputs = false;
    }
  }

}
