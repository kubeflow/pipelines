import 'iron-icons/iron-icons.html';
import 'neon-animation/web-animations.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-dropdown-menu/paper-dropdown-menu.html';
import 'paper-input/paper-input.html';
import 'paper-item/paper-item.html';
import 'paper-listbox/paper-listbox.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import './pipeline-new.html';

import { customElement, property } from '../../decorators';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { Parameter } from '../../model/parameter';
import { Pipeline } from '../../model/pipeline';
import { PipelinePackage } from '../../model/pipeline_package';

interface NewPipelineQueryParams {
  packageId?: number;
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

  public async refresh(_: string, queryParams: NewPipelineQueryParams) {
    this._busy = true;
    const packageList = this.$.packagesListbox as any;
    try {
      this.packages = await Apis.getPackages();
      if (queryParams.packageId) {
        let packageIdx = -1;
        this.packages.forEach((p, i) => {
          if (p.id === queryParams.packageId) {
            packageIdx = i;
          }
        });
        if (packageIdx === -1) {
          Utils.log.error('Cannot find package with id ' + queryParams.packageId);
          return;
        }
        packageList.selected = packageIdx;
      }
    } finally {
      this._busy = false;
    }
  }

  protected async _packageChanged(e: any) {
    const selectedEl = (this.$.packagesListbox as any).selectedItem;
    if (!selectedEl) {
      return;
    }
    this.packageId = selectedEl.packageId;
    const pkg = this.packages.filter((p) => p.id === this.packageId)[0];
    if (!pkg) {
      Utils.log.error('No package found with id ' + this.packageId);
      return;
    }

    this.parameters = pkg.parameters.map((p) => {
      return {
        description: p.description,
        name: p.name,
        value: p.value || '',
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
      name: (this.$.name as HTMLInputElement).value,
      packageId: this.packageId,
      parameters: this.parameters,
      recurring: false,
      recurringIntervalHours: 0,
    };
    await Apis.newPipeline(newPipeline);

    this.dispatchEvent(new RouteEvent('/pipelines'));
  }
}
