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
import { NodePhase, Workflow } from '../../model/argo_template';
import { RouteEvent } from '../../model/events';
import { OutputMetadata, PlotMetadata } from '../../model/output_metadata';
import { PageElement } from '../../model/page_element';
import { JobGraph } from '../job-graph/job-graph';

import '../data-plotter/data-plot';
import '../job-graph/job-graph';
import './job-details.html';

export interface OutputInfo {
  index?: number;
  path: string;
  step: string;
}

@customElement('job-details')
export class JobDetails extends PageElement {

  @property({ type: Array })
  public outputPlots: PlotMetadata[] = [];

  @property({ type: Object })
  public workflow: Workflow;

  @property({ type: Object })
  public pipeline: Pipeline;

  @property({ type: Number })
  public selectedTab = 0;

  @property({ type: Boolean })
  protected _loadingJob = false;

  @property({ type: Boolean })
  protected _loadingOutputs = false;

  @property({ type: String })
  protected _pipelineId = '';

  private _jobId = '';

  public get refreshButton(): PaperButtonElement {
    return this.$.refreshButton as PaperButtonElement;
  }

  public get cloneButton(): PaperButtonElement {
    return this.$.cloneButton as PaperButtonElement;
  }

  public get tabs(): PaperTabsElement {
    return this.$.tabs as PaperTabsElement;
  }

  public get outputList(): HTMLDivElement {
    return this.$.outputList as HTMLDivElement;
  }

  public get plotContainer(): HTMLDivElement {
    return this.$.plotContainer as HTMLDivElement;
  }

  public get jobGraph(): JobGraph {
    return this.$.jobGraph as JobGraph;
  }

  public async load(_: string, queryParams: { jobId?: string, pipelineId: string }): Promise<void> {
    this._reset();

    if (queryParams.jobId !== undefined && !!queryParams.pipelineId) {
      this._pipelineId = queryParams.pipelineId;
      this._jobId = queryParams.jobId;

      return this._loadJob();
    }
  }

  protected async _loadJob(): Promise<void> {
    this._loadingJob = true;
    try {
      const response = await Apis.getJob(this._pipelineId, this._jobId);
      this.workflow = JSON.parse(response.workflow);
      this.pipeline = await Apis.getPipeline(this._pipelineId);
    } catch (err) {
      this.showPageError(
          'There was an error while loading details for job: ' + this._jobId, err.message);
      Utils.log.verbose('Error loading job details:', err);
      return;
    } finally {
      this._loadingJob = false;
    }

    // Render the job graph
    try {
      (this.$.jobGraph as JobGraph).refresh(this.workflow);
    } catch (err) {
      this.showPageError('There was an error while loading the job graph', err.message);
      Utils.log.verbose('Could not draw job graph from object:', this.workflow, '\n', err);
    }

    // If pipeline params include output, retrieve them so they can be rendered by the data-plot
    // component.
    try {
      await this._loadJobOutputs();
    } catch (err) {
      this.showPageError('There was an error while loading the job outputs', err.message);
      Utils.log.verbose('Could not load job outputs from object:', this.workflow, '\n', err);
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
            parameters: this.workflow.spec.arguments ?
                (this.workflow.spec.arguments.parameters || []) : [],
          }));
  }

  protected _formatDateString(date: string): string {
    return Utils.formatDateString(date);
  }

  protected _getStatusIcon(status: NodePhase): string {
    return Utils.nodePhaseToIcon(status);
  }

  protected _getRunTime(start: string, end: string, status: NodePhase): string {
    return Utils.getRunTime(start, end, status);
  }

  protected _getProgressColor(status: NodePhase): string {
    return Utils.nodePhaseToColor(status);
  }

  private _reset(): void {
    // Clear any preexisting page error.
    this._pageError = '';
    // Reset the selected tab each time the user navigates to this page.
    this.selectedTab = 0;
    // Clear outputPlots to keep from re-adding the same outputs over and over.
    this.set('outputPlots', []);
  }

  private async _loadJobOutputs(): Promise<void> {
    if (!this.workflow) {
      throw new Error('Job workflow object is null.');
    } else if (!this.workflow.status) {
      throw new Error('Job workflow object has no status component.');
    }

    const outputPaths: OutputInfo[] = [];
    Object.keys(this.workflow.status.nodes || []).forEach((id) => {
      const node = this.workflow.status.nodes[id];
      if (!node.inputs) {
        return;
      }
      (node.inputs.parameters || []).filter((p) => p.name === 'output').forEach((p) =>
          outputPaths.push({
            path: p.value!,
            step: node.displayName,
          }));
    });

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
        const metadataFile = fileList.find((f) => f.endsWith('/metadata.json'));
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
      }).reduce((flattenedOutputs, currentOutputs) => flattenedOutputs.concat(currentOutputs), []);
    } catch (err) {
      this.showPageError('There was an error while loading details for this job');
      Utils.log.verbose('Error loading job details:', err.message);
    } finally {
      this._loadingOutputs = false;
    }
  }

}
