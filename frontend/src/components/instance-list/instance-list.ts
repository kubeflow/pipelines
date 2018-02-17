import 'polymer/polymer.html';

import { customElement, property } from '../../decorators';
import * as Apis from '../../lib/apis';
import { InstanceClickEvent, RouteEvent } from '../../lib/events';
import { Instance } from '../../lib/instance';
import { PageElement } from '../../lib/page_element';

import './instance-list.html';

@customElement
export class InstanceList extends Polymer.Element implements PageElement {

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
