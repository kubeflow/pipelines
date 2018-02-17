import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'polymer/polymer.html';

import { customElement, property } from '../../decorators';
import * as Apis from '../../lib/apis';
import { Instance } from '../../lib/instance';
import { PageElement } from '../../lib/page_element';
import * as Utils from '../../lib/utils';

import './instance-details.html';

@customElement
export class InstanceDetails extends Polymer.Element implements PageElement {

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
}
