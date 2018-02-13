import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';

import { customElement, property } from '../../decorators';
import { RouteEvent, TemplateClickEvent } from '../../lib/events';
import { PageElement } from '../../lib/page_element';
import { Template } from '../../lib/template';

import './template-list.html';

@customElement
export class TemplateList extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public templates: Template[] = [];

  public async refresh(_: string) {
    this.templates = await Apis.getTemplates();
  }

  protected _navigate(ev: TemplateClickEvent) {
    const index = ev.model.template.id;
    this.dispatchEvent(new RouteEvent(`/templates/details/${index}`));
  }
}
