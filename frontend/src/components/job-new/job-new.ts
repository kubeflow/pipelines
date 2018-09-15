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
import 'neon-animation/web-animations.html';
import 'paper-dropdown-menu/paper-dropdown-menu.html';
import 'paper-input/paper-input.html';
import 'paper-input/paper-textarea.html';
import 'paper-item/paper-item-body.html';
import 'paper-item/paper-item.html';
import 'paper-listbox/paper-listbox.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import { apiJob } from '../../api/job';
import { apiParameter, apiPipeline } from '../../api/pipeline';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { JobSchedule } from '../job-schedule/job-schedule';
import { PipelineUploadDialog } from '../pipeline-upload-dialog/pipeline-upload-dialog';
import { DialogResult } from '../popup-dialog/popup-dialog';

import '../date-time-picker/date-time-picker';

import './job-new.html';

interface NewJobQueryParams {
  pipelineId?: string;
}

interface NewJobData {
  pipelineId?: string;
  parameters?: apiParameter[];
}

@customElement('job-new')
export class JobNew extends PageElement {

  @property({ type: Array })
  public pipelines: apiPipeline[] = [];

  @property({ type: Number })
  protected _pipelineIndex = -1;

  @property({ type: Number })
  protected _pipelineId: string | undefined = '';

  @property({ type: String })
  protected _description = '';

  @property({ type: String })
  protected _name = '';

  @property({ type: Array })
  protected _parameters: apiParameter[] | undefined = [];

  @property({ type: Boolean })
  protected _busy = false;

  @property({
    computed: '_updateDeployButtonState(_pipelineIndex, _name, _scheduleIsValid)',
    type: Boolean
  })
  protected _inputIsValid = true;

  @property({ type: Boolean })
  protected _scheduleIsValid = true;

  protected _overwriteData?: NewJobData;

  private _schedule: JobSchedule | undefined = undefined;

  public get schedule(): JobSchedule | undefined {
    return this._schedule;
  }

  public get listBox(): PaperListboxElement {
    return this.$.pipelinesListbox as PaperListboxElement;
  }

  public get nameInput(): PaperInputElement {
    return this.$.name as PaperInputElement;
  }

  public get descriptionInput(): PaperInputElement {
    return this.$.description as PaperInputElement;
  }

  public get deployButton(): PaperButtonElement {
    return this.$.deployButton as PaperButtonElement;
  }

  public async load(_: string, queryParams: NewJobQueryParams,
      jobData?: NewJobData): Promise<void> {
    this._busy = true;

    // Clear previous state.
    this._reset();

    this._overwriteData = jobData;
    this._pipelineId = '';
    if (queryParams.pipelineId !== undefined) {
      this._pipelineId = queryParams.pipelineId;
    }
    if (this._overwriteData && this._overwriteData.pipelineId) {
      this._pipelineId = this._overwriteData.pipelineId;
    }

    try {
      // TODO: It's still being decided if this Pipeline selector will be part of the new job page,
      // and if it is, how it will work. Using 25 as a page size here as a temporary measure until
      // that's worked out.
      const response = await Apis.listPipelines({ pageSize: 25 });
      this.pipelines = response.pipelines || [];

      if (this._pipelineId) {
        // Try to match incoming job's pipeline to known pipeline.
        this.pipelines.forEach((p, i) => {
          if (p.id === this._pipelineId) {
            // This will cause the observer below to fire before continuing to overwrite the data
            // below.
            this._pipelineIndex = i;
          }
        });
      }
      if (this._overwriteData && this._overwriteData.parameters) {
        // Augment the list of parameters with the overwrite data parameters. To achieve this, check
        // if there one with the same name in the overwrite data, Object.assign them.
        this._overwriteData.parameters.forEach((p) => {
          let param = (this._parameters || []).find((_p) => _p.name === p.name);
          if (param) {
            param = Object.assign(param, p);
          }
        });
      }
    } catch (err) {
      this.showPageError('There was an error while loading pipelines.', err);
    } finally {
      this._busy = false;
    }

    // Reset focus to Job name. Called last to avoid focusing being taken by property changes.
    (this.$.name as PaperInputElement).focus();
  }

  protected _scheduleValidationUpdated(): void {
    if (this._schedule) {
      this._scheduleIsValid = this._schedule.scheduleIsValid;
    }
  }

  // Sets Disabled attribute. true === enabled, false === disabled
  protected _updateDeployButtonState(
      pipelineIndex: number, jobName: string, scheduleIsValid: boolean): boolean {
    return pipelineIndex >= 0 && !!jobName && scheduleIsValid;
  }

  @observe('_pipelineIndex')
  protected _pipelineSelectionChanged(newIndex: number): void {
    if (newIndex >= 0) {
      this._pipelineId = this.pipelines[newIndex].id;
      this._parameters = this.pipelines[newIndex].parameters || [];
    }
  }

  protected async _upload(): Promise<void> {
    const result = await new PipelineUploadDialog().open();

    // BUTTON1 is Upload
    if (result.buttonPressed === DialogResult.BUTTON1 && result.pipeline) {
      Utils.showNotification(`Successfully uploaded pipeline: ${result.pipeline.name}`);
      // Add the parsed pipeline to the dropdown list, and select it
      this.push('pipelines', result.pipeline);
      this.listBox.selected = this.listBox.items!.length;
    }
  }

  protected async _deploy(): Promise<void> {
    const newJob: apiJob = {
      description: this._description,
      enabled: true,
      name: this._name,
      parameters: this._parameters as any,
      pipeline_id: this._pipelineId || '',
    };
    if (this._schedule) {
      newJob.max_concurrency = this._schedule.maxConcurrentRuns.toString();
      const trigger = this._schedule.toTrigger();
      if (trigger) {
        newJob.trigger = trigger;
      }
    }
    this._busy = true;
    try {
      await Apis.newJob(newJob);
      this.dispatchEvent(new RouteEvent('/jobs'));
      Utils.showNotification(`Successfully deployed Job!`);
    } catch (err) {
      Utils.showDialog('There was an error deploying the job.', err);
    } finally {
      this._busy = false;
    }
  }

  private _reset(): void {
    this._name = '';
    this._description = '';
    this._pageError = '';

    // Clear pipeline selection on each component load
    const pipelineList = this.listBox;
    pipelineList.select(-1);
    this._parameters = [];
    this._pipelineId = '';

    // Initialize input to valid to avoid error messages on page load.
    (this.$.name as PaperInputElement).invalid = false;

    // Reset schedule component.
    Utils.deleteAllChildren(this.$.schedule as HTMLElement);

    this._schedule = new JobSchedule();
    this.$.schedule.appendChild(this._schedule);
    this._schedule.addEventListener(
        'schedule-is-valid-changed', this._scheduleValidationUpdated.bind(this));
  }
}
