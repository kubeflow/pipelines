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
import { env } from '../../lib/config';

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
  protected _isDev = env === 'dev';

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
    this._busy = true;
    const pkg = await Apis.uploadPackage(file);
    // Add the parsed package to the dropdown list, and select it
    this.push('packages', pkg);
    (this.$.packagesListbox as any).selected = (this.$.packagesListbox as any).items.length;
    this._busy = false;

    (this.$.altFileUpload as HTMLInputElement).value = '';
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
