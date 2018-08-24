// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import 'paper-spinner/paper-spinner.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import { Job } from '../../api/job';
import { RouteEvent } from '../../model/events';
import { OutputMetadata, PlotMetadata } from '../../model/output_metadata';
import { PageElement } from '../../model/page_element';
import { StoragePath, StorageService } from '../../model/storage';
import { RuntimeGraph } from '../runtime-graph/runtime-graph';

import '../data-plotter/data-plot';
import '../runtime-graph/runtime-graph';
import './run-details.html';

export interface OutputInfo {
  index?: number;
  path: StoragePath;
  step: string;
}

@customElement('run-details')
export class RunDetails extends PageElement {

  @property({ type: Array })
  public outputPlots: PlotMetadata[] = [];

  @property({ type: Object })
  public workflow: any = undefined;

  @property({ type: Object })
  public job: Job | undefined = undefined;

  @property({ type: Number })
  public selectedTab = -1;

  @property({ type: Boolean })
  protected _loadingRun = false;

  @property({ type: Boolean })
  protected _loadingOutputs = false;

  @property({ type: String })
  protected _jobId = '';

  private _runId = '';

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

  public get runtimeGraph(): RuntimeGraph {
    return this.$.runtimeGraph as RuntimeGraph;
  }

  public get configDetails(): HTMLDivElement {
    return this.$.configDetails as HTMLDivElement;
  }

  public async load(_: string, queryParams: { runId?: string, jobId: string }): Promise<void> {
    this._reset();

    if (queryParams.runId !== undefined && !!queryParams.jobId) {
      this._jobId = queryParams.jobId;
      this._runId = queryParams.runId;

      return this._loadRun();
    }
  }

  protected async _loadRun(): Promise<void> {
    this._loadingRun = true;
    try {
      const response = await Apis.getRun(this._jobId, this._runId);
      this.workflow = JSON.parse(response.workflow) as any;
      this.job = await Apis.getJob(this._jobId);
    } catch (err) {
      this.showPageError(
          'There was an error while loading details for run: ' + this._runId, err.message);
      Utils.log.verbose('Error loading run details:', err);
      return;
    } finally {
      this._loadingRun = false;
    }

    // Render the runtime graph
    try {
      (this.$.runtimeGraph as RuntimeGraph).refresh(this.workflow);
    } catch (err) {
      this.showPageError('There was an error while loading the runtime graph', err.message);
      Utils.log.verbose('Could not draw runtime graph from object:', this.workflow, '\n', err);
    }

    // If job params include output, retrieve them so they can be rendered by the data-plot
    // component.
    try {
      await this._loadRunOutputs();
    } catch (err) {
      this.showPageError('There was an error while loading the run outputs', err.message);
      Utils.log.verbose('Could not load run outputs from object:', this.workflow, '\n', err);
    }
  }

  protected _refresh(): void {
    this._loadRun();
  }

  protected _clone(): void {
    if (!this.workflow || !this.job) {
      return;
    }
    this.dispatchEvent(
        new RouteEvent('/jobs/new',
          {
            parameters: this.workflow.spec.arguments ?
                (this.workflow.spec.arguments.parameters || []) : [],
            pipelineId: this.job.pipeline_id,
          }));
  }

  protected _formatDateString(date: string): string {
    return Utils.formatDateString(date);
  }

  protected _getStatusIcon(status: any): string {
    return Utils.nodePhaseToIcon(status);
  }

  protected _getRunTime(start: string, end: string, status: any): string {
    return Utils.getRunTime(start, end, status);
  }

  protected _getProgressColor(status: any): string {
    return Utils.nodePhaseToColor(status);
  }

  @observe('selectedTab')
  protected _selectedTabChanged(): void {
    const tab = (this.tabs.selectedItem || this.tabs.children[0]) as PaperTabElement | undefined;
    if (this.selectedTab === -1) {
      return;
    }
    if (tab) {
      location.hash = tab.getAttribute('href') || '';
    }
  }

  private _reset(): void {
    const tabElement = this.tabs.querySelector(`[href="${location.hash}"]`);
    this.selectedTab = tabElement ? this.tabs.indexOf(tabElement) : 0;

    // Clear any preexisting page error.
    this._pageError = '';
    // Clear outputPlots to keep from re-adding the same outputs over and over.
    this.set('outputPlots', []);
  }

  private async _loadRunOutputs(): Promise<void> {
    if (!this.workflow) {
      throw new Error('Run workflow object is null.');
    } else if (!this.workflow.status) {
      throw new Error('Run workflow object has no status component.');
    }

    const outputPaths: OutputInfo[] = [];
    Object.keys(this.workflow.status.nodes || []).forEach((id) => {
      const node = this.workflow!.status.nodes[id];
      if (!node.outputs) {
        return;
      }
      (node.outputs.artifacts || [])
        .filter((p: any) => p.name === 'mlpipeline-ui-metadata' && !!p.s3)
        .forEach((p: any) =>
          outputPaths.push({
            path: { source: StorageService.MINIO, bucket: p.s3!.bucket, key: p.s3!.key },
            step: node.displayName,
          })
        );
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
        const metadataFile = await Apis.readFile(outputInfo.path);
        if (metadataFile) {
          try {
            const metadata = JSON.parse(metadataFile) as OutputMetadata;
            outputsMap.set(metadata.outputs, outputInfo);
          } catch (e) {
            Utils.log.verbose('Could not parse metadata file for path: ' +
                JSON.stringify(outputInfo.path));
          }
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
      this.showPageError('There was an error while loading details for this run');
      Utils.log.verbose('Error loading run details:', err.message);
    } finally {
      this._loadingOutputs = false;
    }
  }

}
