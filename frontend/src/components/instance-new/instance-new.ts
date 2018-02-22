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
import { Template } from '../../lib/template';

interface NewInstanceQueryParams {
  templateId?: string;
}

@customElement
export class InstanceNew extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public template: Template;

  @property({ type: String })
  public startDate = '';

  @property({ type: String })
  public endDate = '';

  @property({ type: Object })
  public parameters: Parameter[];

  public async refresh(_: string, queryParams: NewInstanceQueryParams) {
    let id;
    if (queryParams.templateId) {
      id = Number.parseInt(queryParams.templateId);
      if (!isNaN(id)) {
        this.template = await Apis.getTemplate(id);

        this.parameters = this.template.parameters.map((p) => {
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
      parameterValues: this.parameters,
      recurring: false,
      recurringIntervalHours: 0,
      starts: Date.parse(this.startDate),
      tags: (this.$.tags as HTMLInputElement).value.split(','),
      templateId: this.template.id,
    };
    await Apis.newInstance(newInstance);

    this.dispatchEvent(new RouteEvent('/instances'));
  }
}
