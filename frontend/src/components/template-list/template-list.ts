/// <reference path="../../../bower_components/polymer/types/polymer.d.ts" />
import { customElement } from '../../decorators';
import Template from '../../modules/template';
import * as Apis from '../../modules/apis';
import PageElement from '../../modules/page_element';
import { DomRepeatMouseEvent, RouteEvent } from '../../modules/events';

@customElement
export default class TemplateList extends Polymer.Element implements PageElement {

  public templates: Template[] = [];

  static get properties() {
    return {
      templates: { type: Array },
    };
  }

  public async refresh(_: string) {
    this.templates = await Apis.getTemplates();
  }

  protected _navigate(ev: DomRepeatMouseEvent) {
    const index = ev.model.template.id;
    this.dispatchEvent(new RouteEvent(`/templates/view/${index}`));
  }
}
