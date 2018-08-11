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
import { Job } from '../../api/job';
import { ListPipelinesRequest } from '../../api/list_pipelines_request';
import { Parameter } from '../../api/parameter';
import { Pipeline } from '../../api/pipeline';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { JobSchedule } from '../job-schedule/job-schedule';

import './job-new.html';

interface NewJobQueryParams {
  pipelineId?: number;
}

interface NewJobData {
  pipelineId?: number;
  parameters?: Parameter[];
}

@customElement('job-new')
export class JobNew extends PageElement {

  @property({ type: Array })
  public pipelines: Pipeline[] = [];

  @property({ type: Number })
  protected _pipelineIndex = -1;

  @property({ type: Number })
  protected _pipelineId = -1;

  @property({ type: String })
  protected _description = '';

  @property({ type: String })
  protected _name = '';

  @property({ type: Array })
  protected _parameters: Parameter[] = [];

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

  private _schedule: JobSchedule;

  public get schedule(): JobSchedule {
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
    this._pipelineId = -1;
    if (queryParams.pipelineId !== undefined) {
      this._pipelineId = queryParams.pipelineId;
    }
    if (this._overwriteData && this._overwriteData.pipelineId) {
      this._pipelineId = this._overwriteData.pipelineId;
    }

    try {
      const response = await Apis.listPipelines(new ListPipelinesRequest());
      this.pipelines = response.pipelines || [];

      if (this._pipelineId > -1) {
        // Try to match incoming job's pipeline to known pipeline.
        this.pipelines.forEach((p, i) => {
          if (p.id === +this._pipelineId) {
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
          let param = this._parameters.find((_p) => _p.name === p.name);
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
    this._scheduleIsValid = this._schedule.scheduleIsValid;
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

  protected _altUpload(): void {
    (this.$.altFileUpload as HTMLInputElement).click();
  }

  protected async _upload(): Promise<void> {
    const files = (this.$.altFileUpload as HTMLInputElement).files;

    if (!files) {
      return;
    }

    const file = files[0];
    this._busy = true;
    try {
      const pkg = await Apis.uploadPipeline(file);
      // Add the parsed pipeline to the dropdown list, and select it
      this.push('pipelines', pkg);
      (this.$.pipelinesListbox as PaperListboxElement).selected =
          (this.$.pipelinesListbox as PaperListboxElement).items!.length;
    } catch (err) {
      Utils.showDialog('There was an error uploading the pipeline.', err);
    } finally {
      (this.$.altFileUpload as HTMLInputElement).value = '';
      this._busy = false;
    }
  }

  protected async _deploy(): Promise<void> {
    const newJob = new Job();
    newJob.name = this._name;
    newJob.description = this._description;
    newJob.enabled = true;
    newJob.pipeline_id = this._pipelineId;
    newJob.parameters = this._parameters;
    newJob.max_concurrency = this._schedule.maxConcurrentRuns;
    const trigger = this._schedule.toTrigger();
    if (trigger) {
      newJob.trigger = trigger;
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
    const pipelineList = this.$.pipelinesListbox as PaperListboxElement;
    pipelineList.select(-1);
    this._parameters = [];
    this._pipelineId = -1;

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
