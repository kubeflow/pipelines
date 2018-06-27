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
import { ListPackagesRequest } from '../../api/list_packages_request';
import { Parameter } from '../../api/parameter';
import { Pipeline } from '../../api/pipeline';
import { PipelinePackage } from '../../api/pipeline_package';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { PipelineSchedule } from '../pipeline-schedule/pipeline-schedule';

import './pipeline-new.html';

interface NewPipelineQueryParams {
  packageId?: number;
}

interface NewPipelineData {
  packageId?: number;
  parameters?: Parameter[];
}

@customElement('pipeline-new')
export class PipelineNew extends PageElement {

  @property({ type: Array })
  public packages: PipelinePackage[];

  @property({ type: Number })
  protected _packageIndex = -1;

  @property({ type: Number })
  protected _packageId = -1;

  @property({ type: String })
  protected _description = '';

  @property({ type: String })
  protected _name = '';

  @property({ type: Array })
  protected _parameters: Parameter[] = [];

  @property({ type: Boolean })
  protected _busy = false;

  @property({
    computed: '_updateDeployButtonState(_packageIndex, _name, _scheduleIsValid)',
    type: Boolean
  })
  protected _inputIsValid = true;

  @property({ type: Boolean })
  protected _scheduleIsValid = true;

  protected _overwriteData?: NewPipelineData;

  private _schedule: PipelineSchedule;

  public get listBox(): PaperListboxElement {
    return this.$.packagesListbox as PaperListboxElement;
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

  public async load(_: string, queryParams: NewPipelineQueryParams,
      pipelineData?: NewPipelineData): Promise<void> {
    this._busy = true;

    // Clear previous state.
    this._reset();

    this._overwriteData = pipelineData;
    this._packageId = -1;
    if (queryParams.packageId !== undefined) {
      this._packageId = queryParams.packageId;
    }
    if (this._overwriteData && this._overwriteData.packageId) {
      this._packageId = this._overwriteData.packageId;
    }

    try {
      const response = await Apis.getPackages(new ListPackagesRequest());
      this.packages = response.packages || [];

      if (this._packageId > -1) {
        // Try to match incoming Pipeline's package to known package.
        this.packages.forEach((p, i) => {
          if (p.id === +this._packageId) {
            // This will cause the observer below to fire before continuing to overwrite the data
            // below.
            this._packageIndex = i;
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
      this.showPageError('There was an error while loading packages.', err);
    } finally {
      this._busy = false;
    }

    // Reset focus to Pipeline name. Called last to avoid focusing being taken by property changes.
    (this.$.name as PaperInputElement).focus();
  }

  protected _scheduleValidationUpdated(): void {
    this._scheduleIsValid = this._schedule.scheduleIsValid;
  }

  // Sets Disabled attribute. true === enabled, false === disabled
  protected _updateDeployButtonState(
      packageIndex: number, pipelineName: string, scheduleIsValid: boolean): boolean {
    return packageIndex >= 0 && !!pipelineName && scheduleIsValid;
  }

  @observe('_packageIndex')
  protected _packageSelectionChanged(newIndex: number): void {
    if (newIndex >= 0) {
      this._packageId = this.packages[newIndex].id;
      this._parameters = this.packages[newIndex].parameters || [];
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
      const pkg = await Apis.uploadPackage(file);
      // Add the parsed package to the dropdown list, and select it
      this.push('packages', pkg);
      (this.$.packagesListbox as PaperListboxElement).selected =
          (this.$.packagesListbox as PaperListboxElement).items!.length;
    } catch (err) {
      Utils.showDialog('There was an error uploading the package.', err);
    } finally {
      (this.$.altFileUpload as HTMLInputElement).value = '';
      this._busy = false;
    }
  }

  protected async _deploy(): Promise<void> {
    const newPipeline = new Pipeline();
    newPipeline.name = this._name;
    newPipeline.description = this._description;
    // TODO: The frontend shouldn't really be sending this, but currently the
    // backend breaks if it receives an empty string, undefined, or null.
    newPipeline.created_at = new Date().toISOString();
    newPipeline.package_id = this._packageId;
    newPipeline.parameters = this._parameters;
    newPipeline.schedule = this._schedule.scheduleAsUTCCrontab();
    this._busy = true;
    try {
      await Apis.newPipeline(newPipeline);
      this.dispatchEvent(new RouteEvent('/pipelines'));
      Utils.showNotification(`Successfully deployed Pipeline!`);
    } catch (err) {
      Utils.showDialog('There was an error deploying the pipeline.', err);
    } finally {
      this._busy = false;
    }
  }

  private _reset(): void {
    this._name = '';
    this._description = '';
    this._pageError = '';

    // Clear package selection on each component load
    const packageList = this.$.packagesListbox as PaperListboxElement;
    packageList.select(-1);
    this._parameters = [];
    this._packageId = -1;

    // Initialize input to valid to avoid error messages on page load.
    (this.$.name as PaperInputElement).invalid = false;

    // Reset schedule component.
    Utils.deleteAllChildren(this.$.schedule as HTMLElement);

    this._schedule = new PipelineSchedule();
    this.$.schedule.appendChild(this._schedule);
    this._schedule.addEventListener(
        'schedule-is-valid-changed', this._scheduleValidationUpdated.bind(this));
  }
}
