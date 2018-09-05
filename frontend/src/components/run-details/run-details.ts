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
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import { apiJob } from '../../api/job';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { RuntimeGraph } from '../runtime-graph/runtime-graph';

import '../runtime-graph/runtime-graph';
import './run-details.html';

@customElement('run-details')
export class RunDetails extends PageElement {

  @property({ type: Object })
  public workflow: any = undefined;

  @property({ type: Object })
  public job: apiJob | undefined = undefined;

  @property({ type: Number })
  public selectedTab = -1;

  @property({ type: Boolean })
  protected _loadingRun = false;

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
      this.workflow = JSON.parse(response.workflow!) as any;
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

  protected _formatDateString(date: Date | string): string {
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
  }
}
