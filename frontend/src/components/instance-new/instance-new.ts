import 'polymer/polymer.html';

import 'app-datepicker/app-datepicker-dialog.html';
import 'iron-icons/iron-icons.html';
import 'paper-input/paper-input.html';

import * as Apis from '../../lib/apis';
import PageElement from '../../lib/page_element';
import { customElement, property } from '../../decorators';

import './instance-new.html';
import Template from 'src/lib/template';

interface NewInstanceQueryParams {
  templateId?: string;
}

@customElement
export default class InstanceNew extends Polymer.Element implements PageElement {

  @property({ type: String })
  public instanceId = '';

  @property({ type: Object })
  public template: Template;

  @property({ type: String })
  public startDate = '';

  @property({ type: String })
  public endDate = '';

  public async refresh(_: string, queryParams: NewInstanceQueryParams) {
    let id = undefined;
    if (queryParams.templateId) {
      id = Number.parseInt(queryParams.templateId);
      if (!isNaN(id)) {
        this.template = await Apis.getTemplate(id);
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
