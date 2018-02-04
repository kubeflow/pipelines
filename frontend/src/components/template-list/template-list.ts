import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import PageElement from '../../lib/page_element';
import Template from '../../lib/template';
import { TemplateClickEvent, RouteEvent } from '../../lib/events';
import { customElement, property } from '../../decorators';

import './template-list.html';

@customElement
export default class TemplateList extends Polymer.Element implements PageElement {

  @property({type: Array})
  public templates: Template[] = [];

  public async refresh(_: string) {
    this.templates = await Apis.getTemplates();
  }

  protected _navigate(ev: TemplateClickEvent) {
    const index = ev.model.template.id;
    this.dispatchEvent(new RouteEvent(`/templates/details/${index}`));
  }
}
