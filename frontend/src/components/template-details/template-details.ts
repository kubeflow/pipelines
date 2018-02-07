import 'polymer/polymer.html';

import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';
import PageElement from '../../lib/page_element';
import Template from '../../lib/template';
import { customElement, property } from '../../decorators';

import './template-details.html';

@customElement
export default class TemplateDetails extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public template: Template | null = null;

  public async refresh(path: string) {
    if (path !== '') {
      const id = Number.parseInt(path);
      if (isNaN(id)) {
        Utils.log.error(`Bad template path: ${id}`);
        return;
      }
      this.template = await Apis.getTemplate(id);
    }
  }
}
