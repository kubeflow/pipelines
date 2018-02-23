import 'app-datepicker/app-datepicker-dialog.html';
import 'iron-icons/iron-icons.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-input/paper-input.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';

import './instance-new.html';

import { customElement, property } from '../../decorators';
import { RouteEvent } from '../../lib/events';
import { Instance } from '../../lib/instance';
import { PageElement } from '../../lib/page_element';
import { Parameter } from '../../lib/parameter';
import { PipelinePackage } from '../../lib/pipeline_package';

interface NewInstanceQueryParams {
  packageId?: string;
}

@customElement('instance-new')
export class InstanceNew extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public package: PipelinePackage;

  @property({ type: String })
  public startDate = '';

  @property({ type: String })
  public endDate = '';

  @property({ type: Object })
  public parameters: Parameter[];

  public async refresh(_: string, queryParams: NewInstanceQueryParams) {
    let id;
    if (queryParams.packageId) {
      id = Number.parseInt(queryParams.packageId);
      if (!isNaN(id)) {
        this.package = await Apis.getPackage(id);

        this.parameters = this.package.parameters.map((p) => {
          return {
            description: p.description,
            name: p.name,
            value: '',
          };
        });
      }
    }
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
    const newInstance: Instance = {
      author: '',
      description: (this.$.description as HTMLInputElement).value,
      ends: Date.parse(this.endDate),
      name: (this.$.name as HTMLInputElement).value,
      packageId: this.package.id,
      parameterValues: this.parameters,
      recurring: false,
      recurringIntervalHours: 0,
      starts: Date.parse(this.startDate),
      tags: (this.$.tags as HTMLInputElement).value.split(','),
    };
    await Apis.newInstance(newInstance);

    this.dispatchEvent(new RouteEvent('/instances'));
  }
}
