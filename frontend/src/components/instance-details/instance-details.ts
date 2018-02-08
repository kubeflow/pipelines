import 'polymer/polymer.html';
import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';
import PageElement from '../../lib/page_element';
import { Instance } from '../../lib/instance';
import { customElement, property } from '../../decorators';

import './instance-details.html';

@customElement
export default class InstanceDetails extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public instance: Instance | null = null;

  public async refresh(path: string) {
    if (path !== '') {
      const id = Number.parseInt(path);
      if (isNaN(id)) {
        Utils.log.error(`Bad instance path: ${id}`);
        return;
      }
      this.instance = await Apis.getInstance(id);
    }
  }

  protected _paramsToArray(paramsObject: { [key: string]: string | number }) {
    return Utils.objectToArray(paramsObject);
  }
}
