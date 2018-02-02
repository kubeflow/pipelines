/// <reference path="../../../bower_components/polymer/types/polymer.d.ts" />
/// <reference path="../../../bower_components/polymer/types/lib/utils/templatize.d.ts" />
import { customElement } from '../../decorators';
import Template from '../../modules/template';
import * as Apis from '../../modules/apis';
import * as Utils from '../../modules/utils';
import PageElement from '../../modules/page_element';
import { RouteEvent } from '../../modules/events';

@customElement
export default class TemplateDetails extends Polymer.Element implements PageElement {

  public template: Template | null = null;

  static get properties() {
    return {
      template: { type: Object },
    };
  }

  public async refresh(path: string) {
    if (path !== '') {
      const id = Number.parseInt(path);
      if (id === NaN) {
        Utils.log.error(`Bad template path: ${id}`);
        return;
      }
      this.template = await Apis.getTemplate(id);
    }
  }

  protected _back() {
    this.dispatchEvent(new RouteEvent('/templates'));
  }
}
