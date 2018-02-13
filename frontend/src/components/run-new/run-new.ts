import 'app-datepicker/app-datepicker-dialog.html';
import 'iron-icons/iron-icons.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-input/paper-input.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';

import './run-new.html';

import { customElement, property } from '../../decorators';
import { PageElement } from '../../lib/page_element';
import { Parameter } from '../../lib/parameter';
import { Template } from '../../lib/template';

interface NewRunQueryParams {
  templateId?: string;
}

@customElement
export class RunNew extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public template: Template;

  @property({ type: String })
  public startDate = '';

  @property({ type: String })
  public endDate = '';

  @property({ type: Object })
  public parameterValues: Parameter[];

  public async refresh(_: string, queryParams: NewRunQueryParams) {
    let id;
    if (queryParams.templateId) {
      id = Number.parseInt(queryParams.templateId);
      if (!isNaN(id)) {
        this.template = await Apis.getTemplate(id);

        this.parameterValues = this.template.parameters.map((p) => {
          return {
            description: p.description,
            from: 0,
            isSweep: false,
            name: p.name,
            step: 0,
            to: 0,
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
}
