import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';

import { customElement, property } from '../../decorators';
import { PackageClickEvent, RouteEvent } from '../../lib/events';
import { PageElement } from '../../lib/page_element';
import { PipelinePackage } from '../../lib/pipeline_package';

import './package-list.html';

@customElement('package-list')
export class PackageList extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public packages: PipelinePackage[] = [];

  public async refresh(_: string) {
    this.packages = await Apis.getPackages();
  }

  protected _navigate(ev: PackageClickEvent) {
    const index = ev.model.package.id;
    this.dispatchEvent(new RouteEvent(`/packages/details/${index}`));
  }
}
