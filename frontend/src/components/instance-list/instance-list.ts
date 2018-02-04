import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import PageElement from '../../lib/page_element';
import { Instance } from '../../lib/instance';
import { InstanceClickEvent, RouteEvent } from '../../lib/events';
import { customElement, property } from '../../decorators';

import './instance-list.html';

@customElement
export default class InstanceList extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public instances: Instance[] = [];

  public async refresh(_: string) {
    this.instances = await Apis.getInstances();
  }

  protected _navigate(ev: InstanceClickEvent) {
    const index = ev.model.instance.id;
    this.dispatchEvent(new RouteEvent(`/instances/details/${index}`));
  }
}
