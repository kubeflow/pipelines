import 'app-datepicker/app-datepicker-dialog.html';
import 'iron-icons/iron-icons.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-dropdown-menu/paper-dropdown-menu-light.html';
import 'paper-input/paper-input.html';
import 'paper-item/paper-item.html';
import 'paper-listbox/paper-listbox.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import './pipeline-new.html';

import { customElement, property } from '../../decorators';
import { RouteEvent } from '../../lib/events';
import { PageElement } from '../../lib/page_element';
import { Parameter } from '../../lib/parameter';
import { Pipeline } from '../../lib/pipeline';
import { PipelinePackage } from '../../lib/pipeline_package';

interface NewPipelineQueryParams {
  packageId?: string;
}

@customElement('pipeline-new')
export class PipelineNew extends Polymer.Element implements PageElement {

  @property({ type: Number })
  public packageId: number;

  @property({ type: Array })
  public packages: PipelinePackage[];

  @property({ type: String })
  public startDate = '';

  @property({ type: String })
  public endDate = '';

  @property({ type: Object })
  public parameters: Parameter[];

  protected _busy = false;

  private _uploadFileSizeWarningLimit = 25 * 1024 * 1024;

  public async refresh(_: string, queryParams: NewPipelineQueryParams) {
    let id;
    this._busy = true;
    this.packages = await Apis.getPackages();
    if (queryParams.packageId) {
      id = Number.parseInt(queryParams.packageId);
      if (!isNaN(id)) {
        (this.$.packagesListbox as any).selected = id;
      }
    }
    this._busy = false;
  }

  protected async _packageChanged(e: any) {
    const id = (this.$.packagesListbox as any).selected;
    const pkg = this.packages[id];

    this.parameters = pkg.parameters.map((p) => {
      return {
        description: p.description,
        name: p.name,
        value: '',
      };
    });
  }

  protected _altUpload() {
    (this.$.altFileUpload as HTMLInputElement).click();
  }

  protected async _upload() {
    const files = (this.$.altFileUpload as HTMLInputElement).files;

    if (!files) {
      return;
    }

    const file = files[0];
    const isLarge = file.size > this._uploadFileSizeWarningLimit;
    if (isLarge) {
      Utils.log.warn(
        'Trying to upload a large file, browser might experience slowness or freezing.');
    }

    this._busy = true;
    // First, load the file data into memory.
    const readPromise = new Promise((resolve, reject) => {

      const reader = new FileReader();

      reader.onload = () => resolve(reader.result);
      // TODO: handle file reading errors.
      reader.onerror = () => {
        reject(new Error('Error reading file.'));
      };

      // TODO: this will freeze the UI on large files (>~20MB on my laptop) until
      // they're loaded into memory, and very large files (>~100MB) will crash
      // the browser.
      // One possible solution is to slice the file into small chunks and upload
      // each separately, but this requires the backend to support partial
      // chunk uploads.
      reader.readAsText(file);
    });

    // Now upload the file data to the backend server.
    const pkg = await Apis.uploadPackage(await readPromise);
    this.push('packages', pkg);
    (this.$.packagesListbox as any).selected = (this.$.packagesListbox as any).items.length;
    this._busy = false;
  }

  protected _pickStartDate() {
    const datepicker = this.$.startDatepicker as any;
    datepicker.open();
  }

  protected _pickEndDate() {
    const datepicker = this.$.endDatepicker as any;
    datepicker.open();
  }

  protected async _deploy() {
    const newPipeline: Pipeline = {
      author: '',
      description: (this.$.description as HTMLInputElement).value,
      ends: Date.parse(this.endDate),
      name: (this.$.name as HTMLInputElement).value,
      packageId: this.packageId,
      parameterValues: this.parameters,
      recurring: false,
      recurringIntervalHours: 0,
      starts: Date.parse(this.startDate),
      tags: (this.$.tags as HTMLInputElement).value.split(','),
    };
    await Apis.newPipeline(newPipeline);

    this.dispatchEvent(new RouteEvent('/pipelines'));
  }
}
