import 'iron-icons/iron-icons.html';
import 'neon-animation/web-animations.html';
import 'paper-dropdown-menu/paper-dropdown-menu.html';
import 'paper-input/paper-input.html';
import 'paper-item/paper-item-body.html';
import 'paper-item/paper-item.html';
import 'paper-listbox/paper-listbox.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { Parameter } from '../../model/parameter';
import { Pipeline } from '../../model/pipeline';
import { PipelinePackage } from '../../model/pipeline_package';
import { PipelineSchedule } from '../pipeline-schedule/pipeline-schedule';

import './pipeline-new.html';

interface NewPipelineQueryParams {
  packageId?: number;
}

interface NewPipelineData {
  packageId: number;
  parameters: Parameter[];
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
    computed: '_updateDeployButtonState(_name, _scheduleIsValid)',
    type: Boolean
  })
  protected _inputIsValid = true;

  @property({ type: Boolean })
  protected _scheduleIsValid = true;

  protected _overwriteData?: NewPipelineData;

  private _schedule: PipelineSchedule;

  public async load(_: string, queryParams: NewPipelineQueryParams,
      pipelineData?: NewPipelineData): Promise<void> {
    this._busy = true;
    this._overwriteData = pipelineData;
    const packageList = this.$.packagesListbox as PaperListboxElement;

    // Clear package selection on each component load
    packageList.select(-1);
    this._parameters = [];
    this._packageId = -1;

    // Initialize input to valid to avoid error messages on page load.
    (this.$.name as PaperInputElement).invalid = false;

    this._packageId =
        this._overwriteData ? this._overwriteData.packageId : queryParams.packageId || -1;

    // Reset schedule component on each page load.
    Utils.deleteAllChildren(this.$.schedule as HTMLElement);

    this._schedule = new PipelineSchedule();
    this.$.schedule.appendChild(this._schedule);
    this._schedule.addEventListener(
        'schedule-is-valid-changed', this._scheduleValidationUpdated.bind(this));

    try {
      this.packages = await Apis.getPackages();

      if (this._packageId > -1) {
        // Try to match incoming Pipeline's package to known package.
        this.packages.forEach((p, i) => {
          if (p.id === +this._packageId) {
            // This will cause the observer below to fire before continuing to
            // overwrite the data below.
            this._packageIndex = i;
          }
        });
      }
      if (this._overwriteData) {
        this._packageId = this._overwriteData.packageId;
        // Augment the list of parameters with the overwrite data parameters. To
        // achieve this, first deep clone the parameters array, then for each
        // parameter, check if there one with the same name in the overwrite
        // data, Object.assign them.
        this._parameters = this._parameters || [];
        const augmentedParams = this._parameters.map((p) => ({ ...p }));
        this._overwriteData.parameters.forEach((p) => {
          const param = augmentedParams.filter((_p) => _p.name === p.name);
          if (param.length === 1) {
            param[0] = Object.assign(param[0], p);
          }
        });
        this._parameters = augmentedParams;
      }
    } catch (err) {
      this.showPageError('There was an error while loading packages.');
      Utils.log.error('Error loading packages:', err);
    } finally {
      this._busy = false;
    }
  }

  protected _scheduleValidationUpdated(): void {
    this._scheduleIsValid = this._schedule.scheduleIsValid;
  }

  // Sets Disabled attribute. true === enabled, false === disabled
  protected _updateDeployButtonState(pipelineName: string, scheduleIsValid: boolean): boolean {
    return !!pipelineName && scheduleIsValid;
  }

  protected _packageSelectionChanged(ev: CustomEvent): void {
    // paper-listbox has a known issue where changing the selected item results in two events being
    // fired, one of which has no data.
    // See: https://github.com/PolymerElements/iron-selector/issues/170
    if (ev.detail.value) {
      this._packageId = ev.detail.value.packageId;
      this._parameters = ev.detail.value.parameters;
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
      (this.$.altFileUpload as HTMLInputElement).value = '';
    } catch (err) {
      Utils.showDialog('There was an error uploading the package.');
    } finally {
      this._busy = false;
    }
  }

  protected async _deploy(): Promise<void> {
    const newPipeline = new Pipeline();
    newPipeline.name = this._name;
    newPipeline.description = this._description;
    // TODO: The frontend shouldn't really be sending this, but currently the
    // backend breaks if it receives an empty string, undefined, or null.
    newPipeline.createdAt = Math.floor(Date.now() / 1000);
    newPipeline.packageId = this._packageId;
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
}
